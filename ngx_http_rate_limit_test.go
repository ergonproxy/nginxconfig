package main

import "testing"

func TestParseRate(t *testing.T) {
	sample := []struct {
		r      string
		expect int
	}{
		{"1r/s", 1},
		{"300r/m", 5},
		{"5r/s", 5},
	}
	for _, v := range sample {
		got, err := parseRate(v.r)
		if err != nil {
			t.Fatal(err)
		}
		if got != v.expect {
			t.Errorf("%s: expected %d got %d", v.r, v.expect, got)
		}
	}
}
