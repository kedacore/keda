// Copyright Splunk Inc.
// SPDX-License-Identifier: Apache-2.0

package signalflow

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/signalfx/signalflow-client-go/v2/signalflow/messages"
	"github.com/signalfx/signalfx-go/idtool"
)

// Computation is a single running SignalFlow job
type Computation struct {
	sync.Mutex
	channel <-chan messages.Message
	name    string
	client  *Client
	dataCh  chan *messages.DataMessage
	// An intermediate channel for data messages where they can be buffered if
	// nothing is currently pulling data messages.
	dataChBuffer       chan *messages.DataMessage
	eventCh            chan *messages.EventMessage
	infoCh             chan *messages.InfoMessage
	eventChBuffer      chan *messages.EventMessage
	expirationCh       chan *messages.ExpiredTSIDMessage
	expirationChBuffer chan *messages.ExpiredTSIDMessage
	infoChBuffer       chan *messages.InfoMessage

	errMutex  sync.RWMutex
	lastError error

	handle                   asyncMetadata[string]
	resolutionMS             asyncMetadata[int]
	lagMS                    asyncMetadata[int]
	maxDelayMS               asyncMetadata[int]
	matchedSize              asyncMetadata[int]
	limitSize                asyncMetadata[int]
	matchedNoTimeseriesQuery asyncMetadata[string]
	groupByMissingProperties asyncMetadata[[]string]

	tsidMetadata map[idtool.ID]*asyncMetadata[*messages.MetadataProperties]
}

// ComputationError exposes the underlying metadata of a computation error
type ComputationError struct {
	Code      int
	Message   string
	ErrorType string
}

func (e *ComputationError) Error() string {
	err := fmt.Sprintf("%v", e.Code)
	if e.ErrorType != "" {
		err = fmt.Sprintf("%v (%v)", e.Code, e.ErrorType)
	}
	if e.Message != "" {
		err = fmt.Sprintf("%v: %v", err, e.Message)
	}
	return err
}

func newComputation(channel <-chan messages.Message, name string, client *Client) *Computation {
	comp := &Computation{
		channel:            channel,
		name:               name,
		client:             client,
		dataCh:             make(chan *messages.DataMessage),
		dataChBuffer:       make(chan *messages.DataMessage),
		eventCh:            make(chan *messages.EventMessage),
		infoCh:             make(chan *messages.InfoMessage),
		eventChBuffer:      make(chan *messages.EventMessage),
		expirationCh:       make(chan *messages.ExpiredTSIDMessage),
		expirationChBuffer: make(chan *messages.ExpiredTSIDMessage),
		infoChBuffer:       make(chan *messages.InfoMessage),
		tsidMetadata:       make(map[idtool.ID]*asyncMetadata[*messages.MetadataProperties]),
	}

	go bufferMessages(comp.dataChBuffer, comp.dataCh)
	go bufferMessages(comp.expirationChBuffer, comp.expirationCh)
	go bufferMessages(comp.eventChBuffer, comp.eventCh)
	go bufferMessages(comp.infoChBuffer, comp.infoCh)

	go func() {
		err := comp.watchMessages()

		if !errors.Is(err, errChannelClosed) {
			comp.errMutex.Lock()
			comp.lastError = err
			comp.errMutex.Unlock()
		}

		comp.shutdown()
	}()

	return comp
}

// Handle of the computation. Will wait as long as the given ctx is not closed. If ctx is closed an
// error will be returned.
func (c *Computation) Handle(ctx context.Context) (string, error) {
	return c.handle.Get(ctx)
}

// Resolution of the job. Will wait as long as the given ctx is not closed. If ctx is closed an
// error will be returned.
func (c *Computation) Resolution(ctx context.Context) (time.Duration, error) {
	resMS, err := c.resolutionMS.Get(ctx)
	return time.Duration(resMS) * time.Millisecond, err
}

// Lag detected for the job. Will wait as long as the given ctx is not closed. If ctx is closed an
// error will be returned.
func (c *Computation) Lag(ctx context.Context) (time.Duration, error) {
	lagMS, err := c.lagMS.Get(ctx)
	return time.Duration(lagMS) * time.Millisecond, err
}

// MaxDelay detected of the job. Will wait as long as the given ctx is not closed. If ctx is closed an
// error will be returned.
func (c *Computation) MaxDelay(ctx context.Context) (time.Duration, error) {
	maxDelayMS, err := c.maxDelayMS.Get(ctx)
	return time.Duration(maxDelayMS) * time.Millisecond, err
}

