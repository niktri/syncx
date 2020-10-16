package syncx

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func testIncrDecr(counter ConcurrentInt64, goroutines int, n int) int {
	ups, downs := 10*goroutines, 7*goroutines
	arr := buildUpDowns(ups, downs)
	counter.Set(0)
	wg := sync.WaitGroup{}
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		g := g
		go func() {
			defer wg.Done()
			for i := 0; i < n; i++ {
				for j := 0; j < 17; j++ {
					b := arr[17*g+j]
					if b {
						counter.Incr()
					} else {
						counter.Decr()
					}
				}
			}
		}()
	}
	wg.Wait()
	count := int(counter.Get())
	//This will fail for ChannelCounter, as it's lazy.
	if count != 3*goroutines*n {
		panic(fmt.Errorf("Invalid count=%v go=%v n=%v", count, goroutines, n))
	}
	return count
}

func BenchmarkAtomicInt64(b *testing.B) {
	a := NewAtomicInt64(0)
	for i := 0; i < b.N; i++ {
		testIncrDecr(a, 100, 100)
	}
}

func BenchmarkMutexInt64(b *testing.B) {
	a := NewMutexInt64(0)
	for i := 0; i < b.N; i++ {
		testIncrDecr(a, 100, 100)
	}
}

func BenchmarkChannelInt64_16(b *testing.B) {
	a := NewChannelInt64(0, 16)
	for i := 0; i < b.N; i++ {
		testIncrDecr(a, 100, 100)
	}
}

func BenchmarkChannelInt64_1024(b *testing.B) {
	a := NewChannelInt64(0, 1024)
	for i := 0; i < b.N; i++ {
		testIncrDecr(a, 100, 100)
	}
}
func BenchmarkChannelInt64_64(b *testing.B) {
	a := NewChannelInt64(0, 64)
	for i := 0; i < b.N; i++ {
		testIncrDecr(a, 100, 100)
	}
}
func BenchmarkChannelInt64_1(b *testing.B) {
	a := NewChannelInt64(0, 1)
	for i := 0; i < b.N; i++ {
		testIncrDecr(a, 100, 100)
	}
}

func ExampleChannelInt64() {
	a := NewChannelInt64(0, 64)
	testIncrDecr(a, 500, 500)
	time.Sleep(1 * time.Second) //Let channel drained
	fmt.Println(a.Get())
	//Output:
	//750000
}
