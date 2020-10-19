package syncx

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// ******************** L O C K - B E N C H M A R K S ********************

func buildUpDowns(ups int, downs int) []bool {
	updowns := make([]bool, ups+downs)
	for i := 0; i < ups; i++ {
		updowns[i] = true
	}
	rand.Shuffle(ups+downs, func(i, j int) {
		updowns[i], updowns[j] = updowns[j], updowns[i]
	})
	return updowns
}

func testLocks(locker sync.Locker, goroutines int, n int) int {
	ups, downs := 10*goroutines, 7*goroutines
	arr := buildUpDowns(ups, downs)
	count := 0
	wg := sync.WaitGroup{}
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		g := g
		go func() {
			defer wg.Done()
			for i := 0; i < n; i++ {
				for j := 0; j < 17; j++ {
					b := arr[17*g+j]
					locker.Lock()
					if b {
						count++
					} else {
						count--
					}
					// time.Sleep(5000 * time.Millisecond)
					locker.Unlock()
				}
			}
		}()
	}
	wg.Wait()
	if count != 3*goroutines*n {
		panic(fmt.Errorf("Invalid count=%v go=%v n=%v", count, goroutines, n))
	}
	return count
}

func BenchmarkPlainMutex_Lock(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testLocks(&sync.Mutex{}, 250, 1)
	}
}

func BenchmarkMutexTryLocker_Lock(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testLocks(NewMutexTryLocker(), 250, 1)
	}
}
func BenchmarkChannelTryLocker_Lock(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testLocks(NewChannelTryLocker(), 250, 1)
	}
}

func BenchmarkHackTryLocker_Lock(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testLocks(NewHackTryLocker(), 250, 1)
	}
}

func BenchmarkAtomicTryLocker_Lock(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testLocks(NewAtomicTryLocker(), 250, 10)
	}
}

// ******************** T R Y L O C K - B E N C H M A R K S ********************

func testTryLock(locker TryLocker, goroutines int, n int) int {
	ups, downs := 10*goroutines, 7*goroutines
	arr := buildUpDowns(ups, downs)
	// fmt.Println(arr)
	count := 0
	wg := sync.WaitGroup{}
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		g := g
		go func() {
			defer wg.Done()
			for i := 0; i < n; i++ {
				for j := 0; j < 17; j++ {
					b := arr[17*g+j]
					if j <= 10 {
						for !locker.TryLock() {
							// fmt.Println("TryAgain")
						}
					} else {
						locker.Lock()
					}
					if b {
						count++
					} else {
						count--
					}
					time.Sleep(100 * time.Millisecond)
					locker.Unlock()
				}
			}
		}()
	}
	wg.Wait()
	if count != 3*goroutines*n {
		panic(fmt.Errorf("Invalid count=%v go=%v n=%v", count, goroutines, n))
	}
	return count
}

func BenchmarkMutexTryLocker_TryLock(b *testing.B) {
	// m := NewMutexTryLockerWithThresholds(&sync.Mutex{}, 1000000*1000000, 1000000*1000000+1)
	m := NewMutexTryLocker()
	for i := 0; i < b.N; i++ {
		testTryLock(m, 12, 1)
	}
}

func BenchmarkChannelTryLocker_TryLock(b *testing.B) {
	m := NewChannelTryLocker()
	for i := 0; i < b.N; i++ {
		testTryLock(m, 50, 30)
	}
}

func BenchmarkHackTryLocker_TryLock(b *testing.B) {
	m := NewHackTryLocker()
	for i := 0; i < b.N; i++ {
		testTryLock(m, 50, 30)
	}
}
func BenchmarkAtomicTryLocker_TryLock(b *testing.B) {
	// m := NewAtomicTryLockerWithThresholds(1000000*1000000, 1000000*1000000+1)
	m := NewAtomicTryLocker()
	for i := 0; i < b.N; i++ {
		testTryLock(m, 12, 1)
	}
}
