package main

import (
	"context"
	"sync"

	"github.com/dgraph-io/badger/v2"
	"github.com/open-policy-agent/opa/storage"
)

// This implements storage interface with opa that is based on badger key/value
// store.
type opaStore struct {
	db   *badger.DB
	txns *sync.Map
}

func (o *opaStore) NewTransaction(ctx context.Context, params ...storage.TransactionParams) (storage.Transaction, error) {
	var p storage.TransactionParams
	if len(params) > 0 {
		p = params[0]
	}
	txn := newTxn(ctx, o.db, p)
	id, err := txn.getID()
	if err != nil {
		return nil, err
	}
	o.txns.Store(id, txn)
	return id, nil
}

type opaStoreTxn struct {
	id     txnID
	params storage.TransactionParams
	events chan *txnEvent
	output chan *txnEvent
	err    error
	closed bool
}

type txnID uint64

func (id txnID) ID() uint64 {
	return uint64(id)
}

func (otxn *opaStoreTxn) start(ctx context.Context, db *badger.DB) (err error) {
	defer func() {
		if err != nil {
			otxn.err = err
		}
	}()
	if otxn.params.Write {
		err = db.Update(func(txn *badger.Txn) error {
			return otxn.manage(ctx, txn)
		})
	} else {
		err = db.View(func(txn *badger.Txn) error {
			return otxn.manage(ctx, txn)
		})
	}
	return
}

func (otxn *opaStoreTxn) manage(ctx context.Context, txn *badger.Txn) error {
	defer func() {
		otxn.closed = true
	}()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case e := <-otxn.events:
			switch e.name {
			case "read":
				key := e.payload.([]byte)
				it, err := txn.Get(key)
				if err != nil {
					otxn.output <- &txnEvent{name: "error", payload: err}
				} else {
					v := make([]byte, it.ValueSize())
					it.ValueCopy(v)
					otxn.output <- &txnEvent{name: "value", payload: v}
				}
			case "write":
				kv := e.payload.(keyValue)
				err := txn.Set(kv.key, kv.value)
				if err != nil {
					otxn.output <- &txnEvent{name: "error", payload: err}
				} else {
					otxn.output <- &txnEvent{name: "value", payload: true}
				}
			case "id":
				if otxn.id == 0 {
					otxn.id = txnID(txn.ReadTs())
				}
				otxn.output <- &txnEvent{name: "value", payload: otxn.id}
			case "abort", "commit":
				return nil
			}
		}
	}
}

func (otxn *opaStoreTxn) send(e *txnEvent) (*txnEvent, error) {
	if otxn.err != nil {
		return nil, otxn.err
	}
	otxn.events <- e
	return <-otxn.output, nil
}

func (otxn *opaStoreTxn) getID() (txnID, error) {
	e, err := otxn.send(&txnEvent{name: "id"})
	if err != nil {
		return 0, err
	}
	otxn.id = e.payload.(txnID)
	return otxn.id, nil
}

type keyValue struct {
	key   []byte
	value []byte
}

type txnEvent struct {
	name    string
	payload interface{}
}

func newTxn(ctx context.Context, db *badger.DB, params storage.TransactionParams) *opaStoreTxn {
	txn := &opaStoreTxn{
		events: make(chan *txnEvent),
		output: make(chan *txnEvent),
	}
	go txn.start(ctx, db)
	return txn
}
