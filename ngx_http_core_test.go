package main

import "testing"

func TestParseListen(t *testing.T) {
	sample := []struct {
		args         []string
		net, addHost string
	}{
		{[]string{"127.0.0.1:8000"}, "tcp", "127.0.0.1:8000"},
		{[]string{"127.0.0.1"}, "tcp", "127.0.0.1:8000"},
		{[]string{"8000"}, "tcp", ":8000"},
		{[]string{"*:8000"}, "tcp", "*:8000"},
		{[]string{"localhost:8000"}, "tcp", "localhost:8000"},
		{[]string{"[::]:8000"}, "tcp", "[::]:8000"},
		{[]string{"[::1]"}, "tcp", "[::1]:8000"},
		{[]string{"unix:/var/run/nginx.sock"}, "unix", "/var/run/nginx.sock"},
	}
	stmt := &rule{name: "listen"}
	for _, s := range sample {
		t.Run(s.args[0], func(ts *testing.T) {
			stmt.args = s.args
			o := parseListen(stmt, "8000")
			if o.net != s.net {
				ts.Errorf("net: expected %q got %q", s.net, o.net)
			}
			if o.addrPort != s.addHost {
				ts.Errorf("addr:port : expected %q got %q", s.addHost, o.addrPort)
			}
		})
	}
}
