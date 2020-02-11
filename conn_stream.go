package main

import (
	"context"
	"net"
	"time"

	"github.com/ergongate/vince/buffers"
	"github.com/uber-go/tally"
	"go.uber.org/atomic"
)

type connConfig struct {
	readTimeout  time.Duration
	writeTimeout time.Duration
}

type proxyConnOpts struct {
	local, remote connConfig
	scope         tally.Scope
}

func histogramBucket() tally.Buckets {
	return tally.DefaultBuckets
}

func proxyConn(ctx context.Context, ts *metricsCollector, opts proxyConnOpts, local, remote net.Conn) error {
	watch := ts.tcp.duration.Start()
	defer func() {
		watch.Stop()
	}()
	var n int
	var err error
	var firstRemote, firstLocal atomic.Bool
	firstRemote.Store(true)
	firstLocal.Store(true)
	go func() {
		buf := buffers.GetSlice()
		defer func() {
			buffers.PutSlice(buf)
		}()
		buf = buf[:0]
		for {
			if ctx.Err() != nil {
				show(ctx, err)
			}
			// read local and write remote
			if opts.local.readTimeout != 0 {
				local.SetReadDeadline(time.Now().Add(opts.local.readTimeout))
			}
			n, err = local.Read(buf)
			if err != nil {
				return
			}
			if firstLocal.Load() {
				firstLocal.Store(false)
			}
			ts.tcp.local.bytesRead.RecordValue(float64(n))

			if opts.remote.writeTimeout != 0 {
				remote.SetWriteDeadline(time.Now().Add(opts.remote.writeTimeout))
			}
			n, err = remote.Write(buf[:n])
			if err != nil {
				return
			}
			if firstRemote.Load() {
				firstRemote.Store(false)
			}
			ts.tcp.upstream.bytesWritten.RecordValue(float64(n))
		}
	}()
	buf := buffers.GetSlice()
	defer func() {
		buffers.PutSlice(buf)
	}()
	for {
		if ctx.Err() != nil {
			show(ctx, err)
			return err
		}
		buf = buf[:0]
		//read remote and write local
		if opts.remote.readTimeout != 0 {
			remote.SetReadDeadline(time.Now().Add(opts.remote.readTimeout))
		}
		n, err = remote.Read(buf)
		if err != nil {
			show(ctx, err)
			return err
		}
		if firstRemote.Load() {
			firstRemote.Store(false)
		}
		ts.tcp.local.bytesWritten.RecordValue(float64(n))

		if opts.local.writeTimeout != 0 {
			local.SetWriteDeadline(time.Now().Add(opts.local.writeTimeout))
		}
		n, err = local.Write(buf[:n])
		if err != nil {
			show(ctx, err)
			return err
		}
		if firstLocal.Load() {
			firstLocal.Store(false)
		}
		ts.tcp.local.bytesWritten.RecordValue(float64(n))
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

func streamListener(ctx context.Context, mx *metricsCollector, ls net.Listener, srv stream) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		l, err := ls.Accept()
		if err != nil {
			return err
		}
		mx.tcp.conn.accepted.Inc(1)
		go streamConn(ctx, mx, l, srv)
	}
}

func streamConn(ctx context.Context, mx *metricsCollector, conn net.Conn, srv stream) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	remote, err := srv.upstream()
	if err != nil {
		return err
	}
	defer remote.Close()
	opts := srv.config()
	return proxyConn(ctx, mx, opts, conn, remote)
}

func show(ctx context.Context, err error) {
	if err != nil {
		// TODO log this error
	}
}

type streamServer struct {
	stream stream
	ctx    context.Context
	mx     *metricsCollector
	cancel func()
}

func (s *streamServer) init(ctx context.Context, sm stream, m *connManager) {
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.stream = sm
}

func (s *streamServer) Serve(ls net.Listener) error {
	return streamListener(s.ctx, s.mx, ls, s.stream)
}

func (s *streamServer) Close() error {
	s.cancel()
	<-s.ctx.Done() // make sure we didn't mess up context
	return nil
}
