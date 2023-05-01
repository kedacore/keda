package utils

import "sync"

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
	inner  sync.Once
	result Out
	err    error
	done   bool
}

type onceWithInit[Out any] struct {
	inner Once[Out]
	f     func() (Out, error)
}

func NewOnce[Out any]() Once[Out] {
	var empty Out
	return &once[Out]{
		inner:  sync.Once{},
		result: empty,
		err:    nil,
		done:   false,
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
	o.inner.Do(func() {
		o.result, o.err = f()
		o.done = true
	})
	return o.result, o.err
}

func (o *once[Out]) Done() bool {
	return o.done
}

func (o *once[Out]) Result() (bool, Out, error) {
	return o.done, o.result, o.err
}
