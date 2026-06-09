package circular

import "iter"

// Queue is a bounded FIFO queue implemented using a circular array.
// It uses head and tail pointers to avoid slice re-allocations.
// When full, new elements are rejected rather than overwriting old ones.
type Queue[T any] struct {
	data []T
	head int // index of the first element
	tail int // index where the next element will be inserted
	len  int // number of elements in the queue
}

// New creates a new circular queue with the given capacity.
func New[T any](capacity int) *Queue[T] {
	return &Queue[T]{data: make([]T, capacity)}
}

// Len returns the number of elements in the queue.
func (q *Queue[T]) Len() int {
	return q.len
}

// Cap returns the capacity of the queue.
func (q *Queue[T]) Cap() int {
	return len(q.data)
}

// IsFull returns true if the queue is at capacity.
func (q *Queue[T]) IsFull() bool {
	return q.len == len(q.data)
}

// IsEmpty returns true if the queue is empty.
func (q *Queue[T]) IsEmpty() bool {
	return q.len == 0
}

// Push adds an element to the tail of the queue.
// Returns false if the queue is full.
func (q *Queue[T]) Push(value T) bool {
	if q.IsFull() {
		return false
	}

	q.data[q.tail] = value
	q.tail = q.next(q.tail)
	q.len++
	return true
}

// Pull removes and returns an element from the head of the queue.
// Returns the zero value and false if the queue is empty.
func (q *Queue[T]) Pull() (value T, ok bool) {
	if q.IsEmpty() {
		return
	}

	value = q.data[q.head]
	var zero T
	q.data[q.head] = zero
	q.head = q.next(q.head)
	q.len--
	return value, true
}

// all returns an iterator over all elements in the queue in FIFO order.
// The iterator yields (index, value) pairs where index is 0-based from the head.
func (q *Queue[T]) all() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		if q.IsEmpty() {
			return
		}

		current := q.head
		for idx := 0; idx < q.len; idx++ {
			if !yield(idx, q.data[current]) {
				return
			}
			current = q.next(current)
		}
	}
}

// DeleteFunc removes elements from the queue based on a predicate function.
// Returns an iterator over the removed elements.
// Elements for which shouldRemove returns true are removed from the queue.
func (q *Queue[T]) DeleteFunc(shouldRemove func(T) bool) (removed iter.Seq[T]) {
	return func(yield func(T) bool) {
		if q.IsEmpty() {
			return
		}

		newTail := q.head
		current := q.head
		stopYielding := false
		newLen := 0

		for i := 0; i < q.len; i++ {
			value := q.data[current]
			var zero T

			if !shouldRemove(value) {
				// Keep this element - move it to newTail if needed
				if current != newTail {
					q.data[newTail] = value
					q.data[current] = zero
				}
				newTail = q.next(newTail)
				current = q.next(current)
				newLen++
				continue
			}

			// Remove this element
			q.data[current] = zero
			current = q.next(current)

			// Try to yield the removed value if we haven't stopped
			stopYielding = stopYielding || !yield(value)
		}

		q.tail = newTail
		q.len = newLen
	}
}

// Clear removes all elements from the queue.
// Returns an iterator over the removed elements.
func (q *Queue[T]) Clear() iter.Seq[T] {
	return func(yield func(T) bool) {
		if q.IsEmpty() {
			return
		}

		current := q.head
		stopYielding := false

		for i := 0; i < q.len; i++ {
			value := q.data[current]
			var zero T
			q.data[current] = zero
			current = q.next(current)

			stopYielding = stopYielding || !yield(value)
		}

		q.head = 0
		q.tail = 0
		q.len = 0
	}
}

// next returns the next index in the circular queue.
func (q *Queue[T]) next(index int) int {
	index++
	if index >= len(q.data) {
		return 0
	}
	return index
}
