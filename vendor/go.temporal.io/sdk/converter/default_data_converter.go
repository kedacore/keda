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

package converter

var (
	defaultDataConverter = NewCompositeDataConverter(
		NewNilPayloadConverter(),
		NewByteSlicePayloadConverter(),

		// Order is important here. Both ProtoJsonPayload and ProtoPayload converters check for the same proto.Message
		// interface. The first match (ProtoJsonPayload in this case) will always be used for serialization.
		// Deserialization is controlled by metadata, therefore both converters can deserialize corresponding data format
		// (JSON or binary proto).
		NewProtoJSONPayloadConverter(),
		NewProtoPayloadConverter(),

		NewJSONPayloadConverter(),
	)
)

// GetDefaultDataConverter returns default data converter used by Temporal worker.
func GetDefaultDataConverter() DataConverter {
	return defaultDataConverter
}
