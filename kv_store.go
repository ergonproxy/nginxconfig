package main

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/hashicorp/raft"
)

const (
	retainSnapshotCount = 2
	raftTimeout         = 10 * time.Second
)

var errNotLeader = errors.New("kv: Not a leader")

type kv struct {
	store *store
	raft  *raft.Raft
}

type command struct {
	Op         string
	Key, Value string
}

func (s *kv) Set(key, value string) error {
	if s.raft.State() != raft.Leader {
		return errNotLeader
	}
	c := &command{
		Op:    "set",
		Key:   key,
		Value: value,
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	f := s.raft.Apply(b, raftTimeout)
	return f.Error()
}
