/*
Copyright 2024 The KEDA Authors

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

package util

import (
	"fmt"

	"go.uber.org/automaxprocs/maxprocs"
	"k8s.io/klog/v2"
)

// ConfigureMaxProcs sets up automaxprocs with proper logging configuration.
// It wraps the automaxprocs logger to handle structured logging with string keys
// to prevent panics when automaxprocs tries to pass numeric keys.
func ConfigureMaxProcs(logger klog.Logger) error {
	_, err := maxprocs.Set(maxprocs.Logger(func(format string, args ...interface{}) {
		logger.Info(fmt.Sprintf(format, args...))
	}))
	return err
}
