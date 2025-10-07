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

	bwo := baseWorkerOptions{
		pollerRate:       defaultPollerRate,
		slotSupplier:     params.Tuner.GetNexusSlotSupplier(),
		maxTaskPerSecond: defaultWorkerTaskExecutionRate,
		taskPollers: []scalableTaskPoller{
			newScalableTaskPoller(
				poller,
				opts.executionParameters.Logger,
				params.NexusTaskPollerBehavior),
		},
		taskProcessor:  poller,
		workerType:     "NexusWorker",
		identity:       params.Identity,
		buildId:        params.getBuildID(),
		logger:         params.Logger,
		stopTimeout:    params.WorkerStopTimeout,
		fatalErrCb:     params.WorkerFatalErrorCallback,
		metricsHandler: params.MetricsHandler,
		slotReservationData: slotReservationData{
			taskQueue: params.TaskQueue,
		},
		isInternalWorker: params.isInternalWorker(),
	}

	baseWorker := newBaseWorker(bwo)

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
