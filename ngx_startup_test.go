package main

import "testing"

func TestOveride(t *testing.T) {
	r := []*rule{
		{name: "first"},
		{name: "second"},
		{name: "third"},
		{name: "fourth"},
		{name: "first"},
	}
	got := overide(r)
	if len(got) != 4 {
		t.Fatalf("expected %d got %d", 4, len(got))
	}
	if got[3].name != "first" {
		t.Errorf("expected %q got %q", "first", got[3].name)
	}
}

func TesVinceHandler(t *testing.T) {

}
