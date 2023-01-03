// Copyright (c) 2021 VMware, Inc. or its affiliates. All Rights Reserved.
// Copyright (c) 2012-2021, Sean Treadway, SoundCloud Ltd.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package amqp091

import (
	"context"
	"sync"
)

// confirms resequences and notifies one or multiple publisher confirmation listeners
type confirms struct {
	m                     sync.Mutex
	listeners             []chan Confirmation
	sequencer             map[uint64]Confirmation
	deferredConfirmations *deferredConfirmations
	published             uint64
	publishedMut          sync.Mutex
	expecting             uint64
}

// newConfirms allocates a confirms
func newConfirms() *confirms {
	return &confirms{
		sequencer:             map[uint64]Confirmation{},
		deferredConfirmations: newDeferredConfirmations(),
		published:             0,
		expecting:             1,
	}
}

func (c *confirms) Listen(l chan Confirmation) {
	c.m.Lock()
	defer c.m.Unlock()

	c.listeners = append(c.listeners, l)
}

// Publish increments the publishing counter
func (c *confirms) Publish(ctx context.Context) *DeferredConfirmation {
	c.publishedMut.Lock()
	defer c.publishedMut.Unlock()

	c.published++
	return c.deferredConfirmations.Add(ctx, c.published)
}

// confirm confirms one publishing, increments the expecting delivery tag, and
// removes bookkeeping for that delivery tag.
func (c *confirms) confirm(confirmation Confirmation) {
	delete(c.sequencer, c.expecting)
	c.expecting++
	for _, l := range c.listeners {
		l <- confirmation
	}
}

// resequence confirms any out of order delivered confirmations
func (c *confirms) resequence() {
	c.publishedMut.Lock()
	defer c.publishedMut.Unlock()

	for c.expecting <= c.published {
		sequenced, found := c.sequencer[c.expecting]
		if !found {
			return
		}
		c.confirm(sequenced)
	}
}

// One confirms one publishing and all following in the publishing sequence
func (c *confirms) One(confirmed Confirmation) {
	c.m.Lock()
	defer c.m.Unlock()

	c.deferredConfirmations.Confirm(confirmed)

	if c.expecting == confirmed.DeliveryTag {
		c.confirm(confirmed)
	} else {
		c.sequencer[confirmed.DeliveryTag] = confirmed
	}
	c.resequence()
}

// Multiple confirms all publishings up until the delivery tag
func (c *confirms) Multiple(confirmed Confirmation) {
	c.m.Lock()
	defer c.m.Unlock()

	c.deferredConfirmations.ConfirmMultiple(confirmed)

	for c.expecting <= confirmed.DeliveryTag {
		c.confirm(Confirmation{c.expecting, confirmed.Ack})
	}
	c.resequence()
}

// Cleans up the confirms struct and its dependencies.
// Closes all listeners, discarding any out of sequence confirmations
func (c *confirms) Close() error {
	c.m.Lock()
	defer c.m.Unlock()

	c.deferredConfirmations.Close()

	for _, l := range c.listeners {
		close(l)
	}
	c.listeners = nil
	return nil
}

type deferredConfirmations struct {
	m             sync.Mutex
	confirmations map[uint64]*DeferredConfirmation
}

func newDeferredConfirmations() *deferredConfirmations {
	return &deferredConfirmations{
		confirmations: map[uint64]*DeferredConfirmation{},
	}
}

func (d *deferredConfirmations) Add(ctx context.Context, tag uint64) *DeferredConfirmation {
	d.m.Lock()
	defer d.m.Unlock()

	dc := &DeferredConfirmation{DeliveryTag: tag}
	dc.ctx, dc.cancel = context.WithCancel(ctx)
	d.confirmations[tag] = dc
	return dc
}

func (d *deferredConfirmations) Confirm(confirmation Confirmation) {
	d.m.Lock()
	defer d.m.Unlock()

	dc, found := d.confirmations[confirmation.DeliveryTag]
	if !found {
		// we should never receive a confirmation for a tag that hasn't been published, but a test causes this to happen
		return
	}
	dc.Confirm(confirmation.Ack)
	delete(d.confirmations, confirmation.DeliveryTag)
}

func (d *deferredConfirmations) ConfirmMultiple(confirmation Confirmation) {
	d.m.Lock()
	defer d.m.Unlock()

	for k, v := range d.confirmations {
		if k <= confirmation.DeliveryTag {
			v.Confirm(confirmation.Ack)
			delete(d.confirmations, k)
		}
	}
}

// Nacks all pending DeferredConfirmations being blocked by dc.Wait()
func (d *deferredConfirmations) Close() {
	d.m.Lock()
	defer d.m.Unlock()

	for k, v := range d.confirmations {
		v.Confirm(false)
		delete(d.confirmations, k)
	}
}

// Confirm ack confirmation.
func (d *DeferredConfirmation) Confirm(ack bool) {
	d.m.Lock()
	defer d.m.Unlock()

	d.confirmation.Ack = ack
	d.cancel()
}

// Waits for publisher confirmation. Returns true if server successfully received the publishing.
func (d *DeferredConfirmation) Wait() bool {
	<-d.ctx.Done()

	d.m.Lock()
	defer d.m.Unlock()
	return d.confirmation.Ack
}
