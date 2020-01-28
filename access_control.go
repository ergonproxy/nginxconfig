package main

import (
	"bytes"
	"encoding/json"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
)

var accessControlPrefix = []byte("/access/")
var accessControlModelPrefix = []byte("/access/model")

var _ persist.Adapter = (*accessControlAdapter)(nil)

type accessControlAdapter struct {
	store kvStore
	file  []byte
}

func (a *accessControlAdapter) LoadPolicy(m model.Model) error {
	b, err := a.store.get(joinSlice(accessControlModelPrefix, a.file))
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &m)
}

func (a *accessControlAdapter) SavePolicy(m model.Model) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return a.store.set(joinSlice(accessControlModelPrefix, a.file), b)
}

func (a *accessControlAdapter) AddPolicy(sec string, ptype string, rule []string) error {
	m := make(model.Model)
	if err := a.LoadPolicy(m); err != nil {
		return err
	}
	m.AddPolicy(sec, ptype, rule)
	return a.SavePolicy(m)
}

func (a *accessControlAdapter) RemovePolicy(sec string, ptype string, rule []string) error {
	m := make(model.Model)
	if err := a.LoadPolicy(m); err != nil {
		return err
	}
	m.RemovePolicy(sec, ptype, rule)
	return a.SavePolicy(m)
}

func (a *accessControlAdapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	m := make(model.Model)
	if err := a.LoadPolicy(m); err != nil {
		return err
	}
	m.RemoveFilteredPolicy(sec, ptype, fieldIndex, fieldValues...)
	return a.SavePolicy(m)
}

type accessControl struct {
	adopt   *accessControlAdapter
	enforce *casbin.Enforcer
}

func (a *accessControl) init(adopt *accessControlAdapter) error {
	e := new(casbin.Enforcer)
	if err := e.InitWithModelAndAdapter(make(model.Model), adopt); err != nil {
		return err
	}
	a.adopt = adopt
	a.enforce = e
	adopt.store.onSet(a.reload)
	adopt.store.onRemove(a.reload)
	return nil
}

func (a *accessControl) with(file string) (*accessControl, error) {
	n := new(accessControl)
	err := n.init(&accessControlAdapter{file: []byte(file), store: a.adopt.store.clone()})
	return n, err
}

func (a *accessControl) Enforce(vals ...interface{}) (bool, error) {
	return a.enforce.Enforce(vals...)
}

func (a *accessControl) reload(key []byte) {
	if bytes.HasPrefix(key, accessControlModelPrefix) {
		err := a.enforce.LoadPolicy()
		if err != nil {
			// TODO:(gernest) log error
		}
	}
}
