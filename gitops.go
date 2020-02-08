package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ergongate/vince/buffers"
	"github.com/labstack/echo/v4"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/protocol/packp"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
)

// This provide api for managing vince configuration directory. By default
// configuration are managed as a git repository. This allows events to be fired
// upon change on the configuration file a.k.a hooks.
type gitOps struct {
	opts   gitOpsOptions
	repo   *git.Repository
	server transport.Transport
	hand   http.Handler
}

type gitOpsOptions struct {
	dir  string
	repo struct {
		path   string
		remote string
		clone  git.CloneOptions
		pull   git.PullOptions
		latest bool
	}
}

func (o gitOpsOptions) path() string {
	return filepath.Join(o.dir, o.repo.path)
}

func (o *gitOps) init(ctx context.Context, opts gitOpsOptions) (err error) {
	o.opts = opts
	path := opts.path()
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		if opts.repo.remote == "" {
			return err
		}
		// we are cloning the new repo repo from remote
		o.repo, err = git.PlainCloneContext(ctx, path, false, &opts.repo.clone)
	} else {
		if !stat.IsDir() {
			return errors.New("gitops: repository path is not a directory")
		}
		o.repo, err = git.PlainOpen(opts.path())
		if err != nil {
			return
		}
	}
	e := echo.New()
	e.GET("/info/refs", echo.WrapHandler(http.HandlerFunc(o.refs)))
	e.GET("/git-upload-pack", echo.WrapHandler(http.HandlerFunc(o.up)))
	e.GET("/git-receive-pack", echo.WrapHandler(http.HandlerFunc(o.down)))
	o.hand = e
	return
}

func (o *gitOps) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	o.hand.ServeHTTP(w, r)
}

func (o *gitOps) refs(w http.ResponseWriter, r *http.Request) {
	ad := packp.NewAdvRefs()
	refs, err := o.repo.References()
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

func (o *gitOps) endpoint(r *http.Request) (*transport.Endpoint, error) {
	return nil, nil
}

func (o *gitOps) upSession(r *http.Request) (transport.UploadPackSession, error) {
	return nil, nil
}

func (o *gitOps) downSession(r *http.Request) (transport.ReceivePackSession, error) {
	return nil, nil
}

func (o *gitOps) setHeaders(h http.Header, cmd string) {
	h.Add(HeaderContentType, fmt.Sprintf("application/x-%s-result", cmd))
	h.Add("Cache-Control", "no-cache")
}
