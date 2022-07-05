package trybuffer

import (
	"sync/atomic"
	"time"
)

type TryBuffer struct {
	buffer  chan struct{}
	fun     func()
	isClose uint32
}

func NewTryBuffer(fun func(), duration time.Duration) *TryBuffer {
	t := &TryBuffer{
		buffer: make(chan struct{}, 1),
		fun:    fun,
	}

	go t.run(duration)
	return t
}

func (t *TryBuffer) run(duration time.Duration) {
	for range t.buffer {
	next:
		for {
			select {
			case _, ok := <-t.buffer:
				if !ok {
					return
				}
			case <-time.After(duration):
				break next
			}
		}
		if atomic.LoadUint32(&t.isClose) != 0 {
			return
		}
		t.fun()
	}
}

func (t *TryBuffer) Try() {
	if t == nil || t.buffer == nil || atomic.LoadUint32(&t.isClose) != 0 {
		return
	}
	select {
	case t.buffer <- struct{}{}:
	default:
	}
}

func (t *TryBuffer) Close() {
	if t == nil || t.buffer == nil || atomic.LoadUint32(&t.isClose) != 0 {
		return
	}
	atomic.StoreUint32(&t.isClose, 1)
	close(t.buffer)
}
