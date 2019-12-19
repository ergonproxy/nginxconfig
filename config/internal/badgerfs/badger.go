package badgerfs

import (
	"github.com/dgraph-io/badger"
)

var _ KV = (*B)(nil)

type B struct {
	db *badger.DB
}

func NewB(db *badger.DB) *B {
	return &B{db: db}
}

func (b *B) Set(key string, value []byte) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), value)
	})
}

func (b *B) Get(key string) (o []byte, err error) {
	err = b.db.View(func(txn *badger.Txn) error {
		item, rerr := txn.Get([]byte(key))
		if rerr != nil {
			return rerr
		}
		o, err = item.ValueCopy(nil)
		return err
	})
	return
}

func (b *B) Has(key string) bool {
	return b.db.View(func(txn *badger.Txn) error {
		_, rerr := txn.Get([]byte(key))
		return rerr
	}) == nil
}

func (b *B) Remove(key string) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

func (b *B) Walk(prefix string, fn WalkFn) error {
	if fn == nil {
		return nil
	}
	return b.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		p := []byte(prefix)
		for it.Seek(p); it.ValidForPrefix(p); it.Next() {
			item := it.Item()
			k := item.Key()
			err := fn(string(k), nil, func() ([]byte, error) {
				return item.ValueCopy(nil)
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}
