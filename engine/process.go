package engine

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"go.uber.org/zap"
)

const (
	// ChildProcess environment vairble used to signal that the process is a child
	// process.
	ChildProcess = "VINCE_CHILD_PROCESS"
	// FileDescriptions env var for passing open filedescriptors to child process.
	// The descriptors are for sockets.
	FileDescriptions = "VINCE_FD"
	// PIDFile is the default name for the nginx pid file.
	PIDFile = "vince.pid"
)

// IsChild returns true if the binary was invoked as a worker/child process
func IsChild() bool {
	return os.Getenv(ChildProcess) == "true"
}

type errorList []string

func (e errorList) Error() string {
	return strings.Join([]string(e), "\n")
}

type Stats struct {
	Open, Idle, Active, Hijacked int64
}

type Server interface {
	Serve(net.Listener) error
	Shutdown(context.Context) error
	Close() error
}

// Process defines nginx process. Main process spans child processes.
type Process struct {
	main           bool
	pid            int
	pidFile        string
	ppid           int
	env            []string
	binary         string
	sockets        []net.Listener
	socketFiles    []*os.File
	children       []*exec.Cmd
	servers        []Server
	closed         bool
	average        Stats
	stats          Stats
	heartBeat      time.Duration
	statColllector struct {
		readers []func() (Msg, error)
		writers []func(Msg) error
	}
	connManager *ConnManager
}

func New() *Process {
	return &Process{
		main:        !IsChild(),
		pid:         os.Getpid(),
		pidFile:     PIDFile,
		ppid:        os.Getppid(),
		binary:      os.Args[0],
		heartBeat:   3 * time.Second,
		connManager: NewConnManager(),
	}
}

type fileD struct {
	desc uintptr
	name string
}

func (f fileD) String() string {
	return fmt.Sprint(f.desc) + "|" + f.name
}

func (f fileD) File() *os.File {
	return os.NewFile(f.desc, f.name)
}

func parseFD(s string) ([]*fileD, error) {
	parts := strings.Split(s, ",")
	var o []*fileD
	for _, v := range parts {
		f, err := envToFile(v)
		if err != nil {
			return nil, err
		}
		o = append(o, f)
	}
	return o, nil
}

func socketToFile(ls net.Listener) (*os.File, error) {
	switch e := ls.(type) {
	case *net.TCPListener:
		return e.File()
	case *net.UnixListener:
		return e.File()
	default:
		return nil, errors.New("unknown listener")
	}
}

// encodes os.File to a string that can be passed in environment
// variable
func fileToEnv(f *os.File) string {
	return (fileD{desc: f.Fd(), name: f.Name()}).String()
}

// ErrInvalidFile is returned when wron format of encoded file description.
var ErrInvalidFile = errors.New("invalid file description")

func envToFile(s string) (*fileD, error) {
	p := strings.Split(s, "|")
	if len(p) != 2 {
		return nil, ErrInvalidFile
	}
	var fd fileD
	fmt.Sscan(p[0], &fd.desc)
	fd.name = p[1]
	return &fd, nil
}

func (p *Process) genChildEnv() []string {
	var env []string
	childEnv := ChildProcess + "=true"
	var fdEnv []string

	for i, fd := range p.socketFiles {
		fdEnv = append(fdEnv, (fileD{desc: uintptr(3 + i), name: fd.Name()}).String())
	}
	env = append(env, childEnv)
	if fdEnv != nil {
		env = append(env, FileDescriptions+"="+strings.Join(fdEnv, ","))
	}
	return env
}

// StartChildren starts child worker process inheriting file descriptors of
// sockets that are held by p.
//
// TODO:
// Start the child process with unprivileged user with no read/write access to
// system resources except the incoming requests.
func (p *Process) StartChildren() error {
	if !p.main {
		// NoOp when this is not a main process
		return nil
	}
	env := p.genChildEnv()
	for i := 0; i < runtime.NumCPU(); i++ {
		cmd := exec.Command(p.binary)
		cmd.ExtraFiles = p.socketFiles
		cmd.Stderr = os.Stderr
		cmd.Env = append(p.env, env...)
		in, err := cmd.StdinPipe()
		if err != nil {
			return err
		}
		out, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		p.statColllector.readers = append(p.statColllector.readers, outReader(out))
		p.statColllector.writers = append(p.statColllector.writers, inWriter(in))
		if err := cmd.Start(); err != nil {
			return err
		}
		p.children = append(p.children, cmd)
	}
	return nil
}

