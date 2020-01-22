package main

import (
	"context"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
)

type (
	vinceConfigKey struct{}
	connManagerKey struct{}
)

type httpConnManager struct {
	conns  *sync.Map
	status httpConnStatus
	serial func() int64
}

type connInfo struct {
	id    int64
	state http.ConnState
}

type httpConnStatus struct {
	open, active, idle, hijacked int64
}

func (c *httpConnStatus) Add(othe *httpConnStatus) {
	atomic.AddInt64(&c.open, othe.open)
	atomic.AddInt64(&c.active, othe.active)
	atomic.AddInt64(&c.idle, othe.idle)
	atomic.AddInt64(&c.hijacked, othe.hijacked)
}

func (c *httpConnStatus) Set(othe *httpConnStatus) {
	atomic.StoreInt64(&c.open, othe.open)
	atomic.StoreInt64(&c.active, othe.active)
	atomic.StoreInt64(&c.idle, othe.idle)
	atomic.StoreInt64(&c.hijacked, othe.hijacked)
}

func (c *httpConnStatus) Reset() httpConnStatus {
	return httpConnStatus{
		open:     atomic.SwapInt64(&c.open, 0),
		active:   atomic.SwapInt64(&c.active, 0),
		idle:     atomic.SwapInt64(&c.idle, 0),
		hijacked: atomic.SwapInt64(&c.hijacked, 0),
	}
}

func newHTTPConnManager(serial func() int64) *httpConnManager {
	return &httpConnManager{
		conns:  new(sync.Map),
		serial: serial,
	}
}

func (m *httpConnManager) manageConnState(conn net.Conn, state http.ConnState) {
	switch state {
	case http.StateNew:
		m.changeState(conn, func(i *connInfo) {
			i.state = state
		})
		m.inc(state)
	case http.StateActive:
		m.changeState(conn, func(i *connInfo) {
			if i.state != http.StateNew {
				// When we transition from new , we still mark the connection as open but in
				// active state, so here we have two records of the same connection since
				// the connection is both open and active.
				// This means we don't reduce open connections.
				m.dec(i.state)
			}
			i.state = state
		})
		m.inc(state)
	case http.StateIdle:
		m.changeState(conn, func(i *connInfo) {
			m.dec(i.state)
			i.state = state
		})
		m.inc(state)
	case http.StateHijacked:
		m.changeState(conn, func(i *connInfo) {
			m.dec(i.state)
			i.state = state
		})
		m.inc(state)
	case http.StateClosed:
		m.changeState(conn, func(i *connInfo) {
			m.dec(i.state)
			if i.state != http.StateNew {
				m.dec(http.StateNew)
			}
			i.state = state
		})
		m.conns.Delete(conn)
	}
}

// Close update connection state to closed. This is mainly used with connections
// that have StateHijacked its a noop for other states.
func (m *httpConnManager) Close(conn net.Conn) {
	if v, ok := m.conns.Load(conn); ok {
		s := v.(*connInfo)
		if s.state == http.StateHijacked {
			m.dec(s.state)
			m.dec(http.StateNew)
			m.conns.Delete(conn)
		}
	}
}

func (m *httpConnManager) id() int64 {
	return m.serial()
}

func (m *httpConnManager) changeState(conn net.Conn, fn func(*connInfo)) {
	if v, ok := m.conns.Load(conn); ok {
		s := v.(*connInfo)
		if fn != nil {
			fn(s)
		}
	}
}

func (m *httpConnManager) inc(state http.ConnState) {
	m.track(state, 1)
}

func (m *httpConnManager) dec(state http.ConnState) {
	m.track(state, -1)
}

func (m *httpConnManager) track(state http.ConnState, n int64) {
	switch state {
	case http.StateNew:
		atomic.AddInt64(&m.status.open, n)
	case http.StateActive:
		atomic.AddInt64(&m.status.active, n)
	case http.StateIdle:
		atomic.AddInt64(&m.status.idle, n)
	case http.StateHijacked:
		atomic.AddInt64(&m.status.hijacked, n)
	}
}

func (m *httpConnManager) GetStatus() httpConnStatus {
	return httpConnStatus{
		open:     atomic.LoadInt64(&m.status.open),
		active:   atomic.LoadInt64(&m.status.active),
		idle:     atomic.LoadInt64(&m.status.idle),
		hijacked: atomic.LoadInt64(&m.status.hijacked),
	}
}

// this ensures hijacked connections are closed and all connections references
// are cleared.
func (m *httpConnManager) graceful() {
	m.conns.Range(func(key, value interface{}) bool {
		i := value.(*connInfo)
		if i.state == http.StateHijacked {
			// this will never be closed by the server so we need to do it now
			key.(net.Conn).Close()
		}
		m.conns.Delete(key)
		return true
	})
}

func (m *httpConnManager) connContext(baseCtx context.Context, conn net.Conn) context.Context {
	reqID := m.id()
	m.conns.Store(conn, &connInfo{
		id: reqID,
	})
	baseCtx = context.WithValue(baseCtx, requestID{}, reqID)
	setVariable(baseCtx, vRequestID, reqID)
	return baseCtx
}

func (m *httpConnManager) baseCtx(ctx context.Context, ls net.Listener) context.Context {
	v := new(sync.Map)
	baseCtx := context.WithValue(ctx, variables{}, v)
	baseCtx = context.WithValue(baseCtx, connManagerKey{}, m)
	return baseCtx
}
