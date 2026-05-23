package utils

import (
	"sync"
	"sync/atomic"
)

type Once[Out any] interface {
	Do(f func() (Out, error)) (Out, error)
	Done() bool
	Result() (bool, Out, error)
}

type OnceWithInit[Out any] interface {
	Once[Out]
	DoWithInit() (Out, error)
}

type once[Out any] struct {
	mutex  sync.Mutex
	result Out
	err    error
	done   uint32
}

type onceWithInit[Out any] struct {
	inner Once[Out]
	f     func() (Out, error)
}

func NewOnce[Out any]() Once[Out] {
	var empty Out
	return &once[Out]{
		mutex:  sync.Mutex{},
		result: empty,
		err:    nil,
		done:   0,
	}
}

func NewOnceWithInit[Out any](f func() (Out, error)) OnceWithInit[Out] {
	return &onceWithInit[Out]{
		inner: NewOnce[Out](),
		f:     f,
	}
}
func (o *onceWithInit[Out]) DoWithInit() (Out, error) {
	return o.inner.Do(o.f)
}

func (o *onceWithInit[Out]) Do(f func() (Out, error)) (Out, error) {
	return o.inner.Do(f)
}

func (o *onceWithInit[Out]) Done() bool {
	return o.inner.Done()
}

func (o *onceWithInit[Out]) Result() (bool, Out, error) {
	return o.inner.Result()
}

func (o *once[Out]) Do(f func() (Out, error)) (Out, error) {
	if atomic.LoadUint32(&o.done) != 0 {
		return o.result, o.err
	}

	o.mutex.Lock()
	defer o.mutex.Unlock()
	if o.done == 0 {
		o.result, o.err = f()

		if o.err == nil {
			defer atomic.StoreUint32(&o.done, 1)
		}
	}

	return o.result, o.err
}

func (o *once[Out]) Done() bool {
	return o.done != 0
}

func (o *once[Out]) Result() (bool, Out, error) {
	return o.Done(), o.result, o.err
}
