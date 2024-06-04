package mtaskctl

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrCanceled = errors.New("canceled")
var ErrTimeout = errors.New("timeout")

const CHANNEL_NUMBER = 16

type MTaskCtl struct {
	max [CHANNEL_NUMBER]int // 最大并发数量

	waitGroup sync.WaitGroup                // 控制整体
	ch        [CHANNEL_NUMBER]chan struct{} // 控制各个channel的数量

	mtxPause  sync.Mutex    //
	pause     bool          // pause 暂停状态
	pauseChan chan struct{} // 暂停时有chan造成阻塞，恢复后close解除阻塞
	cancel    error         // 运行/取消状态

	timer   *time.Timer
	timeout time.Time
}

func NewTaskCtl(maxs []int) *MTaskCtl {
	if len(maxs) > CHANNEL_NUMBER {
		panic("不支持16以上的通道")
	}

	task := &MTaskCtl{
		pause:     true,
		pauseChan: make(chan struct{}),
	}
	for i, v := range maxs {
		task.max[i] = v
		task.ch[i] = make(chan struct{}, v)
		for j := 0; j < v; j++ {
			task.ch[i] <- struct{}{}
		}
	}
	task.Resume() // 默认不暂停
	return task
}

// func example() {
// 	ctl := NewTaskCtl([]int{8})
// 	for {
// 		channel, e := ctl.New()
// 		if e != nil {
// 			break
// 		}
// 		go func(channel int) {
// 			// do something
// 			ctl.Recycle(channel)
// 		}(channel)
// 	}
// 	ctl.Wait()
// }

// 分配一个可用的channel
// nil为正常继续，error为终止（Cancel），并发数量不足或pause时将阻塞
func (t *MTaskCtl) New() (channel int, e error) {
	t.waitGroup.Add(1)

	// 如果已经取消，则直接返回
	if e = t.Err(); e != nil {
		t.waitGroup.Done()
		return 0, e
	}

	// 等待空闲channel，或者close
	fmt.Println("mtaskctl.new: channel wait")
	// t.mtxCh.Lock()
	var ok bool
	select {
	case _, ok = <-t.ch[0]:
		channel = 0
	case _, ok = <-t.ch[1]:
		channel = 1
	case _, ok = <-t.ch[2]:
		channel = 2
	case _, ok = <-t.ch[3]:
		channel = 3
	case _, ok = <-t.ch[4]:
		channel = 4
	case _, ok = <-t.ch[5]:
		channel = 5
	case _, ok = <-t.ch[6]:
		channel = 6
	case _, ok = <-t.ch[7]:
		channel = 7
	case _, ok = <-t.ch[8]:
		channel = 8
	case _, ok = <-t.ch[9]:
		channel = 9
	case _, ok = <-t.ch[10]:
		channel = 10
	case _, ok = <-t.ch[11]:
		channel = 11
	case _, ok = <-t.ch[12]:
		channel = 12
	case _, ok = <-t.ch[13]:
		channel = 13
	case _, ok = <-t.ch[14]:
		channel = 14
	case _, ok = <-t.ch[15]:
		channel = 15
	}
	// t.mtxCh.Unlock()
	fmt.Println("mtaskctl.new: channel", channel)

	// 如果已经取消，则直接返回
	if e = t.Err(); e != nil {
		t.waitGroup.Done()
		return 0, e
	}

	// 如果通道关闭也返回
	if !ok {
		t.waitGroup.Done()
		return 0, ErrCanceled
	}
	return
}

// 完成后调用
func (t *MTaskCtl) Recycle(channel int) {
	fmt.Println("mtaskctl.recycle:", channel, "start")
	// t.mtxCh.Lock()
	// if len(t.ch[channel]) < cap(t.ch[channel]) {
	t.ch[channel] <- struct{}{}
	// }
	// t.mtxCh.Unlock()
	fmt.Println("mtaskctl.recycle:", channel, "finished")

	t.waitGroup.Done()
}

