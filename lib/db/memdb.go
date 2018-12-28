// Package db offer different database implementations used by transports
package db

import (
	"sync"

	"github.com/patrickmn/go-cache"
)

// MemDB is in internal Key value store, thread safe
type MemDB struct {
	db *cache.Cache
}

var mux sync.Mutex

// Get number of pending messages and number of message consumer for the queue
func getNewMemDB() (memdb MemDB) {
	memdb.db = cache.New(0, 0)
	return memdb
}

/*
should add IncrBy, Set, Get, Delete with locks
*/

// Get gets the content for key
func (d MemDB) Get(key string) (val interface{}, found bool) {
	val, found = d.db.Get(key)
	return val, found
}

// Delete deletes the key from db
func (d MemDB) Delete(key string) {
	d.db.Delete(key)
}

// Set sets the key to val
func (d MemDB) Set(key string, val interface{}) {
	d.db.Set(key, val, 0)
}

// IncrBy increments the key by val, if not present new value will be val
func (d MemDB) IncrBy(key string, val int64) {
	mux.Lock()
	x, found := d.db.Get(key)
	var prevValue int64
	if found {
		prevValue = x.(int64)
	} else {
		prevValue = 0
	}
	newValue := prevValue + val
	d.db.Set(key, newValue, 0)
	mux.Unlock()
}
