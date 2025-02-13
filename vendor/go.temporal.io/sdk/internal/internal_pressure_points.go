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

import (
	"errors"
	"math/rand"
	"strconv"
	"time"

	"go.temporal.io/api/workflowservice/v1"

	"go.temporal.io/sdk/log"
)

// ** This is for internal stress testing framework **

// PressurePoints
const (
	pressurePointTypeWorkflowTaskStartTimeout    = "workflow-task-start-timeout"
	pressurePointTypeWorkflowTaskCompleted       = "workflow-task-complete"
	pressurePointTypeActivityTaskScheduleTimeout = "activity-task-schedule-timeout"
	pressurePointTypeActivityTaskStartTimeout    = "activity-task-start-timeout"
	pressurePointConfigProbability               = "probability"
	pressurePointConfigSleep                     = "sleep"
	workerOptionsConfig                          = "worker-options"
	workerOptionsConfigConcurrentPollRoutineSize = "ConcurrentPollRoutineSize"
)

type (
	pressurePointMgr interface {
		Execute(pressurePointName string) error
	}

	pressurePointMgrImpl struct {
		config map[string]map[string]string
		logger log.Logger
	}
)

// newWorkflowWorkerWithPressurePoints returns an instance of a workflow worker.
func newWorkflowWorkerWithPressurePoints(service workflowservice.WorkflowServiceClient, params workerExecutionParameters, pressurePoints map[string]map[string]string, registry *registry) (worker *workflowWorker) {
	return newWorkflowWorker(service, params, &pressurePointMgrImpl{config: pressurePoints, logger: params.Logger}, registry)
}

func (p *pressurePointMgrImpl) Execute(pressurePointName string) error {
	if config, ok := p.config[pressurePointName]; ok {
		// If probability is configured.
		if value, ok2 := config[pressurePointConfigProbability]; ok2 {
			if probability, err := strconv.Atoi(value); err == nil {
				if rand.Int31n(100) < int32(probability) {
					// Drop the task.
					p.logger.Debug("pressurePointMgrImpl.Execute drop task.",
						"PressurePointName", pressurePointName,
						"probability", probability)
					return errors.New("pressurepoint configured")
				}
			}
		} else if value, ok3 := config[pressurePointConfigSleep]; ok3 {
			if timeoutSeconds, err := strconv.Atoi(value); err == nil {
				if timeoutSeconds > 0 {
					p.logger.Debug("pressurePointMgrImpl.Execute sleep.",
						"PressurePointName", pressurePointName,
						"DurationSeconds", timeoutSeconds)
					d := time.Duration(timeoutSeconds) * time.Second
					time.Sleep(d)
					return nil
				}
			}
		}
	}
	return nil
}
