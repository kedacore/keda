// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azeventhubs

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	azlog "github.com/Azure/azure-sdk-for-go/sdk/internal/log"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/eh"
)

// processorOwnerLevel is the owner level we assign to every ProcessorPartitionClient
// created by this Processor.
var processorOwnerLevel = to.Ptr[int64](0)

// ProcessorStrategy specifies the load balancing strategy used by the Processor.
type ProcessorStrategy string

const (
	// ProcessorStrategyBalanced will attempt to claim a single partition during each update interval, until
	// each active owner has an equal share of partitions. It can take longer for Processors to acquire their
	// full share of partitions, but minimizes partition swapping.
	// This is the default strategy.
	ProcessorStrategyBalanced ProcessorStrategy = "balanced"

	// ProcessorStrategyGreedy will attempt to claim all partitions it can during each update interval, respecting
	// balance. This can lead to more partition swapping, as Processors steal partitions to get to their fair share,
	// but can speed up initial startup.
	ProcessorStrategyGreedy ProcessorStrategy = "greedy"
)

// ProcessorOptions are the options for the NewProcessor
// function.
type ProcessorOptions struct {
	// LoadBalancingStrategy dictates how concurrent Processor instances distribute
	// ownership of partitions between them.
	// The default strategy is ProcessorStrategyBalanced.
	LoadBalancingStrategy ProcessorStrategy

	// UpdateInterval controls how often attempt to claim partitions.
	// The default value is 10 seconds.
	UpdateInterval time.Duration

	// PartitionExpirationDuration is the amount of time before a partition is considered
	// unowned.
	// The default value is 60 seconds.
	PartitionExpirationDuration time.Duration

	// StartPositions are the default start positions (configurable per partition, or with an overall
	// default value) if a checkpoint is not found in the CheckpointStore.
	//
	// - If the Event Hubs namespace has geo-replication enabled, the default is Earliest.
	// - If the Event Hubs namespace does NOT have geo-replication enabled, the default position is Latest
	StartPositions StartPositions

	// Prefetch represents the size of the internal prefetch buffer for each ProcessorPartitionClient
	// created by this Processor. When set, this client will attempt to always maintain
	// an internal cache of events of this size, asynchronously, increasing the odds that
	// ReceiveEvents() will use a locally stored cache of events, rather than having to
	// wait for events to arrive from the network.
	//
	// Defaults to 300 events if Prefetch == 0.
	// Disabled if Prefetch < 0.
	Prefetch int32
}

// StartPositions are used if there is no checkpoint for a partition in
// the checkpoint store.
type StartPositions struct {
	// PerPartition controls the start position for a specific partition,
	// by partition ID. If a partition is not configured here it will default
	// to Default start position.
	PerPartition map[string]StartPosition

	// Default is used if the partition is not found in the PerPartition map.
	Default StartPosition
}

type state int32

const (
	stateNone    state = 0
	stateStopped state = 1
	stateRunning state = 2
)

// Processor uses a [ConsumerClient] and [CheckpointStore] to provide automatic
// load balancing between multiple Processor instances, even in separate
// processes or on separate machines.
//
// See [example_consuming_with_checkpoints_test.go] for an example, and the function documentation
// for [Run] for a more detailed description of how load balancing works.
//
// [example_consuming_with_checkpoints_test.go]: https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/messaging/azeventhubs/example_consuming_with_checkpoints_test.go
type Processor struct {
	stateMu sync.Mutex
	state   state

	ownershipUpdateInterval time.Duration
	defaultStartPositions   StartPositions
	checkpointStore         CheckpointStore
	prefetch                int32

	// consumerClient is actually a *azeventhubs.ConsumerClient
	// it's an interface here to make testing easier.
	consumerClient consumerClientForProcessor

	nextClients           chan *ProcessorPartitionClient
	nextClientsReady      chan struct{}
	consumerClientDetails consumerClientDetails

	lb *processorLoadBalancer

	// claimedOwnerships is set to whatever our current ownerships are. The underlying
	// value is a []Ownership.
	currentOwnerships *atomic.Value
}

type consumerClientForProcessor interface {
	GetEventHubProperties(ctx context.Context, options *GetEventHubPropertiesOptions) (EventHubProperties, error)
	NewPartitionClient(partitionID string, options *PartitionClientOptions) (*PartitionClient, error)
	getDetails() consumerClientDetails
}

// NewProcessor creates a Processor.
//
// More information can be found in the documentation for the [Processor]
// type or the [example_consuming_with_checkpoints_test.go] for an example.
//
// [example_consuming_with_checkpoints_test.go]: https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/messaging/azeventhubs/example_consuming_with_checkpoints_test.go
func NewProcessor(consumerClient *ConsumerClient, checkpointStore CheckpointStore, options *ProcessorOptions) (*Processor, error) {
	return newProcessorImpl(consumerClient, checkpointStore, options)
}

