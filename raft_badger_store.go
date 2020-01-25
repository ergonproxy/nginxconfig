package main

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"sync"

	"github.com/dgraph-io/badger/v2"
	"github.com/hashicorp/raft"
)

var stablePrefix = []byte("/stable")
var entryPrefix = []byte("/entry")
var _ raft.StableStore = (*store)(nil)
var _ raft.LogStore = (*store)(nil)

type firstKey struct{}

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

func (s *store) FirstIndex() (uint64, error) {
	if v, ok := s.cache.Load(firstKey{}); ok {
		return v.(uint64), nil
	}
	index, err := s.seekEntry(nil, 0, false)
	if err == nil {
		s.cache.Store(firstKey{}, index+1)
	}
	return index + 1, err
}

func (s *store) LastIndex() (uint64, error) {
	return s.seekEntry(nil, math.MaxUint64, true)
}

func (s *store) seekEntry(e *raft.Log, seekTo uint64, reverse bool) (uint64, error) {
	var index uint64
	err := s.db.View(func(txn *badger.Txn) error {
		opt := badger.DefaultIteratorOptions
		opt.PrefetchValues = false
		opt.Prefix = entryPrefix
		opt.Reverse = reverse
		itr := txn.NewIterator(opt)
		defer itr.Close()

		itr.Seek(entryKey(seekTo))
		if !itr.Valid() {
			return raft.ErrLogNotFound
		}
		item := itr.Item()
		index = parseKeyIndex(item.Key())
		if e == nil {
			return nil
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, e)
		})
	})
	return index, err
}

func (s *store) GetLog(index uint64, e *raft.Log) error {
	return s.db.View(func(txn *badger.Txn) error {
		it, err := txn.Get(entryKey(index))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return raft.ErrLogNotFound
			}
			return err
		}
		return it.Value(func(val []byte) error {
			return json.Unmarshal(val, e)
		})
	})
}

func (s *store) StoreLog(e *raft.Log) error {
	return s.db.Update(func(txn *badger.Txn) error {
		b, err := json.Marshal(e)
		if err != nil {
			return err
		}
		return txn.Set(entryKey(e.Index), b)
	})
}
func (s *store) StoreLogs(logs []*raft.Log) error {
	batch := s.db.NewWriteBatch()
	defer batch.Cancel()
	for _, e := range logs {
		b, err := json.Marshal(e)
		if err != nil {
			return err
		}
		if err := batch.Set(entryKey(e.Index), b); err != nil {
			return err
		}
	}
	return batch.Flush()
}

func entryKey(key uint64) []byte {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], key)
	return joinSlice(entryPrefix, b[:])
}

func parseKeyIndex(key []byte) uint64 {
	return binary.BigEndian.Uint64(key[len(entryPrefix):])
}

func (s *store) deleteRage(batch *badger.WriteBatch, from, to uint64) error {
	var keys []string
	err := s.db.Update(func(txn *badger.Txn) error {
		start := entryKey(from)
		opt := badger.DefaultIteratorOptions
		opt.PrefetchValues = false
		opt.Prefix = entryPrefix
		itr := txn.NewIterator(opt)
		defer itr.Close()

		for itr.Seek(start); itr.Valid(); itr.Next() {
			key := itr.Item().Key()
			idx := parseKeyIndex(key)
			if idx > to {
				break
			}
			keys = append(keys, string(key))
		}
		return nil
	})
	if err != nil {
		return err
	}
	return s.deleteKeys(batch, keys)
}

func (s *store) DeleteRange(min, max uint64) error {
	batch := s.db.NewWriteBatch()
	defer batch.Cancel()
	if err := s.deleteRage(batch, min, max); err != nil {
		return err
	}
	return batch.Flush()
}
func (s *store) deleteKeys(batch *badger.WriteBatch, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	for _, k := range keys {
		if err := batch.Delete([]byte(k)); err != nil {
			return err
		}
	}
	return nil
}
