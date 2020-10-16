package syncx

import (
	"strconv"
	"sync"
	"sync/atomic"
)

// ********************* A T O M I C - I N T *********************

// AtomicInt64 is for frequent usecase of concurrent shared counter.
// It is simpler & safer alternative of sync/atomic, avoiding accidental unsafe access of shared variable & self-documenting its purpose.
// When set as field on logrus.WithFields(), it will always print dynamic value.
// Always use returned *AtomicInt64 using NewAtomicInt64. Never deref returned pointer.
type AtomicInt64 int64

// NewAtomicInt64 creates *AtomicInt64. Never deref this pointer
func NewAtomicInt64(val int64) *AtomicInt64 {
	a := AtomicInt64(val)
	return &a
}

// Get returns current value atomically
func (a *AtomicInt64) Get() int64 {
	return atomic.LoadInt64((*int64)(a))
}

// Set sets new and returns old value atomically
func (a *AtomicInt64) Set(new int64) int64 {
	return atomic.SwapInt64((*int64)(a), new)
}

// Incr increments and returns new value atomically
func (a *AtomicInt64) Incr() int64 {
	return atomic.AddInt64((*int64)(a), 1)
}

// Decr decrements and returns new value atomically
func (a *AtomicInt64) Decr() int64 {
	return atomic.AddInt64((*int64)(a), -1)
}

// Add adds and returns new value atomically
func (a *AtomicInt64) Add(delta int64) int64 {
	return atomic.AddInt64((*int64)(a), delta)
}

// Sub substracts and returns new value atomically
func (a *AtomicInt64) Sub(delta int64) int64 {
	return atomic.AddInt64((*int64)(a), -delta)
}

// SetIfOld sets new value if current value is old, return false otherwise.
func (a *AtomicInt64) SetIfOld(old int64, new int64) bool {
	return atomic.CompareAndSwapInt64((*int64)(a), old, new)
}

//String Threadsafe Stringer
func (a *AtomicInt64) String() string {
	return strconv.FormatInt(atomic.LoadInt64((*int64)(a)), 10)
}

// IncrString .
func (a *AtomicInt64) IncrString() string {
	return strconv.FormatInt(atomic.AddInt64((*int64)(a), 1), 10)
}

// DecrString .
func (a *AtomicInt64) DecrString() string {
	return strconv.FormatInt(atomic.AddInt64((*int64)(a), -1), 10)
}

// ********************* M I S C *********************

// Below constructs are only for benchmarking purpose. They are much slower in practice.

//ConcurrentInt64 .
type ConcurrentInt64 interface {
	Get() int64
	Set(new int64) (old int64)
	Add(delta int64) (new int64)
	Sub(delta int64) (new int64)
	Incr() (new int64)
	Decr() (new int64)
}

func _() {
	var _ ConcurrentInt64 = (*AtomicInt64)(nil)
	var _ ConcurrentInt64 = (*MutexInt64)(nil)
	var _ ConcurrentInt64 = (*ChannelInt64)(nil)
}

//MutexInt64 is 3x slower to AtomicInt64
type MutexInt64 struct {
	in  sync.Mutex
	val int64
}

func NewMutexInt64(val int64) *MutexInt64 {
	return &MutexInt64{val: val}
}

func (m *MutexInt64) Get() int64 {
	m.in.Lock()
	val := m.val
	m.in.Unlock()
	return val
}
func (m *MutexInt64) Set(new int64) int64 {
	m.in.Lock()
	ret := m.val
	m.val = new
	m.in.Unlock()
	return ret
}
func (m *MutexInt64) Add(delta int64) int64 {
	m.in.Lock()
	m.val += delta
	ret := m.val
	m.in.Unlock()
	return ret
}
func (m *MutexInt64) Sub(delta int64) int64 {
	m.in.Lock()
	m.val -= delta
	ret := m.val
	m.in.Unlock()
	return ret
}
func (m *MutexInt64) Incr() int64 {
	m.in.Lock()
	m.val++
	ret := m.val
	m.in.Unlock()
	return ret
}
func (m *MutexInt64) Decr() int64 {
	m.in.Lock()
	m.val--
	ret := m.val
	m.in.Unlock()
	return ret
}

//ChannelInt64 is 30x slower to AtomicInt64
type ChannelInt64 struct {
	deltach chan int64   // channel of delta
	av      atomic.Value // Keeps int64 val
}

func NewChannelInt64(val int64, chancap int) *ChannelInt64 {
	deltach := make(chan int64, chancap)
	ret := &ChannelInt64{deltach, atomic.Value{}}
	ret.av.Store(int64(val))
	go ret.process()
	return ret
}
func (m *ChannelInt64) process() {
	for {
		delta := <-m.deltach
		val := m.av.Load().(int64)
		val += delta
		m.av.Store(val)
	}
}
func (m *ChannelInt64) Get() int64 {
	return m.av.Load().(int64)
}
func (m *ChannelInt64) Set(new int64) int64 {
	ret := m.av.Load().(int64)
	m.av.Store(new)
	return ret
}
func (m *ChannelInt64) Add(delta int64) int64 {
	val := m.av.Load().(int64)
	m.deltach <- delta
	val += delta
	return val
}
func (m *ChannelInt64) Sub(delta int64) int64 {
	val := m.av.Load().(int64)
	m.deltach <- -delta
	val -= delta
	return val
}
func (m *ChannelInt64) Incr() int64 {
	val := m.av.Load().(int64)
	m.deltach <- 1
	val++
	return val
}
func (m *ChannelInt64) Decr() int64 {
	val := m.av.Load().(int64)
	m.deltach <- -1
	val--
	return val
}