func newProcessorImpl(consumerClient consumerClientForProcessor, checkpointStore CheckpointStore, options *ProcessorOptions) (*Processor, error) {
	if options == nil {
		options = &ProcessorOptions{}
	}

	updateInterval := 10 * time.Second

	if options.UpdateInterval != 0 {
		updateInterval = options.UpdateInterval
	}

	partitionDurationExpiration := time.Minute

	if options.PartitionExpirationDuration != 0 {
		partitionDurationExpiration = options.PartitionExpirationDuration
	}

	startPosPerPartition := map[string]StartPosition{}

	if options.StartPositions.PerPartition != nil {
		for k, v := range options.StartPositions.PerPartition {
			startPosPerPartition[k] = v
		}
	}

	strategy := options.LoadBalancingStrategy

	switch strategy {
	case ProcessorStrategyBalanced:
	case ProcessorStrategyGreedy:
	case "":
		strategy = ProcessorStrategyBalanced
	default:
		return nil, fmt.Errorf("invalid load balancing strategy '%s'", strategy)
	}

	currentOwnerships := &atomic.Value{}
	currentOwnerships.Store([]Ownership{})

	return &Processor{
		ownershipUpdateInterval: updateInterval,
		consumerClient:          consumerClient,
		checkpointStore:         checkpointStore,

		defaultStartPositions: StartPositions{
			PerPartition: startPosPerPartition,
			Default:      options.StartPositions.Default,
		},
		prefetch:              options.Prefetch,
		consumerClientDetails: consumerClient.getDetails(),
		nextClientsReady:      make(chan struct{}),
		lb:                    newProcessorLoadBalancer(checkpointStore, consumerClient.getDetails(), strategy, partitionDurationExpiration),
		currentOwnerships:     currentOwnerships,

		// `nextClients` will be properly initialized when the user calls
		// Run() since it needs to query the # of partitions on the Event Hub.
		nextClients: make(chan *ProcessorPartitionClient),
	}, nil
}

// NextPartitionClient will get the next owned [ProcessorPartitionClient] if one is acquired
// or will block until a new one arrives or [Processor.Run] is cancelled. When the Processor
// stops running this function will return nil.
//
// NOTE: You MUST call [ProcessorPartitionClient.Close] on the returned client to avoid
// leaking resources.
//
// See [example_consuming_with_checkpoints_test.go] for an example of typical usage.
//
// [example_consuming_with_checkpoints_test.go]: https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/messaging/azeventhubs/example_consuming_with_checkpoints_test.go
func (p *Processor) NextPartitionClient(ctx context.Context) *ProcessorPartitionClient {
	select {
	case <-ctx.Done():
		return nil
	case <-p.nextClientsReady:
	}

	select {
	case nextClient := <-p.nextClients:
		return nextClient
	case <-ctx.Done():
		return nil
	}
}

func (p *Processor) checkState() error {
	switch p.state {
	case stateNone:
		// not running so we can start. And lock out any other users.
		p.state = stateRunning
		return nil
	case stateRunning:
		return errors.New("the Processor is currently running, concurrent calls to Run() are not allowed")
	case stateStopped:
		return errors.New("the Processor has been stopped. Create a new instance to start processing again")
	default:
		return fmt.Errorf("unhandled state value %v", p.state)
	}
}

// Run handles the load balancing loop, blocking until the passed in context is cancelled
// or it encounters an unrecoverable error. On cancellation, it will return a nil error.
//
// This function should run for the lifetime of your application, or for as long as you want
// to continue to claim and process partitions.
//
// Once a Processor has been stopped it cannot be restarted and a new instance must
// be created.
//
// As partitions are claimed new [ProcessorPartitionClient] instances will be returned from
// [Processor.NextPartitionClient]. This can happen at any time, based on new Processor instances
// coming online, as well as other Processors exiting.
//
// [ProcessorPartitionClient] are used like a [PartitionClient] but provide an [ProcessorPartitionClient.UpdateCheckpoint]
// function that will store a checkpoint into the [CheckpointStore]. If the client were to crash, or be restarted
// it will pick up from the last checkpoint.
//
// See [example_consuming_with_checkpoints_test.go] for an example of typical usage.
//
// [example_consuming_with_checkpoints_test.go]: https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/messaging/azeventhubs/example_consuming_with_checkpoints_test.go
func (p *Processor) Run(ctx context.Context) error {
	p.stateMu.Lock()
	err := p.checkState()
	p.stateMu.Unlock()

	if err != nil {
		return err
	}

	err = p.runImpl(ctx)

	// the context is the proper way to close down the Run() loop, so it's not
	// an error and doesn't need to be returned.
	if ctx.Err() != nil {
		return nil
	}

	return err
}

