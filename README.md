# syncx
syncx adds lock-free features to golang sync package like `TryLock`, `AtomicInt`, `Once.IsDone`.
Get syncx using:
```
go get github.com/niktri/syncx
```


# Features
* Various [`TryLock`](https://github.com/niktri/syncx/blob/main/trylock.go) implementations - Lockfree way to acquire mutex Lock.
* Various lock-free functions like [`Once.IsDone()`]((https://github.com/niktri/syncx/blob/main/once.go)), [`Locker.IsLocked()`]((https://github.com/niktri/syncx/blob/main/trylock.go)) extending golang/sync
* [`AtomicInt64`](https://github.com/niktri/syncx/blob/main/atomic_int64.go) - Safer alternative of sync/atomic, avoiding accidental unsafe access of shared variable & self-documenting its purpose.

# TryLock
`TryLock` is [Much discussed](https://github.com/golang/go/issues/6123) in golang community & it seems(rightly so) golang won't implement it.
## MutexTryLocker
* [`MutexTryLocker`](https://github.com/niktri/syncx/blob/main/trylock.go) is best implementation of TryLocker. It uses a Mutex + Atomic flag. 
* Multiple `Lock()` or Multiple `TryLock()` calls won't race with each other. However single `TryLock()` may race with concurrnet `Lock()` calls.
In this scenario race will be ended on best effort basis.
This is why it's TryLock is psuedo-non-blocking.
* It's Lock-Unlock performance is very close(~8%) to Plain [sync.Mutex](https://golang.org/pkg/sync/#Mutex).

### Example MutexTryLocker
```
    import "github.com/niktri/syncx"
    ...
    locker:=syncx.NewMutexTryLocker()
    m.Lock()
	fmt.Println(m.TryLock())//false
	m.Unlock()
	fmt.Println(m.TryLock())//true
	m.Unlock()
```

## ChannelTryLocker
* [`ChannelTryLocker`](https://github.com/niktri/syncx/blob/main/trylock.go) is implemented using go channels. 
* It does not have live-lock issue of MutexTryLocker, but Lock() & TryLock() are 3x & 100x slower.

## HackTryLocker
* [`HackTryLocker`](https://github.com/niktri/syncx/blob/main/trylock.go) is implemented hacking sync.Mutex as it's [first variable is `state`](https://github.com/golang/go/blob/af8748054b40e9a1e529e42a0f83cc2c90a35af6/src/sync/mutex.go#L26).
* It's 1000x slower than MutexTryLocker.

# Once.IsDone()
* golang `sync.Once` already keeps a [flag](https://github.com/golang/go/blob/af8748054b40e9a1e529e42a0f83cc2c90a35af6/src/sync/once.go#L18) if it's 
done or not. It does not expose it.
* [syncx.Once.IsDone()](https://github.com/niktri/syncx/blob/main/once.go) just exposes this flag. It's simpler to use without duplicating flag and messing with atomics.
* It can be used when we just want to query the status without doing actual work.
* As [discussed here](https://github.com/golang/go/issues/41690), it's concluded that there aren't enough usecases to include to standard library.

### Example Once.IsDone()
```
    once := syncx.Once{}
	fmt.Println(once.IsDone()) //false
	go once.Do(func() {
		time.Sleep(3 * time.Second)
	})
	fmt.Println(once.IsDone()) //false
	time.Sleep(1 * time.Second)
	fmt.Println(once.IsDone()) //false
	time.Sleep(5 * time.Second)
	fmt.Println(once.IsDone()) //true
```

# AtomicInt64
* [`AtomicInt64`](https://github.com/niktri/syncx/blob/main/atomic_int64.go) is for usecases of keeping a shared counter.
* It self-documents its purpose & protects accidental unsafe access to shared counter.
* It's just a plain int64 type with safe convenient functions: `Get, Set, Incr, Decr, Add, Sub, SetIf, String, IncrString, DecrString`.
* It implements [Stringer](https://golang.org/pkg/fmt/#Stringer). So it conveniently returns live value if stored in a repo. _e.g. If stored as a field in logrus.WithField, it will always log live value._
* There are 2 more implementations of counters only for benchmarking purpose: 
    * `MutexInt64` Plain Mutex guards variable. 3x slower than AtomicInt64.
    * `ChannelInt64`. Every mutation happens in another goroutine, communicated by channel. 30x slower than AtomicInt64

### Example AtomicInt64
```
    a := syncx.NewAtomicInt64(0)
	a.Set(100)
	a.Incr()
	a.Add(50)
	a.Sub(150)
	fmt.Println(a.Decr())   //0
	fmt.Println(a.String()) //0
	wg := sync.WaitGroup{}
	f := func(incr int64) {
		for i := 0; i < 100000; i++ {
			a.Add(incr)
		}
		wg.Done()
	}
	wg.Add(2)
	go f(1)
	go f(-1)
	wg.Wait()
	fmt.Println(a) //0
```
##### Example AtomicInt64 live logging
```
    //On startup
    globalCounter := syncx.NewAtomicInt64(0)

    //On Every Request
	log:=logrus.WithField("counter", globalCounter)
    a.Incr()
    processRequest(contextWithLogger)
    a.Decr()

    //Deep down stack inside processRequest()
    log.Log("Connected to Service") //Prints live globalCounter value    
```