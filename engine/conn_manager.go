package engine

import (
	"net"
	"net/http"
	"sync"
	"sync/atomic"
)

type ConnManager struct {
	conns                        *sync.Map
	open, active, idle, hijacked int64
	serial                       int64
}

func NewConnManager() *ConnManager {
	return &ConnManager{
		conns: &sync.Map{},
	}
}

func (m *ConnManager) Manage(conn net.Conn, state http.ConnState) {
	switch state {
	case http.StateNew:
		m.conns.Store(conn, &ConnInfo{
			ID:    m.id(),
			State: state,
		})
		m.inc(state)
	case http.StateActive:
		m.changeState(conn, func(i *ConnInfo) {
			if i.State != http.StateNew {
				// When we transition from new , we still mark the connection as open but in
				// active state, so here we have two records of the same connection since
				// the connection is both open and active.
				// This means we don't reduce open connections.
				m.dec(i.State)
			}
			i.State = state
		})
		m.inc(state)
	case http.StateIdle:
		m.changeState(conn, func(i *ConnInfo) {
			m.dec(i.State)
			i.State = state
		})
		m.inc(state)
	case http.StateHijacked:
		m.changeState(conn, func(i *ConnInfo) {
			m.dec(i.State)
			i.State = state
		})
		m.inc(state)
	case http.StateClosed:
		m.changeState(conn, func(i *ConnInfo) {
			m.dec(i.State)
			if i.State != http.StateNew {
				m.dec(http.StateNew)
			}
			i.State = state
		})
		m.conns.Delete(conn)
	}
}

// Close update connection state to closed. This is mainly used with connections
// that have StateHijacked its a noop for other states.
func (m *ConnManager) Close(conn net.Conn) {
	if v, ok := m.conns.Load(conn); ok {
		s := v.(*ConnInfo)
		if s.State == http.StateHijacked {
			m.dec(s.State)
			m.dec(http.StateNew)
			m.conns.Delete(conn)
		}
	}
}

func (m *ConnManager) id() int64 {
	return atomic.AddInt64(&m.serial, 1)
}

func (m *ConnManager) changeState(conn net.Conn, fn func(*ConnInfo)) {
	if v, ok := m.conns.Load(conn); ok {
		s := v.(*ConnInfo)
		if fn != nil {
			fn(s)
		}
	}
}

func (m *ConnManager) inc(state http.ConnState) {
	m.track(state, 1)
}

func (m *ConnManager) dec(state http.ConnState) {
	m.track(state, -1)
}

func (m *ConnManager) track(state http.ConnState, n int64) {
	switch state {
	case http.StateNew:
		atomic.AddInt64(&m.open, n)
	case http.StateActive:
		atomic.AddInt64(&m.active, n)
	case http.StateIdle:
		atomic.AddInt64(&m.idle, n)
	case http.StateHijacked:
		atomic.AddInt64(&m.hijacked, n)
	}
}

func (m *ConnManager) GetInfo(conn net.Conn) (*ConnInfo, bool) {
	v, ok := m.conns.Load(conn)
	if ok {
		return v.(*ConnInfo), ok
	}
	return nil, ok
}

// GetStatus returns connection status of the manager.
func (m *ConnManager) GetStatus() ConnStatus {
	return ConnStatus{
		Open:     atomic.LoadInt64(&m.open),
		Active:   atomic.LoadInt64(&m.active),
		Idle:     atomic.LoadInt64(&m.idle),
		Hijacked: atomic.LoadInt64(&m.hijacked),
	}
}

type ConnInfo struct {
	ID    int64
	State http.ConnState
}

type ConnStatus struct {
	Open, Active, Idle, Hijacked int64
}
