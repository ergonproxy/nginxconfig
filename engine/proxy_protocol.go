package engine

import "net"

var _ net.Listener = (*ProxyListener)(nil)

type ProxyListener struct {
	Listener net.Listener
}

type ProxyConn struct {
	Local  net.Addr
	Remote net.Addr
	net.Conn
}

func (pls *ProxyListener) Accept() (net.Conn, error) {
	conn, err := pls.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &ProxyConn{
		Local:  conn.LocalAddr(),
		Remote: conn.RemoteAddr(),
		Conn:   conn,
	}, nil
}

func (pls ProxyListener) Close() error {
	return pls.Listener.Close()
}

func (pls *ProxyListener) Addr() net.Addr {
	return pls.Listener.Addr()
}
