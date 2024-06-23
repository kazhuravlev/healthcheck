package logr

import (
	"container/ring"
	"sync"
	"time"
)

const maxStatesToStore = 5

type Ring struct {
	mu   *sync.RWMutex
	data *ring.Ring
}

func New() *Ring {
	return &Ring{
		mu:   new(sync.RWMutex),
		data: ring.New(maxStatesToStore),
	}
}

func (r *Ring) Put(rec Rec) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.data.Value = rec
	r.data = r.data.Prev()
}

func (r *Ring) GetLast() (Rec, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	last := r.data.Next()

	if last.Value == nil {
		return Rec{}, false
	}

	return last.Value.(Rec), true
}

func (r *Ring) SlicePrev() []Rec {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make([]Rec, 0, r.data.Len())
	r.data.Do(func(val any) {
		if val == nil {
			return
		}

		res = append(res, val.(Rec))
	})

	if len(res) == 0 {
		return nil
	}

	return res[1:]
}

type Rec struct {
	Time  time.Time
	Error error
}
