package mtaskctl

import (
	"fmt"
	"math/rand"
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

	for index := 0; ; index++ {
		channel, e := ctl.New()
		if e != nil {
			break
		}
		go func(channel, index int) {
			defer ctl.Recycle(channel)

			time.Sleep(1 * time.Second)

			if e := ctl.Err(); e != nil {
				pf.p("check out", e.Error())
				return
			}

			time.Sleep(1 * time.Second)

			if e := ctl.Err(); e != nil {
				pf.p("check out", e.Error())
				return
			}

			pf.p("index:", index, " channel:", channel, "--- end")

		}(channel, index)
	}
	ctl.Wait()

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
	fmt.Printf("%6.2fs ", float64(time.Since(t.start).Milliseconds())/1000)
	fmt.Println(a...)
	t.mu.Unlock()
}

// 测试并发
func Test_coo(t *testing.T) {
	pgm_start := time.Now()
	ctl := NewTaskCtl([]int{8})
	for index := 0; index < 20; index++ {
		channel, e := ctl.New()
		if e != nil {
			break
		}
		go func(channel, index int) {
			defer ctl.Recycle(channel)

			start := time.Now()
			n := rand.Intn(5) + 1
			time.Sleep(time.Duration(n) * time.Second)
			// time.Sleep(2 * time.Second)

			fmt.Printf("index:%d \tchannel:%d \t start:%d \tduration:%d \n",
				index, channel, start.Sub(pgm_start).Milliseconds(), time.Since(start).Milliseconds())
		}(channel, index)
	}
	ctl.Wait()
	ctl.Close()
	time.Sleep(2 * time.Second)
}

func Benchmark_(b *testing.B) {
	cnter := [2]int{}
	ctl := NewTaskCtl([]int{4, 4})
	for i := 0; i < b.N; i++ {
		channel, e := ctl.New()
		if e != nil {
			break
		}
		go func(channel, index int) {
			cnter[channel]++
			ctl.Recycle(channel)
		}(channel, i)
	}
	ctl.Wait()
	ctl.Close()
	fmt.Println(cnter)
}

func Test_Timeout(t *testing.T) {
	c := NewTaskCtl([]int{1})

	go func() {
		for i := 0; i < 20; i++ {
			if e := c.Err(); e != nil {
				fmt.Println(e, time.Now())
			}
			time.Sleep(1 * time.Second)
		}
	}()

	c.Timeout(3 * time.Second)

	time.Sleep(2 * time.Second)
	c.UnTimeout()

	c.Timeout(3 * time.Second)
	time.Sleep(7 * time.Second)

	c.UnTimeout()
	c.UnCancel()

	c.Timeout(3 * time.Second)
	time.Sleep(7 * time.Second)
}

func Test_Done(t *testing.T) {
	c := NewTaskCtl([]int{1})

	channel, e := c.New()
	if e != nil {
		fmt.Println(e)
		return
	}
	go func() {
		time.Sleep(2 * time.Second)
		c.Recycle(channel)
	}()

	fmt.Println(time.Now())
	c.Done()
	fmt.Println(time.Now())
}
