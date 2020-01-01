package engine

import (
	"context"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"

	"github.com/ergongate/vince/version"
	"go.uber.org/zap"
)

type ConnManager struct {
	conns                        *sync.Map
	open, active, idle, hijacked int64
	serial                       int64
	pid                          int
	version                      string
	logger                       *zap.Logger
}

func NewConnManager(lg *zap.Logger) *ConnManager {
	return &ConnManager{
		conns:   &sync.Map{},
		pid:     os.Getpid(),
		version: version.Version,
		logger:  lg,
	}
}

func (m *ConnManager) Manage(conn net.Conn, state http.ConnState) {
	switch state {
	case http.StateNew:
		if _, ok := m.conns.Load(conn); !ok {
			m.conns.Store(conn, &ConnInfo{
				ID:    m.id(),
				State: state,
			})
		}
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
			i.NumRequests++
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

func (m *ConnManager) BaseContext(ls net.Listener) context.Context {
	v := &sync.Map{}
	v.Store("$nginx_version", version.Version)
	v.Store("$pid", m.pid)
	return context.WithValue(withLog(context.Background(), m.logger), variables{}, v)
}

func (m *ConnManager) ConnContext(ctx context.Context, conn net.Conn) context.Context {
	var info *ConnInfo
	if v, ok := m.conns.Load(conn); ok {
		info = v.(*ConnInfo)
	} else {
		info = &ConnInfo{
			ID: m.id(),
		}
		m.conns.Store(conn, info)
	}
	v := ctx.Value(variables{}).(*sync.Map)
	v.Store("$connection", info.ID)
	v.Store("$connection_requests", info.NumRequests)
	switch e := conn.(type) {
	case *ProxyConn:
		h, p, _ := net.SplitHostPort(e.RemoteAddr().String())
		v.Store("$proxy_protocol_addr", h)
		v.Store("$proxy_protocol_port", p)

		h, p, _ = net.SplitHostPort(e.LocalAddr().String())
		v.Store("$proxy_protocol_server_addr", h)
		v.Store("$proxy_protocol_server_port", p)

		h, p, _ = net.SplitHostPort(e.Remote.String())
		v.Store("$remote_addr", h)
		v.Store("$remote_port", p)
	default:
		h, p, _ := net.SplitHostPort(e.RemoteAddr().String())
		v.Store("$remote_addr", h)
		v.Store("$remote_port", p)
	}
	return context.WithValue(ctx, variables{}, v)
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
	ID          int64
	NumRequests int64
	State       http.ConnState
}

type ConnStatus struct {
	Open, Active, Idle, Hijacked int64
}

func (c *ConnStatus) Add(othe *ConnStatus) {
	atomic.AddInt64(&c.Open, othe.Open)
	atomic.AddInt64(&c.Active, othe.Active)
	atomic.AddInt64(&c.Idle, othe.Idle)
	atomic.AddInt64(&c.Hijacked, othe.Hijacked)
}

func (c *ConnStatus) Set(othe *ConnStatus) {
	atomic.StoreInt64(&c.Open, othe.Open)
	atomic.StoreInt64(&c.Active, othe.Active)
	atomic.StoreInt64(&c.Idle, othe.Idle)
	atomic.StoreInt64(&c.Hijacked, othe.Hijacked)
}

func (c *ConnStatus) Reset() ConnStatus {
	return ConnStatus{
		Open:     atomic.SwapInt64(&c.Open, 0),
		Active:   atomic.SwapInt64(&c.Active, 0),
		Idle:     atomic.SwapInt64(&c.Idle, 0),
		Hijacked: atomic.SwapInt64(&c.Hijacked, 0),
	}
}