// Run starts children process if this is the main process and listens for
// control signals.
func (p *Process) Run(ctx context.Context) error {
	defer func() {
		p.releaseResources(ctx)
	}()
	lg := log(ctx)
	if p.main {
		lg.Info("Start main process", zap.Int("pid", p.pid))
	} else {
		lg.Info("Start child process", zap.Int("pid", p.pid))
	}
	if err := p.WritePID(); err != nil {
		return err
	}

	if err := p.manageSockets(ctx); err != nil {
		return err
	}

	if err := p.StartChildren(); err != nil {
		return err
	}

	// start servers that listen on registered sockets
	if err := p.manageServers(ctx); err != nil {
		return err
	}

	// start stats collection/publishing loop
	if err := p.manageStats(ctx); err != nil {
		return err
	}
	ch := make(chan os.Signal, 2)
	signal.Notify(
		ch,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGHUP,
		syscall.SIGQUIT,
		syscall.SIGUSR1,
		syscall.SIGUSR2,
		syscall.SIGWINCH,
	)
	for {
		sig := <-ch
		switch sig {
		case syscall.SIGTERM, syscall.SIGINT:
			lg.Debug("fast shutdown")
			return p.FastClose(ctx)
		case syscall.SIGQUIT:
			lg.Debug("graceful shutdown")
			return p.Graceful(ctx)
		case syscall.SIGHUP:
			lg.Debug("reload configuration")
		case syscall.SIGUSR1:
			lg.Debug("reopening log files")
		case syscall.SIGUSR2:
			lg.Debug("upgrading an executable files")
		case syscall.SIGWINCH:
			lg.Debug("graceful shutdown of worker process")
			if p.main {
				// we send the signal to worker process
				for _, ch := range p.children {
					if err := ch.Process.Signal(syscall.SIGUSR2); err != nil {
						lg.Error("Sending signal to child process",
							zap.String("signal", sig.String()),
							zap.Int("pid", ch.Process.Pid),
							zap.Error(err),
						)
					}
				}
				// we are not exiting this loop since the main process remains operational.
			} else {
				return p.Graceful(ctx)
			}
		default:
			lg.Debug("received unknown signal ", zap.String("signal", sig.String()))
		}
	}
}

// WritePID creates apid file if this is a main process. If there is already a
// pid file then it it will be renamed with .old extension and then overwritten
// with the new value.
func (p *Process) WritePID() error {
	if !p.main {
		return nil
	}
	pid := strconv.FormatInt(int64(p.pid), 10)
	if _, err := os.Stat(p.pidFile); err == nil {
		err = os.Rename(p.pidFile, p.pidFile+".old")
		if err != nil {
			return err
		}
	}
	return ioutil.WriteFile(p.pidFile, []byte(pid), 0600)
}

// Close sends kill signal to all child process and exits
func (p *Process) Close(ctx context.Context) error {
	if p.closed {
		return nil
	}
	var e errorList
	if p.main {
		for _, ch := range p.children {
			if err := ch.Process.Kill(); err != nil {
				e = append(e, err.Error())
			}
		}
	}
	if err := p.releaseResources(ctx); err != nil {
		e = append(e, err.Error())
	}
	return e
}

func (p *Process) FastClose(ctx context.Context) error {
	if p.closed {
		return nil
	}
	var e errorList
	if p.main {
		for _, ch := range p.children {
			if err := ch.Process.Kill(); err != nil {
				e = append(e, err.Error())
			}
		}
	}
	if err := p.releaseResources(ctx); err != nil {
		e = append(e, err.Error())
	}
	return e
}

func (p *Process) Graceful(ctx context.Context) error {
	if p.main {
		// There is nothing that is served from this main process appart from cache
		// and configuration.
		return p.Close(ctx)
	}
	var e errorList
	for _, s := range p.servers {
		if err := s.Shutdown(ctx); err != nil {
			e = append(e, err.Error())
		}
	}
	if err := p.releaseResources(ctx); err != nil {
		e = append(e, err.Error())
	}
	return e
}

func (p *Process) releaseResources(ctx context.Context) error {
	if p.closed {
		return nil
	}
	var e errorList
	lg := log(ctx)
	lg.Debug("Releasing sockets",
		zap.Int("pid", p.pid),
		zap.Int("ppid", p.ppid),
		zap.Int("sockets", len(p.sockets)),
	)
	for _, ls := range p.sockets {
		if err := ls.Close(); err != nil {
			lg.Error("Failed to close socket", zap.Error(err))
			e = append(e, err.Error())
		}
	}
	lg.Debug("Releasing socket files",
		zap.Int("pid", p.pid),
		zap.Int("ppid", p.ppid),
		zap.Int("sockets", len(p.socketFiles)),
	)
	for _, ls := range p.socketFiles {
		if err := ls.Close(); err != nil {
			lg.Error("Failed to close socket file", zap.Error(err))
			e = append(e, err.Error())
		}
	}
	p.closed = true
	return e
}

