package syncx

import (
	"fmt"
	"math/rand"
	"sync"
)

func ExampleAtomicInt64_usage() {
	a := NewAtomicInt64(0)
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
	//Output:
	//0
	//0
	//0
}

var loggerFields map[string]interface{}

func addToLogField(key string, val interface{}) {
	if loggerFields == nil {
		loggerFields = map[string]interface{}{}
	}
	loggerFields[key] = val
}

func logFields() {
	fmt.Println(loggerFields)
}

func ExampleAtomicInt64_print() {
	a := NewAtomicInt64(0)
	addToLogField("counter", a)
	a.Incr()
	logFields()
	a.Add(100)
	logFields()
	//Output:
	//map[counter:1]
	//map[counter:101]
}

func randomOnesAndMinusOnes(ones, minusones int) []int {
	total := ones + minusones
	ret := make([]int, total)
	for i := 0; i < ones; i++ {
		ret[i] = 1
	}
	for i := ones; i < total; i++ {
		ret[i] = -1
	}
	rand.Shuffle(total, func(i, j int) {
		ret[i], ret[j] = ret[j], ret[i]
	})
	return ret
}

func ExampleAtomicInt64() {
	a := NewAtomicInt64(0)
	arr := randomOnesAndMinusOnes(10000, 5000) //create 15k `1` & `-1`
	wg := sync.WaitGroup{}
	for i := 0; i < 150; i++ { //150 threads
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ { //Each thread gets 100
				incr := arr[i*100+j]
				if incr == 1 {
					a.Incr()
				} else {
					a.Decr()
				}
			}
		}()
	}
	wg.Wait()
	fmt.Println(a)
	//Output: 5000
}
