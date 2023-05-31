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

import (
	"fmt"

	commonpb "go.temporal.io/api/common/v1"
)

type (
	// CompositeDataConverter applies PayloadConverters in specified order.
	CompositeDataConverter struct {
		payloadConverters map[string]PayloadConverter
		orderedEncodings  []string
	}
)

// NewCompositeDataConverter creates new instance of CompositeDataConverter from ordered list of PayloadConverters.
// Order is important here because during serialization DataConverter will try PayloadsConverters in
// that order until PayloadConverter returns non nil payload.
// Last PayloadConverter should always serialize the value (JSONPayloadConverter is good candidate for it).
func NewCompositeDataConverter(payloadConverters ...PayloadConverter) DataConverter {
	dc := &CompositeDataConverter{
		payloadConverters: make(map[string]PayloadConverter, len(payloadConverters)),
		orderedEncodings:  make([]string, len(payloadConverters)),
	}

	for i, payloadConverter := range payloadConverters {
		dc.payloadConverters[payloadConverter.Encoding()] = payloadConverter
		dc.orderedEncodings[i] = payloadConverter.Encoding()
	}

	return dc
}

// ToPayloads converts a list of values.
func (dc *CompositeDataConverter) ToPayloads(values ...interface{}) (*commonpb.Payloads, error) {
	if len(values) == 0 {
		return nil, nil
	}

	result := &commonpb.Payloads{}
	for i, value := range values {
		payload, err := dc.ToPayload(value)
		if err != nil {
			return nil, fmt.Errorf("values[%d]: %w", i, err)
		}

		result.Payloads = append(result.Payloads, payload)
	}

	return result, nil
}

// FromPayloads converts to a list of values of different types.
func (dc *CompositeDataConverter) FromPayloads(payloads *commonpb.Payloads, valuePtrs ...interface{}) error {
	if payloads == nil {
		return nil
	}

	for i, payload := range payloads.GetPayloads() {
		if i >= len(valuePtrs) {
			break
		}

		err := dc.FromPayload(payload, valuePtrs[i])
		if err != nil {
			return fmt.Errorf("payload item %d: %w", i, err)
		}
	}

	return nil
}

// ToPayload converts single value to payload.
func (dc *CompositeDataConverter) ToPayload(value interface{}) (*commonpb.Payload, error) {
	for _, enc := range dc.orderedEncodings {
		payloadConverter := dc.payloadConverters[enc]
		payload, err := payloadConverter.ToPayload(value)
		if err != nil {
			return nil, err
		}
		if payload != nil {
			return payload, nil
		}
	}

	return nil, fmt.Errorf("value: %v of type: %T: %w", value, value, ErrUnableToFindConverter)
}

// FromPayload converts single value from payload.
func (dc *CompositeDataConverter) FromPayload(payload *commonpb.Payload, valuePtr interface{}) error {
	if payload == nil {
		return nil
	}

	enc, err := encoding(payload)
	if err != nil {
		return err
	}

	payloadConverter, ok := dc.payloadConverters[enc]
	if !ok {
		return fmt.Errorf("encoding %s: %w", enc, ErrEncodingIsNotSupported)
	}

	return payloadConverter.FromPayload(payload, valuePtr)
}

// ToString converts payload object into human readable string.
func (dc *CompositeDataConverter) ToString(payload *commonpb.Payload) string {
	if payload == nil {
		return ""
	}

	enc, err := encoding(payload)
	if err != nil {
		return err.Error()
	}

	payloadConverter, ok := dc.payloadConverters[enc]
	if !ok {
		return fmt.Errorf("encoding %s: %w", enc, ErrEncodingIsNotSupported).Error()
	}

	return payloadConverter.ToString(payload)
}

// ToStrings converts payloads object into human readable strings.
func (dc *CompositeDataConverter) ToStrings(payloads *commonpb.Payloads) []string {
	if payloads == nil {
		return nil
	}

	var result []string
	for _, payload := range payloads.GetPayloads() {
		result = append(result, dc.ToString(payload))
	}

	return result
}

func encoding(payload *commonpb.Payload) (string, error) {
	metadata := payload.GetMetadata()
	if metadata == nil {
		return "", ErrMetadataIsNotSet
	}

	if e, ok := metadata[MetadataEncoding]; ok {
		return string(e), nil
	}

	return "", ErrEncodingIsNotSet
}
