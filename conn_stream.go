package main

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/uber-go/tally"
	"go.uber.org/atomic"
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
	scope         tally.Scope
}

type tcpAnalytics struct {
	local struct {
		bytesRead    tally.Histogram
		bytesWritten tally.Histogram
	}
	upstream struct {
		bytesWritten tally.Histogram
		bytesRead    tally.Histogram
	}
	duration tally.Histogram
}

func (t *tcpAnalytics) init(scope tally.Scope, local, remote net.Conn, m *connManager) {
	s := scope.Tagged(map[string]string{
		"local":    strconv.FormatInt(m.getID(local), 10),
		"upstream": strconv.FormatInt(m.getID(remote), 10),
	})
	t.local.bytesRead = s.Histogram("stream_local_bytes_read", histogramBucket())
	t.local.bytesWritten = s.Histogram("stream_local_bytes_written", histogramBucket())
	t.upstream.bytesRead = s.Histogram("stream_upstream_bytes_read", histogramBucket())
	t.upstream.bytesWritten = s.Histogram("stream_upstream_bytes_written", histogramBucket())
	t.duration = s.Histogram("stream_total_duration", tally.MustMakeLinearDurationBuckets(0, time.Millisecond, 60))
}

func histogramBucket() tally.Buckets {
	return tally.DefaultBuckets
}

func proxyConn(ctx context.Context, opts proxyConnOpts, local, remote net.Conn, m *connManager) error {
	var ts tcpAnalytics
	ts.init(opts.scope, local, remote, m)
	watch := ts.duration.Start()
	defer func() {
		watch.Stop()
	}()
	var n int
	var err error
	var firstRemote, firstLocal atomic.Bool
	firstRemote.Store(true)
	firstLocal.Store(true)
	go func() {
		buf := bytesPool.Get().([]byte)
		defer func() {
			bytesPool.Put(buf[:0])
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
				m.manageConnState(local, http.StateActive)
				firstLocal.Store(false)
			}
			ts.local.bytesRead.RecordValue(float64(n))

			if opts.remote.writeTimeout != 0 {
				remote.SetWriteDeadline(time.Now().Add(opts.remote.writeTimeout))
			}
			n, err = remote.Write(buf[:n])
			if err != nil {
				return
			}
			if firstRemote.Load() {
				m.manageConnState(remote, http.StateActive)
				firstRemote.Store(false)
			}
			ts.upstream.bytesWritten.RecordValue(float64(n))
		}
	}()
	buf := bytesPool.Get().([]byte)
	defer func() {
		bytesPool.Put(buf[:0])
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
			m.manageConnState(remote, http.StateActive)
			firstRemote.Store(false)
		}
		ts.local.bytesWritten.RecordValue(float64(n))

		if opts.local.writeTimeout != 0 {
			local.SetWriteDeadline(time.Now().Add(opts.local.writeTimeout))
		}
		n, err = local.Write(buf[:n])
		if err != nil {
			show(ctx, err)
			return err
		}
		if firstLocal.Load() {
			m.manageConnState(local, http.StateActive)
			firstLocal.Store(false)
		}
		ts.local.bytesWritten.RecordValue(float64(n))
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

func streamListener(ctx context.Context, ls net.Listener, srv stream, m *connManager) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		baseCtx := m.baseCtx(ctx, ls)

		l, err := ls.Accept()
		if err != nil {
			return err
		}
		ctx = m.connContext(baseCtx, l)
		m.manageConnState(l, http.StateNew)
		go streamConn(ctx, l, srv, m)
	}
}

func streamConn(ctx context.Context, conn net.Conn, srv stream, m *connManager) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	remote, err := srv.upstream()
	if err != nil {
		return err
	}
	m.manageConnState(remote, http.StateNew)
	opts := srv.config()
	defer func() {
		m.manageConnState(conn, http.StateClosed)
		m.manageConnState(remote, http.StateClosed)
	}()
	return proxyConn(ctx, opts, conn, remote, m)
}

func show(ctx context.Context, err error) {
	if err != nil {
		// TODO log this error
	}
}

type streamServer struct {
	stream stream
	m      *connManager
	ctx    context.Context
	cancel func()
}

func (s *streamServer) init(ctx context.Context, sm stream, m *connManager) {
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.m = m
	s.stream = sm
}

func (s *streamServer) Serve(ls net.Listener) error {
	return streamListener(s.ctx, ls, s.stream, s.m)
}

func (s *streamServer) Close() error {
	s.cancel()
	<-s.ctx.Done() // make sure we didn't mess up context
	return nil
}
