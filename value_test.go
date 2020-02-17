package main

import "testing"

func TestStringTemplateValue(t *testing.T) {
	sample := []struct {
		src    string
		ctx    interface{}
		expect string
	}{
		{"empty", nil, "empty"},
		{"empty $key", map[string]string{"key": "1"}, "empty 1"},
		{"empty $1", map[string]string{"n_1": "1"}, "empty 1"},
	}
	for _, v := range sample {
		s := new(stringTemplateValue)
		s.store(v.src)
		got := s.Value(v.ctx)
		if got != v.expect {
			t.Errorf("%s: expected %q got %q", v.src, v.expect, got)
		}
	}
}
