// The MIT License
//
// Copyright (c) 2022 Temporal Technologies Inc.  All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: temporal/api/enums/v1/update.proto

package enums

import (
	fmt "fmt"
	math "math"
	strconv "strconv"

	proto "github.com/gogo/protobuf/proto"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// UpdateWorkflowExecutionLifecycleStage is specified by clients invoking
// workflow execution updates and used to indicate to the server how long the
// client wishes to wait for a return value from the RPC. If any value other
// than UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_COMPLETED is sent by the
// client then the RPC will complete before the update is finished and will
// return a handle to the running update so that it can later be polled for
// completion.
type UpdateWorkflowExecutionLifecycleStage int32

const (
	// An unspecified vale for this enum.
	UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_UNSPECIFIED UpdateWorkflowExecutionLifecycleStage = 0
	// The gRPC call will not return until the update request has been admitted
	// by the server - it may be the case that due to a considerations like load
	// or resource limits that an update is made to wait before the server will
	// indicate that it has been received and will be processed. This value
	// does not wait for any sort of acknowledgement from a worker.
	UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_ADMITTED UpdateWorkflowExecutionLifecycleStage = 1
	// The gRPC call will not return until the update has passed validation on
	// a worker.
	UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_ACCEPTED UpdateWorkflowExecutionLifecycleStage = 2
	// The gRPC call will not return until the update has executed to completion
	// on a worker and has either been rejected or returned a value or an error.
	UPDATE_WORKFLOW_EXECUTION_LIFECYCLE_STAGE_COMPLETED UpdateWorkflowExecutionLifecycleStage = 3
)

var UpdateWorkflowExecutionLifecycleStage_name = map[int32]string{
	0: "Unspecified",
	1: "Admitted",
	2: "Accepted",
	3: "Completed",
}

var UpdateWorkflowExecutionLifecycleStage_value = map[string]int32{
	"Unspecified": 0,
	"Admitted":    1,
	"Accepted":    2,
	"Completed":   3,
}

func (UpdateWorkflowExecutionLifecycleStage) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_62be6b87506ae277, []int{0}
}

func init() {
	proto.RegisterEnum("temporal.api.enums.v1.UpdateWorkflowExecutionLifecycleStage", UpdateWorkflowExecutionLifecycleStage_name, UpdateWorkflowExecutionLifecycleStage_value)
}

func init() {
	proto.RegisterFile("temporal/api/enums/v1/update.proto", fileDescriptor_62be6b87506ae277)
}

var fileDescriptor_62be6b87506ae277 = []byte{
	// 343 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x52, 0x2a, 0x49, 0xcd, 0x2d,
	0xc8, 0x2f, 0x4a, 0xcc, 0xd1, 0x4f, 0x2c, 0xc8, 0xd4, 0x4f, 0xcd, 0x2b, 0xcd, 0x2d, 0xd6, 0x2f,
	0x33, 0xd4, 0x2f, 0x2d, 0x48, 0x49, 0x2c, 0x49, 0xd5, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x12,
	0x85, 0xa9, 0xd1, 0x4b, 0x2c, 0xc8, 0xd4, 0x03, 0xab, 0xd1, 0x2b, 0x33, 0xd4, 0xea, 0x66, 0xe2,
	0x52, 0x0d, 0x05, 0xab, 0x0b, 0xcf, 0x2f, 0xca, 0x4e, 0xcb, 0xc9, 0x2f, 0x77, 0xad, 0x48, 0x4d,
	0x2e, 0x2d, 0xc9, 0xcc, 0xcf, 0xf3, 0xc9, 0x4c, 0x4b, 0x4d, 0xae, 0x4c, 0xce, 0x49, 0x0d, 0x2e,
	0x49, 0x4c, 0x4f, 0x15, 0xb2, 0xe4, 0x32, 0x0d, 0x0d, 0x70, 0x71, 0x0c, 0x71, 0x8d, 0x0f, 0xf7,
	0x0f, 0xf2, 0x76, 0xf3, 0xf1, 0x0f, 0x8f, 0x77, 0x8d, 0x70, 0x75, 0x0e, 0x0d, 0xf1, 0xf4, 0xf7,
	0x8b, 0xf7, 0xf1, 0x74, 0x73, 0x75, 0x8e, 0x74, 0xf6, 0x71, 0x8d, 0x0f, 0x0e, 0x71, 0x74, 0x77,
	0x8d, 0x0f, 0xf5, 0x0b, 0x0e, 0x70, 0x75, 0xf6, 0x74, 0xf3, 0x74, 0x75, 0x11, 0x60, 0x10, 0x32,
	0xe3, 0x32, 0x22, 0x5e, 0xab, 0xa3, 0x8b, 0xaf, 0x67, 0x48, 0x88, 0xab, 0x8b, 0x00, 0x23, 0x89,
	0xfa, 0x9c, 0x9d, 0x5d, 0x03, 0x40, 0xfa, 0x98, 0x84, 0xcc, 0xb9, 0x8c, 0x89, 0xd7, 0xe7, 0xec,
	0xef, 0x1b, 0xe0, 0xe3, 0x0a, 0xd2, 0xc8, 0xec, 0xb4, 0x9d, 0xf1, 0xc2, 0x43, 0x39, 0x86, 0x1b,
	0x0f, 0xe5, 0x18, 0x3e, 0x3c, 0x94, 0x63, 0x6c, 0x78, 0x24, 0xc7, 0xb8, 0xe2, 0x91, 0x1c, 0xe3,
	0x89, 0x47, 0x72, 0x8c, 0x17, 0x1e, 0xc9, 0x31, 0x3e, 0x78, 0x24, 0xc7, 0xf8, 0xe2, 0x91, 0x1c,
	0xc3, 0x87, 0x47, 0x72, 0x8c, 0x13, 0x1e, 0xcb, 0x31, 0x5c, 0x78, 0x2c, 0xc7, 0x70, 0xe3, 0xb1,
	0x1c, 0x03, 0x97, 0x44, 0x66, 0xbe, 0x1e, 0xd6, 0xe0, 0x75, 0xe2, 0x86, 0x84, 0x6d, 0x00, 0x28,
	0x0a, 0x02, 0x18, 0xa3, 0x14, 0xd3, 0x91, 0x14, 0x66, 0xe6, 0xa3, 0x44, 0x97, 0x35, 0x98, 0xb1,
	0x8a, 0x49, 0x3c, 0x04, 0xaa, 0x20, 0x33, 0x5f, 0xcf, 0xb1, 0x20, 0x53, 0xcf, 0x15, 0x6c, 0x56,
	0x98, 0xe1, 0x2b, 0x26, 0x29, 0x84, 0x8c, 0x95, 0x95, 0x63, 0x41, 0xa6, 0x95, 0x15, 0x58, 0xce,
	0xca, 0x2a, 0xcc, 0x30, 0x89, 0x0d, 0x1c, 0xcb, 0xc6, 0x80, 0x00, 0x00, 0x00, 0xff, 0xff, 0xe6,
	0x41, 0x41, 0x5f, 0x0b, 0x02, 0x00, 0x00,
}

func (x UpdateWorkflowExecutionLifecycleStage) String() string {
	s, ok := UpdateWorkflowExecutionLifecycleStage_name[int32(x)]
	if ok {
		return s
	}
	return strconv.Itoa(int(x))
}