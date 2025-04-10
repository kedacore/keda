// The MIT License
//
// Copyright (c) 2024 Temporal Technologies Inc.  All rights reserved.
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
	"github.com/nexus-rpc/sdk-go/nexus"
	"go.temporal.io/api/workflowservice/v1"
)

type nexusWorkerOptions struct {
	executionParameters workerExecutionParameters
	client              Client
	workflowService     workflowservice.WorkflowServiceClient
	handler             nexus.Handler
	registry            *registry
}

type nexusWorker struct {
	executionParameters workerExecutionParameters
	workflowService     workflowservice.WorkflowServiceClient
	worker              *baseWorker
	stopC               chan struct{}
}

func newNexusWorker(opts nexusWorkerOptions) (*nexusWorker, error) {
	workerStopChannel := make(chan struct{})
	params := opts.executionParameters
	params.WorkerStopChannel = getReadOnlyChannel(workerStopChannel)
	ensureRequiredParams(&params)
	poller := newNexusTaskPoller(
		newNexusTaskHandler(
			opts.handler,
			opts.executionParameters.Identity,
			opts.executionParameters.Namespace,
			opts.executionParameters.TaskQueue,
			opts.client,
			opts.executionParameters.DataConverter,
			opts.executionParameters.FailureConverter,
			opts.executionParameters.Logger,
			opts.executionParameters.MetricsHandler,
			opts.registry,
		),
		opts.workflowService,
		params,
	)

	baseWorker := newBaseWorker(baseWorkerOptions{
		pollerCount:      params.MaxConcurrentNexusTaskQueuePollers,
		pollerRate:       defaultPollerRate,
		slotSupplier:     params.Tuner.GetNexusSlotSupplier(),
		maxTaskPerSecond: defaultWorkerTaskExecutionRate,
		taskWorker:       poller,
		workerType:       "NexusWorker",
		identity:         params.Identity,
		buildId:          params.getBuildID(),
		logger:           params.Logger,
		stopTimeout:      params.WorkerStopTimeout,
		fatalErrCb:       params.WorkerFatalErrorCallback,
		metricsHandler:   params.MetricsHandler,
		slotReservationData: slotReservationData{
			taskQueue: params.TaskQueue,
		},
	},
	)

	return &nexusWorker{
		executionParameters: opts.executionParameters,
		workflowService:     opts.workflowService,
		worker:              baseWorker,
		stopC:               workerStopChannel,
	}, nil
}

// Start the worker.
func (w *nexusWorker) Start() error {
	err := verifyNamespaceExist(w.workflowService, w.executionParameters.MetricsHandler, w.executionParameters.Namespace, w.worker.logger)
	if err != nil {
		return err
	}
	w.worker.Start()
	return nil
}

// Stop the worker.
func (w *nexusWorker) Stop() {
	close(w.stopC)
	w.worker.Stop()
}
