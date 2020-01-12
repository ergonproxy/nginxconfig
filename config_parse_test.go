package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestParseIncludeReqular(t *testing.T) {
	filename := "fixture/crossplane/includes-regular/nginx.conf"
	p := parse(filename, defaultParseOpts())
	b, _ := json.Marshal(p)
	expect, err := ioutil.ReadFile(filepath.Join(filepath.Dir(filename), "expect.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, expect) {
		t.Error("failed to match expectation")
	}
}

func TestParseIncludesGlobbed(t *testing.T) {
	filename := "fixture/crossplane/includes-globbed/nginx.conf"
	expectFile := filepath.Join(filepath.Dir(filename), "expect_globbed.json")
	p := parse(filename, defaultParseOpts())
	b, _ := json.Marshal(p)
	expect, err := ioutil.ReadFile(expectFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, expect) {
		t.Error("failed to match expectation")
	}
}
func TestParseIncludesBlobbedCombined(t *testing.T) {
	filename := "fixture/crossplane/includes-globbed/nginx.conf"
	expectFile := filepath.Join(filepath.Dir(filename), "expect_globbed_combine.json")
	opts := defaultParseOpts()
	opts.combine = true
	p := parse(filename, opts)
	b, _ := json.Marshal(p)
	expect, err := ioutil.ReadFile(expectFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, expect) {
		t.Error("failed to match expectation")
	}
}
