package main

import (
	"bytes"
	"crypto/rand"
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
	"github.com/ergongate/vince/templates"
	"golang.org/x/crypto/bcrypt"
)

var _ kvStore = (*kvStoreDB)(nil)

type kvStore interface {
	get(key []byte) ([]byte, error)
	set(key, value []byte) error
	remove(key []byte) error
	onSet(func([]byte))
	onRemove(func([]byte))
	clone() kvStore // without callbacks
}

type kvStoreDB struct {
	db              *badger.DB
	setCallbacks    []func([]byte)
	removeCallbacks []func([]byte)
}

func (kv *kvStoreDB) clone() kvStore {
	return &kvStoreDB{db: kv.db}
}

func (kv *kvStoreDB) set(key, value []byte) error {
	err := kv.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
	if err != nil {
		return err
	}
	if len(kv.setCallbacks) > 0 {
		go kv.setCb(key)
	}
	return nil
}

func (kv *kvStoreDB) remove(key []byte) error {
	err := kv.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
	if err != nil {
		return err
	}
	if len(kv.setCallbacks) > 0 {
		go kv.removeCb(key)
	}
	return nil
}

func (kv *kvStoreDB) get(key []byte) (value []byte, err error) {
	err = kv.db.View(func(txn *badger.Txn) error {
		i, err := txn.Get(key)
		if err != nil {
			return err
		}
		value, err = i.ValueCopy(nil)
		return err
	})
	return
}

func (kv *kvStoreDB) onSet(fn func([]byte)) {
	kv.setCallbacks = append(kv.setCallbacks, fn)
}

func (kv *kvStoreDB) onRemove(fn func([]byte)) {
	kv.removeCallbacks = append(kv.removeCallbacks, fn)
}

func (kv *kvStoreDB) setCb(key []byte) {
	for _, v := range kv.setCallbacks {
		v(key)
	}
}

