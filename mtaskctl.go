package mtaskctl

import (
	"errors"
	"sync"
)

var Canceled = errors.New("canceled")

type MTaskCtl struct {
	i      int           // 当前并发数量
	n      int           // 最大并发数量
	e      error         // 运行/取消状态
	mu     sync.Mutex    //
	pa     bool          // pause 暂停状态
	resume chan struct{} // 恢复信号
	ch     chan error    // 控制数量，cancel后error为非空
}

func NewTaskCtl(n int) *MTaskCtl {
	task := &MTaskCtl{
		n:      n,
		resume: make(chan struct{}),
		ch:     make(chan error, n),
	}
	for i := 0; i < n; i++ {
		task.ch <- nil
	}
	return task
}

// Start在运行goroutine之前执行，检查是否可以开始。
// nil为正常继续，error为终止（Cancel），并发数量不足或pause时将阻塞
func (t *MTaskCtl) Start() error {
	if t.e != nil {
		return t.e
	}
	e := <-t.ch
	if e == nil {
		t.i++
	}
	return e
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
func (t *MTaskCtl) Done() {
	t.mu.Lock()
	t.i--
	if !t.pa && len(t.ch) < cap(t.ch) {
		t.ch <- t.e
	}
	t.mu.Unlock()
}

// Wait等待所有线程执行完
func (t *MTaskCtl) Wait() {
	for t.i != 0 {
		<-t.ch
	}
}

// Close关闭
func (t *MTaskCtl) Close() {
	if t.ch != nil {
		close(t.ch)
		t.ch = nil
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
	for i := t.i; i < t.n; i++ {
		<-t.ch
	}
	t.mu.Unlock()
}

// TaskResume恢复任务
func (t *MTaskCtl) TaskResume() {
	t.mu.Lock()
	t.pa = false
	for i := t.i; i < t.n; i++ {
		t.ch <- t.e
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
