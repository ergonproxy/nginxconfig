package main

import (
	"testing"
)

func TestBuildNestedAndMultipleArgs(t *testing.T) {
	sample := []*Stmt{
		&Stmt{
			Directive: "events",
			Blocks: []*Stmt{
				&Stmt{
					Directive: "worker_connections",
					Args:      []string{"1024"},
				},
			},
		},
		&Stmt{
			Directive: "http",
			Blocks: []*Stmt{
				&Stmt{
					Directive: "server",
					Blocks: []*Stmt{
						&Stmt{Directive: "listeb", Args: []string{"127.0.0.1:8080"}},
						&Stmt{Directive: "server_name", Args: []string{"default_server"}},
						&Stmt{
							Directive: "location",
							Args:      []string{"/"},
							Blocks: []*Stmt{
								&Stmt{
									Directive: "return",
									Args:      []string{"200", "foo bar baz"},
								},
							},
						},
					},
				},
			},
		},
	}
	expect := `events {
    worker_connections 1024;
}
http {
    server {
        listeb 127.0.0.1:8080;
        server_name default_server;
        location / {
            return 200 foo bar baz;
        }
    }
}`
	buf := build(sample, 4, false)
	if buf != expect {
		t.Errorf("===expected \n%s\n === got \n%s", expect, buf)
	}
}
