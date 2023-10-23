package mtaskctl

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func Test_TaskDo(t *testing.T) {
	pf := newpf()
	ctl := NewTaskCtl([]int{2})

	go func() {
		time.Sleep(2 * time.Second)
		pf.p("======= pause =======")
		ctl.Pause()
		// pf.p("======= pause =======")

		time.Sleep(8 * time.Second)
		pf.p("======= resume =======")
		ctl.Resume()
		// pf.p("======= resume =======")

		// time.Sleep(8 * time.Second) // start_error
		time.Sleep(5 * time.Second) // check_error
		pf.p("======= pause =======")
		ctl.Pause()
		// pf.p("======= pause =======")

		time.Sleep(8 * time.Second)
		pf.p("======= cancel =======")
		ctl.Cancel(nil)
		pf.p("======= cancel =======")
	}()

	ctl.Do(func(index, channel int) {
		pf.p("index:", index, " channel:", channel, "start")
		if index == 10 {
			ctl.Cancel(nil)
			pf.p("cancel")
			return
		}

		time.Sleep(1 * time.Second)

		if e := ctl.Check(); e != nil {
			pf.p("check out", e.Error())
			return
		}

		time.Sleep(1 * time.Second)

		if e := ctl.Check(); e != nil {
			pf.p("check out", e.Error())
			return
		}

		pf.p("index:", index, " channel:", channel, "--- end")
	})
	pf.p("Finished")
}

type pf struct {
	start time.Time
	mu    sync.Mutex
}

func newpf() *pf {
	return &pf{
		start: time.Now(),
	}
}

func (t *pf) p(a ...any) {
	t.mu.Lock()
	fmt.Printf("%6.2fs ", float64(time.Now().Sub(t.start).Milliseconds())/1000)
	fmt.Println(a...)
	t.mu.Unlock()
}
