//go:build go1.23

package proto

import "iter"

// RowRange returns a [iter.Seq] iterator over i-th row.
func (c ColArr[T]) RowRange(i int) iter.Seq[T] {
	var start int
	end := int(c.Offsets[i])
	if i > 0 {
		start = int(c.Offsets[i-1])
	}

	return func(yield func(T) bool) {
		for idx := start; idx < end; idx++ {
			if !yield(c.Data.Row(idx)) {
				return
			}
		}
	}
}
