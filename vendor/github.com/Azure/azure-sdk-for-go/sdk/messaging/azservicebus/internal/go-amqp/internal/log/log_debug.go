// Copyright (C) 2017 Kale Blankenship
// Portions Copyright (c) Microsoft Corporation

//go:build debug
// +build debug

package log

import "log"
import "os"
import "strconv"

var (
	debugLevel = 1
	logger     = log.New(os.Stderr, "", log.Lmicroseconds)
)

func init() {
	level, err := strconv.Atoi(os.Getenv("DEBUG_LEVEL"))
	if err != nil {
		return
	}

	debugLevel = level
}

func Debug(level int, format string, v ...interface{}) {
	if level <= debugLevel {
		logger.Printf(format, v...)
	}
}
