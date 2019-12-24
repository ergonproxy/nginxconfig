package engine

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
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

type Server interface {
	Serve(net.Listener) error
	Shutdown(context.Context) error
	Close() error
}

// Process defines nginx process. Main process spans child processes.
type Process struct {
	main        bool
	pid         int
	pidFile     string
	ppid        int
	env         []string
	binary      string
	sockets     []net.Listener
	socketFiles []*os.File
	children    []*exec.Cmd
	servers     []Server
	closed      bool
}

func New() *Process {
	return &Process{
		main:    !IsChild(),
		pid:     os.Getpid(),
		pidFile: PIDFile,
		ppid:    os.Getppid(),
		binary:  os.Args[0],
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
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = append(p.env, env...)
		if err := cmd.Start(); err != nil {
			return err
		}
		p.children = append(p.children, cmd)
	}
	return nil
}

func (p *Process) Info() {
	m := map[string]interface{}{
		"main": p.main,
		"pid":  p.pid,
		"ppid": p.ppid,
	}
	fmt.Printf("%v\n", m)
}

// Run starts children process if this is the main process and listens for
// control signals.
func (p *Process) Run() (err error) {
	defer func() {
		err = p.releaseResources()
	}()
	p.Info()
	if err := p.WritePID(); err != nil {
		return err
	}
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
	if err := p.StartChildren(); err != nil {
		return err
	}
	if !p.main {
		for _, ls := range p.sockets {
			srv := defaultServer()
			p.servers = append(p.servers, srv)
			fmt.Printf("[%d] starting serving at %s\n", p.pid, ls.Addr().String())
			go srv.Serve(ls)
		}
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
			fmt.Println("fast shutdown")
			return p.FastClose()
		case syscall.SIGQUIT:
			fmt.Println("graceful shutdown")
			return p.Graceful()
		case syscall.SIGHUP:
			fmt.Println("reload configuration")
		case syscall.SIGUSR1:
			fmt.Println("reopening log files")
		case syscall.SIGUSR2:
			fmt.Println("upgrading an executable files")
		case syscall.SIGWINCH:
			fmt.Println("graceful shutdown of worker process")
			if p.main {
				// we send the signal to worker process
				var e errorList
				for _, ch := range p.children {
					if err := ch.Process.Signal(syscall.SIGUSR2); err != nil {
						e = append(e, err.Error())
					}
				}
				if e != nil {
					fmt.Println(e)
				}
				// we are not exiting this loop since the main process remains operational.
			} else {
				return p.Graceful()
			}
		default:
			fmt.Println("received unknown signal ", sig.String())
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
func (p *Process) Close() error {
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
	if err := p.releaseResources(); err != nil {
		e = append(e, err.Error())
	}
	return e
}

func (p *Process) FastClose() error {
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
	if err := p.releaseResources(); err != nil {
		e = append(e, err.Error())
	}
	return e
}

func (p *Process) Graceful() error {
	ctx := context.Background()
	if p.main {
		// There is nothing that is served from this main process appart from cache
		// and configuration.
		return p.Close()
	}
	var e errorList
	for _, s := range p.servers {
		if err := s.Shutdown(ctx); err != nil {
			e = append(e, err.Error())
		}
	}
	if err := p.releaseResources(); err != nil {
		e = append(e, err.Error())
	}
	return e
}

func (p *Process) releaseResources() error {
	if p.closed {
		return nil
	}
	var e errorList

	for _, ls := range p.sockets {
		if err := ls.Close(); err != nil {
			e = append(e, err.Error())
		}
	}
	for _, ls := range p.socketFiles {
		if err := ls.Close(); err != nil {
			e = append(e, err.Error())
		}
	}
	p.closed = true
	return e
}

// Run
func Run() error {
	runFlags()
	return New().Run()
}