// MatchedSize detected of the job. Will wait as long as the given ctx is not closed. If ctx is closed an
// error will be returned.
func (c *Computation) MatchedSize(ctx context.Context) (int, error) {
	return c.matchedSize.Get(ctx)
}

// LimitSize detected of the job. Will wait as long as the given ctx is not closed. If ctx is closed an
// error will be returned.
func (c *Computation) LimitSize(ctx context.Context) (int, error) {
	return c.limitSize.Get(ctx)
}

// MatchedNoTimeseriesQuery if it matched no active timeseries. Will wait as long as the given ctx
// is not closed. If ctx is closed an error will be returned.
func (c *Computation) MatchedNoTimeseriesQuery(ctx context.Context) (string, error) {
	return c.matchedNoTimeseriesQuery.Get(ctx)
}

// GroupByMissingProperties are timeseries that don't contain the required dimensions. Will wait as
// long as the given ctx is not closed. If ctx is closed an error will be returned.
func (c *Computation) GroupByMissingProperties(ctx context.Context) ([]string, error) {
	return c.groupByMissingProperties.Get(ctx)
}

// TSIDMetadata for a particular tsid. Will wait as long as the given ctx is not closed. If ctx is closed an
// error will be returned.
func (c *Computation) TSIDMetadata(ctx context.Context, tsid idtool.ID) (*messages.MetadataProperties, error) {
	c.Lock()
	if _, ok := c.tsidMetadata[tsid]; !ok {
		c.tsidMetadata[tsid] = &asyncMetadata[*messages.MetadataProperties]{}
	}
	md := c.tsidMetadata[tsid]
	c.Unlock()
	return md.Get(ctx)
}

// Err returns the last fatal error that caused the computation to stop, if
// any.  Will be nil if the computation stopped in an expected manner.
func (c *Computation) Err() error {
	c.errMutex.RLock()
	defer c.errMutex.RUnlock()

	return c.lastError
}

func (c *Computation) watchMessages() error {
	for {
		m, ok := <-c.channel
		if !ok {
			return nil
		}
		if err := c.processMessage(m); err != nil {
			return err
		}
	}
}

var errChannelClosed = errors.New("computation channel is closed")

func (c *Computation) processMessage(m messages.Message) error {
	switch v := m.(type) {
	case *messages.JobStartControlMessage:
		c.handle.Set(v.Handle)
	case *messages.EndOfChannelControlMessage, *messages.ChannelAbortControlMessage:
		return errChannelClosed
	case *messages.DataMessage:
		c.dataChBuffer <- v
	case *messages.ExpiredTSIDMessage:
		c.Lock()
		delete(c.tsidMetadata, idtool.IDFromString(v.TSID))
		c.Unlock()
		c.expirationChBuffer <- v
	case *messages.InfoMessage:
		switch v.MessageBlock.Code {
		case messages.JobRunningResolution:
			c.resolutionMS.Set(v.MessageBlock.Contents.(messages.JobRunningResolutionContents).ResolutionMS())
		case messages.JobDetectedLag:
			c.lagMS.Set(v.MessageBlock.Contents.(messages.JobDetectedLagContents).LagMS())
		case messages.JobInitialMaxDelay:
			c.maxDelayMS.Set(v.MessageBlock.Contents.(messages.JobInitialMaxDelayContents).MaxDelayMS())
		case messages.FindLimitedResultSet:
			c.matchedSize.Set(v.MessageBlock.Contents.(messages.FindLimitedResultSetContents).MatchedSize())
			c.limitSize.Set(v.MessageBlock.Contents.(messages.FindLimitedResultSetContents).LimitSize())
		case messages.FindMatchedNoTimeseries:
			c.matchedNoTimeseriesQuery.Set(v.MessageBlock.Contents.(messages.FindMatchedNoTimeseriesContents).MatchedNoTimeseriesQuery())
		case messages.GroupByMissingProperty:
			c.groupByMissingProperties.Set(v.MessageBlock.Contents.(messages.GroupByMissingPropertyContents).GroupByMissingProperties())
		}
		c.infoChBuffer <- v
	case *messages.ErrorMessage:
		rawData := v.RawData()
		computationError := ComputationError{}
		if code, ok := rawData["error"]; ok {
			computationError.Code = int(code.(float64))
		}
		if msg, ok := rawData["message"]; ok && msg != nil {
			computationError.Message = msg.(string)
		}
		if errType, ok := rawData["errorType"]; ok {
			computationError.ErrorType = errType.(string)
		}
		return &computationError
	case *messages.MetadataMessage:
		c.Lock()
		if _, ok := c.tsidMetadata[v.TSID]; !ok {
			c.tsidMetadata[v.TSID] = &asyncMetadata[*messages.MetadataProperties]{}
		}
		c.tsidMetadata[v.TSID].Set(&v.Properties)
		c.Unlock()
	case *messages.EventMessage:
		c.eventChBuffer <- v
	}
	return nil
}

