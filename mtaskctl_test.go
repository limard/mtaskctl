package mtaskctl

import (
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

	ctl := NewTaskCtl(2)

	go func() {
		time.Sleep(2 * time.Second)
		pf("======= pause =======")
		ctl.TaskPause()

		time.Sleep(8 * time.Second)
		pf("======= resume =======")
		ctl.TaskResume()

		// time.Sleep(8 * time.Second) // start_error
		time.Sleep(5 * time.Second) // check_error
		pf("======= pause =======")
		ctl.TaskPause()

		time.Sleep(8 * time.Second)
		pf("======= cancel =======")
		ctl.TaskCancel(nil)
	}()

	for i := 0; i < 10; i++ {
		pf(i, "start1")
		if ctl.Start() != nil {
			pf(i, "start-error")
			break
		}
		pf(i, "\tstart2")

		go func(i int) {
			defer func() {
				pf(i, "\t\t\tdone")
				ctl.Done()
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