// 等待所有线程执行完，与Done功能相同
func (t *MTaskCtl) Wait() {
	t.waitGroup.Wait()
}

// Err 在运行routine过程中执行，检查是否继续。
// nil为正常继续，error为终止（Cancel），pause时阻塞
// If Done is not yet closed, Err returns nil.
// If Done is closed, Err returns a non-nil error explaining why:
// Canceled if the context was canceled
// or DeadlineExceeded if the context's deadline passed.
// After Err returns a non-nil error, successive calls to Err return the same error.
func (t *MTaskCtl) Err() error {
	// 暂停时阻塞，恢复(close)后解除阻塞
	<-t.pauseChan

	// 如果任务取消，则cancel为非nil
	return t.cancel
}

// Close关闭
func (t *MTaskCtl) Close() {
	for i := 0; i < CHANNEL_NUMBER; i++ {
		if t.ch[i] != nil {
			close(t.ch[i])
			t.ch[i] = nil
		}
	}
}

func (t *MTaskCtl) Timeout(timeout time.Duration) {
	t.timeout = time.Now().Add(timeout)

	run := true
	if t.timer == nil {
		t.timer = time.NewTimer(timeout)
	} else {
		run = !t.timer.Reset(timeout)
	}
	if run {
		go func() {
			fmt.Println("in timeout")
			tm := <-t.timer.C
			fmt.Println("set timeout", tm)
			t.Cancel(ErrTimeout)
		}()
	}
}

func (t *MTaskCtl) UnTimeout() {
	if t.timer != nil {
		t.timer.Stop()
	}
	t.timeout = time.Time{}
}

// 取消任务，cause为选填的取消原因
func (t *MTaskCtl) Cancel(cause error) {
	if cause != nil {
		t.cancel = cause
	} else {
		t.cancel = ErrCanceled
	}
	t.Resume()
}

func (t *MTaskCtl) UnCancel() {
	t.cancel = nil
}

// 暂停任务
func (t *MTaskCtl) Pause() {
	t.mtxPause.Lock()
	t.pause = true
	t.pauseChan = make(chan struct{})
	t.mtxPause.Unlock()
}

// 恢复任务
func (t *MTaskCtl) Resume() {
	t.mtxPause.Lock()
	if t.pause {
		close(t.pauseChan)
	}
	t.pause = false
	t.mtxPause.Unlock()
}

// Support Context interface

// Deadline returns the time when work done on behalf of this context
// should be canceled.  Deadline returns ok==false when no deadline is
// set.  Successive calls to Deadline return the same results.
func (t *MTaskCtl) Deadline() (deadline time.Time, ok bool) {
	if t.timeout.IsZero() {
		return time.Time{}, false
	}
	return t.timeout, true
}

func (t *MTaskCtl) Value(key interface{}) interface{} {
	return nil
}

// Done returns a channel that's closed when work done on behalf of this
// context should be canceled. Done may return nil if this context can
// never be canceled. Successive calls to Done return the same value.
// The close of the Done channel may happen asynchronously,
// after the cancel function returns.
//
// WithCancel arranges for Done to be closed when cancel is called;
// WithDeadline arranges for Done to be closed when the deadline
// expires; WithTimeout arranges for Done to be closed when the timeout
// elapses.
//
// Done is provided for use in select statements:
//
//	// Stream generates values with DoSomething and sends them to out
//	// until DoSomething returns an error or ctx.Done is closed.
//	func Stream(ctx context.Context, out chan<- Value) error {
//		for {
//			v, err := DoSomething(ctx)
//			if err != nil {
//				return err
//			}
//			select {
//			case <-ctx.Done():
//				return ctx.Err()
//			case out <- v:
//			}
//		}
//	}
//
// See https://blog.golang.org/pipelines for more examples of how to use
// a Done channel for cancellation.
func (t *MTaskCtl) Done() <-chan struct{} {
	c := make(chan struct{})
	go func() {
		t.waitGroup.Wait()
		c <- struct{}{}
	}()
	return c
}
