package engine

import (
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

// Process defines nginx process. Main process spans child processes.
type Process struct {
	main       bool
	pid        int
	pidFile    string
	ppid       int
	env        []string
	binary     string
	extraFiles []*os.File
	sockets    []net.Listener
	children   []*exec.Cmd
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

// encodes os.File to a string that can be passed in environment
// variable
func fileToEnv(f *os.File) string {
	return fmt.Sprintf("%d|%s", f.Fd(), f.Name())
}

// ErrInvalidFile is returned when wron format of encoded file description.
var ErrInvalidFile = errors.New("invalid file description")

func envToFile(s string) (*os.File, error) {
	p := strings.Split(s, "|")
	if len(p) != 2 {
		return nil, ErrInvalidFile
	}
	fd, err := strconv.ParseUint(p[0], 10, 64)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), p[1]), nil
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
	childEnv := ChildProcess + "=true"
	for i := 0; i < runtime.NumCPU(); i++ {
		cmd := exec.Command(p.binary)
		cmd.ExtraFiles = p.extraFiles
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = append(p.env, childEnv)
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
func (p *Process) Run() error {
	p.Info()
	if err := p.WritePID(); err != nil {
		return err
	}
	if err := p.StartChildren(); err != nil {
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
			fmt.Println("fast shutdown")
		case syscall.SIGQUIT:
			fmt.Println("graceful shutdown")
		case syscall.SIGHUP:
			fmt.Println("reload configuration")
		case syscall.SIGUSR1:
			fmt.Println("reopening log files")
		case syscall.SIGUSR2:
			fmt.Println("upgrading an executable files")
		case syscall.SIGWINCH:
			fmt.Println("graceful shutdown of worker process")
		default:
			fmt.Println("received unknown signal ", sig.String())
		}
		return p.Close()
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
	var e errorList
	for _, ch := range p.children {
		if err := ch.Process.Kill(); err != nil {
			e = append(e, err.Error())
		}
	}
	for _, ls := range p.sockets {
		if err := ls.Close(); err != nil {
			e = append(e, err.Error())
		}
	}
	return e
}

func Run() error {
	return New().Run()
}
