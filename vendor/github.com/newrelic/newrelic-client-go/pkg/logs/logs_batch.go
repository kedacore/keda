package logs

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// BatchMode enables the Logs client to accept, queue, and post
// Logs on behalf of the consuming application
func (e *Logs) BatchMode(ctx context.Context, accountID int, opts ...BatchConfigOption) (err error) {
	if e.logQueue != nil {
		return errors.New("the Logs client is already in batch mode")
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
	e.logQueue = make(chan interface{}, e.batchSize)
	e.flushQueue = make([]chan bool, e.batchWorkers)
	e.logTimer = time.NewTimer(e.batchTimeout)

	// Handle timer based flushing
	go func() {
		err := e.watchdog(ctx)
		if err != nil {
			e.logger.Error(fmt.Sprintf("watchdog returned error: %v", err))
		}
	}()

	// Spin up some workers
	for x := range e.flushQueue {
		e.flushQueue[x] = make(chan bool, 1)

		go func(id int) {
			e.logger.Trace("inside anonymous function")
			err := e.batchWorker(ctx, id)
			if err != nil {
				e.logger.Error(fmt.Sprintf("batch worker returned error: %v", err))
			}
		}(x)
	}

	return nil
}

type BatchConfigOption func(*Logs) error

// BatchConfigWorkers sets how many background workers will process
// logs as they are queued
func BatchConfigWorkers(count int) BatchConfigOption {
	return func(e *Logs) error {
		if count <= 0 {
			return errors.New("logs: invalid worker count specified")
		}

		e.batchWorkers = count

		return nil
	}
}

// BatchConfigQueueSize is how many logs to queue before sending
// to New Relic.  If this limit is hit before the Timeout, the queue
// is flushed.
func BatchConfigQueueSize(size int) BatchConfigOption {
	return func(e *Logs) error {
		if size <= 0 {
			return errors.New("logs: invalid queue size specified")
		}

		e.batchSize = size
		return nil
	}
}

// BatchConfigTimeout is the maximum amount of time to queue logs
// before sending to New Relic.  If this is reached before the Size
// limit, the queue is flushed.
func BatchConfigTimeout(seconds int) BatchConfigOption {
	return func(e *Logs) error {
		if seconds <= 0 {
			return errors.New("logs: invalid timeout specified")
		}

		e.batchTimeout = time.Duration(seconds) * time.Second
		return nil
	}
}

// EnqueueLogEntry handles the queueing. Only works in batch mode. If you wish to be able to avoid blocking
// forever until the log can be queued, provide a ctx with a deadline or timeout as this function will
// bail when ctx.Done() is closed and return and error.
func (e *Logs) EnqueueLogEntry(ctx context.Context, msg interface{}) (err error) {
	if e.logQueue == nil {
		return errors.New("queueing not enabled for this client")
	}

	select {
	case e.logQueue <- msg:
		e.logger.Trace("EnqueueLogEntry: log entry queued ")
		return nil
	case <-ctx.Done():
		e.logger.Trace("EnqueueLogEntry: exiting per context Done")
		return ctx.Err()
	}
}

// Flush gives the user a way to manually flush the queue in the foreground.
// This is also used by watchdog when the timer expires.
func (e *Logs) Flush() error {
	if e.flushQueue == nil {
		return errors.New("queueing not enabled for this client")
	}

	e.logger.Debug("flushing queues")
	for x := range e.flushQueue {
		e.logger.Trace(fmt.Sprintf("flushing logs queue: %d", x))
		e.flushQueue[x] <- true
	}

	return nil
}

//
// batchWorker reads []byte from the queue until a threshold is passed,
// then copies the []byte it has read and sends that batch along to Logs
// in its own goroutine.
//
func (e *Logs) batchWorker(ctx context.Context, id int) (err error) {
	e.logger.Trace("batchWorker")

	if id < 0 || len(e.flushQueue) < id {
		return errors.New("batchWorker: invalid worker id specified")
	}

	logBuf := make([]interface{}, e.batchSize)
	count := 0

	for {
		select {
		case item := <-e.logQueue:
			logBuf[count] = item
			count++
			if count >= e.batchSize {
				e.grabAndConsumeLogs(count, logBuf)
				count = 0
			}
		case <-e.flushQueue[id]:
			if count > 0 {
				e.grabAndConsumeLogs(count, logBuf)
				count = 0
			}
		case <-ctx.Done():
			e.logger.Trace("batchWorker[", id, "]: exiting per context Done")
			return ctx.Err()
		}
	}
}

//
// watchdog has a Timer that will send the results once the
// it has expired.
//
func (e *Logs) watchdog(ctx context.Context) (err error) {
	e.logger.Trace("watchdog")
	if e.logTimer == nil {
		return errors.New("invalid timer for watchdog()")
	}

	for {
		select {
		case <-e.logTimer.C:
			e.logger.Debug("Timeout expired, flushing queued logs")
			if err = e.Flush(); err != nil {
				return
			}
			e.logTimer.Reset(e.batchTimeout)
		case <-ctx.Done():
			e.logger.Trace("watchdog exiting: context finished")
			return ctx.Err()
		}
	}
}

// grabAndConsumeLogs makes a copy of the log handles,
// and asynchronously writes those logs in its own goroutine.
func (e *Logs) grabAndConsumeLogs(count int, logBuf []interface{}) {
	e.logger.Trace("grabAndConsumeLogs")
	saved := make([]interface{}, count)
	for i := 0; i < count; i++ {
		saved[i] = logBuf[i]
		logBuf[i] = nil
	}

	go func(count int, saved []interface{}) {
		if sendErr := e.sendLogs(saved[0:count]); sendErr != nil {
			e.logger.Error("failed to send logs")
		}
	}(count, saved)
}

func (e *Logs) sendLogs(logs []interface{}) error {
	e.logger.Trace(fmt.Sprintf("sendLogs: entry count: %d", len(logs)))
	_, err := e.client.Post(e.config.Region().LogsURL(), nil, logs, nil)

	if err != nil {
		return err
	}

	return nil
}
