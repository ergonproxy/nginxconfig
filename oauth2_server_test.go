package main

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOauth2(t *testing.T) {
	t.Run("Context", func(t *testing.T) {
		w := httptest.NewRecorder()
		var ctx oauth2Context
		ctx.init()
		link := "http://www.example.com"
		ctx.setErrURI(oauth2ErrInvalidClient, "", link, "")
		ctx.setRedirect(link)
		rdir, err := ctx.getRedirectURL()
		if err != nil {
			t.Error(err)
		}
		if !strings.Contains(rdir, link+"?") {
			t.Errorf("expected a normal query got %s", rdir)
		}
		ctx.redirectInFragment = true
		rdir, err = ctx.getRedirectURL()
		if err != nil {
			t.Error(err)
		}
		if !strings.Contains(rdir, link+"#") {
			t.Errorf("expected a normal query got %s", rdir)
		}
		ctx.kind = oauth2ResponseData
		err = ctx.commit(w)
		if err != nil {
			t.Error(err)
		}
	})
}
