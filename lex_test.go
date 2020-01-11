package main

import (
	"os"
	"reflect"
	"testing"
)

func TestLexSimple(t *testing.T) {
	file := "fixture/crossplane/simple/nginx.conf"
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	tokens, err := lex(f, file)
	if err != nil {
		t.Fatal(err)
	}
	expect := []token{
		{text: "events", line: 1, quote: false},
		{text: "{", line: 1, quote: false},
		{text: "worker_connections", line: 2, quote: false},
		{text: "1024", line: 2, quote: false},
		{text: ";", line: 2, quote: false},
		{text: "}", line: 3, quote: false},
		{text: "http", line: 5, quote: false},
		{text: "{", line: 5, quote: false},
		{text: "server", line: 6, quote: false},
		{text: "{", line: 6, quote: false},
		{text: "listen", line: 7, quote: false},
		{text: "127.0.0.1:8080", line: 7, quote: false},
		{text: ";", line: 7, quote: false},
		{text: "server_name", line: 8, quote: false},
		{text: "default_server", line: 8, quote: false},
		{text: ";", line: 8, quote: false},
		{text: "location", line: 9, quote: false},
		{text: "/", line: 9, quote: false},
		{text: "{", line: 9, quote: false},
		{text: "return", line: 10, quote: false},
		{text: "200", line: 10, quote: false},
		{text: "foo bar baz", line: 10, quote: true},
		{text: ";", line: 10, quote: false},
		{text: "}", line: 11, quote: false},
		{text: "}", line: 12, quote: false},
		{text: "}", line: 13, quote: false},
	}
	if !reflect.DeepEqual(tokens, expect) {
		t.Errorf("expected %#v\n got %#v", expect, tokens)
	}
}

func TestLexWithComments(t *testing.T) {
	file := "fixture/crossplane/with-comments/nginx.conf"
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	tokens, err := lex(f, file)
	if err != nil {
		t.Fatal(err)
	}
	expect := []token{
		{text: "events", line: 1, quote: false},
		{text: "{", line: 1, quote: false},
		{text: "worker_connections", line: 2, quote: false},
		{text: "1024", line: 2, quote: false},
		{text: ";", line: 2, quote: false},
		{text: "}", line: 3, quote: false},
		{text: "#comment", line: 4, quote: false},
		{text: "http", line: 5, quote: false},
		{text: "{", line: 5, quote: false},
		{text: "server", line: 6, quote: false},
		{text: "{", line: 6, quote: false},
		{text: "listen", line: 7, quote: false},
		{text: "127.0.0.1:8080", line: 7, quote: false},
		{text: ";", line: 7, quote: false},
		{text: "#listen", line: 7, quote: false},
		{text: "server_name", line: 8, quote: false},
		{text: "default_server", line: 8, quote: false},
		{text: ";", line: 8, quote: false},
		{text: "location", line: 9, quote: false},
		{text: "/", line: 9, quote: false},
		{text: "{", line: 9, quote: false},
		{text: "## this is brace", line: 9, quote: false},
		{text: "# location /", line: 10, quote: false},
		{text: "return", line: 11, quote: false},
		{text: "200", line: 11, quote: false},
		{text: "foo bar baz", line: 11, quote: true},
		{text: ";", line: 11, quote: false},
		{text: "}", line: 12, quote: false},
		{text: "}", line: 13, quote: false},
		{text: "}", line: 14, quote: false},
	}
	if !reflect.DeepEqual(tokens, expect) {
		t.Errorf("expected %#v\n got %#v", expect, tokens)
	}
}

