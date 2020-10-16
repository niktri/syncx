package syncx

import (
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

//TryLocker extends sync.Locker with a TryLock. 3 implementations are benchmarked.
//MutexTryLocker is the best implementation in most cases, other implementations should be avoided.
type TryLocker interface {
	sync.Locker
	//TryLock tries to acquire lock, returns true if success, false if it is not possible to acquire lock without blocking.
	TryLock() bool
	//IsLocked returns true if locked. There are no real use-cases for this except keeping stats or logging.
	IsLocked() bool
}

// ****************** MutexTryLocker using Mutex + Atomic flag ******************

func _() {
	var _ TryLocker = (*MutexTryLocker)(nil)
}

//NewMutexTryLocker creates TryLocker using mutex + atomic blocked flag. Read tryLockMutex.TryLock() doc for details.
//A rare race scenario will be resolved on best-effort basis governed by thresholds.
//MutexTryLocker is the best implementation to be used in most cases.
func NewMutexTryLocker() *MutexTryLocker {
	const livelockThreshold = 10
	const giveupThreshould = int(^uint(0) >> 1)
	return NewTryLockMutexWithThresholds(&sync.Mutex{}, livelockThreshold, giveupThreshould)
}

//NewTryLockMutexWithThresholds creates TryLocker using mutex + atomic blocked flag. Read tryLockMutex.TryLock() doc for details.
//`giveupThreshould > livelockThreshold >=1`
func NewTryLockMutexWithThresholds(locker sync.Locker, livelockThreshold int, giveupThreshould int) *MutexTryLocker {
	if livelockThreshold < 1 || giveupThreshould <= livelockThreshold {
		panic("Invalid Thresholds: " + strconv.Itoa(livelockThreshold) + " , " + strconv.Itoa(giveupThreshould))
	}
	return &MutexTryLocker{&sync.Mutex{}, 0, livelockThreshold, giveupThreshould}
}

//MutexTryLocker is non-blocking extension of golang sync.Mutex. Read TryLock() doc for details.
type MutexTryLocker struct {
	locker            sync.Locker // Client can pass any Locker
	blocked           int32       // 0: unblocked, 0: blocked
	livelockThreshold int         // After this we consider livelock & sleep
	giveupThreshould  int         // After this we give-up.
}

//Lock locks m as in sync.Mutex.Lock(). It blocks if m is already locked.
//
//Lock() can be called concurrently without race. TryLock() can be called concurrently without race.
//If Single TryLock() is called concurrently to Lock(), we might encounter race.
//Note that further concurrent TryLock() will simply return false without race.
//In this rare case, resolution is made on best-effort basis governed by thresholds.
func (m *MutexTryLocker) Lock() {
	// startTime := time.Now()
	for i := 0; ; i++ {
		m.locker.Lock()                                    //@LOCK
		if !atomic.CompareAndSwapInt32(&m.blocked, 0, 1) { //@BLOCKED
			//There is only 1 reason we are here: Someone reached @TRYLOCK before we reached @BLOCKED. Release & retry @LOCK.
			m.locker.Unlock()
			if i < m.livelockThreshold {
				//Rarely it may happen that many lucky goroutinues are at @LOCK but one unlucky goroutine already on @TRYLOCK.
				//Unlucky is unable to acquire lock, as lucky ones keep winning the @LOCK. Gosched() may dampen this a bit.
				runtime.Gosched()
			} else {
				if i < m.giveupThreshould { //@LIVELOCK
					//Code reaching here should be even rarer. Gosched() wasn't enough. We must sleep.
					time.Sleep(time.Duration(i-m.livelockThreshold) * time.Millisecond)
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
	if !atomic.CompareAndSwapInt32(&m.blocked, 1, 0) {
		panic("syncx: unlock of unblocked mutex")
	}
	m.locker.Unlock()
}

//TryLock is a psuedo-non-blocking way to acquire lock, returns true if it acquires lock, false otherwise.
func (m *MutexTryLocker) TryLock() bool {
	if atomic.CompareAndSwapInt32(&m.blocked, 0, 1) { //@TRYLOCK
		//This is psuedo-non-blocking as if other goroutines are just before @BLOCKED they will try best to release & yield,
		//to give this goroutine a chance to acquire lock without wasting much time.
		m.locker.Lock()
		return true
	}
	return false
}

//IsLocked returns true if locked. There are no real use-cases for this except keeping stats or logging.
func (m *MutexTryLocker) IsLocked() bool {
	return atomic.LoadInt32(&m.blocked) == 1
}

// ****************** ChannelTryLocker using channels ******************

func _() {
	var _ TryLocker = (*ChannelTryLocker)(nil)
}

// NewChannelTryLocker creates TryLocker using channels.
func NewChannelTryLocker() *ChannelTryLocker {
	return &ChannelTryLocker{make(chan int32, 1)}
}

//ChannelTryLocker implements TryLocker using Channels. It does not have live-lock issue of MutexTryLocker, but Lock() & TryLock() are 3x & 100x slower.
type ChannelTryLocker struct {
	ch chan int32
}

//Lock acquires lock
func (m *ChannelTryLocker) Lock() {
	m.ch <- 1
}

//Unlock releases lock
func (m *ChannelTryLocker) Unlock() {
	<-m.ch
}

//TryLock tries to acquire lock, returns true if success, false if it is not possible to acquire lock without blocking.
func (m *ChannelTryLocker) TryLock() bool {
	select {
	case m.ch <- 1:
		return true
	default:
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
