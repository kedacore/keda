package queue

import (
	"container/ring"
)

// Holder provides synchronized access to a *Queue[T].
type Holder[T any] struct {
	// these channels work in tandem to provide exclusive access to the underlying *Queue[T].
	// each channel is created with a buffer size of one.
	// empty behaves like a mutex when there's one or more messages in the queue.
	// populated is like a semaphore when the queue is empty.
	// the *Queue[T] is only ever in one channel. which channel depends on if it contains any items.
	// the initial state is for empty to contain an empty queue.
	empty     chan *Queue[T]
	populated chan *Queue[T]
}

// NewHolder creates a new Holder[T] that contains the provided *Queue[T].
func NewHolder[T any](q *Queue[T]) *Holder[T] {
	h := &Holder[T]{
		empty:     make(chan *Queue[T], 1),
		populated: make(chan *Queue[T], 1),
	}
	h.Release(q)
	return h
}

// Acquire attempts to acquire the *Queue[T]. If the *Queue[T] has already been acquired the call blocks.
// When the *Queue[T] is no longer required, you MUST call Release() to relinquish acquisition.
func (h *Holder[T]) Acquire() *Queue[T] {
	// the queue will be in only one of the channels, it doesn't matter which one
	var q *Queue[T]
	select {
	case q = <-h.empty:
		// empty queue
	case q = <-h.populated:
		// populated queue
	}
	return q
}

// Wait returns a channel that's signaled when the *Queue[T] contains at least one item.
// When the *Queue[T] is no longer required, you MUST call Release() to relinquish acquisition.
func (h *Holder[T]) Wait() <-chan *Queue[T] {
	return h.populated
}

// Release returns the *Queue[T] back to the Holder[T].
// Once the *Queue[T] has been released, it is no longer safe to call its methods.
func (h *Holder[T]) Release(q *Queue[T]) {
	if q.Len() == 0 {
		h.empty <- q
	} else {
		h.populated <- q
	}
}

// Len returns the length of the *Queue[T].
func (h *Holder[T]) Len() int {
	msgLen := 0
	select {
	case q := <-h.empty:
		h.empty <- q
	case q := <-h.populated:
		msgLen = q.Len()
		h.populated <- q
	}
	return msgLen
}

// Queue[T] is a segmented FIFO queue of Ts.
type Queue[T any] struct {
	head *ring.Ring
	tail *ring.Ring
	size int
}

// New creates a new instance of Queue[T].
//   - size is the size of each Queue segment
func New[T any](size int) *Queue[T] {
	r := &ring.Ring{
		Value: &segment[T]{
			items: make([]*T, size),
		},
	}
	return &Queue[T]{
		head: r,
		tail: r,
	}
}

// Enqueue adds the specified item to the end of the queue.
// If the current segment is full, a new segment is created.
func (q *Queue[T]) Enqueue(item T) {
	for {
		r := q.tail
		seg := r.Value.(*segment[T])

		if seg.tail < len(seg.items) {
			seg.items[seg.tail] = &item
			seg.tail++
			q.size++
			return
		}

		// segment is full, can we advance?
		if next := r.Next(); next != q.head {
			q.tail = next
			continue
		}

		// no, add a new ring
		r.Link(&ring.Ring{
			Value: &segment[T]{
				items: make([]*T, len(seg.items)),
			},
		})

		q.tail = r.Next()
	}
}

// Dequeue removes and returns the item from the front of the queue.
func (q *Queue[T]) Dequeue() *T {
	r := q.head
	seg := r.Value.(*segment[T])

	if seg.tail == 0 {
		// queue is empty
		return nil
	}

	// remove first item
	item := seg.items[seg.head]
	seg.items[seg.head] = nil
	seg.head++
	q.size--

	if seg.head == seg.tail {
		// segment is now empty, reset indices
		seg.head, seg.tail = 0, 0

		// if we're not at the last ring, advance head to the next one
		if q.head != q.tail {
			q.head = r.Next()
		}
	}

	return item
}

// Len returns the total count of enqueued items.
func (q *Queue[T]) Len() int {
	return q.size
}

type segment[T any] struct {
	items []*T
	head  int
	tail  int
}
