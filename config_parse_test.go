package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/ergongate/vince/templates"
)

func TestParseIncludeReqular(t *testing.T) {
	filename := "fixture/crossplane/includes-regular/nginx.conf"
	fs, err := templates.NewIncludeFS()
	if err != nil {
		t.Fatal(err)
	}
	p := parse(filename, fs, defaultParseOpts())
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
	fs, err := templates.NewIncludeFS()
	if err != nil {
		t.Fatal(err)
	}
	p := parse(filename, fs, defaultParseOpts())
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
	fs, err := templates.NewIncludeFS()
	if err != nil {
		t.Fatal(err)
	}
	p := parse(filename, fs, opts)
	b, _ := json.Marshal(p)
	expect, err := ioutil.ReadFile(expectFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, expect) {
		t.Error("failed to match expectation")
	}
}