func (p *Processor) runImpl(ctx context.Context) error {
	consumers := &sync.Map{}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		p.close(ctx, consumers)
	}()

	// size the channel to the # of partitions. We can never exceed this size since
	// we'll never reclaim a partition that we already have ownership of.
	eventHubProperties, err := p.initNextClientsCh(ctx)

	if err != nil {
		return err
	}

	// do one dispatch immediately
	if err := p.dispatch(ctx, eventHubProperties, consumers); err != nil {
		return err
	}

	// note randSource is not thread-safe but it's not currently used in a way that requires
	// it to be.
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(calculateUpdateInterval(rnd, p.ownershipUpdateInterval)):
			if err := p.dispatch(ctx, eventHubProperties, consumers); err != nil {
				return err
			}
		}
	}
}

func calculateUpdateInterval(rnd *rand.Rand, updateInterval time.Duration) time.Duration {
	// Introduce some jitter:  [0.0, 1.0) / 2 = [0.0, 0.5) + 0.8 = [0.8, 1.3)
	// (copied from the retry code for calculating jitter)
	return time.Duration(updateInterval.Seconds() * (rnd.Float64()/2 + 0.8) * float64(time.Second))
}

func (p *Processor) initNextClientsCh(ctx context.Context) (EventHubProperties, error) {
	eventHubProperties, err := p.consumerClient.GetEventHubProperties(ctx, nil)

	if err != nil {
		return EventHubProperties{}, err
	}

	p.nextClients = make(chan *ProcessorPartitionClient, len(eventHubProperties.PartitionIDs))
	close(p.nextClientsReady)

	return eventHubProperties, nil
}

type getCheckpoints func() (map[string]Checkpoint, error)

// dispatch uses the checkpoint store to figure out which partitions should be processed by this
// instance and starts a PartitionClient, if there isn't one.
// NOTE: due to random number usage in the load balancer, this function is not thread safe.
func (p *Processor) dispatch(ctx context.Context, eventHubProperties EventHubProperties, consumers *sync.Map) error {
	ownerships, err := p.lb.LoadBalance(ctx, eventHubProperties.PartitionIDs)

	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}

	// store off the set of ownerships we claimed this round - when the processor
	// shuts down we'll clear them (if we still own them).
	tmpOwnerships := make([]Ownership, len(ownerships))
	copy(tmpOwnerships, ownerships)
	p.currentOwnerships.Store(tmpOwnerships)

	getCheckpoints := sync.OnceValues(func() (map[string]Checkpoint, error) {
		return p.getCheckpointsMap(ctx)
	})

	for _, ownership := range ownerships {
		wg.Add(1)

		go func(o Ownership) {
			defer wg.Done()

			err := p.addPartitionClient(ctx, o, getCheckpoints, p.openPartitionClientImpl, consumers)

			if err != nil {
				azlog.Writef(EventConsumer, "failed to create partition client for partition '%s': %s", o.PartitionID, err.Error())
			}
		}(ownership)
	}

	wg.Wait()

	return nil
}

// openPartitionClient creates a PartitionClient and initializes it, causing it to open AMQP links.
// Implemented by [Processor.openPartitionClientImpl]
type openPartitionClient func(ctx context.Context, partitionID string, startPosition StartPosition) (partitionClient *PartitionClient, err error)

// openPartitionClientImpl creates a PartitionClient and initializes it, causing it to open AMQP links.
func (p *Processor) openPartitionClientImpl(ctx context.Context, partitionID string, startPosition StartPosition) (partitionClient *PartitionClient, err error) {
	partitionClient, err = p.consumerClient.NewPartitionClient(partitionID, &PartitionClientOptions{
		StartPosition: startPosition,
		OwnerLevel:    processorOwnerLevel,
		Prefetch:      p.prefetch,
	})

	if err != nil {
		return nil, err
	}

	// make sure we create the link _now_ - if we're stealing we want to stake a claim _now_, rather than
	// later when the user actually calls ReceiveEvents(), since the acquisition of the link is lazy.
	if err := partitionClient.init(ctx); err != nil {
		_ = partitionClient.Close(ctx) // ignore close error here, we're just cleaning up after a failed init()
		return nil, err
	}

	return partitionClient, nil
}