func TestLexmessyConfig(t *testing.T) {
	file := "fixture/crossplane/messy/nginx.conf"
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	tokens, err := lex(f, file)
	if err != nil {
		t.Fatal(err)
	}
	expect := []token{

		{text: "user", line: 1, quote: false},
		{text: "nobody", line: 1, quote: false},
		{text: ";", line: 1, quote: false},
		{text: "# hello\\n\\\\n\\\\\\n worlddd  \\#\\\\#\\\\\\# dfsf\\n \\\\n \\\\\\n \\", line: 2, quote: false},
		{text: "events", line: 3, quote: true},
		{text: "{", line: 3, quote: false},
		{text: "worker_connections", line: 3, quote: true},
		{text: "2048", line: 3, quote: true},
		{text: ";", line: 3, quote: false},
		{text: "}", line: 3, quote: false},
		{text: "http", line: 5, quote: true},
		{text: "{", line: 5, quote: false},
		{text: "#forteen", line: 5, quote: false},
		{text: "# this is a comment", line: 6, quote: false},
		{text: "access_log", line: 7, quote: true},
		{text: "off", line: 7, quote: false},
		{text: ";", line: 7, quote: false},
		{text: "default_type", line: 7, quote: false},
		{text: "text/plain", line: 7, quote: true},
		{text: ";", line: 7, quote: false},
		{text: "error_log", line: 7, quote: false},
		{text: "off", line: 7, quote: true},
		{text: ";", line: 7, quote: false},
		{text: "server", line: 8, quote: false},
		{text: "{", line: 8, quote: false},
		{text: "listen", line: 9, quote: true},
		{text: "8083", line: 9, quote: true},
		{text: ";", line: 9, quote: false},
		{text: "return", line: 10, quote: true},
		{text: "200", line: 10, quote: false},
		{text: "Ser\" ' ' ver\\\\ \\ $server_addr:\\$server_port\\n\\nTime: $time_local\\n\\n", line: 10, quote: true},
		{text: ";", line: 10, quote: false},
		{text: "}", line: 11, quote: false},
		{text: "server", line: 12, quote: true},
		{text: "{", line: 12, quote: false},
		{text: "listen", line: 12, quote: true},
		{text: "8080", line: 12, quote: false},
		{text: ";", line: 12, quote: false},
		{text: "root", line: 13, quote: true},
		{text: "/usr/share/nginx/html", line: 13, quote: false},
		{text: ";", line: 13, quote: false},
		{text: "location", line: 14, quote: false},
		{text: "~", line: 14, quote: false},
		{text: "/hello/world;", line: 14, quote: true},
		{text: "{", line: 14, quote: false},
		{text: "return", line: 14, quote: true},
		{text: "301", line: 14, quote: false},
		{text: "/status.html", line: 14, quote: false},
		{text: ";", line: 14, quote: false},
		{text: "}", line: 14, quote: false},
		{text: "location", line: 15, quote: false},
		{text: "/foo", line: 15, quote: false},
		{text: "{", line: 15, quote: false},
		{text: "}", line: 15, quote: false},
		{text: "location", line: 15, quote: false},
		{text: "/bar", line: 15, quote: false},
		{text: "{", line: 15, quote: false},
		{text: "}", line: 15, quote: false},
		{text: "location", line: 16, quote: false},
		{text: "/\\{\\;\\}\\ #\\ ab", line: 16, quote: false},
		{text: "{", line: 16, quote: false},
		{text: "}", line: 16, quote: false},
		{text: "# hello", line: 16, quote: false},
		{text: "if", line: 17, quote: false},
		{text: "($request_method", line: 17, quote: false},
		{text: "=", line: 17, quote: false},
		{text: "P\\{O\\)\\###\\;ST", line: 17, quote: false},
		{text: ")", line: 17, quote: false},
		{text: "{", line: 17, quote: false},
		{text: "}", line: 17, quote: false},
		{text: "location", line: 18, quote: false},
		{text: "/status.html", line: 18, quote: true},
		{text: "{", line: 18, quote: false},
		{text: "try_files", line: 19, quote: false},
		{text: "/abc/${uri} /abc/${uri}.html", line: 19, quote: false},
		{text: "=404", line: 19, quote: false},
		{text: ";", line: 19, quote: false},
		{text: "}", line: 20, quote: false},
		{text: "location", line: 21, quote: true},
		{text: "/sta;\n                    tus", line: 21, quote: true},
		{text: "{", line: 22, quote: false},
		{text: "return", line: 22, quote: true},
		{text: "302", line: 22, quote: false},
		{text: "/status.html", line: 22, quote: false},
		{text: ";", line: 22, quote: false},
		{text: "}", line: 22, quote: false},
		{text: "location", line: 23, quote: true},
		{text: "/upstream_conf", line: 23, quote: false},
		{text: "{", line: 23, quote: false},
		{text: "return", line: 23, quote: true},
		{text: "200", line: 23, quote: false},
		{text: "/status.html", line: 23, quote: false},
		{text: ";", line: 23, quote: false},
		{text: "}", line: 23, quote: false},
		{text: "}", line: 23, quote: false},
		{text: "server", line: 24, quote: false},
		{text: "{", line: 25, quote: false},
		{text: "}", line: 25, quote: false},
		{text: "}", line: 25, quote: false},
	}
	// for _, tk := range tokens {
	// 	fmt.Printf("%#v,\n", tk)
	// }
	// t.Error("yay")
	if !reflect.DeepEqual(tokens, expect) {
		t.Errorf("expected %#v\n got %#v", expect, tokens)
	}
}

