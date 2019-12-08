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
			switch i.State {
			case http.StateIdle:
				m.dec(i.State)
			}
			i.State = state
		})
		m.inc(state)
	case http.StateIdle:
		m.changeState(conn, func(i *ConnInfo) {
			switch i.State {
			case http.StateActive:
				m.dec(i.State)
			}
			i.State = state
			atomic.AddInt64(&m.active, -1)
		})
		m.inc(state)
	case http.StateHijacked:
		m.changeState(conn, func(i *ConnInfo) {
			switch i.State {
			case http.StateActive:
				m.dec(i.State)
			}
			i.State = state
		})
		m.inc(state)
	case http.StateClosed:
		m.changeState(conn, func(i *ConnInfo) {
			switch i.State {
			case http.StateIdle, http.StateActive:
				m.dec(i.State)
			}
			i.State = state
		})
		m.dec(http.StateNew)
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
