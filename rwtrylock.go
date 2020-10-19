package syncx

import (
	"runtime"
	"strconv"
	"sync"
	"time"
)

//RWTryLocker extends TryLocker with a Read variation: TryRLock
type RWTryLocker interface {
	TryLocker
	RLock()
	RUnlock()
	//TryRLock tries to acquire RLock, returns true if success, false if it is not possible to acquire RLock without blocking.
	TryRLock() bool
	//IsRLocked returns true if RLocked. There are no real use-cases for this except keeping stats or logging.
	IsRLocked() bool
	//IsUnlocked returns true if it is neither RLocked nor Locked.
	IsUnlocked() bool
}

// ****************** MutexRWTryLocker using Mutex + Atomic flag ******************

func _() {
	var _ RWTryLocker = (*MutexRWTryLocker)(nil)
}

//NewMutexRWTryLocker creates TryLocker using mutex + atomic blocked flag. Read MutexRWTryLocker.TryLock() doc for details.
func NewMutexRWTryLocker() *MutexRWTryLocker {
	const livelockThreshold = 10
	const giveupThreshould = int(^uint(0) >> 1)
	return NewMutexRWTryLockerWithThresholds(livelockThreshold, giveupThreshould)
}

//NewMutexRWTryLockerWithThresholds creates TryLocker using mutex + atomic blocked flag. Read tryLockMutex.TryLock() doc for details.
//`giveupThreshould > livelockThreshold >=1`
func NewMutexRWTryLockerWithThresholds(livelockThreshold int, giveupThreshould int) *MutexRWTryLocker {
	if livelockThreshold < 1 || giveupThreshould <= livelockThreshold {
		panic("Invalid Thresholds: " + strconv.Itoa(livelockThreshold) + " , " + strconv.Itoa(giveupThreshould))
	}
	return &MutexRWTryLocker{livelockThreshold: livelockThreshold, giveupThreshould: giveupThreshould}
}

//MutexRWTryLocker is non-blocking extension of golang sync.RWMutex. Read TryLock() doc for details.
type MutexRWTryLocker struct {
	rwmutex           sync.RWMutex
	statemutex        sync.Mutex //Protects state
	state             int32      // 0: Unlocked, -1: Locked, 1+: RLocked
	livelockThreshold int        // After this we consider livelock & sleep
	giveupThreshould  int        // After this we give-up.
}

//Lock locks m as in sync.Mutex.Lock(). It blocks if m is already locked.
//
//Lock() can be called concurrently without race. TryLock() can be called concurrently without race.
//If Single TryLock() is called concurrently to Lock(), we might encounter race.
//Note that further concurrent TryLock() will simply return false without race.
//In this rare case, resolution is made on best-effort basis governed by thresholds.
func (m *MutexRWTryLocker) Lock() {
	// startTime := time.Now()
	for i := 0; ; i++ {
		m.rwmutex.Lock() //@LOCK

		m.statemutex.Lock()
		unblocked := m.state == unblocked
		if unblocked {
			m.state = blocked //@BLOCKED
		}
		m.statemutex.Unlock()
		if !unblocked {
			m.rwmutex.Unlock()
			if i < m.livelockThreshold {
				//Rarely it may happen that many lucky goroutinues are at @LOCK but one unlucky goroutine already on @TRYLOCK.
				//Unlucky is unable to acquire lock, as lucky ones keep winning the @LOCK. Gosched() may dampen this a bit.
				runtime.Gosched()
			} else {
				if i < m.giveupThreshould { //@LIVELOCK
					//Code reaching here should be even rarer. Gosched() wasn't enough. We must sleep.
					d := time.Duration(i-m.livelockThreshold) * time.Millisecond
					if d > 1*time.Second {
						d = 1 * time.Second
					}
					time.Sleep(d)
				} else { //@GIVEUP
					panic("This should never occur: LiveLock in syncx.Mutex.Lock: " + strconv.Itoa(i))
				}
			}
		}
		return
	}
}

//RLock as in RWMutex.RLock().
//
//Multiple RLock() can be individually called concurrently without race.
//If Single TryLock() is called concurrently to RLock(), we might encounter race.
//Note that further concurrent TryLock() will simply return false without race.
//In this rare case, resolution is made on best-effort basis governed by thresholds.
func (m *MutexRWTryLocker) RLock() {
	for i := 0; ; i++ {
		m.rwmutex.RLock() //@LOCK

		m.statemutex.Lock()
		notblocked := m.state != blocked
		if notblocked {
			m.state++ //@BLOCKED
		}
		m.statemutex.Unlock()
		if !notblocked {
			m.rwmutex.RUnlock()
			if i < m.livelockThreshold {
				//Rarely it may happen that many lucky goroutinues are at @LOCK but one unlucky goroutine already on @TRYLOCK.
				//Unlucky is unable to acquire lock, as lucky ones keep winning the @LOCK. Gosched() may dampen this a bit.
				runtime.Gosched()
			} else {
				if i < m.giveupThreshould { //@LIVELOCK
					//Code reaching here should be even rarer. Gosched() wasn't enough. We must sleep.
					d := time.Duration(i-m.livelockThreshold) * time.Millisecond
					if d > 1*time.Second {
						d = 1 * time.Second
					}
					time.Sleep(d)
				} else { //@GIVEUP
					panic("This should never occur: LiveLock in syncx.Mutex.Lock: " + strconv.Itoa(i))
				}
			}
		}
		return
	}
}

// Unlock unlocks m as in sync.Mutex.Unlock()
func (m *MutexRWTryLocker) Unlock() {
	m.statemutex.Lock()
	if m.state != blocked {
		panic("syncx: unlock of unblocked mutex")
	}
	m.state = unblocked
	m.statemutex.Unlock()
	m.rwmutex.Unlock()
}

// RUnlock unlocks m as in sync.Mutex.RUnlock()
func (m *MutexRWTryLocker) RUnlock() {
	m.statemutex.Lock()
	if m.state < 1 {
		panic("syncx: unlock of unblocked mutex")
	}
	m.state--
	m.statemutex.Unlock()
	m.rwmutex.RUnlock()
}

//TryLock is a psuedo-non-blocking way to acquire lock, returns true if it acquires lock, false otherwise.
func (m *MutexRWTryLocker) TryLock() bool {
	m.statemutex.Lock()
	if m.state != unblocked {
		m.statemutex.Unlock()
		return false
	}
	m.state = blocked
	m.statemutex.Unlock()

	m.rwmutex.Lock()
	return true
}

//TryRLock returns true if it acquires Rlocks without blocking, false otherwise.
func (m *MutexRWTryLocker) TryRLock() bool {
	m.statemutex.Lock()
	if m.state == blocked {
		m.statemutex.Unlock()
		return false
	}
	m.state++
	m.statemutex.Unlock()

	m.rwmutex.RLock()
	return true
}

//IsLocked returns true if locked. There are no real use-cases for this except keeping stats or logging.
func (m *MutexRWTryLocker) IsLocked() bool {
	m.statemutex.Lock()
	ret := m.state == blocked
	m.statemutex.Unlock()
	return ret
}

//IsRLocked .
func (m *MutexRWTryLocker) IsRLocked() bool {
	m.statemutex.Lock()
	ret := m.state > 0
	m.statemutex.Unlock()
	return ret
}

//IsUnlocked returns true, if its neither locked nor rlocked.
func (m *MutexRWTryLocker) IsUnlocked() bool {
	m.statemutex.Lock()
	ret := m.state == unblocked
	m.statemutex.Unlock()
	return ret
}
