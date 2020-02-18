package main

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ergongate/vince/buffers"
	"github.com/labstack/echo/v4"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/protocol/packp"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/server"
)

// This provide api for managing vince configuration directory. By default
// configuration are managed as a git repository. This allows events to be fired
// upon change on the configuration file a.k.a hooks.
type gitOps struct {
	opts   gitOpsOptions
	repos  repoLoader
	server transport.Transport
}

type gitOpsOptions struct {
	dir  string
	auth bool
}

type repoLoader struct {
	allowed []string
	paths   *sync.Map
	dir     string
	auth    func(username, password string) bool
}

func (r *repoLoader) init(dir string, auth func(username, password string) bool) {
	r.paths = new(sync.Map)
	r.allowed = []string{"config"}
	r.dir = dir
	r.auth = auth
}

func (r *repoLoader) Load(ep *transport.Endpoint) (storer.Storer, error) {
	o, err := r.load(ep)
	if err != nil {
		return nil, err
	}
	return o.Storer, nil
}

func (r *repoLoader) load(ep *transport.Endpoint) (*git.Repository, error) {
	if r.auth != nil {
		if ep.User == "" || ep.Password == "" {
			return nil, transport.ErrAuthenticationRequired
		}
		if !r.auth(ep.User, ep.Password) {
			return nil, transport.ErrAuthorizationFailed
		}
	}
	e := filepath.FromSlash(ep.Path)
	e = filepath.Join(r.dir, e)
	if s, ok := r.paths.Load(e); ok {
		return s.(*git.Repository), nil
	}
	for _, a := range r.allowed {
		if a == e {
			s, err := git.PlainOpen(e)
			if err != nil {
				if err != git.ErrRepositoryNotExists {
					return nil, err
				}
				s, err = git.PlainInit(e, false)
				if err != nil {
					return nil, err
				}
				r.paths.Store(e, s)
				return s, nil
			}
			r.paths.Store(e, s)
			return s, nil
		}
	}
	return nil, transport.ErrRepositoryNotFound
}

func (o *gitOps) init(opts gitOpsOptions) (err error) {
	o.opts = opts
	o.repos.init(opts.dir, nil)
	o.server = server.NewServer(&o.repos)
	return
}

func (o *gitOps) handler(e *echo.Echo) {
	e.GET("/:project/info/refs", echo.WrapHandler(
		http.HandlerFunc(o.refs),
	))
	e.GET("/git-upload-pack", echo.WrapHandler(
		http.HandlerFunc(o.up)),
	)
	e.GET("/git-receive-pack", echo.WrapHandler(
		http.HandlerFunc(o.down),
	))
}

func (o *gitOps) refs(w http.ResponseWriter, r *http.Request) {
	if !o.setup(w, r) {
		return
	}
	ep, err := o.endpoint(r)
	if err != nil {
		// TODO?
		return
	}
	repo, err := o.repos.load(ep)
	if err != nil {
		// TODO?
		return
	}
	ad := packp.NewAdvRefs()
	refs, err := repo.References()
	if err != nil {
		return
	}
	err = refs.ForEach(func(p *plumbing.Reference) error {
		return ad.AddReference(p)
	})
	if err != nil {
		return
	}
	o.setHeaders(w.Header(), "")
	w.WriteHeader(http.StatusOK)
	if err = ad.Encode(w); err != nil {
		// TODO?
	}
}

func (o *gitOps) up(w http.ResponseWriter, r *http.Request) {
	if !o.setup(w, r) {
		return
	}
	s, err := o.upSession(r)
	if err != nil {
		return
	}
	buf := buffers.GetBytes()
	defer buffers.PutBytes(buf)
	ar, err := s.AdvertisedReferences()
	if err != nil {
		return
	}
	if err := ar.Encode(buf); err != nil {
		return
	}
	req := packp.NewUploadPackRequest()
	if err := req.Decode(r.Body); err != nil {
		return
	}
	resp, err := s.UploadPack(r.Context(), req)
	if err != nil {
		return
	}
	if err = resp.Encode(buf); err != nil {
		return
	}
	o.setHeaders(w.Header(), "git-upload-pack")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, buf)
}

func (o *gitOps) down(w http.ResponseWriter, r *http.Request) {
	if !o.setup(w, r) {
		return
	}
	s, err := o.downSession(r)
	if err != nil {
		return
	}
	ar, err := s.AdvertisedReferences()
	if err != nil {
		return
	}
	buf := buffers.GetBytes()
	defer buffers.PutBytes(buf)

	if err := ar.Encode(buf); err != nil {
		return
	}
	req := packp.NewReferenceUpdateRequest()
	if err := req.Decode(r.Body); err != nil {
		return
	}
	resp, err := s.ReceivePack(r.Context(), req)
	if err != nil {
		return
	}
	if err = resp.Encode(buf); err != nil {
		return
	}
	o.setHeaders(w.Header(), "git-receive-pack")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, buf)
}

func (o *gitOps) setup(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get(HeaderAuthorization) == "" {
		w.Header()["WWW-Authenticate"] = []string{`Basic realm=""`}
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}
	return true
}

func (o *gitOps) endpoint(r *http.Request) (*transport.Endpoint, error) {
	var basic basicAuth
	basic.init(r, false)
	e := &transport.Endpoint{
		Protocol: "http",
		User:     basic.UserName,
		Password: basic.Password,
	}
	p := strings.Split(r.URL.Path, "/") // TODO be robust and remove service prefix
	if len(p) > 1 {
		e.Path = p[1]
	}
	return e, nil
}

func (o *gitOps) upSession(r *http.Request) (transport.UploadPackSession, error) {
	ep, err := o.endpoint(r)
	if err != nil {
		return nil, err
	}
	return o.server.NewUploadPackSession(ep, nil)
}

func (o *gitOps) downSession(r *http.Request) (transport.ReceivePackSession, error) {
	ep, err := o.endpoint(r)
	if err != nil {
		return nil, err
	}
	return o.server.NewReceivePackSession(ep, nil)
}

func (o *gitOps) setHeaders(h http.Header, cmd string) {
	h.Add(HeaderContentType, fmt.Sprintf("application/x-%s-result", cmd))
	h.Add("Cache-Control", "no-cache")
}
