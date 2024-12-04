package events

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"
)

// BatchMode enables the Events client to accept, queue, and post
// Events on behalf of the consuming application
func (e *Events) BatchMode(ctx context.Context, accountID int, opts ...BatchConfigOption) (err error) {
	if e.eventQueue != nil {
		return errors.New("the Events client is already in batch mode")
	}

	// Loop through config options
	for _, fn := range opts {
		if nil != fn {
			if err := fn(e); err != nil {
				return err
			}
		}
	}

	e.accountID = accountID
	e.eventQueue = make(chan []byte, e.batchSize)
	e.flushQueue = make([]chan bool, e.batchWorkers)
	e.eventTimer = time.NewTimer(e.batchTimeout)

	// Handle timer based flushing
	go func() {
		err := e.watchdog(ctx)
		if err != nil {
			e.logger.Error("watchdog returned error", "error", err)
		}
	}()

	// Spin up some workers
	for x := range e.flushQueue {
		e.flushQueue[x] = make(chan bool, 1)

		go func(id int) {
			err := e.batchWorker(ctx, id)
			if err != nil {
				e.logger.Error("batch worker returned error", "error", err)
			}
		}(x)
	}

	return nil
}

type BatchConfigOption func(*Events) error

// BatchConfigWorkers sets how many background workers will process
// events as they are queued
func BatchConfigWorkers(count int) BatchConfigOption {
	return func(e *Events) error {
		if count <= 0 {
			return errors.New("events: invalid worker count specified")
		}

		e.batchWorkers = count

		return nil
	}
}

// BatchConfigQueueSize is how many events to queue before sending
// to New Relic.  If this limit is hit before the Timeout, the queue
// is flushed.
func BatchConfigQueueSize(size int) BatchConfigOption {
	return func(e *Events) error {
		if size <= 0 {
			return errors.New("events: invalid queue size specified")
		}

		e.batchSize = size
		return nil
	}
}

// BatchConfigTimeout is the maximum amount of time to queue events
// before sending to New Relic.  If this is reached before the Size
// limit, the queue is flushed.
func BatchConfigTimeout(seconds int) BatchConfigOption {
	return func(e *Events) error {
		if seconds <= 0 {
			return errors.New("events: invalid timeout specified")
		}

		e.batchTimeout = time.Duration(seconds) * time.Second
		return nil
	}
}

// EnqueueEventContext handles the queueing. Only works in batch mode. If you wish to be able to avoid blocking
// forever until the event can be queued, provide a ctx with a deadline or timeout as this function will
// bail when ctx.Done() is closed and return and error.
func (e *Events) EnqueueEvent(ctx context.Context, event interface{}) (err error) {
	if e.eventQueue == nil {
		return errors.New("queueing not enabled for this client")
	}

	jsonData, err := e.marshalEvent(event)
	if err != nil {
		return err
	}
	if jsonData == nil {
		return errors.New("events: EnqueueEvent marhal returned nil data")
	}

	select {
	case e.eventQueue <- *jsonData:
		return nil
	case <-ctx.Done():
		e.logger.Trace("EnqueueEvent: exiting per context Done")
		return ctx.Err()
	}
}

// Flush gives the user a way to manually flush the queue in the foreground.
// This is also used by watchdog when the timer expires.
func (e *Events) Flush() error {
	if e.flushQueue == nil {
		return errors.New("queueing not enabled for this client")
	}

	e.logger.Debug("flushing events")

	for x := range e.flushQueue {
		e.flushQueue[x] <- true
	}

	return nil
}

// batchWorker reads []byte from the queue until a threshold is passed,
// then copies the []byte it has read and sends that batch along to Insights
// in its own goroutine.
func (e *Events) batchWorker(ctx context.Context, id int) (err error) {
	if e == nil {
		return errors.New("batchWorker: invalid Events, unable to start worker")
	}
	if id < 0 || len(e.flushQueue) < id {
		return errors.New("batchWorker: invalid worker id specified")
	}

	eventBuf := make([][]byte, e.batchSize)
	count := 0

	for {
		select {
		case item := <-e.eventQueue:
			eventBuf[count] = item
			count++
			if count >= e.batchSize {
				e.grabAndConsumeEvents(count, eventBuf)
				count = 0
			}
		case <-e.flushQueue[id]:
			if count > 0 {
				e.grabAndConsumeEvents(count, eventBuf)
				count = 0
			}
		case <-ctx.Done():
			e.logger.Trace(fmt.Sprintf("batchWorker[%d]: exiting per context Done", id))
			return ctx.Err()
		}
	}
}

// watchdog has a Timer that will send the results once the
// it has expired.
func (e *Events) watchdog(ctx context.Context) (err error) {
	if e.eventTimer == nil {
		return errors.New("invalid timer for watchdog()")
	}

	for {
		select {
		case <-e.eventTimer.C:
			e.logger.Debug("Timeout expired, flushing queued events")
			if err = e.Flush(); err != nil {
				return
			}
			e.eventTimer.Reset(e.batchTimeout)
		case <-ctx.Done():
			e.logger.Trace("watchdog exiting: context finished")
			return ctx.Err()
		}
	}
}

// grabAndConsumeEvents makes a copy of the event handles,
// and asynchronously writes those events in its own goroutine.
func (e *Events) grabAndConsumeEvents(count int, eventBuf [][]byte) {
	saved := make([][]byte, count)
	for i := 0; i < count; i++ {
		saved[i] = eventBuf[i]
		eventBuf[i] = nil
	}

	go func(count int, saved [][]byte) {
		if sendErr := e.sendEvents(saved[0:count]); sendErr != nil {
			e.logger.Error("failed to send events")
		}
	}(count, saved)
}

func (e *Events) sendEvents(events [][]byte) error {
	var buf bytes.Buffer

	// Since we already marshalled all of the data into JSON, let's make a
	// hand-crafted, artisanal JSON array
	buf.WriteString("[")
	eventCount := len(events) - 1
	for e := range events {
		buf.Write(events[e])
		if e < eventCount {
			buf.WriteString(",")
		}
	}
	buf.WriteString("]")

	resp := &createEventResponse{}

	_, err := e.client.Post(e.config.Region().InsightsURL(e.accountID), nil, buf.Bytes(), resp)

	if err != nil {
		return err
	}

	if !resp.Success {
		return errors.New("failed creating custom event")
	}

	return nil
}
