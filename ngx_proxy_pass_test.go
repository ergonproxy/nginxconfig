package main

import (
	"encoding/json"
	"net/url"
	"testing"
	"time"
)

func TestPassUnix(t *testing.T) {
	t.Skip()
	s := "http://unix:/tmp/backend.socket:/uri/"
	u, err := parseProxyURL(s)
	if err != nil {
		t.Fatal(err)
	}
	t.Errorf("%v", u)

	ux, err := url.Parse("unix:/var/folders/82/kbw661tj579fcykv_vz2sjbw0000gn/T/vince-test-suite996732971/unix.sock")
	if err != nil {
		t.Fatal(err)
	}
	t.Errorf("%#v", ux)

}

func TestDuration(t *testing.T) {
	t.Skip()
	b, _ := json.Marshal(time.Second)
	t.Error(string(b))
}
