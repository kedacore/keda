// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
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

package internal

// Below are the metadata which will be embedded as part of headers in every RPC call made by this client to Temporal server.
// Update to the metadata below is typically done by the Temporal team as part of a major feature or behavior change.

const (
	// SDKVersion is a semver (https://semver.org/) that represents the version of this Temporal GoSDK.
	// Server validates if SDKVersion fits its supported range and rejects request if it doesn't.
	//
	// Exposed as: [go.temporal.io/sdk/temporal.SDKVersion]
	SDKVersion = "1.33.1"

	// SDKName represents the name of the SDK.
	SDKName = clientNameHeaderValue

	// SupportedServerVersions is a semver rages (https://github.com/blang/semver#ranges) of server versions that
	// are supported by this Temporal SDK.
	// Server validates if its version fits into SupportedServerVersions range and rejects request if it doesn't.
	SupportedServerVersions = ">=1.0.0 <2.0.0"
)
