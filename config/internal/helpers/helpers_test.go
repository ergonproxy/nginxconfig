package helpers

import "testing"

func TestParseConnection(t *testing.T) {
	sample := []struct {
		name, listen, expect string
	}{
		{"ip:port", "127.0.0.1:8000", "127.0.0.1:8000"},
		{"ip", "127.0.0.1", "127.0.0.1"},
		{"port", "8000", "8000"},
		{"all interfaces", "*:8000", "*:8000"},
		{"localhost", "localhost:8000", "localhost:8000"},
		{"ipv6:port", "[::]:8000", "[::]:8000"},
		{"ipv6", "[::1]", "[::1]"},
		{"unix", "unix:/var/run/nginx.sock", "unix:/var/run/nginx.sock"},
	}
	for _, s := range sample {
		t.Run(s.name, func(ts *testing.T) {
			c, err := ParseConnection(s.listen)
			if err != nil {
				ts.Fatal(err)
			}
			got := c.String()
			if got != s.listen {
				ts.Errorf("expected %q got %q", s.listen, got)
			}
		})
	}
}
