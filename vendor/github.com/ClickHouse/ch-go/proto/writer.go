package proto

import (
	"io"
	"net"
)

// Writer is a column writer.
//
// It helps to reduce memory footprint by writing column using vector I/O.
type Writer struct {
	conn io.Writer

	buf       *Buffer
	bufOffset int
	needCut   bool

	vec net.Buffers
}

// NewWriter creates new [Writer].
func NewWriter(conn io.Writer, buf *Buffer) *Writer {
	w := &Writer{
		conn: conn,
		buf:  buf,
		vec:  make(net.Buffers, 0, 16),
	}
	return w
}

// ChainWrite adds buffer to the vector to write later.
//
// Passed byte slice may be captured until [Writer.Flush] is called.
func (w *Writer) ChainWrite(data []byte) {
	w.cutBuffer()
	w.vec = append(w.vec, data)
}

// ChainBuffer creates a temporary buffer and adds it to the vector to write later.
//
// Data is not written immediately, call [Writer.Flush] to flush data.
//
// NB: do not retain buffer.
func (w *Writer) ChainBuffer(cb func(*Buffer)) {
	cb(w.buf)
}

func (w *Writer) cutBuffer() {
	newOffset := len(w.buf.Buf)
	data := w.buf.Buf[w.bufOffset:newOffset:newOffset]
	if len(data) == 0 {
		return
	}
	w.bufOffset = newOffset
	w.vec = append(w.vec, data)
}

func (w *Writer) reset() {
	w.bufOffset = 0
	w.needCut = false
	w.buf.Reset()
	// Do not hold references, to avoid memory leaks.
	clear(w.vec)
	w.vec = w.vec[:0]
}

// Flush flushes all data to writer.
func (w *Writer) Flush() (n int64, err error) {
	w.cutBuffer()
	n, err = w.vec.WriteTo(w.conn)
	w.reset()
	return n, err
}
