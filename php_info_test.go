package main

import "testing"

import "os"

import "encoding/json"

import "io/ioutil"

import "bytes"

func TestParsePHPInfo(t *testing.T) {
	f, err := os.Open("fixture/php_info")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	info := parsePHPInfo(f)
	b, _ := json.MarshalIndent(info, "", "  ")
	expect, err := ioutil.ReadFile("fixture/php_info.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, expect) {
		t.Error("mismatch")
	}
}
