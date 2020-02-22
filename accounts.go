package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ergongate/vince/templates"
	"golang.org/x/crypto/bcrypt"
)

// accounts are implemented specifically to accommodate token based
// authentication so the api shares oauth struct.
//
// users are still expected to have password protection which is not stored as
// plain text for security reasons.

func (o *oauth2) register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := r.ParseForm(); err != nil {
		logDebug(ctx, fmt.Sprintf("oauth: error parsing form %v", err))
		o.invalid(w, r)
		return
	}
	email := r.Form.Get("email")
	password := r.Form.Get("password")
	confirm := r.Form.Get("confirm_password")
	if password == "" || confirm == "" || email == "" {
		o.invalid(w, r)
		return
	}
	if password != confirm {
		o.invalid(w, r)
		return
	}
	hash, err := hashString(password)
	if err != nil {
		logError(ctx, fmt.Sprintf("oauth: error  hashing password %v", err))
		e500(w)
		return
	}
	u := oauth2User{
		Email:     email,
		Password:  hash,
		CreatedAt: time.Now(),
	}
	if err := o.saveUser(&u); err != nil {
		logError(ctx, fmt.Sprintf("oauth: error  saving user %v", err))
		e500(w)
		return
	}
	if err := templates.ExecHTML(w, "oauth/register.html", nil); err != nil {
		logError(ctx, fmt.Sprintf("oauth: error  saving user %v", err))
		e500(w)
		return
	}
}

func hashString(secret string) (string, error) {
	s, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(s), nil
}

func (o *oauth2) invalid(w http.ResponseWriter, r *http.Request) {
}
