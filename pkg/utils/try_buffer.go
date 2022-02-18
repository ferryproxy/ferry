package utils

import (
	"time"
)

type TryBuffer struct {
	buffer chan struct{}
	fun    func()
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
		t.fun()
	}
}

func (t *TryBuffer) Try() {
	t.buffer <- struct{}{}
}

func (t *TryBuffer) Close() {
	close(t.buffer)
}
