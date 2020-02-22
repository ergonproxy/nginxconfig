package templates

import (
	"bytes"
	"testing"
)

func TestHTML(t *testing.T) {
	var buf bytes.Buffer
	err := ExecHTML(&buf, "oauth/login.html", nil)
	if err != nil {
		t.Fatal(err)
	}
}
