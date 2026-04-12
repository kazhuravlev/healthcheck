package logr

import (
	"sync"
	"time"
)

const maxStatesToStore = 5

type Ring struct {
	mu     sync.RWMutex
	data   [maxStatesToStore]Rec
	latest int
	count  int
}

// New creates an empty ring for recent records.
func New() *Ring {
	return &Ring{
		mu:     sync.RWMutex{},
		data:   [5]Rec{},
		latest: -1,
		count:  0,
	}
}

// Put saves a record as the latest one.
func (r *Ring) Put(rec Rec) {
	r.mu.Lock()
	defer r.mu.Unlock()

	next := 0
	if r.count > 0 {
		next = r.latest - 1
		if next < 0 {
			next = maxStatesToStore - 1
		}
	}

	r.data[next] = rec
	r.latest = next
	if r.count < maxStatesToStore {
		r.count++
	}
}

// GetLast returns the latest record.
// It returns false when the ring is empty.
func (r *Ring) GetLast() (Rec, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.count == 0 {
		return Rec{}, false
	}

	return r.data[r.latest], true
}

func (r *Ring) Slice() []Rec {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.count <= 1 {
		return nil
	}

	res := make([]Rec, 0, r.count-1)
	for i := 1; i < r.count; i++ {
		idx := r.latest + i
		if idx >= maxStatesToStore {
			idx -= maxStatesToStore
		}
		res = append(res, r.data[idx])
	}

	return res
}

type Rec struct {
	Time  time.Time
	Error error
}
