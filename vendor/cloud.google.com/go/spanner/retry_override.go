/*
Copyright 2026 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package spanner

import (
	"time"

	"github.com/googleapis/gax-go/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type suppressRetryCodesOption struct {
	code1 codes.Code
	code2 codes.Code
	len   uint8
	extra map[codes.Code]struct{}
}

func newSuppressRetryCodesOption(suppressedCodes ...codes.Code) suppressRetryCodesOption {
	opt := suppressRetryCodesOption{}
	for _, code := range suppressedCodes {
		switch opt.len {
		case 0:
			opt.code1 = code
			opt.len = 1
		case 1:
			if code != opt.code1 {
				opt.code2 = code
				opt.len = 2
			}
		default:
			if code == opt.code1 || code == opt.code2 {
				continue
			}
			if opt.extra == nil {
				opt.extra = map[codes.Code]struct{}{
					opt.code1: {},
					opt.code2: {},
				}
			}
			opt.extra[code] = struct{}{}
		}
	}
	return opt
}

type resourceExhaustedMarkerOption struct {
	mark                         func(error)
	allowRetryWithoutServerDelay bool
}

func appendResourceExhaustedMarkerOptions(base []gax.CallOption, mark func(error), allowRetryWithoutServerDelay bool) []gax.CallOption {
	if mark == nil && !allowRetryWithoutServerDelay {
		return base
	}
	opts := append([]gax.CallOption{}, base...)
	opts = append(opts, resourceExhaustedMarkerOption{
		mark:                         mark,
		allowRetryWithoutServerDelay: allowRetryWithoutServerDelay,
	})
	return opts
}

func (opt resourceExhaustedMarkerOption) Resolve(cs *gax.CallSettings) {
	if cs.Retry == nil {
		return
	}

	originalRetryFactory := cs.Retry
	cs.Retry = func() gax.Retryer {
		originalRetryer := originalRetryFactory()
		if originalRetryer == nil {
			return nil
		}
		if opt.allowRetryWithoutServerDelay {
			if originalSpannerRetryer, ok := originalRetryer.(*spannerRetryer); ok {
				originalRetryer = &spannerRetryer{
					Retryer:                                originalSpannerRetryer.Retryer,
					allowResourceExhaustedWithoutRetryInfo: true,
				}
			}
		}
		if opt.mark == nil {
			return originalRetryer
		}

		return wrapRetryFn(func(err error) (time.Duration, bool) {
			if shouldCooldownEndpointOnRetry(status.Code(err)) {
				opt.mark(err)
			}
			return originalRetryer.Retry(err)
		})
	}
}

func (opt suppressRetryCodesOption) Resolve(cs *gax.CallSettings) {
	if opt.len == 0 || cs.Retry == nil {
		return
	}

	originalRetryFactory := cs.Retry
	cs.Retry = func() gax.Retryer {
		originalRetryer := originalRetryFactory()
		if originalRetryer == nil {
			return nil
		}

		return wrapRetryFn(func(err error) (time.Duration, bool) {
			if opt.contains(status.Code(err)) {
				return 0, false
			}
			return originalRetryer.Retry(err)
		})
	}
}

func (opt suppressRetryCodesOption) contains(code codes.Code) bool {
	if opt.extra != nil {
		_, ok := opt.extra[code]
		return ok
	}
	switch opt.len {
	case 1:
		return code == opt.code1
	case 2:
		return code == opt.code1 || code == opt.code2
	default:
		return false
	}
}
