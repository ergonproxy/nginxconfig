package main

import (
	"context"
	"errors"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/sourcegraph/jsonrpc2"
)

type processManager struct {
	m                   childManager
	onSignal            func(os.Signal) error
	childProcessOptions processOptions
	childCount          int
	rpcHandle           jsonrpc2.Handler
	rpcOpts             []jsonrpc2.ConnOpt
	process             *sync.Map
}

func newProcessManager(m childManager, count int, hand jsonrpc2.Handler, opts ...jsonrpc2.ConnOpt) *processManager {
	if count == 0 {
		count = runtime.NumCPU()
	}
	return &processManager{m: m, childCount: count, rpcHandle: hand, rpcOpts: opts, process: new(sync.Map)}
}

func (p *processManager) start(ctx context.Context) error {
	for i := 0; i < p.childCount; i++ {
		ch, err := p.m.Create(ctx, p.childProcessOptions)
		if err != nil {
			return err
		}
		go p.startProcess(ctx, ch)
	}
	return nil
}

func (p *processManager) startProcess(ctx context.Context, ch childProcess) error {
	if err := ch.Start(ctx); err != nil {
		return err
	}
	stream := jsonrpc2.NewBufferedStream(ch, jsonrpc2.VSCodeObjectCodec{})
	h := jsonrpc2.NewConn(ctx, stream, p.rpcHandle, p.rpcOpts...)
	p.process.Store(ch.Pid(), &childConn{rpc: h, ch: ch})
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-h.DisconnectNotify():
			return nil
		}
	}
}

func (p *processManager) kill() error {
	var errs []string
	p.process.Range(func(key, value interface{}) bool {
		v := value.(*childConn)
		// close rpc server before we kill the child
		if err := v.rpc.Close(); err != nil {
			errs = append(errs, err.Error())
		}

		if err := v.ch.Kill(); err != nil {
			errs = append(errs, err.Error())
		}
		return true
	})
	if errs != nil {
		return errors.New(strings.Join(errs, ","))
	}
	return nil
}
func (p *processManager) Signal(sig os.Signal) error {
	var errs []string
	p.process.Range(func(key, value interface{}) bool {
		v := value.(*childConn)
		if err := v.ch.Signal(sig); err != nil {
			errs = append(errs, err.Error())
		}
		return true
	})
	if p.onSignal != nil {
		if err := p.onSignal(sig); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if errs != nil {
		return errors.New(strings.Join(errs, ","))
	}
	return nil
}

type childConn struct {
	rpc *jsonrpc2.Conn
	ch  childProcess
}

type processOptions struct {
	Path       string
	Env        []string
	Dir        string
	ExtraFiles []*os.File
}

type childManager interface {
	Create(context.Context, processOptions) (childProcess, error)
}

type childProcess interface {
	io.Reader // reads stdout of the child process
	io.Writer // writes to stdin of the child process
	Start(context.Context) error
	Signal(os.Signal) error
	Kill() error
	Close() error
	Pid() int
}
