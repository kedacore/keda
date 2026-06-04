/*
Copyright 2026 Google LLC

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

package spanner

import (
	"context"
	"sync"

	"google.golang.org/grpc"
)

// channelEndpoint represents a routable server endpoint.
type channelEndpoint interface {
	Address() string
	IsHealthy() bool
	IsTransientFailure() bool
	GetConn() *grpc.ClientConn
	IncrementActiveRequests()
	DecrementActiveRequests()
	ActiveRequestCount() int
}

// channelEndpointCache caches endpoints by server address.
type channelEndpointCache interface {
	Get(ctx context.Context, address string) channelEndpoint
	GetIfPresent(address string) channelEndpoint
	Evict(address string)
	DefaultChannel() channelEndpoint
	ClientFor(ep channelEndpoint) spannerClient
	Close() error
}

type passthroughChannelEndpoint struct {
	address string
}

var (
	_ channelEndpoint      = (*passthroughChannelEndpoint)(nil)
	_ channelEndpointCache = (*passthroughChannelEndpointCache)(nil)
)

func (e *passthroughChannelEndpoint) Address() string {
	return e.address
}

func (*passthroughChannelEndpoint) IsHealthy() bool {
	return true
}

func (*passthroughChannelEndpoint) IsTransientFailure() bool {
	return false
}

func (*passthroughChannelEndpoint) GetConn() *grpc.ClientConn {
	return nil
}

func (*passthroughChannelEndpoint) IncrementActiveRequests() {}

func (*passthroughChannelEndpoint) DecrementActiveRequests() {}

func (*passthroughChannelEndpoint) ActiveRequestCount() int {
	return 0
}

type passthroughChannelEndpointCache struct {
	mu              sync.Mutex
	endpoints       map[string]*passthroughChannelEndpoint
	defaultEndpoint *passthroughChannelEndpoint
}

func newPassthroughChannelEndpointCache() *passthroughChannelEndpointCache {
	return &passthroughChannelEndpointCache{
		endpoints:       make(map[string]*passthroughChannelEndpoint),
		defaultEndpoint: &passthroughChannelEndpoint{address: ""},
	}
}

func (c *passthroughChannelEndpointCache) Get(_ context.Context, address string) channelEndpoint {
	if address == "" {
		return c.defaultEndpoint
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if endpoint, ok := c.endpoints[address]; ok {
		return endpoint
	}
	endpoint := &passthroughChannelEndpoint{address: address}
	c.endpoints[address] = endpoint
	return endpoint
}

func (c *passthroughChannelEndpointCache) GetIfPresent(address string) channelEndpoint {
	if address == "" {
		return c.defaultEndpoint
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	endpoint, ok := c.endpoints[address]
	if !ok {
		return nil
	}
	return endpoint
}

func (c *passthroughChannelEndpointCache) Evict(address string) {
	if address == c.defaultEndpoint.Address() {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.endpoints, address)
}

func (c *passthroughChannelEndpointCache) DefaultChannel() channelEndpoint {
	return c.defaultEndpoint
}

func (c *passthroughChannelEndpointCache) ClientFor(_ channelEndpoint) spannerClient {
	return nil
}

func (c *passthroughChannelEndpointCache) Close() error {
	return nil
}
