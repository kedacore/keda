/*
 * Copyright (c) "Neo4j"
 * Neo4j Sweden AB [https://neo4j.com]
 *
 * This file is part of Neo4j.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package log

import (
	"fmt"
	"os"
	"time"
)

// Console is a simple logger that logs to stdout/console.
// Turn the different log levels on/off as wished, all are off by default.
type Console struct {
	Errors bool
	Infos  bool
	Warns  bool
	Debugs bool
}

const timeFormat = "2006-01-02 15:04:05.000"

func (l *Console) Error(name, id string, err error) {
	if !l.Errors {
		return
	}
	now := time.Now()
	fmt.Fprintf(os.Stderr, "%s  ERROR  [%s %s] %s\n", now.Format(timeFormat), name, id, err.Error())
}

func (l *Console) Infof(name, id string, msg string, args ...any) {
	if !l.Infos {
		return
	}
	now := time.Now()
	fmt.Fprintf(os.Stdout, "%s   INFO  [%s %s] %s\n", now.Format(timeFormat), name, id, fmt.Sprintf(msg, args...))
}

func (l *Console) Warnf(name, id string, msg string, args ...any) {
	if !l.Warns {
		return
	}
	now := time.Now()
	fmt.Fprintf(os.Stdout, "%s   WARN  [%s %s] %s\n", now.Format(timeFormat), name, id, fmt.Sprintf(msg, args...))
}

func (l *Console) Debugf(name, id string, msg string, args ...any) {
	if !l.Debugs {
		return
	}
	now := time.Now()
	fmt.Fprintf(os.Stdout, "%s  DEBUG  [%s %s] %s\n", now.Format(timeFormat), name, id, fmt.Sprintf(msg, args...))
}
