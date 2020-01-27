package templates

import "testing"

import "bytes"

func TestHTML(t *testing.T) {
	tpl, err := HTML()
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	err = tpl.ExecuteTemplate(&buf, "oauth2_login.html", nil)
	if err != nil {
		t.Fatal(err)
	}
}
