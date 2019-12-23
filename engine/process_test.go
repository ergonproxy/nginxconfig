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
	env := fileToEnv(f)
	f2, err := envToFile(env)
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()
	if f2.Fd() != f.Fd() {
		t.Errorf("expected %d got %d", f.Fd(), f2.Fd())
	}
	if f.Name() != f2.Name() {
		t.Errorf("expected %s got %s", f.Name(), f2.Name())
	}
}
