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
	exampleTryLock(NewMutexTryLocker())
	//Output:
	//false
	//true
}

func ExampleNewChannelTryLocker() {
	exampleTryLock(NewChannelTryLocker())
	//Output: false
	//true
}

func ExampleNewHackTryLocker() {
	exampleTryLock(NewHackTryLocker())
	//Output: false
	//true
}
