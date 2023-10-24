package mtaskctl

import (
	"errors"
	"sync"
)

var Canceled = errors.New("canceled")

const CHANNEL_NUMBER = 8

type MTaskCtl struct {
	max [CHANNEL_NUMBER]int // 最大并发数量

	mtxN sync.Mutex
	n    [CHANNEL_NUMBER]int           // 当前并发数量
	ch   [CHANNEL_NUMBER]chan struct{} // 控制数量，cancel后error为非空

	mtxPause  sync.Mutex    //
	pause     bool          // pause 暂停状态
	pauseChan chan struct{} // 暂停时有chan造成阻塞，恢复后close解除阻塞
	cancel    error         // 运行/取消状态
}

func NewTaskCtl(max []int) *MTaskCtl {
	if len(max) > CHANNEL_NUMBER {
		panic("不支持8个以上的通道")
	}

	task := &MTaskCtl{
		pause:     true,
		pauseChan: make(chan struct{}),
	}
	for i, v := range max {
		task.max[i] = v
		task.ch[i] = make(chan struct{}, v)
		for j := 0; j < v; j++ {
			task.ch[i] <- struct{}{}
		}
	}
	task.Resume() // 默认不暂停
	return task
}

// perHandler 预分配的handler，外侧决定是否继续
// handle 实际处理的handler，在协程中执行
func (t *MTaskCtl) Do(perHandler func(index int) bool, handle func(index, channel int)) {
	for index := 0; ; index++ {
		// 外侧决定是否需要继续
		if !perHandler(index) {
			break
		}

		// 分配/等待channel，也可能被取消
		channel, e := t.assign()
		if e != nil {
			break
		}

		go func(i, channel int) {
			handle(i, channel)
			t.done(channel)
		}(index, channel)
	}
	t.wait()
}

// 分配一个可用的channel
// nil为正常继续，error为终止（Cancel），并发数量不足或pause时将阻塞
func (t *MTaskCtl) assign() (channel int, e error) {
	// 如果已经取消，则直接返回
	if e = t.Check(); e != nil {
		return 0, e
	}

	// 等待空闲channel，或者close
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
	}

	// 如果已经取消，则直接返回
	if e = t.Check(); e != nil {
		return 0, e
	}

	// 如果通道关闭也返回
	if !ok {
		return 0, Canceled
	}

	t.mtxN.Lock()
	t.n[channel]++
	t.mtxN.Unlock()
	return
}

// 完成后插入新元素
func (t *MTaskCtl) done(index int) {
	t.mtxN.Lock()
	t.n[index]--
	t.mtxN.Unlock()

	if len(t.ch[index]) < cap(t.ch[index]) {
		t.ch[index] <- struct{}{}
	}
}

// 等待所有线程执行完
func (t *MTaskCtl) wait() {
	for i := 0; i < CHANNEL_NUMBER; i++ {
		for t.n[i] > 0 {
			<-t.ch[i]
		}
	}
}

// Check在运行routine过程中执行，检查是否继续。
// nil为正常继续，error为终止（Cancel），pause时阻塞
func (t *MTaskCtl) Check() error {
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

// 取消任务，cause为选填的取消原因
func (t *MTaskCtl) Cancel(cause error) {
	if cause != nil {
		t.cancel = cause
	} else {
		t.cancel = Canceled
	}
	t.Resume()
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