func bufferMessages[T any](in chan *T, out chan *T) {
	buffer := make([]*T, 0)
	var nextMessage *T

	defer func() {
		if nextMessage != nil {
			out <- nextMessage
		}
		for i := range buffer {
			out <- buffer[i]
		}

		close(out)
	}()
	for {
		if len(buffer) > 0 {
			if nextMessage == nil {
				nextMessage, buffer = buffer[0], buffer[1:]
			}

			select {
			case out <- nextMessage:
				nextMessage = nil
			case msg, ok := <-in:
				if !ok {
					return
				}
				buffer = append(buffer, msg)
			}
		} else {
			msg, ok := <-in
			if !ok {
				return
			}
			buffer = append(buffer, msg)
		}
	}
}

// Data returns the channel on which data messages come.  This channel will be closed when the
// computation is finished.  To prevent goroutine leaks, you should read all messages from this
// channel until it is closed.
func (c *Computation) Data() <-chan *messages.DataMessage {
	return c.dataCh
}

// Expirations returns a channel that will be sent messages about expired TSIDs, i.e. time series
// that are no longer valid for this computation. This channel will be closed when the computation
// is finished. To prevent goroutine leaks, you should read all messages from this channel until it
// is closed.
func (c *Computation) Expirations() <-chan *messages.ExpiredTSIDMessage {
	return c.expirationCh
}

// Events returns a channel that receives event/alert messages from the signalflow computation.
func (c *Computation) Events() <-chan *messages.EventMessage {
	return c.eventCh
}

// Info returns a channel that receives info messages from the signalflow computation.
func (c *Computation) Info() <-chan *messages.InfoMessage {
	return c.infoCh
}

// Detach the computation on the backend
func (c *Computation) Detach(ctx context.Context) error {
	return c.DetachWithReason(ctx, "")
}

// DetachWithReason detaches the computation with a given reason. This reason will
// be reflected in the control message that signals the end of the job/channel
func (c *Computation) DetachWithReason(ctx context.Context, reason string) error {
	return c.client.Detach(ctx, &DetachRequest{
		Reason:  reason,
		Channel: c.name,
	})
}

// Stop the computation on the backend.
func (c *Computation) Stop(ctx context.Context) error {
	return c.StopWithReason(ctx, "")
}

// StopWithReason stops the computation with a given reason. This reason will
// be reflected in the control message that signals the end of the job/channel.
func (c *Computation) StopWithReason(ctx context.Context, reason string) error {
	handle, err := c.handle.Get(ctx)
	if err != nil {
		return err
	}
	return c.client.Stop(ctx, &StopRequest{
		Reason: reason,
		Handle: handle,
	})
}

func (c *Computation) shutdown() {
	close(c.dataChBuffer)
	close(c.expirationChBuffer)
	close(c.infoChBuffer)
	close(c.eventChBuffer)
}

var ErrMetadataTimeout = errors.New("metadata value did not come in time")

type asyncMetadata[T any] struct {
	sync.Mutex
	sig   chan struct{}
	isSet bool
	val   T
}

func (a *asyncMetadata[T]) ensureInit() {
	a.Lock()
	if a.sig == nil {
		a.sig = make(chan struct{})
	}
	a.Unlock()
}

func (a *asyncMetadata[T]) Set(val T) {
	a.ensureInit()
	a.Lock()
	a.val = val
	if !a.isSet {
		close(a.sig)
		a.isSet = true
	}
	a.Unlock()
}

func (a *asyncMetadata[T]) Get(ctx context.Context) (T, error) {
	a.ensureInit()
	select {
	case <-ctx.Done():
		var t T
		return t, ErrMetadataTimeout
	case <-a.sig:
		return a.val, nil
	}
}
