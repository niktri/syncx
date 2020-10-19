package syncx

import (
	"fmt"
)

// ******************** E X A M P L E S ********************

func ExampleNewMutexRWTryLocker() {
	m := NewMutexRWTryLocker()
	m.Lock()
	fmt.Println(m.TryLock())  //false
	fmt.Println(m.TryRLock()) //false
	m.Unlock()
	fmt.Println(m.TryLock())  //true
	fmt.Println(m.TryRLock()) //false
	m.Unlock()
	fmt.Println(m.TryRLock()) //true
	fmt.Println(m.TryLock())  //false
	fmt.Println(m.TryRLock()) //true
	fmt.Println(m.TryRLock()) //true
	fmt.Println(m.TryLock())  //false
	m.RUnlock()
	m.RUnlock()
	m.RUnlock()
	fmt.Println(m.TryLock()) //true
	m.Unlock()
	//Output:
	//false
	//false
	//true
	//false
	//true
	//false
	//true
	//true
	//false
	//true
}
