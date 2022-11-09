// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

//go:build !cgo

package datadog

import (
	"bytes"
	"errors"
)

func compressZstd(body []byte) (*bytes.Buffer, error) {
	return nil, errors.New("zstd not supported")
}