// addPartitionClient creates a ProcessorPartitionClient
func (p *Processor) addPartitionClient(ctx context.Context, ownership Ownership, getCheckpoints getCheckpoints, openPartitionClient openPartitionClient, consumers *sync.Map) error {
	processorPartClient := &ProcessorPartitionClient{
		consumerClientDetails: p.consumerClientDetails,
		checkpointStore:       p.checkpointStore,
		innerClient:           nil,
		partitionID:           ownership.PartitionID,
		cleanupFn: func() {
			consumers.Delete(ownership.PartitionID)
		},
	}

	if _, alreadyExists := consumers.LoadOrStore(ownership.PartitionID, processorPartClient); alreadyExists {
		return nil
	}

	preferredStartPosition, err := p.getStartPosition(getCheckpoints, ownership)

	if err != nil {
		return err
	}

	partClient, err := openPartitionClient(ctx, ownership.PartitionID, preferredStartPosition)

	if eh.IsGeoReplicationOffsetError(err) {
		azlog.Writef(EventConsumer, "Event Hub is in geo-replication mode and we only have an integer offset, will fallback to starting at earliest+inclusive: %s", err.Error())

		partClient, err = openPartitionClient(ctx, ownership.PartitionID, StartPosition{
			Earliest:  to.Ptr(true),
			Inclusive: true,
		})
	}

	if err != nil {
		consumers.Delete(ownership.PartitionID)
		return err
	}

	processorPartClient.innerClient = partClient

	select {
	case p.nextClients <- processorPartClient:
		return nil
	default:
		_ = processorPartClient.Close(ctx)
		return fmt.Errorf("partitions channel full, consumer for partition %s could not be returned", ownership.PartitionID)
	}
}

// getStartPosition gets the start position, preferring a stored Checkpoint, then falling back to per-partition defaults
// and then the global default position.
func (p *Processor) getStartPosition(getCheckpoints getCheckpoints, ownership Ownership) (StartPosition, error) {
	checkpoints, err := getCheckpoints()

	if err != nil {
		return StartPosition{}, err
	}

	var checkpoint *Checkpoint

	if tmpCheckpoint, ok := checkpoints[ownership.PartitionID]; ok {
		checkpoint = &tmpCheckpoint
	}

	startPosition := p.defaultStartPositions.Default

	if checkpoint != nil {
		if checkpoint.Offset != nil {
			startPosition = StartPosition{
				Offset: checkpoint.Offset,
			}
		} else if checkpoint.SequenceNumber != nil {
			startPosition = StartPosition{
				SequenceNumber: checkpoint.SequenceNumber,
			}
		} else {
			return StartPosition{}, fmt.Errorf("invalid checkpoint for %s, no offset or sequence number", ownership.PartitionID)
		}
	} else if p.defaultStartPositions.PerPartition != nil {
		defaultStartPosition, exists := p.defaultStartPositions.PerPartition[ownership.PartitionID]

		if exists {
			startPosition = defaultStartPosition
		}
	}

	return startPosition, nil
}

func (p *Processor) getCheckpointsMap(ctx context.Context) (map[string]Checkpoint, error) {
	details := p.consumerClient.getDetails()
	checkpoints, err := p.checkpointStore.ListCheckpoints(ctx, details.FullyQualifiedNamespace, details.EventHubName, details.ConsumerGroup, nil)

	if err != nil {
		return nil, err
	}

	m := map[string]Checkpoint{}

	for _, cp := range checkpoints {
		m[cp.PartitionID] = cp
	}

	return m, nil
}

func (p *Processor) close(ctx context.Context, consumersMap *sync.Map) {
	consumersMap.Range(func(key, value any) bool {
		client := value.(*ProcessorPartitionClient)

		if client != nil {
			_ = client.Close(ctx)
		}

		return true
	})

	currentOwnerships := p.currentOwnerships.Load().([]Ownership)

	for i := 0; i < len(currentOwnerships); i++ {
		currentOwnerships[i].OwnerID = relinquishedOwnershipID
	}

	_, err := p.checkpointStore.ClaimOwnership(ctx, currentOwnerships, nil)

	if err != nil {
		azlog.Writef(EventConsumer, "Failed to relinquish ownerships. New processors will have to wait for ownerships to expire: %s", err.Error())
	}

	p.stateMu.Lock()
	p.state = stateStopped
	p.stateMu.Unlock()

	// NextPartitionClient() will quit out now that p.nextClients is closed.
	close(p.nextClients)

	select {
	case <-p.nextClientsReady:
		// already closed
	default:
		close(p.nextClientsReady)
	}
}

// relinquishedOwnershipID indicates that a partition is immediately available, similar to
// how we treat an ownership that is expired as available.
const relinquishedOwnershipID = ""