func Run() error {
	return runDaemon(run)
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lg, err := newLogger()
	if err != nil {
		return err
	}
	defer lg.Sync()
	ctx = withLog(ctx, lg)
	runFlags(ctx)
	err = New().Run(ctx)
	if err != nil {
		lg.Error("Vince", zap.Error(err))
	}
	return err
}

type Msg struct {
	PID  int
	Body Stats
}

func (p *Process) manageChildStats(ctx context.Context, in io.ReadCloser, out io.WriteCloser) error {
	lg := log(ctx)
	dec := json.NewDecoder(bufio.NewReader(in))
	enc := json.NewEncoder(out)
	tick := time.NewTicker(p.heartBeat)
	defer tick.Stop()
	var m Msg
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			err := dec.Decode(&m)
			if err != nil {
				if err != io.EOF {
					lg.Error("Decoding message from input", zap.Error(err))
				}
			} else {
				p.average = m.Body
			}
			p.syncConnStats()
			err = enc.Encode(Msg{PID: p.pid, Body: p.stats})
			if err != nil {
				lg.Error("Writing stats", zap.Error(err))
			}
		}
	}
}

func (p *Process) manageSockets(ctx context.Context) error {
	if p.main {
		ls, err := net.Listen("tcp", ":8090")
		if err != nil {
			return err
		}
		p.sockets = append(p.sockets, ls)
		f, err := socketToFile(ls)
		if err != nil {
			return err
		}
		p.socketFiles = append(p.socketFiles, f)
	} else {
		fds, err := parseFD(os.Getenv(FileDescriptions))
		if err != nil {
			return err
		}
		for _, fd := range fds {
			f := fd.File()
			p.socketFiles = append(p.socketFiles, f)
			ls, err := net.FileListener(f)
			if err != nil {
				return err
			}
			p.sockets = append(p.sockets, ls)
		}
	}
	return nil
}

func (p *Process) manageServers(ctx context.Context) error {
	if !p.main {
		lg := log(ctx)
		for _, ls := range p.sockets {
			srv := defaultServer()
			srv.ConnState = p.connManager.Manage
			srv.ConnContext = func(ctx context.Context, _ net.Conn) context.Context {
				return withLog(ctx, lg)
			}
			p.servers = append(p.servers, srv)
			lg.Debug("Start HTTP Server", zap.String("address", ls.Addr().String()))
			go srv.Serve(ls)
		}
	}
	return nil
}

func (p *Process) syncConnStats() {
	atomic.StoreInt64(&p.stats.Open, atomic.LoadInt64(&p.connManager.open))
	atomic.StoreInt64(&p.stats.Idle, atomic.LoadInt64(&p.connManager.idle))
	atomic.StoreInt64(&p.stats.Active, atomic.LoadInt64(&p.connManager.active))
	atomic.StoreInt64(&p.stats.Hijacked, atomic.LoadInt64(&p.connManager.hijacked))
}

func (p *Process) manageStats(ctx context.Context) error {
	if p.main {
		return p.manageMainStats(ctx)
	}
	return p.manageChildStats(ctx, os.Stdin, os.Stdout)
}

func (p *Process) manageMainStats(ctx context.Context) error {
	if !p.main {
		return nil
	}
	go p.mainStatsLoop(ctx)
	return nil
}

func (p *Process) mainStatsLoop(ctx context.Context) {
	tick := time.NewTicker(p.heartBeat)
	defer tick.Stop()
	lg := log(ctx)
	msgs := make([]Msg, len(p.children))

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			m := Msg{PID: p.pid, Body: p.average}
			for _, w := range p.statColllector.writers {
				if err := w(m); err != nil {
					lg.Error("Writing ", zap.Error(err))
				}
			}
			for i, r := range p.statColllector.readers {
				v, err := r()
				if err != nil {
					if err != io.EOF {
						lg.Error("Readeing ", zap.Error(err))
					}
				} else {
					lg.Debug("Received stats from", zap.Int("pid", v.PID))
					msgs[i] = v
				}
			}
			var total Stats
			for i := 0; i < len(msgs); i++ {
				total.Open += msgs[i].Body.Open
				total.Idle += msgs[i].Body.Idle
				total.Active += msgs[i].Body.Active
				total.Hijacked += msgs[i].Body.Hijacked
			}
			lg.Sugar().Infof("%#v\n", total)
		}
	}
}

func outReader(in io.Reader) func() (Msg, error) {
	dec := json.NewDecoder(bufio.NewReader(in))
	return func() (Msg, error) {
		var m Msg
		err := dec.Decode(&m)
		return m, err
	}
}

func inWriter(out io.Writer) func(Msg) error {
	enc := json.NewEncoder(out)
	return func(m Msg) error {
		return enc.Encode(m)
	}
}