func (kv *kvStoreDB) removeCb(key []byte) {
	for _, v := range kv.removeCallbacks {
		v(key)
	}
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
var oauth2CSRFTokenPrefix = []byte("/csrf/")

type oauth2Token struct {
	Code      string
	ClientID  oauth2ClientID
	UserID    string
	ExpiresIn int64
	CreatedAT time.Time
	UpdatedAt time.Time
}

type oauth2Grant struct {
	Code           string
	Type           string
	UserID         string
	ClientID       oauth2ClientID
	AccessToken    string
	AuthorizeToken string
	RefreshToken   string
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
	Grants      []string
	Tokens      []string
	RedirectURL string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type oauth2User struct {
	Email     string
	Grants    []string
	Tokens    []string
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
// This also allows managing of tokens.
type oauth2 struct {
	store             kvStore
	redirectSeparator string
	templates         *template.Template
	opts              oauth2Option
}

func (o *oauth2) init(store kvStore, opts oauth2Option) error {
	tpl, err := templates.HTML()
	if err != nil {
		return err
	}
	csrf, err := store.get(oauth2CSRFTokenPrefix)
	if err != nil {
		if err != badger.ErrKeyNotFound {
			return err
		}
		if opts.CsrfSecret == nil {
			var secret [32]byte
			_, err := rand.Read(secret[:])
			if err != nil {
				return err
			}
			opts.CsrfSecret = secret[:]
			if err := store.set(oauth2CSRFTokenPrefix, []byte(opts.CsrfSecret)); err != nil {
				return err
			}
		}
	} else {
		if opts.CsrfSecret != nil {
			if !bytes.Equal(csrf, opts.CsrfSecret) {
				if err := store.set(oauth2CSRFTokenPrefix, []byte(opts.CsrfSecret)); err != nil {
					return err
				}
			}
		} else {
			opts.CsrfSecret = csrf
		}
	}
	o.templates = tpl
	o.opts = opts
	return nil
}

func generateCSRFToken() (string, error) {
	var secret [32]byte
	_, err := rand.Read(secret[:])
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(secret[:]), nil
}

type oauth2Option struct {
	RedirectSeparator   string   `json:"redirect_separator"`
	AuthorizationExpire int64    `json:"authorization_expire"`
	AccessExpire        int64    `json:"access_expire"`
	AllowGetAccess      bool     `json:"allow_get_access"`
	AllowedAccessType   []string `json:"allowed_access_type"`
	TokenType           string   `json:"token_type"`
	ProviderName        string   `json:"provider_name"`
	AuthEndpoint        string   `json:"auth_endpoint"`
	TokenEndpoint       string   `json:"token_endpoint"`
	InfoEndpoint        string   `json:"info_endpoint"`
	Session             struct {
		Path     string        `json:"session_path"`
		MaxAge   int           `json:"session_max_age"`
		Domain   string        `json:"session_domain"`
		Secure   bool          `json:"session_secure"`
		HTTPOnly bool          `json:"session_hhhponly"`
		Name     string        `json:"session_name"`
		SameSite http.SameSite `json:"session_same_site"`
	}
	CsrfSecret []byte `json:"csrf_secret"`
}

func (o *oauth2Option) init() {
	o.AllowedAccessType = []string{"authorization_code", "refresh_token", "password", "client_credentials", "assertion"}
	o.TokenType = "Bearer"
	o.AuthorizationExpire = 200
	o.AccessExpire = 200
	o.AuthEndpoint = "/authorize"
	o.TokenEndpoint = "/tokens"
	o.InfoEndpoint = "/info"
}

func (o *oauth2Option) load(r *rule) error {
	switch r.name {
	}
	return nil
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

func (o *oauth2) generate() (string, error) {
	var b [256]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
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
		return o.templates.ExecuteTemplate(w, "oauth/login.html", map[string]interface{}{
			"Action": r.URL.String(),
			"Title":  "vince oauth login",
		})
	}
	switch reqTyp {
	case "code":
		code, err := o.generate()
		if err != nil {
			ctx.setErrState(oauth2ErrServerError, "", state)
			ctx.internalErr = err
			return ctx.commit(w)
		}
		grant := new(oauth2Grant)
		grant.Code = code
		grant.Scope = scope
		grant.State = state
		grant.ClientID = client.ID
		grant.UserID = usr.Email
		if err := o.saveGrant(grant); err != nil {
			ctx.setErrState(oauth2ErrServerError, "", state)
			ctx.internalErr = err
			return ctx.commit(w)
		}
		usr.Grants = append(usr.Grants, grant.Code)
		if err = o.saveUser(usr); err != nil {
			ctx.setErrState(oauth2ErrServerError, "", state)
			ctx.internalErr = err
			return ctx.commit(w)
		}
		client.Grants = append(client.Grants, grant.Code)
		if err = o.saveClient(client); err != nil {
			ctx.setErrState(oauth2ErrServerError, "", state)
			ctx.internalErr = err
			return ctx.commit(w)
		}
		ctx.data[oauth2ParamCode] = grant.Code
		ctx.data[oauth2ParamState] = state
		return ctx.commit(w)
	case "token":
		code, err := o.generate()
		if err != nil {
			ctx.setErrState(oauth2ErrServerError, "", state)
			ctx.internalErr = err
			return ctx.commit(w)
		}
		ctx.redirectInFragment = true
		grant := new(oauth2Grant)
		grant.Code = code
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
	c.UpdatedAt = time.Now()
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return o.store.set(joinSlice(oauth2ClientPrefix, []byte(c.Code)), b)
}

func (o *oauth2) saveGrant(c *oauth2Grant) error {
	var err error
	c.UpdatedAt = time.Now()
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return o.store.set(joinSlice(oauth2ClientPrefix, []byte(c.Code)), b)
}

func (o *oauth2) finalize(auth *oauth2Grant, ctx *oauth2Context, usr *oauth2User) error {
	access := new(oauth2Grant)
	access.ClientID = auth.ClientID
	access.UserID = auth.UserID
	access.RedirectURL = auth.RedirectURL
	access.Scope = auth.Scope
	access.State = auth.State
	access.ExpiresIn = o.opts.AccessExpire
	code, err := o.generate()
	if err != nil {
		return err
	}
	genAccessToken := oauth2Token{
		Code:     code,
		ClientID: auth.ClientID,
		UserID:   auth.UserID,
	}

	if err := o.saveToken(&genAccessToken); err != nil {
		return err
	}
	code, err = o.generate()
	if err != nil {
		return err
	}

	genRefreshToken := oauth2Token{
		Code:     code,
		ClientID: auth.ClientID,
		UserID:   auth.UserID,
	}
	if err := o.saveToken(&genRefreshToken); err != nil {
		return err
	}

	access.AccessToken = genAccessToken.Code
	access.RefreshToken = genRefreshToken.Code

	if err := o.saveGrant(access); err != nil {
		return err
	}
	ctx.data[oauth2ParamAccessToken] = genAccessToken.Code
	ctx.data[oauth2ParamTokenType] = o.opts.TokenType
	ctx.data[oauth2ParamExpiresIn] = access.ExpiresIn
	ctx.data[oauth2ParamRefreshToken] = genRefreshToken.Code
	if access.Scope != "" {
		ctx.data[oauth2ParamScope] = access.Scope
	}
	if auth.Code != "" {
		return o.store.remove(joinSlice(oauth2GrantPrefix, []byte(auth.Code)))
	}
	return nil
}

func validateURIList(baseList, redirect, sep string) error {
	var list []string
	if sep != "" {
		list = strings.Split(baseList, sep)
	} else {
		list = append(list, baseList)
	}
	for _, item := range list {
		if err := validateURI(item, redirect); err == nil {
			return nil
		}
	}
	return fmt.Errorf("%s : %s / %s", "url dot validate", baseList, redirect)

}

var (
	errOauth2BlankURL    = errors.New("oauth2: urls can not be blank")
	errOauth2FragmentURL = errors.New("oauth2: url must not include fragment")
)

func validateURI(base, redirect string) error {
	if base == "" || redirect == "" {
		return errOauth2BlankURL
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return err
	}

	redirectURL, err := url.Parse(redirect)
	if err != nil {
		return err
	}

	if baseURL.Fragment != "" || redirectURL.Fragment != "" {
		return errOauth2FragmentURL
	}
	if baseURL.Scheme != redirectURL.Scheme {
		return fmt.Errorf("%s : %s / %s", "scheme mismatch", base, redirect)
	}
	if baseURL.Host != redirectURL.Host {
		return fmt.Errorf("%s : %s / %s", "host mismatch", base, redirect)
	}

	if baseURL.Path == redirectURL.Path {
		return nil
	}

	reqPrefix := strings.TrimRight(baseURL.Path, "/") + "/"
	if !strings.HasPrefix(redirectURL.Path, reqPrefix) {
		return fmt.Errorf("%s : %s / %s", "path is not a subpath", base, redirect)
	}

	for _, s := range strings.Split(strings.TrimPrefix(redirectURL.Path, reqPrefix), "/") {
		if s == ".." {
			return fmt.Errorf("%s : %s / %s", "subpath cannot contain path traversal", base, redirect)
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
