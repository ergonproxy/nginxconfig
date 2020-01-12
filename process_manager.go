package main

import (
	"context"
	"io"
	"runtime"
	"sync"

	"github.com/sourcegraph/jsonrpc2"
)

type processManager struct {
	m          childManager
	childCount int
	rpcHandle  jsonrpc2.Handler
	rpcOpts    []jsonrpc2.ConnOpt
	process    *sync.Map
}

func newProcessManager(m childManager, count int, hand jsonrpc2.Handler, opts ...jsonrpc2.ConnOpt) *processManager {
	if count == 0 {
		count = runtime.NumCPU()
	}
	return &processManager{m: m, childCount: count, rpcHandle: hand, rpcOpts: opts, process: new(sync.Map)}
}

func (p *processManager) start(ctx context.Context) error {
	for i := 0; i < p.childCount; i++ {
		ch, err := p.m.Create(ctx)
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
	p.process.Store(ch.Pid(), h)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-h.DisconnectNotify():
			return nil
		}
	}
}

type childManager interface {
	Create(context.Context) (childProcess, error)
}

type childProcess interface {
	io.Reader // reads stdout of the child process
	io.Writer // writes to stdin of the child process
	Start(context.Context) error
	Close() error
	Pid() uint64
}
