package main

import (
	"net"
	"sync"

	"github.com/mikioh/tcp"
	"github.com/mikioh/tcpinfo"
)

var infoBufferPool = &sync.Pool{
	New: func() interface{} {
		return make([]byte, 356)
	},
}

func getTCPConnInfo(conn net.Conn) (*tcpinfo.Info, error) {
	c, err := tcp.NewConn(conn)
	if err != nil {
		return nil, err
	}
	buf := infoBufferPool.Get().([]byte)
	defer func() {
		infoBufferPool.Put(buf[:0])
	}()
	var o tcpinfo.Info
	opts, err := c.Option(o.Level(), o.Name(), buf)
	if err != nil {
		return nil, err
	}
	return opts.(*tcpinfo.Info), nil
}
