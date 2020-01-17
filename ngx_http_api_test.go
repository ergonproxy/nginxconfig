package main

import "testing"

import "time"

func TestISO8601(t *testing.T) {
	sample := []string{"2019-10-01T11:15:44.467Z", "2019-10-01T09:26:07.305Z"}
	for _, v := range sample {
		ts, err := time.Parse(iso8601Milli, v)
		if err != nil {
			t.Fatal(err)
		}
		got := formatISO8601TimeStamp(ts)
		if got != v {
			t.Errorf("expected %v got %v", v, got)
		}
	}
}
