package mtaskctl

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func Test_test(t *testing.T) {
	c := make(chan struct{})
	go func() {
		<-c
		fmt.Println("ok")
	}()

	c = make(chan struct{}, 3)
	fmt.Println(len(c), cap(c))

	time.Sleep(100 * time.Millisecond)
	c <- struct{}{}
	fmt.Println(len(c), cap(c))

	time.Sleep(100 * time.Millisecond)
	c <- struct{}{}
	fmt.Println(len(c), cap(c))
}

func Test_Task(t *testing.T) {
	ctl := NewTaskCtl(2)

	go func() {
		time.Sleep(2 * time.Second)
		fmt.Println("pause")
		ctl.Pause()

		time.Sleep(5 * time.Second)
		fmt.Println("resume")
		ctl.Resume()

		time.Sleep(8 * time.Second)
		fmt.Println("cancel")
		ctl.Cancel()
	}()

	for i := 0; i < 10; i++ {
		fmt.Println(i, "working")
		e := ctl.Working()
		if e != nil {
			fmt.Println(i, "working error")
			break
		}
		fmt.Println(i, "start")

		go func(i int) {
			time.Sleep(3 * time.Second)
			fmt.Println(i, "done")
			ctl.Done()
		}(i)
	}

	fmt.Println("wait")
	ctl.Wait()
}

func Test_value(t *testing.T) {
	v := atomic.Value{}
	v.Store(1)
	v.Store(2)
	fmt.Println(v.Load())
	fmt.Println(v.Load())
	fmt.Println(v.Load())
}
