package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
)

type kvStore interface {
	get(key []byte) ([]byte, error)
	set(key, value []byte) error
}

const (
	oauth2ParamError         = "error"
	oauth2ParamErrDesc       = "error_description"
	oauth2ParamErrURI        = "error_uri"
	oauth2ParamState         = "state"
	oauth2ParamGrantType     = "grant_type"
	oauth2ParamLocation      = "Location"
	oauth2ParamClientID      = "client_id"
	oauth2ParamClientSecret  = "client_secret"
	oauth2ParamAccessToken   = "access_token"
	oauth2ParamTokenType     = "token_type"
	oauth2ParamExpiresIn     = "expires_in"
	oauth2ParamRefreshToken  = "refresh_token"
	oauth2ParamScope         = "scope"
	oauth2ParamRedirectURL   = "redirect_url"
	oauth2ParamCode          = "code"
	oauth2ParamAssertion     = "assertion"
	oauth2ParamAssertionType = "assertion_type"
	oauth2ParamResponseType  = "response_type"
)

type oauth2Errkey string

func (s oauth2Errkey) String() string {
	return string(s)
}

const (
	oauth2ErrInvalidRequest          oauth2Errkey = "invalid_request"
	oauth2ErrUnauthorizedClient      oauth2Errkey = "unauthorized_client"
	oauth2ErrAccessDenied            oauth2Errkey = "access_denied"
	oauth2ErrUnsupportedResponseType oauth2Errkey = "unsupported_response_type"
	oauth2ErrInvalidScope            oauth2Errkey = "invalid_scope"
	oauth2ErrServerError             oauth2Errkey = "server_error"
	oauth2ErrTemporalilyUnavailable  oauth2Errkey = "temporarily_unavailable"
	oauth2ErrUnsupportedGrantType    oauth2Errkey = "unsupported_grant_type"
	oauth2ErrInvalidGrant            oauth2Errkey = "invalid_grant"
	oauth2ErrInvalidClient           oauth2Errkey = "invalid_client"
)

var oauth2Errors = map[oauth2Errkey]string{
	oauth2ErrInvalidRequest:          "The request is missing a required parameter, includes an invalid parameter value, includes a parameter more than once, or is otherwise malformed.",
	oauth2ErrUnauthorizedClient:      "The client is not authorized to request a token using this method.",
	oauth2ErrAccessDenied:            "The resource owner or authorization server denied the request.",
	oauth2ErrUnsupportedResponseType: "The authorization server does not support obtaining a token using this method.",
	oauth2ErrInvalidScope:            "The requested scope is invalid, unknown, or malformed.",
	oauth2ErrServerError:             "The authorization server encountered an unexpected condition that prevented it from fulfilling the request.",
	oauth2ErrTemporalilyUnavailable:  "The authorization server is currently unable to handle the request due to a temporary overloading or maintenance of the server.",
	oauth2ErrUnsupportedGrantType:    "The authorization grant type is not supported by the authorization server.",
	oauth2ErrInvalidGrant:            "The provided authorization grant (e.g., authorization code, resource owner credentials) or refresh token is invalid, expired, revoked, does not match the redirection URI used in the authorization request, or was issued to another client.",
	oauth2ErrInvalidClient:           "Client authentication failed (e.g., unknown client, no client authentication included, or unsupported authentication method).",
}

var oauth2ErrLock sync.Mutex

func getOauth2Err(key string) (value string) {
	oauth2ErrLock.Lock()
	value = oauth2Errors[oauth2Errkey(key)]
	oauth2ErrLock.Unlock()
	return
}

// exposes oauth2 server workflow that uses a key/value store for persistence.
// This also allows managing of tockens.
type oauth2 struct {
	store kvStore
}
type oauth2ResponseType uint

const (
	oauth2ResponseData oauth2ResponseType = iota
	oauth2ResponseRedirect
)

type oauth2Context struct {
	kind               oauth2ResponseType
	statusCode         int
	statusText         string
	url                string
	data               map[string]interface{}
	headers            http.Header
	hasError           bool
	errID              string
	internalErr        error
	redirectInFragment bool
}

func (ctx *oauth2Context) init() {
	ctx.kind = oauth2ResponseData
	ctx.statusCode = http.StatusOK
	ctx.data = make(map[string]interface{})
	ctx.headers = make(http.Header)
	ctx.headers.Add(
		"Cache-Control",
		"no-cache, no-store, max-age=0, must-revalidate",
	)
	ctx.headers.Add("Pragma", "no-cache")
	ctx.headers.Add("Expires", "Fri, 01 Jan 1990 00:00:00 GMT")
}

func (ctx *oauth2Context) setErrURI(id, desc, uri, state string) {
	if desc == "" {
		desc = getOauth2Err(id)
	}
	ctx.hasError = true
	ctx.errID = id
	if ctx.statusCode != http.StatusOK {
		ctx.statusText = desc
	}
	ctx.clearData()
	ctx.data[oauth2ParamError] = id
	ctx.data[oauth2ParamErrDesc] = desc
	ctx.data[oauth2ParamErrURI] = uri
	if state != "" {
		ctx.data[oauth2ParamState] = state
	}
}

func (ctx *oauth2Context) setRedirect(uri string) {
	ctx.kind = oauth2ResponseRedirect
	ctx.url = uri
}

func (ctx *oauth2Context) clearData() {
	for k := range ctx.data {
		delete(ctx.data, k)
	}
}

var errNotOauth2RedirectResponse = errors.New("oauth2: not redirect response")

func (ctx *oauth2Context) getRedirectURL() (string, error) {
	if ctx.kind != oauth2ResponseRedirect {
		return "", errNotOauth2RedirectResponse
	}
	link, err := url.Parse(ctx.url)
	if err != nil {
		return "", err
	}

	q := link.Query()

	for k, v := range ctx.data {
		q.Set(k, fmt.Sprint(v))
	}
	link.RawQuery = q.Encode()
	if ctx.redirectInFragment {
		link.RawQuery = ""
		link.Fragment, err = url.QueryUnescape(q.Encode())
		if err != nil {
			return "", err
		}
	}
	return link.String(), nil
}
func (ctx *oauth2Context) commit(w http.ResponseWriter) error {
	if ctx.internalErr != nil {
		// TODO log this?
	}
	for k, h := range ctx.headers {
		for _, v := range h {
			w.Header().Add(k, v)
		}
	}
	switch ctx.kind {
	case oauth2ResponseRedirect:
		link, err := ctx.getRedirectURL()
		if err != nil {
			return err
		}
		w.Header().Add(oauth2ParamLocation, link)
		w.WriteHeader(http.StatusFound)
		return nil
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(ctx.statusCode)
		return json.NewEncoder(w).Encode(ctx.data)
	}
}
