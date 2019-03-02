/*
Copyright 2019 The Knative Authors

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

// route.go provides methods to perform actions on the route resource.

package test

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	pkgTest "github.com/knative/pkg/test"
	"github.com/knative/pkg/test/spoof"
	"golang.org/x/sync/errgroup"
)

// Prober is the interface for a prober, which checks the result of the probes when stopped.
type Prober interface {
	// SLI returns the "service level indicator" for the prober, which is the observed
	// success rate of the probes.  This will panic if the prober has not been stopped.
	SLI() (total int64, failures int64)

	// Stop terminates the prober, returning any observed errors.
	// Implementations may choose to put additional requirements on
	// the prober, which may cause this to block (e.g. a minimum number
	// of probes to achieve a population suitable for SLI measurement).
	Stop() error
}

type prober struct {
	// These shouldn't change after creation
	t             *testing.T
	domain        string
	minimumProbes int64

	// m guards access to these fields
	m        sync.RWMutex
	requests int64
	failures int64
	stopped  bool

	// This channel is used to send errors encountered probing the domain.
	errCh chan error
	// This channel is simply closed when minimumProbes has been satisfied.
	minDoneCh chan struct{}
}

// prober implements Prober
var _ Prober = (*prober)(nil)

// SLI implements Prober
func (p *prober) SLI() (int64, int64) {
	p.m.RLock()
	defer p.m.RUnlock()

	return p.requests, p.failures
}

// Stop implements Prober
func (p *prober) Stop() error {
	// When we're done stop sending requests.
	defer func() {
		p.m.Lock()
		defer p.m.Unlock()
		p.stopped = true
	}()

	// Check for any immediately available errors
	select {
	case err := <-p.errCh:
		return err
	default:
		// Don't block if there are no errors immediately available.
	}

	// If there aren't any immediately available errors, then
	// wait for either an error or the minimum number of probes
	// to be satisfied.
	select {
	case err := <-p.errCh:
		return err
	case <-p.minDoneCh:
		return nil
	}
}

func (p *prober) handleResponse(response *spoof.Response) (bool, error) {
	p.m.Lock()
	defer p.m.Unlock()

	if p.stopped {
		return p.stopped, nil
	}

	p.requests++
	if response.StatusCode != http.StatusOK {
		p.t.Logf("%q got bad status: %d\nHeaders:%v\nBody: %s", p.domain, response.StatusCode,
			response.Header, string(response.Body))
		p.failures++
	}
	if p.requests == p.minimumProbes {
		close(p.minDoneCh)
	}

	// Returning (false, nil) causes SpoofingClient.Poll to retry.
	return false, nil
}

// ProberManager is the interface for spawning probers, and checking their results.
type ProberManager interface {
	// The ProberManager should expose a way to collectively reason about spawned
	// probes as a sort of aggregating Prober.
	Prober

	// Spawn creates a new Prober
	Spawn(domain string) Prober

	// Foreach iterates over the probers spawned by this ProberManager.
	Foreach(func(domain string, p Prober))
}

type manager struct {
	// Should not change after creation
	t         *testing.T
	clients   *Clients
	minProbes int64

	m      sync.RWMutex
	probes map[string]Prober
}

var _ ProberManager = (*manager)(nil)

// Spawn implements ProberManager
func (m *manager) Spawn(domain string) Prober {
	m.m.Lock()
	defer m.m.Unlock()

	if p, ok := m.probes[domain]; ok {
		return p
	}

	m.t.Logf("Starting Route prober for route domain %s.", domain)
	p := &prober{
		t:             m.t,
		domain:        domain,
		minimumProbes: m.minProbes,
		errCh:         make(chan error, 1),
		minDoneCh:     make(chan struct{}),
	}
	m.probes[domain] = p
	go func() {
		client, err := pkgTest.NewSpoofingClient(m.clients.KubeClient, m.t.Logf, domain,
			ServingFlags.ResolvableDomain)
		if err != nil {
			m.t.Logf("NewSpoofingClient() = %v", err)
			p.errCh <- err
			return
		}

		// RequestTimeout is set to 0 to make the polling infinite.
		client.RequestTimeout = 0
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s", domain), nil)
		if err != nil {
			m.t.Logf("NewRequest() = %v", err)
			p.errCh <- err
			return
		}

		// We keep polling the domain and accumulate success rates
		// to ultimately establish the SLI and compare to the SLO.
		_, err = client.Poll(req, p.handleResponse)
		if err != nil {
			m.t.Logf("Poll() = %v", err)
			p.errCh <- err
			return
		}
	}()
	return p
}

// Stop implements ProberManager
func (m *manager) Stop() error {
	m.m.Lock()
	defer m.m.Unlock()

	m.t.Log("Stopping all probers")

	errgrp := &errgroup.Group{}
	for _, prober := range m.probes {
		errgrp.Go(prober.Stop)
	}
	return errgrp.Wait()
}

// SLI implements Prober
func (m *manager) SLI() (total int64, failures int64) {
	m.m.RLock()
	defer m.m.RUnlock()
	for _, prober := range m.probes {
		pt, pf := prober.SLI()
		total += pt
		failures += pf
	}
	return
}

// Foreach implements ProberManager
func (m *manager) Foreach(f func(domain string, p Prober)) {
	m.m.RLock()
	defer m.m.RUnlock()

	for domain, prober := range m.probes {
		f(domain, prober)
	}
}

func NewProberManager(t *testing.T, clients *Clients, minProbes int64) ProberManager {
	return &manager{
		t:         t,
		clients:   clients,
		minProbes: minProbes,
		probes:    make(map[string]Prober),
	}
}

// RunRouteProber starts a single Prober of the given domain.
func RunRouteProber(t *testing.T, clients *Clients, domain string) Prober {
	// Default to 10 probes
	pm := NewProberManager(t, clients, 10)
	pm.Spawn(domain)
	return pm
}

// AssertProberDefault is a helper for stopping the Prober and checking its SLI
// against the default SLO, which requires perfect responses.
// This takes `testing.T` so that it may be used in `defer`.
func AssertProberDefault(t *testing.T, p Prober) {
	t.Helper()
	if err := p.Stop(); err != nil {
		t.Errorf("Stop() = %v", err)
	}
	// Default to 100% correct (typically used in conjunction with the low probe count above)
	if err := CheckSLO(1.0, t.Name(), p); err != nil {
		t.Errorf("CheckSLO() = %v", err)
	}
}

// CheckSLO compares the SLI of the given prober against the SLO, erroring if too low.
func CheckSLO(slo float64, name string, p Prober) error {
	total, failures := p.SLI()

	successRate := float64(total-failures) / float64(total)
	if successRate < slo {
		return fmt.Errorf("SLI for %q = %f, wanted >= %f", name, successRate, slo)
	}
	return nil
}
