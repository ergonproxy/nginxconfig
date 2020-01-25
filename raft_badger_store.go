package main

import (
	"encoding/binary"
	"sync"

	"github.com/dgraph-io/badger/v2"
	"github.com/hashicorp/raft"
)

var stablePrefix = []byte("/stable")
var _ raft.StableStore = (*store)(nil)

type store struct {
	db    *badger.DB
	cache *sync.Map
}

func (s *store) Set(key, value []byte) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(joinSlice(stablePrefix, key), value)
	})
}

func (s *store) Get(key []byte) (value []byte, err error) {
	err = s.db.View(func(txn *badger.Txn) error {
		i, err := txn.Get(joinSlice(stablePrefix, key))
		if err != nil {
			return err
		}
		value = make([]byte, i.ValueSize())
		_, err = i.ValueCopy(value)
		return err
	})
	return
}

func (s *store) SetUint64(key []byte, value uint64) error {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], value)
	return s.Set(key, b[:])
}

func (s *store) GetUint64(key []byte) (value uint64, err error) {
	err = s.db.View(func(txn *badger.Txn) error {
		i, err := txn.Get(joinSlice(stablePrefix, key))
		if err != nil {
			return err
		}
		return i.Value(func(val []byte) error {
			value = binary.BigEndian.Uint64(val)
			return nil
		})
	})
	if err == badger.ErrKeyNotFound {
		return 0, nil
	}
	return
}

func joinSlice(a, b []byte) []byte {
	k := make([]byte, len(a)+len(b))
	copy(k, a)
	copy(k[len(a):], b)
	return k
}
