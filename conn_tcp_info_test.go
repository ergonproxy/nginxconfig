package main

import (
	"fmt"
	"net"
	"testing"

	"github.com/mikioh/tcpinfo"
)

func TestGetTCPInfo(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if l, err = net.Listen("tcp6", "[::1]:0"); err != nil {
			t.Fatal(fmt.Sprintf("httptest: failed to listen on a port: %v", err))
		}
	}
	defer l.Close()
	ok := make(chan *tcpinfo.Info)
	defer close(ok)
	go func() {
		a, err := l.Accept()
		if err != nil {
			t.Error(a)
		}
		defer a.Close()
		info, err := getTCPConnInfo(a)
		if err != nil {
			t.Error(a)
		}
		ok <- info
	}()
	c, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	c.Write([]byte("hello"))
	c.Close()
	info := <-ok
	if info == nil {
		t.Fatal("expected connection info")
	}
}
