package main

import (
	"context"
	"net"
	"sync"
	"time"
)

var bytesPool = &sync.Pool{
	New: func() interface{} {
		return make([]byte, 1024)
	},
}

type connConfig struct {
	readTimeout  time.Duration
	writeTimeout time.Duration
}

type proxyConnOpts struct {
	local, remote connConfig
	stats         connStats
}

type connStatus uint

const (
	statusConnLocalOpen = 1 + iota
	statusConnRemoteOpen
	statusConnLocalClosed
	statusConnRemoteClosed
)

type connStats interface {
	localBytesRead(int)
	remoteBytesRead(int)
	duration(time.Duration)
	status(connStatus)
	done()
}

func proxyConn(ctx context.Context, opts proxyConnOpts, local, remote net.Conn) error {
	buf := bytesPool.Get().([]byte)
	now := time.Now()
	defer func() {
		bytesPool.Put(buf[:0])
		if opts.stats != nil {
			opts.stats.duration(time.Since(now))
			opts.stats.done()
		}
	}()
	var n int
	var err error
	for {
		if ctx.Err() != nil {
			show(ctx, err)
			return err
		}
		buf = buf[:0]
		n, err = remote.Read(buf)
		if err != nil {
			return err
		}
		if opts.stats != nil {
			opts.stats.remoteBytesRead(n)
		}
		n, err = local.Write(buf[:n])
		if err != nil {
			show(ctx, err)
			return err
		}
		if opts.stats != nil {
			opts.stats.localBytesRead(n)
		}
	}
}

func configConn(conn net.Conn, opts connConfig) error {
	if opts.readTimeout != 0 {
		if err := conn.SetReadDeadline(time.Now().Add(opts.readTimeout)); err != nil {
			return err
		}
	}
	if opts.writeTimeout != 0 {
		if err := conn.SetWriteDeadline(time.Now().Add(opts.writeTimeout)); err != nil {
			return err
		}
	}
	return nil
}

type stream interface {
	upstream() (net.Conn, error)
	config() proxyConnOpts
}

func streamListener(ctx context.Context, ls net.Listener, srv stream) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		l, err := ls.Accept()
		if err != nil {
			return err
		}
		go streamConn(ctx, l, srv)
	}
}

func streamConn(ctx context.Context, conn net.Conn, srv stream) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	remote, err := srv.upstream()
	if err != nil {
		return err
	}
	opts := srv.config()
	if opts.stats != nil {
		opts.stats.status(statusConnLocalOpen)
		opts.stats.status(statusConnRemoteOpen)
	}
	defer func() {
		show(ctx, conn.Close())
		show(ctx, remote.Close())
		if opts.stats != nil {
			opts.stats.status(statusConnLocalOpen)
			opts.stats.status(statusConnRemoteOpen)
		}
	}()
	return proxyConn(ctx, opts, conn, remote)
}

func show(ctx context.Context, err error) {
	if err != nil {
		// TODO log this error
	}
}
