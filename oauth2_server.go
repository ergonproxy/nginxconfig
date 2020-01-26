package main

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

type kvStore interface {
	get(key []byte) ([]byte, error)
	set(key, value []byte) error
	remove(key []byte) error
	serial() (uint64, error)
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
	oauth2ParamLoginUsername = "login_username"
	oauth2ParamLoginPassword = "login_password"
)

// grant types
const (
	oauth2GrantTypeAuthorizationCode = "authorization_code"
	oauth2GrantTypeRefreshToken      = "refresh_token"
	oauth2GrantTypePassword          = "password"
	oauth2GrantTypeClientCredentials = "client_credentials"
	oauth2GrantTypeAssertion         = "assertion"
	oauth2GrantTypeImplicit          = "__implicit"
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

var oauth2ClientPrefix = []byte("/client/")
var oauth2UserPrefix = []byte("/user/")
var oauth2GrantPrefix = []byte("/grant/")
var oauth2TokenPrefix = []byte("/token/")

type oauth2Token struct {
	ID        uint64
	Code      string
	ClientID  oauth2ClientID
	UserID    string
	ExpiresIn int64
	CreatedAT time.Time
	UpdatedAt time.Time
}

type oauth2Grant struct {
	ID             uint64
	Code           string
	Type           string
	UserID         string
	ClientID       oauth2ClientID
	AccessToken    uint64
	AuthorizeToken uint64
	RefreshToken   uint64
	Scope          string
	State          string
	RedirectURL    string
	ExpiresIn      int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type oauth2Client struct {
	ID          oauth2ClientID
	UserID      int64
	Name        string
	Secret      string
	Grants      []uint64
	Tokens      []uint64
	RedirectURL string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type oauth2User struct {
	Email     string
	Grants    []uint64
	Tokens    []uint64
	Clients   []string
	Password  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

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

func getOauth2Err(key oauth2Errkey) (value string) {
	oauth2ErrLock.Lock()
	value = oauth2Errors[oauth2Errkey(key)]
	oauth2ErrLock.Unlock()
	return
}

// exposes oauth2 server workflow that uses a key/value store for persistence.
// This also allows managing of tockens.
type oauth2 struct {
	store             kvStore
	redirectSeparator string
	templates         *template.Template
	tokens            *jwtTokenGen
	expires           int64
	tokenType         string
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

func (ctx *oauth2Context) setErrState(id oauth2Errkey, uri, state string) {
	ctx.setErrURI(id, "", uri, state)
}
func (ctx *oauth2Context) setErrURI(id oauth2Errkey, desc, uri, state string) {
	if desc == "" {
		desc = getOauth2Err(id)
	}
	ctx.hasError = true
	ctx.errID = string(id)
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

type oauth2ClientID string

func createClientID() oauth2ClientID {
	var b [256]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic(err) // if we can't secure create random identifiers then  no need to continue
	}
	return oauth2ClientID(hex.EncodeToString(b[:]))
}

func (o *oauth2) authorize(w http.ResponseWriter, r *http.Request) error {
	_ = r.ParseForm()
	var ctx oauth2Context
	ctx.init()
	redirectURI, err := url.QueryUnescape(r.Form.Get(oauth2ParamRedirectURL))
	if err != nil {
		ctx.setErrState(oauth2ErrInvalidRequest, "", "")
		ctx.internalErr = err
		return ctx.commit(w)
	}
	state := r.Form.Get(oauth2ParamState)
	scope := r.Form.Get(oauth2ParamScope)
	clientID := r.Form.Get(oauth2ParamClientID)
	client, err := o.client(clientID)
	if err != nil {
		id := oauth2ErrServerError
		if err == badger.ErrKeyNotFound {
			id = oauth2ErrUnauthorizedClient
		}
		ctx.setErrState(id, "", state)
		ctx.internalErr = err
		return ctx.commit(w)
	}
	if client.RedirectURL == "" {
		ctx.setErrState(oauth2ErrUnauthorizedClient, "", state)
		return ctx.commit(w)
	}
	if redirectURI == "" && firstURI(client.RedirectURL, o.redirectSeparator) == client.RedirectURL {
		redirectURI = firstURI(client.RedirectURL, o.redirectSeparator)
	}
	if err = validateURIList(client.RedirectURL, redirectURI, o.redirectSeparator); err != nil {
		ctx.setErrState(oauth2ErrInvalidRequest, "", state)
		ctx.internalErr = err
		return ctx.commit(w)
	}
	ctx.setRedirect(redirectURI)

	reqTyp := r.Form.Get(oauth2ParamResponseType)
	var usr *oauth2User
	if r.Method == http.MethodPost {
		username := r.Form.Get(oauth2ParamLoginUsername)
		password := r.Form.Get(oauth2ParamLoginPassword)
		usr, err = o.valid(username, password)
	}
	if usr == nil {
		// serve login page
		return o.templates.ExecuteTemplate(w, "oauth2_login.html", map[string]interface{}{
			"Action": r.URL.String(),
			"Title":  "vince oauth login",
		})
	}
	switch reqTyp {
	case "code":
		grant := new(oauth2Grant)
		grant.Code = o.tokens.Generate(o.claims(usr))
		grant.Scope = scope
		grant.State = state
		grant.ClientID = client.ID
		grant.UserID = usr.Email
		if err := o.saveGrant(grant); err != nil {
			ctx.setErrState(oauth2ErrServerError, "", state)
			ctx.internalErr = err
			return ctx.commit(w)
		}
		usr.Grants = append(usr.Grants, grant.ID)
		if err = o.saveUser(usr); err != nil {
			ctx.setErrState(oauth2ErrServerError, "", state)
			ctx.internalErr = err
			return ctx.commit(w)
		}
		client.Grants = append(client.Grants, grant.ID)
		if err = o.saveClient(client); err != nil {
			ctx.setErrState(oauth2ErrServerError, "", state)
			ctx.internalErr = err
			return ctx.commit(w)
		}
		ctx.data[oauth2ParamCode] = grant.Code
		ctx.data[oauth2ParamState] = state
		return ctx.commit(w)
	case "token":
		ctx.redirectInFragment = true
		grant := new(oauth2Grant)
		grant.Code = o.tokens.Generate(o.claims(usr))
		grant.Type = oauth2GrantTypeImplicit
		grant.Scope = scope
		grant.State = state
		grant.RedirectURL = redirectURI
		grant.ClientID = client.ID
		grant.UserID = usr.Email
		if err = o.finalize(grant, &ctx, usr); err != nil {
			ctx.setErrState(oauth2ErrServerError, "", state)
			ctx.internalErr = err
			return ctx.commit(w)
		}
		if state != "" {
			ctx.data[oauth2ParamState] = state
		}
		return ctx.commit(w)
	default:
		ctx.setErrState(oauth2ErrUnsupportedResponseType, "", state)
		return ctx.commit(w)
	}
}

func (o *oauth2) saveUser(usr *oauth2User) error {
	usr.UpdatedAt = time.Now()
	b, err := json.Marshal(usr)
	if err != nil {
		return err
	}
	return o.store.set(joinSlice(oauth2UserPrefix, []byte(usr.Email)), b)
}

func (o *oauth2) saveClient(c *oauth2Client) error {
	var err error
	if c.ID == "" {
		c.ID = createClientID()
	}
	c.UpdatedAt = time.Now()
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return o.store.set(joinSlice(oauth2ClientPrefix, []byte(c.ID)), b)
}

func (o *oauth2) saveToken(c *oauth2Token) error {
	var err error
	if c.ID == 0 {
		c.ID, err = o.store.serial()
		if err != nil {
			return err
		}
	}
	c.UpdatedAt = time.Now()
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return o.store.set(o.key(oauth2ClientPrefix, c.ID), b)
}

func (o *oauth2) saveGrant(c *oauth2Grant) error {
	var err error
	if c.ID == 0 {
		c.ID, err = o.store.serial()
		if err != nil {
			return err
		}
	}
	c.UpdatedAt = time.Now()
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return o.store.set(o.key(oauth2ClientPrefix, c.ID), b)
}

func (o *oauth2) finalize(auth *oauth2Grant, ctx *oauth2Context, usr *oauth2User) error {
	access := new(oauth2Grant)
	access.ClientID = auth.ClientID
	access.UserID = auth.UserID
	access.RedirectURL = auth.RedirectURL
	access.Scope = auth.Scope
	access.State = auth.State
	access.ExpiresIn = o.expires

	genAccessToken := oauth2Token{
		Code:     o.tokens.Generate(o.claims(usr)),
		ClientID: auth.ClientID,
		UserID:   auth.UserID,
	}

	if err := o.saveToken(&genAccessToken); err != nil {
		return err
	}

	genRefreshToken := oauth2Token{
		Code:     o.tokens.Generate(o.claims(usr)),
		ClientID: auth.ClientID,
		UserID:   auth.UserID,
	}
	if err := o.saveToken(&genRefreshToken); err != nil {
		return err
	}

	access.AccessToken = genAccessToken.ID
	access.RefreshToken = genRefreshToken.ID

	if err := o.saveGrant(access); err != nil {
		return err
	}
	ctx.data[oauth2ParamAccessToken] = genAccessToken.Code
	ctx.data[oauth2ParamTokenType] = o.tokenType
	ctx.data[oauth2ParamExpiresIn] = access.ExpiresIn
	ctx.data[oauth2ParamRefreshToken] = genRefreshToken.Code
	if access.Scope != "" {
		ctx.data[oauth2ParamScope] = access.Scope
	}
	if auth.ID != 0 {
		return o.remove(oauth2GrantPrefix, auth.ID)
	}
	return nil
}

func (o *oauth2) remove(prefix []byte, id uint64) error {
	return o.store.remove(o.key(prefix, id))
}

func (o *oauth2) claims(usr *oauth2User) jwt.StandardClaims {
	now := time.Now()
	return jwt.StandardClaims{
		ExpiresAt: now.Add(365 * 24 * time.Hour).Unix(),
		IssuedAt:  now.Unix(),
		Issuer:    usr.Email,
	}
}

func validateURIList(baseList, redir, sep string) error {
	var list []string
	if sep != "" {
		list = strings.Split(baseList, sep)
	} else {
		list = append(list, baseList)
	}
	for _, item := range list {
		if err := validateURI(item, redir); err == nil {
			return nil
		}
	}
	return fmt.Errorf("%s : %s / %s", "url dot validate", baseList, redir)

}

var (
	errOauth2BlankURL    = errors.New("oauth2: urls can not be blank")
	errOauth2FragmentURL = errors.New("oauth2: url must not include fragment")
)

func validateURI(base, redir string) error {
	if base == "" || redir == "" {
		return errOauth2BlankURL
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return err
	}

	redirectURL, err := url.Parse(redir)
	if err != nil {
		return err
	}

	if baseURL.Fragment != "" || redirectURL.Fragment != "" {
		return errOauth2FragmentURL
	}
	if baseURL.Scheme != redirectURL.Scheme {
		return fmt.Errorf("%s : %s / %s", "scheme mismatch", base, redir)
	}
	if baseURL.Host != redirectURL.Host {
		return fmt.Errorf("%s : %s / %s", "host mismatch", base, redir)
	}

	if baseURL.Path == redirectURL.Path {
		return nil
	}

	reqPrefix := strings.TrimRight(baseURL.Path, "/") + "/"
	if !strings.HasPrefix(redirectURL.Path, reqPrefix) {
		return fmt.Errorf("%s : %s / %s", "path is not a subpath", base, redir)
	}

	for _, s := range strings.Split(strings.TrimPrefix(redirectURL.Path, reqPrefix), "/") {
		if s == ".." {
			return fmt.Errorf("%s : %s / %s", "subpath cannot contain path traversial", base, redir)
		}
	}
	return nil
}

// firstURI returns the first string after spliting base using sep. if sep is an empty string
// then base is returned.
//
// This is used to find the first redirect url from a url list.
func firstURI(base, sep string) string {
	if sep != "" {
		l := strings.Split(base, sep)
		if len(l) > 0 {
			return l[0]
		}
	}
	return base
}

func (o *oauth2) client(id string) (*oauth2Client, error) {
	b, err := o.store.get(joinSlice(oauth2ClientPrefix, []byte(id)))
	if err != nil {
		return nil, err
	}
	var c oauth2Client
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (o *oauth2) user(email string) (*oauth2User, error) {
	b, err := o.store.get(joinSlice(oauth2UserPrefix, []byte(email)))
	if err != nil {
		return nil, err
	}
	var u oauth2User
	if err := json.Unmarshal(b, &u); err != nil {
		return nil, err
	}
	return nil, nil
}

func (o *oauth2) key(prefix []byte, id uint64) []byte {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], id)
	return joinSlice(prefix, b[:])
}

func (o *oauth2) valid(username, password string) (*oauth2User, error) {
	usr, err := o.user(username)
	if err != nil {
		return nil, err
	}
	err = compareHashedString(usr.Password, password)
	if err != nil {
		return nil, err
	}
	return usr, nil
}

func compareHashedString(hashed, str string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(str))
}
