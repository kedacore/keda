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

type ConsumerKind uint8

const (
	ConsumerKindSingle ConsumerKind = 1
	ConsumerKindGroup  ConsumerKind = 2
)

type Consumer struct {
	Kind ConsumerKind
	Id   Identifier
}

func DefaultConsumer() Consumer {
    defaultID, _ := NewIdentifier(uint32(0))
	return Consumer{
		Kind: ConsumerKindSingle,
		Id:   defaultID,
	}
}

// NewSingleConsumer create a new Consumer whose kind is ConsumerKindSingle from the Identifier
func NewSingleConsumer(id Identifier) Consumer {
	return Consumer{
		Kind: ConsumerKindSingle,
		Id:   id,
	}
}

// NewGroupConsumer create a new Consumer whose kind is ConsumerKindGroup from the Identifier
func NewGroupConsumer(id Identifier) Consumer {
	return Consumer{
		Kind: ConsumerKindGroup,
		Id:   id,
	}
}
