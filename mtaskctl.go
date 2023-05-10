package mtaskctl

import (
	"context"
	"errors"
	"sync"
)

var Canceled = errors.New("canceled")

const SPOOL_NUMBER = 8

type MTaskCtl struct {
	n      [SPOOL_NUMBER]int        // 当前并发数量
	max    [SPOOL_NUMBER]int        // 最大并发数量
	ch     [SPOOL_NUMBER]chan error // 控制数量，cancel后error为非空
	mu     sync.Mutex               //
	e      error                    // 运行/取消状态
	pa     bool                     // pause 暂停状态
	resume chan struct{}            // 恢复信号
}

func NewTaskCtl(max [SPOOL_NUMBER]int) *MTaskCtl {
	task := &MTaskCtl{
		max:    max,
		resume: make(chan struct{}),
	}

	for i, v := range max {
		task.ch[i] = make(chan error, v)
		for j := 0; j < v; j++ {
			task.ch[i] <- nil
		}
	}
	return task
}

// 代理Start、Check、Done
func (t *MTaskCtl) Do(cb func(i, channel int, cancel context.CancelFunc)) {
	ctx, cancel := context.WithCancel(context.Background())
	for i := 0; ; i++ {
		channel, e := t.Start()
		if e != nil {
			break
		}
		if ctx.Err() != nil {
			t.Done(channel)
			break
		}
		go func(i, channel int) {
			cb(i, channel, cancel)
			t.Done(channel)
		}(i, channel)
	}
	t.Wait()
	cancel()
}

// Start在运行goroutine之前执行，检查是否可以开始。
// nil为正常继续，error为终止（Cancel），并发数量不足或pause时将阻塞
func (t *MTaskCtl) Start() (index int, e error) {
	if t.e != nil {
		return 0, t.e
	}

	select {
	case e = <-t.ch[0]:
		index = 0
	case e = <-t.ch[1]:
		index = 1
	case e = <-t.ch[2]:
		index = 2
	case e = <-t.ch[3]:
		index = 3
	case e = <-t.ch[4]:
		index = 4
	case e = <-t.ch[5]:
		index = 5
	case e = <-t.ch[6]:
		index = 6
	case e = <-t.ch[7]:
		index = 7
	}
	if e == nil {
		t.n[index]++
	}
	return
}

// Check在运行routine过程中执行，检查是否继续。
// nil为正常继续，error为终止（Cancel），pause时阻塞
func (t *MTaskCtl) Check() error {
	if t.pa {
		<-t.resume
	}
	return t.e
}

// Done线程完成后回收资源。Check为error时，也需要调用Done。
func (t *MTaskCtl) Done(index int) {
	t.mu.Lock()
	t.n[index]--
	if !t.pa && len(t.ch[index]) < cap(t.ch[index]) {
		t.ch[index] <- t.e
	}
	t.mu.Unlock()
}

// Wait等待所有线程执行完
func (t *MTaskCtl) Wait() {
	for i := 0; i < SPOOL_NUMBER; i++ {
		for t.n[i] != 0 {
			<-t.ch[i]
		}
	}
}

// Close关闭
func (t *MTaskCtl) Close() {
	for i := 0; i < SPOOL_NUMBER; i++ {
		if t.ch[i] != nil {
			close(t.ch[i])
			t.ch[i] = nil
		}
	}
	if t.resume != nil {
		close(t.resume)
		t.resume = nil
	}
}

// TaskCancel取消任务，cause为选填的取消原因
func (t *MTaskCtl) TaskCancel(cause error) {
	if cause != nil {
		t.e = cause
	} else {
		t.e = Canceled
	}
	if t.pa {
		t.TaskResume()
	}
}

// TaskPause暂停任务
func (t *MTaskCtl) TaskPause() {
	t.mu.Lock()
	t.pa = true
	for i := 0; i < SPOOL_NUMBER; i++ {
		for j := t.n[i]; j < t.max[i]; j++ {
			<-t.ch[i]
		}
	}
	t.mu.Unlock()
}

// TaskResume恢复任务
func (t *MTaskCtl) TaskResume() {
	t.mu.Lock()
	t.pa = false
	for i := 0; i < SPOOL_NUMBER; i++ {
		for j := t.n[i]; j < t.max[i]; j++ {
			t.ch[i] <- t.e
		}
	}
	for {
		select {
		case t.resume <- struct{}{}:
		default:
			goto end
		}
	}
end:
	t.mu.Unlock()
}
