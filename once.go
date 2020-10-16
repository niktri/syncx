package syncx

import (
	"sync"
	"sync/atomic"
)

// Once is similar to sync.Once with IsDone.
// Often we need a lock-free way to retrieve init/close state.
// Need to be careful when close or init in progress, it's better to use locked IsDone, as it prevents partially done resource.
// See proposal discussion at: https://github.com/golang/go/issues/41690
type Once struct {
	done uint32
	m    sync.Mutex
}

//Do does once.
func (o *Once) Do(f func()) {
	if atomic.LoadUint32(&o.done) == 0 {
		o.doSlow(f)
	}
}

func (o *Once) doSlow(f func()) {
	o.m.Lock()
	defer o.m.Unlock()
	if o.done == 0 {
		defer atomic.StoreUint32(&o.done, 1)
		f()
	}
}

// IsDone returns true if done. Lock-free.
func (o *Once) IsDone() bool {
	return atomic.LoadUint32(&o.done) != 0
}
