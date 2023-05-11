package mtaskctl

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func Test_Task(t *testing.T) {
	start := time.Now()
	mu := sync.Mutex{}
	pf := func(a ...any) {
		mu.Lock()
		fmt.Printf("%6.2fs ", float64(time.Now().Sub(start).Milliseconds())/1000)
		fmt.Println(a...)
		mu.Unlock()
	}

	ctl := NewTaskCtl([]int{2})

	go func() {
		time.Sleep(2 * time.Second)
		pf("======= pause1 =======")
		ctl.TaskPause()
		pf("======= pause2 =======")

		time.Sleep(8 * time.Second)
		pf("======= resume1 =======")
		ctl.TaskResume()
		pf("======= resume2 =======")

		// time.Sleep(8 * time.Second) // start_error
		time.Sleep(5 * time.Second) // check_error
		pf("======= pause1 =======")
		ctl.TaskPause()
		pf("======= pause2 =======")

		time.Sleep(8 * time.Second)
		pf("======= cancel1 =======")
		ctl.TaskCancel(nil)
		pf("======= cancel2 =======")
	}()

	for i := 0; i < 10; i++ {
		pf(i, "start1")
		index, e := ctl.Start()
		if e != nil {
			pf(i, "start-error")
			break
		}
		pf(i, "\tstart2")

		go func(i int) {
			defer func() {
				pf(i, "\t\t\tdone")
				ctl.Done(index)
			}()

			time.Sleep(3 * time.Second)
			pf(i, "\t\tcheck1")
			if ctl.Check() != nil {
				pf(i, "\t\tcheck-error")
				return
			}
			pf(i, "\t\tcheck2")

			time.Sleep(3 * time.Second)
		}(i)
	}

	pf("======= wait =======")
	ctl.Wait()
	ctl.Close()
}

func Test_do(t *testing.T) {
	ctl := NewTaskCtl([]int{2, 3})

	ctl.Do(func(n, channel int, cancel context.CancelFunc) {
		fmt.Println(n, channel, "start")
		if n == 10 {
			cancel()
			fmt.Println("cancel")
			return
		}
		time.Sleep(1 * time.Second)
		fmt.Println(n, channel, "       --- end")
	})
}
