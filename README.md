# syncx
syncx is extension of golang sync package with `TryLock` like safer Atomics.

# Features
* Various [`TryLock`](https://github.com/niktri/syncx/blob/main/trylock.go) implementations - Lockfree way to acquire mutex Lock.
* Various lock-free functions like [`Once.IsDone()`]((https://github.com/niktri/syncx/blob/main/once.go)), [`Locker.IsLocked()`]((https://github.com/niktri/syncx/blob/main/trylock.go)) extending golang/sync
* [`AtomicInt64`](https://github.com/niktri/syncx/blob/main/atomic_int64.go) - Safer alternative of sync/atomic, avoiding accidental unsafe access of shared variable & self-documenting its purpose.

# TryLock
`TryLock` is [Much discussed](https://github.com/golang/go/issues/6123) in golang community & it seems(rightly so) golang won't implement it.
## MutexTryLocker
* [`MutexTryLocker`](https://github.com/niktri/syncx/blob/main/trylock.go) is best implementation I could find to implement TryLocker using
a Mutex + Atomic flag. 
* We try to udpate this flag, if we couldn't we simply return false. If true, we try to acquire lock.
* Only critical scenario is: Someone already got lock in `Lock()` but hasn't updated flag yet. 
That `Lock()` thread will release the lock so TryLock thread is unblocked asap.
* This is why it's TryLock is psuedo-non-blocking. In benchmarks its found not be blocked in most cases.
* It's Lock-Unlock performance is very close(~8%) to Plain [sync.Mutex](https://golang.org/pkg/sync/#Mutex).

## ChannelTryLocker
* [`ChannelTryLocker`](https://github.com/niktri/syncx/blob/main/trylock.go) is implemented using go channels. 
* It does not have live-lock issue of MutexTryLocker, but Lock() & TryLock() are 3x & 100x slower.

## HackTryLocker
* [`HackTryLocker`](https://github.com/niktri/syncx/blob/main/trylock.go) is implemented hacking sync.Mutex as it's first variable is [`state`](https://github.com/golang/go/blob/af8748054b40e9a1e529e42a0f83cc2c90a35af6/src/sync/mutex.go#L26).
* It looks like smart hack, but 1000x slower than MutexTryLocker.

# Once.IsDone()
* golang `sync.Once` already keeps a [flag](https://github.com/golang/go/blob/af8748054b40e9a1e529e42a0f83cc2c90a35af6/src/sync/once.go#L18) if it's 
done or not. It does not expose it.
* [syncx.Once](https://github.com/niktri/syncx/blob/main/once.go) just exposes this flag. 
* It can be used when we just want to query the status without doing actual work.
* As [discussed here](https://github.com/golang/go/issues/41690), it's concluded that there aren't enough usecases to include to standard library.
* `syncx.Once` is simpler to use without duplicating Once.done flag and messing with atomics.

# AtomicInt64
* [`AtomicInt64`](https://github.com/niktri/syncx/blob/main/atomic_int64.go) is for usecases of keeping a shared counter.
* AtomicInt64 self-documents its purpose & protects accidental unsafe access to shared counter.
* It's just a plain int64 type, safely incrementing, adding, getting, setting functions.
* It's conveniently returns live value if stored in a repo. _e.g. If stored as a field in logrus.WithField, it will always print live value._
* There are 2 more implementations of counters only for benchmarking purpose: 
    * `MutexInt64` Plain Mutex guards variable. 3x slower than AtomicInt64.
    * `ChannelInt64`. Every mutation happens in another goroutine, communicated by channel. 30x slower than AtomicInt64.


