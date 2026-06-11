//go:build go1.23

package proto

import "iter"

// RowRange returns a [iter.Seq2] iterator over i-th row.
func (c ColMap[K, V]) RowRange(i int) iter.Seq2[K, V] {
	var start int
	end := int(c.Offsets[i])
	if i > 0 {
		start = int(c.Offsets[i-1])
	}

	return func(yield func(K, V) bool) {
		for idx := start; idx < end; idx++ {
			if !yield(
				c.Keys.Row(idx),
				c.Values.Row(idx),
			) {
				return
			}
		}
	}
}
