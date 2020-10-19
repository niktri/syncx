package syncx

import (
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

//TryLocker extends sync.Locker with a TryLock.
//4 implementations are benchmarked. MutexTryLocker & AtomicTryLocker are best implementation in most cases, other implementations should be avoided.
type TryLocker interface {
	sync.Locker
	//TryLock tries to acquire lock, returns true if success, false if it is not possible to acquire lock without blocking.
	//TryLock() may be used when we can do something else if lock is busy, dont call this in empty infinite loop.
	TryLock() bool
	//IsLocked returns true if locked. There are no real use-cases for this except keeping stats or logging.
	IsLocked() bool
}

// ****************** MutexTryLocker using Mutex + Atomic flag ******************

func _() {
	var _ TryLocker = (*MutexTryLocker)(nil)
}

//NewMutexTryLocker creates TryLocker using mutex + atomic state. Read MutexTryLocker.TryLock() doc for details.
//There is no race for 2 Lock() or 2 TryLock() calls. There is a race for 1 TryLock() & multiple Lock().
//That will be resolved on best-effort basis governed by thresholds.
func NewMutexTryLocker() *MutexTryLocker {
	const livelockThreshold = 10
	const giveupThreshould = int(^uint(0) >> 1)
	return NewMutexTryLockerWithThresholds(&sync.Mutex{}, livelockThreshold, giveupThreshould)
}

//NewMutexTryLockerWithThresholds creates TryLocker using mutex + atomic state. Read NewMutexTryLocker.TryLock() doc for details.
//`giveupThreshould > livelockThreshold >=1`
func NewMutexTryLockerWithThresholds(locker sync.Locker, livelockThreshold int, giveupThreshould int) *MutexTryLocker {
	if livelockThreshold < 1 || giveupThreshould <= livelockThreshold {
		panic("Invalid Thresholds: " + strconv.Itoa(livelockThreshold) + " , " + strconv.Itoa(giveupThreshould))
	}
	return &MutexTryLocker{&sync.Mutex{}, unblocked, livelockThreshold, giveupThreshould}
}

const unblocked = 0
const blocked = -1

//MutexTryLocker is non-blocking extension of golang sync.Mutex. Read TryLock() doc for details.
type MutexTryLocker struct {
	locker            sync.Locker // Client can pass any Locker
	state             int32       // 0: unblocked, -1: blocked
	livelockThreshold int         // After this we consider livelock & sleep
	giveupThreshould  int         // After this we give-up.
}

//Lock locks m as in sync.Mutex.Lock(). It blocks if m is already locked.
//
//Lock() can be called concurrently without race. TryLock() can be called concurrently without race.
//If Single TryLock() is called concurrently to Lock(), we might encounter race.
//Note that further concurrent TryLock() will simply return false without race.
//In that case, resolution is made on best-effort basis governed by thresholds.
func (m *MutexTryLocker) Lock() {
	// startTime := time.Now()
	for i := 0; ; i++ {
		m.locker.Lock()                                                //@LOCK
		if !atomic.CompareAndSwapInt32(&m.state, unblocked, blocked) { //@BLOCKED
			//There is only 1 reason we are here: Someone reached @TRYLOCK before we reached @BLOCKED. Release & retry @LOCK.
			m.locker.Unlock()
			if i < m.livelockThreshold {
				//Rarely it may happen that many lucky goroutinues are at @LOCK but one unlucky goroutine already on @TRYLOCK.
				//Unlucky is unable to acquire lock, as lucky ones keep winning the @LOCK. Gosched() may dampen this a bit.
				runtime.Gosched()
			} else {
				// fmt.Println("Mutex threshold ", i, time.Now().Sub(startTime))
				if i < m.giveupThreshould { //@LIVELOCK
					//Code reaching here should be even rarer. Gosched() wasn't enough. We must sleep.
					d := time.Duration(i-m.livelockThreshold) * time.Millisecond
					if d > 1*time.Second {
						d = 1 * time.Second
					}
					time.Sleep(d)
					runtime.Gosched()
				} else { //@GIVEUP
					panic("This should never occur: LiveLock in syncx.Mutex.Lock: " + strconv.Itoa(i))
				}
			}
		} else {
			return
		}
	}
}

// Unlock unlocks m as in sync.Mutex.Unlock()
func (m *MutexTryLocker) Unlock() {
	if !atomic.CompareAndSwapInt32(&m.state, blocked, unblocked) {
		panic("syncx: unlock of unblocked mutex")
	}
	m.locker.Unlock()
}

//TryLock is a psuedo-non-blocking way to acquire lock, returns true if it acquires lock, false otherwise.
//TryLock() may be used when we can do something else if lock is busy, dont call this in empty infinite loop.
func (m *MutexTryLocker) TryLock() bool {
	if atomic.CompareAndSwapInt32(&m.state, unblocked, blocked) { //@TRYLOCK
		//This is psuedo-non-blocking as if other goroutines are just before @BLOCKED they will try best to release & yield,
		//to give this goroutine a chance to acquire lock without wasting much time.
		m.locker.Lock()
		return true
	}
	return false
}

//IsLocked returns true if locked. There are no real use-cases for this except keeping stats or logging.
func (m *MutexTryLocker) IsLocked() bool {
	return atomic.LoadInt32(&m.state) == blocked
}

// ****************** AtomicTryLocker using Atomic flag ******************

func _() {
	var _ TryLocker = (*AtomicTryLocker)(nil)
}

//NewAtomicTryLocker creates TryLocker using atomic state. Read AtomicTryLocker.TryLock() doc for details.
//Lock() & TryLock() can race with each other.
//It will be resolved on best-effort basis governed by thresholds.
func NewAtomicTryLocker() *AtomicTryLocker {
	const livelockThreshold = 10
	const giveupThreshould = int(^uint(0) >> 1)
	return NewAtomicTryLockerWithThresholds(livelockThreshold, giveupThreshould)
}

//NewAtomicTryLockerWithThresholds creates TryLocker using atomic state. Read AtomicTryLocker.TryLock() doc for details.
//`giveupThreshould > livelockThreshold >=1`
func NewAtomicTryLockerWithThresholds(livelockThreshold int, giveupThreshould int) *AtomicTryLocker {
	if livelockThreshold < 1 || giveupThreshould <= livelockThreshold {
		panic("Invalid Thresholds: " + strconv.Itoa(livelockThreshold) + " , " + strconv.Itoa(giveupThreshould))
	}
	return &AtomicTryLocker{unblocked, livelockThreshold, giveupThreshould}
}

//AtomicTryLocker is non-blocking extension of golang sync.Mutex. Read TryLock() doc for details.
type AtomicTryLocker struct {
	state             int32 // 0: unblocked, -1: blocked
	livelockThreshold int   // After this we consider livelock & sleep
	giveupThreshould  int   // After this we give-up.
}

//Lock locks m as in sync.Mutex.Lock(). It blocks if m is already locked.
//
//Lock() calls TryLock() in loop until success. This will result in race, which is dampened using Gosched() and Sleep().
func (m *AtomicTryLocker) Lock() {
	// startTime := time.Now()
	for i := 0; !m.TryLock(); i++ {
		if i < m.livelockThreshold {
			runtime.Gosched()
		} else {
			if i < m.giveupThreshould { //@LIVELOCK
				// fmt.Println("Atomic threshold ", i, time.Now().Sub(startTime))
				d := time.Duration(i-m.livelockThreshold) * time.Millisecond
				if d > 1*time.Second {
					d = 1 * time.Second
				}
				time.Sleep(d)
				runtime.Gosched()
			} else { //@GIVEUP
				panic("This should never occur: LiveLock in syncx.Mutex.Lock: " + strconv.Itoa(i))
			}
		}
	}
}

// Unlock unlocks m as in sync.Mutex.Unlock()
func (m *AtomicTryLocker) Unlock() {
	if !atomic.CompareAndSwapInt32(&m.state, blocked, unblocked) {
		panic("syncx: unlock of unblocked mutex")
	}
}

//TryLock is a non-blocking way to acquire lock, returns true if it acquires lock, false otherwise.
//TryLock() may be used when we can do something else if lock is busy, dont call this in empty infinite loop.
func (m *AtomicTryLocker) TryLock() bool {
	if atomic.CompareAndSwapInt32(&m.state, unblocked, blocked) { //@TRYLOCK
		return true
	}
	return false
}

//IsLocked returns true if locked. There are no real use-cases for this except keeping stats or logging.
func (m *AtomicTryLocker) IsLocked() bool {
	return atomic.LoadInt32(&m.state) == blocked
}

// ****************** ChannelTryLocker using channels ******************

func _() {
	var _ TryLocker = (*ChannelTryLocker)(nil)
}

// NewChannelTryLocker creates TryLocker using channels.
func NewChannelTryLocker() *ChannelTryLocker {
	return &ChannelTryLocker{make(chan struct{}, 1)}
}

//ChannelTryLocker implements TryLocker using Channels. It does not have live-lock issue of MutexTryLocker, but Lock() & TryLock() are 3x & 100x slower.
type ChannelTryLocker struct {
	ch chan struct{}
}

//Lock acquires lock
func (m *ChannelTryLocker) Lock() {
	m.ch <- struct{}{}
}

//Unlock releases lock
func (m *ChannelTryLocker) Unlock() {
	<-m.ch
}

//TryLock tries to acquire lock, returns true if success, false if it is not possible to acquire lock without blocking.
func (m *ChannelTryLocker) TryLock() bool {
	select {
	case m.ch <- struct{}{}:
		return true
	default:
		return false
	}
}

//TryLockWithTimeout acquires lock if it can do so before timeout returning true, returns false otherwise.
func (m *ChannelTryLocker) TryLockWithTimeout(timeout time.Duration) bool {
	select {
	case m.ch <- struct{}{}:
		return true
	case <-time.After(timeout):
		return false
	}
}

//IsLocked returns true if locked. There are no real use-cases for this except keeping stats or logging.
func (m *ChannelTryLocker) IsLocked() bool {
	return len(m.ch) == 1
}

// ****************** HackTryLocker using unsafe.Pointer ******************

func _() {
	var _ TryLocker = (*HackTryLocker)(nil)
}

const hackMutexLocked = 1 << iota

// NewHackTryLocker creates TryLocker using HackTryLocker.
func NewHackTryLocker() *HackTryLocker {
	return &HackTryLocker{}
}

//HackTryLocker hacks plain sync.Mutex as it stores lockstatus in 1st variable `state`. But TryLock() is 1000x slower than MutexTryLocker.
type HackTryLocker struct {
	mutex sync.Mutex
}

//Lock as in sync.Mutex
func (m *HackTryLocker) Lock() {
	m.mutex.Lock()
}

//Unlock as in sync.Mutex
func (m *HackTryLocker) Unlock() {
	m.mutex.Unlock()
}

//TryLock tries to update sync.Mutex 1st variable `state` using unsafe.Pointer. This is 1000x slower than MutexTryLocker.
func (m *HackTryLocker) TryLock() bool {
	return atomic.CompareAndSwapInt32((*int32)(unsafe.Pointer(&m.mutex)), 0, hackMutexLocked)
}

//IsLocked returns true if locked. There are no real use-cases for this except keeping stats or logging.
func (m *HackTryLocker) IsLocked() bool {
	return atomic.LoadInt32((*int32)(unsafe.Pointer(&m.mutex))) == hackMutexLocked
}
