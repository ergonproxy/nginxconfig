package engine

import (
	"context"
	"io"
	"os"
	"os/exec"
)

func NewCommand(ctx context.Context, name string, args ...string) *Command {
	return &Command{name: name, args: args, ctx: ctx}
}

type Command struct {
	stdout        io.WriteCloser
	stdin         io.Reader
	ShouldRestart func(*os.ProcessState) bool
	ExtraFiles    []*os.File
	Env           []string
	Stderr        io.Writer
	exec          *exec.Cmd
	ctx           context.Context
	name          string
	args          []string
	err           error
	reader        func() (Msg, error)
	writer        func(Msg) error
	logWriter     func(io.Writer) error
}

func (cmd *Command) Run() error {
	if cmd.err != nil {
		return cmd.err
	}
	exe := exec.CommandContext(cmd.ctx, cmd.name, cmd.args...)
	exe.ExtraFiles = cmd.ExtraFiles
	exe.Env = cmd.Env
	cmd.exec = exe
	in, err := exe.StdinPipe()
	if err != nil {
		return err
	}
	out, err := exe.StdoutPipe()
	if err != nil {
		return err
	}
	ioerr, err := exe.StderrPipe()
	if err != nil {
		return err
	}
	cmd.reader = outReader(cmd.ctx, out)
	cmd.writer = inWriter(cmd.ctx, in)
	cmd.logWriter = ioErrWriter(cmd.ctx, ioerr)
	err = cmd.exec.Start()
	if err != nil {
		return err
	}
	go cmd.wait(cmd.exec)
	return nil
}

func (cmd *Command) wait(exe *exec.Cmd) {
	err := exe.Wait()
	if err != nil {
		if cmd.ShouldRestart != nil {
			if cmd.ShouldRestart(cmd.exec.ProcessState) {
				if err := cmd.restart(); err != nil {
					cmd.err = err
				}
			}
		}
	}
}

func (cmd *Command) restart() error {
	if cmd.ctx.Err() != nil {
		return cmd.ctx.Err()
	}
	return cmd.Run()
}

func (cmd *Command) ReadMSG() (Msg, error) {
	return cmd.reader()
}

func (cmd *Command) WriteMSG(msg Msg) error {
	return cmd.writer(msg)
}

func (cmd *Command) WriteLogs(to io.Writer) error {
	return cmd.logWriter(to)
}

func (cmd *Command) Signal(sig os.Signal) error {
	return cmd.exec.Process.Signal(sig)
}

func (cmd *Command) PID() int {
	return cmd.exec.Process.Pid
}

func (cmd *Command) Kill() error {
	return cmd.exec.Process.Kill()
}
