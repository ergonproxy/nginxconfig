package main

import "github.com/dgraph-io/badger/v2"

import "fmt"

import "strings"

import "path"

type vinceConfiguration struct {
	// The working directory where vince stores configuration and all the databases
	// that vince uses.
	//
	// By default this is the directory in which vince.conf is specified.
	dir string

	// vince.conf
	confFile string
}

type vinceDatabases struct {
	raft struct {
		stable *badger.DB
		logs   *badger.DB
		snap   *badger.DB
	}
	kv     *badger.DB
	config *badger.DB
	auth   *badger.DB
}

func (db *vinceDatabases) init(dir string, opts badger.Options) (err error) {
	db.raft.stable, err = db.open(opts, dir, "raft", "stable")
	if err != nil {
		return
	}
	db.raft.logs, err = db.open(opts, dir, "raft", "logs")
	if err != nil {
		return
	}
	db.raft.logs, err = db.open(opts, dir, "raft", "snaps")
	if err != nil {
		return
	}
	db.kv, err = db.open(opts, dir, "kv")
	if err != nil {
		return
	}
	db.config, err = db.open(opts, dir, "config")
	if err != nil {
		return
	}
	db.auth, err = db.open(opts, dir, "auth")
	return
}

func (db *vinceDatabases) open(opts badger.Options, dir ...string) (*badger.DB, error) {
	opts.Dir = path.Join(dir...)
	return badger.Open(opts)
}

func (db *vinceDatabases) Close() error {
	var errs []string
	if db.raft.stable != nil {
		if err := db.raft.stable.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if db.raft.logs != nil {
		if err := db.raft.logs.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if db.raft.snap != nil {
		if err := db.raft.snap.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if db.kv != nil {
		if err := db.kv.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if db.config != nil {
		if err := db.config.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if db.auth != nil {
		if err := db.auth.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if errs != nil {
		return fmt.Errorf("vince: error trying to close databases %q", strings.Join(errs, ","))
	}
	return nil
}
