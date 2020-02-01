package main

import (
	"net"

	"github.com/ergongate/vince/buffers"
	"github.com/mikioh/tcp"
	"github.com/mikioh/tcpinfo"
)

func getTCPConnInfo(conn net.Conn) (*tcpinfo.Info, error) {
	c, err := tcp.NewConn(conn)
	if err != nil {
		return nil, err
	}
	buf := buffers.GetSlice()
	defer func() {
		buffers.PutSlice(buf)
	}()
	var o tcpinfo.Info
	opts, err := c.Option(o.Level(), o.Name(), buf)
	if err != nil {
		return nil, err
	}
	return opts.(*tcpinfo.Info), nil
}
