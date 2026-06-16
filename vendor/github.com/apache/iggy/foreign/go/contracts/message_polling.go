// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package iggcon

type PollingStrategy struct {
	Kind  MessagePolling
	Value uint64
}

type MessagePolling byte

const (
	POLLING_OFFSET    MessagePolling = 1
	POLLING_TIMESTAMP MessagePolling = 2
	POLLING_FIRST     MessagePolling = 3
	POLLING_LAST      MessagePolling = 4
	POLLING_NEXT      MessagePolling = 5
)

func NewPollingStrategy(kind MessagePolling, value uint64) PollingStrategy {
	return PollingStrategy{
		Kind:  kind,
		Value: value,
	}
}

func OffsetPollingStrategy(value uint64) PollingStrategy {
	return NewPollingStrategy(POLLING_OFFSET, value)
}

func TimestampPollingStrategy(value uint64) PollingStrategy {
	return NewPollingStrategy(POLLING_TIMESTAMP, value)
}

func FirstPollingStrategy() PollingStrategy {
	return NewPollingStrategy(POLLING_FIRST, 0)
}

func LastPollingStrategy() PollingStrategy {
	return NewPollingStrategy(POLLING_LAST, 0)
}

func NextPollingStrategy() PollingStrategy {
	return NewPollingStrategy(POLLING_NEXT, 0)
}
