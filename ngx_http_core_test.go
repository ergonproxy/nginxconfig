package main

import (
	"reflect"
	"sort"
	"testing"
)

func TestParseListen(t *testing.T) {
	sample := []struct {
		args         []string
		net, addHost string
	}{
		{[]string{"127.0.0.1:8000"}, "tcp", "127.0.0.1:8000"},
		{[]string{"127.0.0.1"}, "tcp", "127.0.0.1:8000"},
		{[]string{"8000"}, "tcp", "0.0.0.0:8000"},
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

func TestOpenSSLCiphers(t *testing.T) {
	t.Skip()
	r, err := openSSLCiphers("EECDH+AESGCM:ECDHE+AESGCM:HIGH:!MD5:!RC4:!aNULL")
	if err != nil {
		t.Fatal(err)
	}
	expect := []string{
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256",
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA",
		"TLS_RSA_WITH_AES_128_GCM_SHA256",
		"TLS_RSA_WITH_AES_128_CBC_SHA256",
		"TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA",
		"TLS_RSA_WITH_3DES_EDE_CBC_SHA",
	}
	sort.Strings(expect)
	std := standardCiphers(r)
	var got []string
	for _, v := range std {
		got = append(got, v.String())
	}
	sort.Strings(got)
	if !reflect.DeepEqual(got, expect) {
		t.Errorf(" expected\n%#v\n got \n%#v", expect, got)
	}
}
