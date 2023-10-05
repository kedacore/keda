// Copyright (c) 2021 VMware, Inc. or its affiliates. All Rights Reserved.
// Copyright (c) 2012-2021, Sean Treadway, SoundCloud Ltd.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package amqp091

import (
	"os"
	"strconv"
	"sync"
	"sync/atomic"
)

var consumerSeq uint64

const consumerTagLengthMax = 0xFF // see writeShortstr

func uniqueConsumerTag() string {
	return commandNameBasedUniqueConsumerTag(os.Args[0])
}

func commandNameBasedUniqueConsumerTag(commandName string) string {
	tagPrefix := "ctag-"
	tagInfix := commandName
	tagSuffix := "-" + strconv.FormatUint(atomic.AddUint64(&consumerSeq, 1), 10)

	if len(tagPrefix)+len(tagInfix)+len(tagSuffix) > consumerTagLengthMax {
		tagInfix = "streadway/amqp"
	}

	return tagPrefix + tagInfix + tagSuffix
}

type consumerBuffers map[string]chan *Delivery

// Concurrent type that manages the consumerTag ->
// ingress consumerBuffer mapping
type consumers struct {
	sync.WaitGroup               // one for buffer
	closed         chan struct{} // signal buffer

	sync.Mutex // protects below
	chans      consumerBuffers
}

func makeConsumers() *consumers {
	return &consumers{
		closed: make(chan struct{}),
		chans:  make(consumerBuffers),
	}
}

func (subs *consumers) buffer(in chan *Delivery, out chan Delivery) {
	defer close(out)
	defer subs.Done()

	var inflight = in
	var queue []*Delivery

	for delivery := range in {
		queue = append(queue, delivery)

		for len(queue) > 0 {
			select {
			case <-subs.closed:
				// closed before drained, drop in-flight
				return

			case delivery, consuming := <-inflight:
				if consuming {
					queue = append(queue, delivery)
				} else {
					inflight = nil
				}

			case out <- *queue[0]:
				/*
				* https://github.com/rabbitmq/amqp091-go/issues/179
				* https://github.com/rabbitmq/amqp091-go/pull/180
				*
				* Comment from @lars-t-hansen:
				*
				* Given Go's slice semantics, and barring any information
				* available to the compiler that proves that queue is the only
				* pointer to the memory it references, the only meaning that
				* queue = queue[1:] can have is basically queue += sizeof(queue
				* element), ie, it bumps a pointer. Looking at the generated
				* code for a simple example (on ARM64 in this case) bears this
				* out. So what we're left with is an array that we have a
				* pointer into the middle of. When the GC traces this pointer,
				* it too does not know whether the array has multiple
				* referents, and so its only sensible choice is to find the
				* beginning of the array, and if the array is not already
				* visited, mark every element in it, including the "dead"
				* pointer.
				*
				* (Depending on the program dynamics, an element may eventually
				* be appended to the queue when the queue is at capacity, and
				* in this case the live elements are copied into a new array
				* and the old array is left to be GC'd eventually, along with
				* the dead object. But that can take time.)
				 */
				queue[0] = nil
				queue = queue[1:]
			}
		}
	}
}

// On key conflict, close the previous channel.
func (subs *consumers) add(tag string, consumer chan Delivery) {
	subs.Lock()
	defer subs.Unlock()

	if prev, found := subs.chans[tag]; found {
		close(prev)
	}

	in := make(chan *Delivery)
	subs.chans[tag] = in

	subs.Add(1)
	go subs.buffer(in, consumer)
}

func (subs *consumers) cancel(tag string) (found bool) {
	subs.Lock()
	defer subs.Unlock()

	ch, found := subs.chans[tag]

	if found {
		delete(subs.chans, tag)
		close(ch)
	}

	return found
}

func (subs *consumers) close() {
	subs.Lock()
	defer subs.Unlock()

	close(subs.closed)

	for tag, ch := range subs.chans {
		delete(subs.chans, tag)
		close(ch)
	}

	subs.Wait()
}

// Sends a delivery to a the consumer identified by `tag`.
// If unbuffered channels are used for Consume this method
// could block all deliveries until the consumer
// receives on the other end of the channel.
func (subs *consumers) send(tag string, msg *Delivery) bool {
	subs.Lock()
	defer subs.Unlock()

	buffer, found := subs.chans[tag]
	if found {
		buffer <- msg
	}

	return found
}
