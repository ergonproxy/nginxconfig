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
	t.Run("ExtraScopes", func(t *testing.T) {
		sample := []struct {
			access, refresh string
			result          bool
		}{
			{"one,two,three", "one", true},
			{"one,two,three", "none", false},
		}
		var o oauth2

		for _, scope := range sample {
			if e := o.extraScopes(scope.access, scope.refresh); e != scope.result {
				t.Errorf("expected %v got %v  aceess: %s refresh:: %s", scope.result, e, scope.access, scope.refresh)
			}
		}
	})
	t.Run("ValidURL", func(t *testing.T) {
		link := "http://www.example.com"
		sample := []struct {
			info, base, redir string
			valid             bool
		}{
			{"exact match", "/vince", "/vince", true},
			{"trailing slash", "/vince", "/vince/", true},
			{"exact match with trailing slash", "/vince/", "/vince/", true},
			{"subpath", "/vince", "/vince/sub/path", true},
			{"subpath with trailing slash", "/vince/", "/vince/sub/path", true},
			{"subpath with traversal like", "/vince", "/vince/.../..sub../...", true},
			{"traversal", "/vince/../allow", "/vince/../allow/sub/path", true},
			{"base path mismatch", "/vince", "/vinceine", false},
			{"base path mismatch slash", "/vince/", "/vince", false},
			{"traversal", "/vince", "/vince/..", false},
			{"embed traversal", "/vince", "/vince/../sub", false},
			{"not subpath", "/vince", "/vince../sub", false},
		}

		for _, v := range sample {
			if v.valid {
				err := validateURI(link+v.base, link+v.redir)
				if err != nil {
					t.Errorf("some fish for %s : %v", v.info, err)
				}
			} else {
				err := validateURI(link+v.base, link+v.redir)
				if err == nil {
					t.Errorf("expected error for for %s : got %v", v.info, err)
				}
			}
		}

		sampleList := []struct {
			base, redir, sep string
			valid            bool
		}{
			{"http://www.example.com/vince", "http://www.example.com/vince", "", true},
			{"http://www.example.com/vince", "http://www.example.com/app", "", false},
			{"http://xxx:14000/vince;http://www.example.com/vince", "http://www.example.com/vince", ";", true},
			{"http://xxx:14000/vince;http://www.example.com/vince", "http://www.example.com/app", ";", false},
		}

		for _, v := range sampleList {
			if v.valid {
				err := validateURIList(v.base, v.redir, v.sep)
				if err != nil {
					t.Error(err)
				}
			} else {
				err := validateURIList(v.base, v.redir, v.sep)
				if err == nil {
					t.Error("expected an error")
				}
			}
		}
	})
}
