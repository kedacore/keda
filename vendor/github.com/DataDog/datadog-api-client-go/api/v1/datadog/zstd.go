// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

//go:build cgo

package datadog

import (
	"bytes"

	"github.com/DataDog/zstd"
)

func compressZstd(body []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	compressor := zstd.NewWriter(&buf)
	if _, err := compressor.Write(body); err != nil {
		return nil, err
	}
	if err := compressor.Close(); err != nil {
		return nil, err
	}
	return &buf, nil
}
