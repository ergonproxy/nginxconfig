package main

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
)

var _ childProcess = (*child)(nil)

type child struct {
	cmd     *exec.Cmd
	inPipe  io.WriteCloser
	outPipe io.ReadCloser
}

func (ch *child) Start(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	in, err := ch.cmd.StdinPipe()
	if err != nil {
		return err
	}
	ch.inPipe = in
	out, err := ch.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	ch.outPipe = out
	return ch.cmd.Start()
}

// Close this only closes stdin/stout pipes it doesn't kill the child process
// you should use Kill for that.
func (ch *child) Close() error {
	var errs []string
	if err := ch.inPipe.Close(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := ch.outPipe.Close(); err != nil {
		errs = append(errs, err.Error())
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, ","))
	}
	return nil
}

func (ch *child) Kill() error {
	return ch.cmd.Process.Kill()
}

func (ch *child) Signal(sig os.Signal) error {
	return ch.cmd.Process.Signal(sig)
}

func (ch *child) Pid() int {
	return ch.cmd.Process.Pid
}

func (ch *child) Read(p []byte) (int, error) {
	return ch.outPipe.Read(p)
}

func (ch *child) Write(p []byte) (int, error) {
	return ch.inPipe.Write(p)
}
