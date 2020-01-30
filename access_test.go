package main

import "testing"

func TestAccess(t *testing.T) {
	t.Skip()
	file := `daemon off;
events {
}
http {
    {{test_http_globals .dir}}
    server {
        listen       127.0.0.1:8080;
        server_name  localhost;
        location /inet/ {
            proxy_pass http://127.0.0.1:8081/;
        }
        location /inet6/ {
            proxy_pass http://[::1]:{{port 8081}}/;
        }
        location /unix/ {
            proxy_pass http://unix:{{.dir}}/unix.sock:/;
        }
    }
    server {
        listen       127.0.0.1:8081;
        listen       [::1]:{{port 8081}};
        listen       unix:{{.dir}}/unix.sock;
        location /allow_all {
            allow all;
        }
        location /allow_unix {
            allow unix:;
        }
        location /deny_all {
            deny all;
        }
        location /deny_unix {
            deny unix:;
        }
    }
}
`
	c, clear, err := setup(file)
	if err != nil {
		t.Fatal(err)
	}
	defer clear()
	runTest(t, c)
}
