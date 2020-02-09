package main

import "testing"

func TestLogFormat(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		var lf logFormat
		lf.defaults()

		if lf.name != "combined" {
			t.Errorf("expected combined got %q", lf.name)
		}
		if lf.escape != "default" {
			t.Errorf("expected default got %q", lf.escape)
		}
		if lf.template != defaultLogFormat {
			t.Errorf("expected %q got %q", defaultLogFormat, lf.template)
		}
	})
}
