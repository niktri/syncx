package syncx

import (
	"fmt"
)

// ******************** E X A M P L E S ********************

func exampleTryLock(m TryLocker) {
	m.Lock()
	fmt.Println(m.TryLock())
	m.Unlock()
	fmt.Println(m.TryLock())
	m.Unlock()
}

func ExampleNewMutexTryLocker() {
	m := NewMutexTryLocker()
	m.Lock()
	fmt.Println(m.TryLock())
	m.Unlock()
	fmt.Println(m.TryLock())
	m.Unlock()
	//Output:
	//false
	//true
}

func ExampleNewChannelTryLocker() {
	m := NewChannelTryLocker()
	m.Lock()
	fmt.Println(m.TryLock())
	m.Unlock()
	fmt.Println(m.TryLock())
	m.Unlock()
	//Output: false
	//true
}

func ExampleNewHackTryLocker() {
	m := NewHackTryLocker()
	m.Lock()
	fmt.Println(m.TryLock())
	m.Unlock()
	fmt.Println(m.TryLock())
	m.Unlock()
	//Output: false
	//true
}

func ExampleNewAtomicTryLocker() {
	m := NewHackTryLocker()
	m.Lock()
	fmt.Println(m.TryLock())
	m.Unlock()
	fmt.Println(m.TryLock())
	m.Unlock()
	//Output: false
	//true
}
