package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	fileList := []struct {
		name string
		pass bool
	}{
		{"./fixture/nginx.conf", true},
	}
	for _, f := range fileList {
		t.Run(f.name, func(ts *testing.T) {
			if f.pass {
				pass(t, f.name)
			}
		})
	}
}

func pass(t *testing.T, filename string) {
	f, err := os.Open(filename)
	if err != nil {
		t.Error(err)
		return
	}
	defer f.Close()
	c, err := LoadDirective(filename, f)
	if err != nil {
		t.Error(err)
		return
	}
	if err := writeJSON(filename+".json", c); err != nil {
		t.Error(err)
	}
}

func writeJSON(filename string, v interface{}) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, b, 0600)
}
