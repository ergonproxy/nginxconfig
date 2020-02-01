package templates

import (
	"bytes"
	"testing"
)

func TestHTML(t *testing.T) {
	tpl := HTML()
	var buf bytes.Buffer
	err := tpl.ExecuteTemplate(&buf, "oauth/login.html", nil)
	if err != nil {
		t.Fatal(err)
	}
}
