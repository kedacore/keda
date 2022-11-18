// Copyright (C) 2017 Kale Blankenship
// Portions Copyright (c) Microsoft Corporation

//go:build !debug
// +build !debug

package log

// dummy functions used when debugging is not enabled

func Debug(_ int, _ string, _ ...interface{}) {}
