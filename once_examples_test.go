package syncx

import (
	"fmt"
	"time"
)

func ExampleOnce_IsDone() {
	once := Once{}
	fmt.Println(once.IsDone()) //false
	go once.Do(func() {
		time.Sleep(3 * time.Second)
	})
	fmt.Println(once.IsDone()) //false
	time.Sleep(1 * time.Second)
	fmt.Println(once.IsDone()) //false
	time.Sleep(5 * time.Second)
	fmt.Println(once.IsDone()) //true
	//Output:
	//false
	//false
	//false
	//true
}
