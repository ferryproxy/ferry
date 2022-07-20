/*
Copyright 2022 FerryProxy Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
