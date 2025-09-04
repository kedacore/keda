//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package file

import (
	"bytes"
	"context"
	"errors"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/streaming"
	"io"
	"sync"
)

// chunkWriter provides methods to upload chunks that represent a file to a server.
// This allows us to provide a local implementation that fakes the server for hermetic testing.
type chunkWriter interface {
	AppendData(context.Context, int64, io.ReadSeekCloser, *AppendDataOptions) (AppendDataResponse, error)
	FlushData(context.Context, int64, *FlushDataOptions) (FlushDataResponse, error)
}

// bufferManager provides an abstraction for the management of buffers.
// this is mostly for testing purposes, but does allow for different implementations without changing the algorithm.
type bufferManager[T ~[]byte] interface {
	// Acquire returns the channel that contains the pool of buffers.
	Acquire() <-chan T

	// Release releases the buffer back to the pool for reuse/cleanup.
	Release(T)

	// Grow grows the number of buffers, up to the predefined max.
	// It returns the total number of buffers or an error.
	// No error is returned if the number of buffers has reached max.
	// This is called only from the reading goroutine.
	Grow() (int, error)

	// Free cleans up all buffers.
	Free()
}

// copyFromReader copies a source io.Reader to file storage using concurrent uploads.
func copyFromReader[T ~[]byte](ctx context.Context, src io.Reader, dst chunkWriter, options UploadStreamOptions, getBufferManager func(maxBuffers int, bufferSize int64) bufferManager[T]) error {
	options.setDefaults()
	actualSize := int64(0)
	wg := sync.WaitGroup{}       // Used to know when all outgoing chunks have finished processing
	errCh := make(chan error, 1) // contains the first error encountered during processing
	var err error

	buffers := getBufferManager(int(options.Concurrency), options.ChunkSize)
	defer buffers.Free()

	// this controls the lifetime of the uploading goroutines.
	// if an error is encountered, cancel() is called which will terminate all uploads.
	// NOTE: the ordering is important here.  cancel MUST execute before
	// cleaning up the buffers so that any uploading goroutines exit first,
	// releasing their buffers back to the pool for cleanup.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// This goroutine grabs a buffer, reads from the stream into the buffer,
	// then creates a goroutine to upload/stage the chunk.
	for chunkNum := uint32(0); true; chunkNum++ {
		var buffer T
		select {
		case buffer = <-buffers.Acquire():
			// got a buffer
		default:
			// no buffer available; allocate a new buffer if possible
			if _, err := buffers.Grow(); err != nil {
				return err
			}

			// either grab the newly allocated buffer or wait for one to become available
			buffer = <-buffers.Acquire()
		}

		var n int
		n, err = io.ReadFull(src, buffer)

		if n > 0 {
			// some data was read, upload it
			wg.Add(1) // We're posting a buffer to be sent

			// NOTE: we must pass chunkNum as an arg to our goroutine else
			// it's captured by reference and can change underneath us!
			go func(chunkNum uint32) {
				// Upload the outgoing chunk, matching the number of bytes read
				offset := int64(chunkNum) * options.ChunkSize
				appendDataOpts := options.getAppendDataOptions()
				actualSize += int64(len(buffer[:n]))
				_, err := dst.AppendData(ctx, offset, streaming.NopCloser(bytes.NewReader(buffer[:n])), appendDataOpts)
				if err != nil {
					select {
					case errCh <- err:
						// error was set
					default:
						// some other error is already set
					}
					cancel()
				}
				buffers.Release(buffer) // The goroutine reading from the stream can reuse this buffer now

				// signal that the chunk has been staged.
				// we MUST do this after attempting to write to errCh
				// to avoid it racing with the reading goroutine.
				wg.Done()
			}(chunkNum)
		} else {
			// nothing was read so the buffer is empty, send it back for reuse/clean-up.
			buffers.Release(buffer)
		}

		if err != nil { // The reader is done, no more outgoing buffers
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				// these are expected errors, we don't surface those
				err = nil
			} else {
				// some other error happened, terminate any outstanding uploads
				cancel()
			}
			break
		}
	}

	wg.Wait() // Wait for all outgoing chunks to complete

	if err != nil {
		// there was an error reading from src, favor this error over any error during staging
		return err
	}

	select {
	case err = <-errCh:
		// there was an error during staging
		return err
	default:
		// no error was encountered
	}

	// All chunks uploaded, return nil error
	flushOpts := options.getFlushDataOptions()
	_, err = dst.FlushData(ctx, actualSize, flushOpts)
	return err
}

// mmbPool implements the bufferManager interface.
// it uses anonymous memory mapped files for buffers.
// don't use this type directly, use newMMBPool() instead.
type mmbPool struct {
	buffers chan mmb
	count   int
	max     int
	size    int64
}

func newMMBPool(maxBuffers int, bufferSize int64) bufferManager[mmb] {
	return &mmbPool{
		buffers: make(chan mmb, maxBuffers),
		max:     maxBuffers,
		size:    bufferSize,
	}
}

func (pool *mmbPool) Acquire() <-chan mmb {
	return pool.buffers
}

func (pool *mmbPool) Grow() (int, error) {
	if pool.count < pool.max {
		buffer, err := newMMB(pool.size)
		if err != nil {
			return 0, err
		}
		pool.buffers <- buffer
		pool.count++
	}
	return pool.count, nil
}

func (pool *mmbPool) Release(buffer mmb) {
	pool.buffers <- buffer
}

func (pool *mmbPool) Free() {
	for i := 0; i < pool.count; i++ {
		buffer := <-pool.buffers
		buffer.delete()
	}
	pool.count = 0
}
