package main

import (
	"errors"
	"net"
	"time"

	"go.uber.org/atomic"
)

type pool struct {
	size     uint64
	conns    chan net.Conn
	dial     func() (net.Conn, error)
	created  atomic.Uint64
	duration time.Duration
}

func (p *pool) init(size int,
	duration time.Duration,
	dial func() (net.Conn, error)) {
	p.conns = make(chan net.Conn, size)
	p.dial = dial
	p.duration = duration
}

func (p *pool) take() (net.Conn, error) {
	select {
	case conn := <-p.conns:
		return conn, nil
	default:
		if p.created.Load() < p.size {
			p.created.Inc()
			return p.dial()
		}
		return p.wait(p.duration)
	}
}

func (p *pool) wait(duration time.Duration) (net.Conn, error) {
	t := time.NewTimer(duration)
	defer t.Stop()
	select {
	case <-t.C:
		return nil, errors.New("Timeout")
	case conn := <-p.conns:
		return conn, nil
	}
}

func (p *pool) put(conn net.Conn) {
	p.conns <- conn
}
