// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driver

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"sync"

	"github.com/golang/snappy"
	"github.com/klauspost/compress/zstd"
	"go.mongodb.org/mongo-driver/x/mongo/driver/wiremessage"
)

// CompressionOpts holds settings for how to compress a payload
type CompressionOpts struct {
	Compressor       wiremessage.CompressorID
	ZlibLevel        int
	ZstdLevel        int
	UncompressedSize int32
}

var zstdEncoders sync.Map // map[zstd.EncoderLevel]*zstd.Encoder

func getZstdEncoder(level zstd.EncoderLevel) (*zstd.Encoder, error) {
	if v, ok := zstdEncoders.Load(level); ok {
		return v.(*zstd.Encoder), nil
	}
	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(level))
	if err != nil {
		return nil, err
	}
	zstdEncoders.Store(level, encoder)
	return encoder, nil
}

var zlibEncoders sync.Map // map[int /*level*/]*zlibEncoder

func getZlibEncoder(level int) (*zlibEncoder, error) {
	if v, ok := zlibEncoders.Load(level); ok {
		return v.(*zlibEncoder), nil
	}
	writer, err := zlib.NewWriterLevel(nil, level)
	if err != nil {
		return nil, err
	}
	encoder := &zlibEncoder{writer: writer, buf: new(bytes.Buffer)}
	zlibEncoders.Store(level, encoder)

	return encoder, nil
}

type zlibEncoder struct {
	mu     sync.Mutex
	writer *zlib.Writer
	buf    *bytes.Buffer
}

func (e *zlibEncoder) Encode(dst, src []byte) ([]byte, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.buf.Reset()
	e.writer.Reset(e.buf)

	_, err := e.writer.Write(src)
	if err != nil {
		return nil, err
	}
	err = e.writer.Close()
	if err != nil {
		return nil, err
	}
	dst = append(dst[:0], e.buf.Bytes()...)
	return dst, nil
}

// CompressPayload takes a byte slice and compresses it according to the options passed
func CompressPayload(in []byte, opts CompressionOpts) ([]byte, error) {
	switch opts.Compressor {
	case wiremessage.CompressorNoOp:
		return in, nil
	case wiremessage.CompressorSnappy:
		return snappy.Encode(nil, in), nil
	case wiremessage.CompressorZLib:
		encoder, err := getZlibEncoder(opts.ZlibLevel)
		if err != nil {
			return nil, err
		}
		return encoder.Encode(nil, in)
	case wiremessage.CompressorZstd:
		encoder, err := getZstdEncoder(zstd.EncoderLevelFromZstd(opts.ZstdLevel))
		if err != nil {
			return nil, err
		}
		return encoder.EncodeAll(in, nil), nil
	default:
		return nil, fmt.Errorf("unknown compressor ID %v", opts.Compressor)
	}
}

// DecompressPayload takes a byte slice that has been compressed and undoes it according to the options passed
func DecompressPayload(in []byte, opts CompressionOpts) (uncompressed []byte, err error) {
	switch opts.Compressor {
	case wiremessage.CompressorNoOp:
		return in, nil
	case wiremessage.CompressorSnappy:
		uncompressed = make([]byte, opts.UncompressedSize)
		return snappy.Decode(uncompressed, in)
	case wiremessage.CompressorZLib:
		r, err := zlib.NewReader(bytes.NewReader(in))
		if err != nil {
			return nil, err
		}
		defer func() {
			err = r.Close()
		}()
		uncompressed = make([]byte, opts.UncompressedSize)
		_, err = io.ReadFull(r, uncompressed)
		if err != nil {
			return nil, err
		}
		return uncompressed, nil
	case wiremessage.CompressorZstd:
		r, err := zstd.NewReader(bytes.NewBuffer(in))
		if err != nil {
			return nil, err
		}
		defer r.Close()
		uncompressed = make([]byte, opts.UncompressedSize)
		_, err = io.ReadFull(r, uncompressed)
		if err != nil {
			return nil, err
		}
		return uncompressed, nil
	default:
		return nil, fmt.Errorf("unknown compressor ID %v", opts.Compressor)
	}
}
