package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/format/pktline"
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
	dir     string
	auth    func(username, password string) bool
}

func (r *repoLoader) init(dir string, auth func(username, password string) bool) {
	r.allowed = []string{"vince", "project", "project.git"}
	r.dir = dir
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0777)
		if err != nil {
			panic(err)
		}
	}
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
	for _, a := range r.allowed {
		a = filepath.Join(r.dir, a)
		if a == e {
			var s *git.Repository
			_, err := os.Stat(e)
			if os.IsNotExist(err) {
				err = os.Mkdir(e, 0777)
				if err != nil {
					return nil, err
				}
				s, err = git.PlainInit(e, false)
			} else {
				s, err = git.PlainOpen(e)
			}
			if err != nil {
				return nil, err
			}
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
	g := e.Group("/git")
	g.GET("/:project/info/refs", echo.WrapHandler(
		http.HandlerFunc(o.refs),
	))
	g.POST("/:project/git-upload-pack", echo.WrapHandler(
		http.HandlerFunc(o.up)),
	)
	g.POST("/:project/git-receive-pack", echo.WrapHandler(
		http.HandlerFunc(o.down),
	))
}

func (o *gitOps) refs(w http.ResponseWriter, r *http.Request) {
	rpc := r.URL.Query().Get("service")
	if !o.setup(w, r) {
		return
	}
	ep, err := o.endpoint(r, "/info/refs")
	if err != nil {
		// TODO?
		return
	}
	h := w.Header()
	h.Add(HeaderContentType, fmt.Sprintf("application/x-%s-advertisement", rpc))
	h.Add("Cache-Control", "no-cache")
	switch rpc {
	case "git-upload-pack":
		w.WriteHeader(http.StatusOK)
		o.service(w, rpc)
		err := o.upload(w, ep)
		if err != nil {
			//TODO
		}
	case "git-receive-pack":
		w.WriteHeader(http.StatusOK)
		o.service(w, rpc)
		err := o.receive(w, ep)
		if err != nil {
			//TODO
		}
	default:
		http.Error(w, "Not Found", 404)
		return
	}
}

func (o *gitOps) service(w io.Writer, rpc string) {
	enc := pktline.NewEncoder(w)
	enc.EncodeString(fmt.Sprintf("# service=%s\n", rpc))
	enc.Flush()
}

func (o *gitOps) upload(w io.Writer, ep *transport.Endpoint) error {
	s, err := o.server.NewUploadPackSession(ep, nil)
	if err != nil {
		return err
	}
	ar, err := s.AdvertisedReferences()
	if err != nil {
		return err
	}
	return ar.Encode(w)
}

func (o *gitOps) receive(w io.Writer, ep *transport.Endpoint) error {
	s, err := o.server.NewReceivePackSession(ep, nil)
	if err != nil {
		return err
	}
	ar, err := s.AdvertisedReferences()
	if err != nil {
		return err
	}
	return ar.Encode(w)
}

func (o *gitOps) up(w http.ResponseWriter, r *http.Request) {
	if !o.setup(w, r) {
		return
	}
	s, err := o.upSession(r)
	if err != nil {
		o.e500(w)
		return
	}
	rd := r.Body
	if r.Header.Get(HeaderContentEncoding) == "gzip" {
		rd, err = gzip.NewReader(r.Body)
		if err != nil {
			o.e500(w)
			return
		}
	}
	req := packp.NewUploadPackRequest()
	if err := req.Decode(rd); err != nil {
		o.e500(w)
		return
	}
	resp, err := s.UploadPack(r.Context(), req)
	if err != nil {
		o.e500(w)
		return
	}
	if err = resp.Encode(w); err != nil {
		return
	}
	o.setHeaders(w.Header(), "git-upload-pack")
	w.WriteHeader(http.StatusOK)
}

func (o *gitOps) e500(w http.ResponseWriter) {
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (o *gitOps) down(w http.ResponseWriter, r *http.Request) {
	if !o.setup(w, r) {
		return
	}
	s, err := o.downSession(r)
	if err != nil {
		return
	}
	req := packp.NewReferenceUpdateRequest()
	rd := r.Body
	if r.Header.Get(HeaderContentEncoding) == "gzip" {
		rd, err = gzip.NewReader(r.Body)
		if err != nil {
			o.e500(w)
			return
		}
	}
	if err := req.Decode(rd); err != nil {
		o.e500(w)
		return
	}
	resp, err := s.ReceivePack(r.Context(), req)
	if err != nil {
		o.e500(w)
		return
	}
	if err = resp.Encode(w); err != nil {
		o.e500(w)
		return
	}
	o.setHeaders(w.Header(), "git-receive-pack")
	w.WriteHeader(http.StatusOK)
}

func (o *gitOps) setup(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get(HeaderAuthorization) == "" {
		w.Header()["WWW-Authenticate"] = []string{`Basic realm=""`}
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}
	return true
}

func (o *gitOps) endpoint(r *http.Request, service string) (*transport.Endpoint, error) {
	var basic basicAuth
	basic.init(r, false)
	p := strings.TrimSuffix(r.URL.Path, service)
	p = strings.TrimPrefix(p, "/git")
	p = filepath.Join(o.opts.dir, filepath.FromSlash(p))
	e := &transport.Endpoint{
		Protocol: "http",
		User:     basic.UserName,
		Password: basic.Password,
		Path:     filepath.Clean(p),
	}
	return e, nil
}

func (o *gitOps) upSession(r *http.Request) (transport.UploadPackSession, error) {
	ep, err := o.endpoint(r, "/git-upload-pack")
	if err != nil {
		return nil, err
	}
	return o.server.NewUploadPackSession(ep, nil)
}

func (o *gitOps) downSession(r *http.Request) (transport.ReceivePackSession, error) {
	ep, err := o.endpoint(r, "/git-receive-pack")
	if err != nil {
		return nil, err
	}
	return o.server.NewReceivePackSession(ep, nil)
}

func (o *gitOps) setHeaders(h http.Header, cmd string) {
	h.Add(HeaderContentType, fmt.Sprintf("application/x-%s-result", cmd))
	h.Add("Cache-Control", "no-cache")
}
