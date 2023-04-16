package mtaskctl

import (
	"errors"
	"sync"
)

var Canceled = errors.New("canceled")

type TaskCtl struct {
	i  int
	n  int
	e  error
	mu sync.Mutex
	pa bool
	ch chan error
}

func NewTaskCtl(n int) *TaskCtl {
	task := &TaskCtl{
		n: n,
	}
	task.ch = make(chan error, n)
	for i := 0; i < n; i++ {
		task.ch <- nil
	}
	return task
}

// Working返回nil为继续，error为终止，并发数量不足或pause时阻塞
func (t *TaskCtl) Working() error {
	e := <-t.ch
	if e == nil {
		t.i++
	}
	return e
}

// Done线程完成后回收资源
func (t *TaskCtl) Done() {
	t.i--
	if !t.pa && len(t.ch) < cap(t.ch) {
		t.ch <- t.e
	}
}

// Wait等待所有线程执行完
func (t *TaskCtl) Wait() {
	for t.i != 0 {
		<-t.ch
	}
}

// Cancel取消任务
func (t *TaskCtl) Cancel() {
	t.e = Canceled
}

// Pause暂停任务
func (t *TaskCtl) Pause() {
	t.mu.Lock()
	t.pa = true
	for i := t.i; i < t.n; i++ {
		<-t.ch
	}
	t.mu.Unlock()
}

// Resume恢复任务
func (t *TaskCtl) Resume() {
	t.mu.Lock()
	t.pa = false
	for i := t.i; i < t.n; i++ {
		t.ch <- t.e
	}
	t.mu.Unlock()
}