func TestLexQuoteBehavior(t *testing.T) {
	file := "fixture/crossplane/quote-behavior/nginx.conf"
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	tokens, err := lex(f, file)
	if err != nil {
		t.Fatal(err)
	}
	expect := []token{
		{text: "outer-quote", line: 1, quote: true},
		{text: "left", line: 1, quote: true},
		{text: "-quote", line: 1, quote: false},
		{text: "right-\"quote\"", line: 1, quote: false},
		{text: "inner\"-\"quote", line: 1, quote: false},
		{text: ";", line: 1, quote: false},
		{text: "", line: 2, quote: true},
		{text: "", line: 2, quote: true},
		{text: "left-empty", line: 2, quote: false},
		{text: "right-empty\"\"", line: 2, quote: false},
		{text: "inner\"\"empty", line: 2, quote: false},
		{text: "right-empty-single\"", line: 2, quote: false},
		{text: ";", line: 2, quote: false},
	}
	if !reflect.DeepEqual(tokens, expect) {
		t.Errorf("expected %#v\n got %#v", expect, tokens)
	}
}

func TestQUoteRightBrace(t *testing.T) {
	file := "fixture/crossplane/quoted-right-brace/nginx.conf"
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	tokens, err := lex(f, file)
	if err != nil {
		t.Fatal(err)
	}
	expect := []token{
		{text: "events", line: 1, quote: false},
		{text: "{", line: 1, quote: false},
		{text: "}", line: 1, quote: false},
		{text: "http", line: 2, quote: false},
		{text: "{", line: 2, quote: false},
		{text: "log_format", line: 3, quote: false},
		{text: "main", line: 3, quote: false},
		{text: "escape=json", line: 3, quote: false},
		{text: "{ \"@timestamp\": \"$time_iso8601\", ", line: 4, quote: true},
		{text: "\"server_name\": \"$server_name\", ", line: 5, quote: true},
		{text: "\"host\": \"$host\", ", line: 6, quote: true},
		{text: "\"status\": \"$status\", ", line: 7, quote: true},
		{text: "\"request\": \"$request\", ", line: 8, quote: true},
		{text: "\"uri\": \"$uri\", ", line: 9, quote: true},
		{text: "\"args\": \"$args\", ", line: 10, quote: true},
		{text: "\"https\": \"$https\", ", line: 11, quote: true},
		{text: "\"request_method\": \"$request_method\", ", line: 12, quote: true},
		{text: "\"referer\": \"$http_referer\", ", line: 13, quote: true},
		{text: "\"agent\": \"$http_user_agent\"", line: 14, quote: true},
		{text: "}", line: 15, quote: true},
		{text: ";", line: 15, quote: false},
		{text: "}", line: 16, quote: false},
	}
	if !reflect.DeepEqual(tokens, expect) {
		t.Errorf("expected %#v\n got %#v", expect, tokens)
	}
}
