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
	p := parse(filename, templates.IncludeFS, defaultParseOpts())
	b, _ := json.Marshal(p)
	file := filepath.Join(filepath.Dir(filename), "includes_regular")
	// ioutil.WriteFile(file, b, 0600)
	expect, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, expect) {
		t.Error("failed to match expectation")
	}
}

func TestParseIncludesGlobbed(t *testing.T) {
	filename := "fixture/crossplane/includes-globbed/nginx.conf"
	p := parse(filename, templates.IncludeFS, defaultParseOpts())
	b, _ := json.Marshal(p)
	file := filepath.Join(filepath.Dir(filename), "includes_globbed")
	// ioutil.WriteFile(file, b, 0600)
	expect, err := ioutil.ReadFile(file)
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
	p := parse(filename, templates.IncludeFS, opts)
	b, _ := json.Marshal(p)
	expect, err := ioutil.ReadFile(expectFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, expect) {
		t.Error("failed to match expectation")
	}
}
