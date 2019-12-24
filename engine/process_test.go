package engine

import (
	"net"
	"testing"
)

func TestEncodefile(t *testing.T) {
	ls, err := net.Listen("tcp", ":8090")
	if err != nil {
		t.Fatal(err)
	}
	defer ls.Close()
	tcp, ok := ls.(*net.TCPListener)
	if !ok {
		t.Fatal("not ok")
	}
	f, err := tcp.File()
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	t.Error(tcp.Addr())
	t.Error(f.Name())
}
