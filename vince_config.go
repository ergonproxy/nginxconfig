package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dgraph-io/badger/v2"
	"github.com/urfave/cli/v2"
)

type vinceConfiguration struct {
	// The working directory where vince stores configuration and all the databases
	// that vince uses.
	//
	// By default this is the directory in which vince.conf is specified.
	dir  string
	dirs struct {
		metrics string
		auth    string
		backups string
		vince   string
	}
	// vince.conf
	confFile    string
	defaultPort int
	management  struct {
		enabled bool
		port    int
	}
}

func (c *vinceConfiguration) setup() error {
	fi, err := os.Stat(c.dir)
	if err != nil {
		return err
	}
	c.dirs.metrics = filepath.Join(c.dir, "metrics")
	c.dirs.auth = filepath.Join(c.dir, "auth")
	c.dirs.backups = filepath.Join(c.dir, "backups")
	c.dirs.vince = filepath.Join(c.dir, "vince")
	return c.checkDirs(fi.Mode(),
		c.dirs.auth,
		c.dirs.metrics,
		c.dirs.backups,
		c.dirs.vince,
	)
}

func (c *vinceConfiguration) checkDirs(mode os.FileMode, dirs ...string) error {
	for _, dir := range dirs {
		if err := c.checkDir(dir, mode); err != nil {
			return err
		}
	}
	return nil
}

func (c *vinceConfiguration) checkDir(path string, mode os.FileMode) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(path, mode)
		}
	}
	if !info.IsDir() {
		return fmt.Errorf("vince: %q is not a directory", path)
	}
	return nil
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
	opts.Dir = filepath.Join(dir...)
	opts.Logger = nil // don't log its very verbose
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

func getConfig(ctx *cli.Context) (*vinceConfiguration, error) {
	file := ctx.String("c")
	var c vinceConfiguration
	c.management.enabled = true // TODO make this configurable
	c.management.port = 9000
	c.defaultPort = ctx.Int("p")
	if file != "" {
		stat, err := os.Stat(file)
		if err != nil {
			return nil, err
		}
		if !stat.IsDir() {
			return nil, errors.New("vince: -c flag should be a valid directory")
		}
		c.dir = file
		c.confFile = filepath.Join(file, "conf", "vince.conf")
		return &c, nil
	}
	for _, file := range defaultConfigFiles() {
		_, err := os.Stat(file)
		if err == nil {
			c.dir = filepath.Dir(file)
			c.confFile = file
			return &c, nil
		}
	}
	return nil, errors.New("vince: missing configuration file")
}
