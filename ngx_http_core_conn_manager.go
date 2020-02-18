package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	"go.uber.org/atomic"
)

type (
	vinceConfigKey struct{}
)

type connManager struct {
	conns  *sync.Map
	status httpConnStatus
	serial func() int64
}

type connInfo struct {
	id    int64
	state http.ConnState
}

type httpConnStatus struct {
	open, active, idle, hijacked atomic.Int64
}

func (c *httpConnStatus) Add(other *httpConnStatus) {
	c.open.Add(other.open.Load())
	c.active.Add(other.active.Load())
	c.idle.Add(other.idle.Load())
	c.hijacked.Add(other.hijacked.Load())
}

func (c *httpConnStatus) Set(other *httpConnStatus) {
	c.open.Store(other.open.Load())
	c.active.Store(other.active.Load())
	c.idle.Store(other.idle.Load())
	c.hijacked.Store(other.hijacked.Load())
}

func (c *httpConnStatus) Reset() httpConnStatus {
	var h httpConnStatus
	h.open.Store(c.open.Swap(0))
	h.active.Store(c.active.Swap(0))
	h.idle.Store(c.idle.Swap(0))
	return h
}

func (c *connManager) init() {
	c.conns = new(sync.Map)
	c.serial = nextID
}

func (m *connManager) manageConnState(conn net.Conn, state http.ConnState) {
	switch state {
	case http.StateNew:
		if _, ok := m.conns.Load(conn); !ok {
			m.conns.Store(conn, &connInfo{
				id: m.id(),
			})
		}
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

// CloseConn update connection state to closed. This is mainly used with connections
// that have StateHijacked its a noop for other states.
func (m *connManager) CloseConn(conn net.Conn) {
	if v, ok := m.conns.Load(conn); ok {
		s := v.(*connInfo)
		if s.state == http.StateHijacked {
			m.dec(s.state)
			m.dec(http.StateNew)
			m.conns.Delete(conn)
		}
	}
}

func (m *connManager) Close() error {
	var errs []string
	m.conns.Range(func(key, value interface{}) bool {
		if err := key.(net.Conn).Close(); err != nil {
			errs = append(errs, err.Error())
		}
		m.conns.Delete(key)
		return true
	})
	if len(errs) > 0 {
		return fmt.Errorf("conn_manager: error closing connections %s", strings.Join(errs, ","))
	}
	return nil
}

func (m *connManager) id() int64 {
	return m.serial()
}

func (m *connManager) changeState(conn net.Conn, fn func(*connInfo)) {
	if v, ok := m.conns.Load(conn); ok {
		s := v.(*connInfo)
		if fn != nil {
			fn(s)
		}
	}
}

func (m *connManager) inc(state http.ConnState) {
	m.track(state, 1)
}

func (m *connManager) dec(state http.ConnState) {
	m.track(state, -1)
}

func (m *connManager) track(state http.ConnState, n int64) {
	switch state {
	case http.StateNew:
		m.status.open.Add(n)
	case http.StateActive:
		m.status.active.Add(n)
	case http.StateIdle:
		m.status.idle.Add(n)
	case http.StateHijacked:
		m.status.hijacked.Add(n)
	}
}

func (m *connManager) GetStatus() httpConnStatus {
	return m.status.Reset()
}

// this ensures hijacked connections are closed and all connections references
// are cleared.
func (m *connManager) graceful() {
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

func (m *connManager) connContext(baseCtx context.Context, conn net.Conn) context.Context {
	reqID := m.id()
	m.conns.Store(conn, &connInfo{
		id: reqID,
	})
	baseCtx = context.WithValue(baseCtx, requestID{}, reqID)
	setVariable(baseCtx, vRequestID, reqID)
	return baseCtx
}

func (c *connManager) baseCtx(ctx context.Context, ls net.Listener) context.Context {
	return context.WithValue(ctx, variables{}, make(map[string]interface{}))
}

func (m *connManager) getID(conn net.Conn) int64 {
	if v, ok := m.conns.Load(conn); ok {
		return v.(*connInfo).id
	}
	return 0
}
