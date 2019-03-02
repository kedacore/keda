// +build e2e

/*
Copyright 2018 The Knative Authors

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

package conformance

import (
	"testing"

	"context"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"

	pkgTest "github.com/knative/pkg/test"
	"github.com/knative/pkg/test/spoof"
	"github.com/knative/serving/test"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	// Mysteriously required to support GCP auth (required by k8s libs). Apparently just importing it is enough. @_@ side effects @_@. https://github.com/kubernetes/client-go/issues/242
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

// Constants for test images located in test/test_images.
const (
	pizzaPlanet1        = "pizzaplanetv1"
	pizzaPlanet2        = "pizzaplanetv2"
	helloworld          = "helloworld"
	httpproxy           = "httpproxy"
	singleThreadedImage = "singlethreaded"
	timeout             = "timeout"
	printport           = "printport"
	runtime             = "runtime"
	protocols           = "protocols"

	concurrentRequests = 50
	// We expect to see 100% of requests succeed for traffic sent directly to revisions.
	// This might be a bad assumption.
	minDirectPercentage = 1
	// We expect to see at least 25% of either response since we're routing 50/50.
	// This might be a bad assumption.
	minSplitPercentage = 0.25
)

// Constants for test image output.
const (
	pizzaPlanetText1 = "What a spaceport!"
	pizzaPlanetText2 = "Re-energize yourself with a slice of pepperoni!"
	helloWorldText   = "Hello World! How about some tasty noodles?"
)

func setup(t *testing.T) *test.Clients {
	t.Helper()
	clients, err := test.NewClients(pkgTest.Flags.Kubeconfig, pkgTest.Flags.Cluster, test.ServingNamespace)
	if err != nil {
		t.Fatalf("Couldn't initialize clients: %v", err)
	}
	return clients
}

func waitForExpectedResponse(t *testing.T, clients *test.Clients, domain, expectedResponse string) error {
	client, err := pkgTest.NewSpoofingClient(clients.KubeClient, t.Logf, domain, test.ServingFlags.ResolvableDomain)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s", domain), nil)
	if err != nil {
		return err
	}
	_, err = client.Poll(req, pkgTest.EventuallyMatchesBody(expectedResponse))
	return err
}

func validateDomains(
	t *testing.T, clients *test.Clients, baseDomain string,
	baseExpected, trafficTargets, targetsExpected []string) error {
	var subdomains []string
	for _, target := range trafficTargets {
		subdomains = append(subdomains, fmt.Sprintf("%s.%s", target, baseDomain))
	}

	g, _ := errgroup.WithContext(context.Background())
	// We don't have a good way to check if the route is updated so we will wait until a subdomain has
	// started returning at least one expected result to key that we should validate percentage splits.
	// In order for tests to succeed reliably, we need to make sure that all domains succeed.
	for i, s := range subdomains {
		i, s := i, s
		g.Go(func() error {
			t.Logf("Waiting for route to update domain: %s", s)
			return waitForExpectedResponse(t, clients, s, targetsExpected[i])
		})
	}
	if err := g.Wait(); err != nil {
		return errors.Wrap(err, "error with initial domain probing")
	}

	g.Go(func() error {
		minBasePercentage := minSplitPercentage
		if len(baseExpected) == 1 {
			minBasePercentage = minDirectPercentage
		}
		min := int(math.Floor(concurrentRequests * minBasePercentage))
		return checkDistribution(t, clients, baseDomain, concurrentRequests, min, baseExpected)
	})
	for i, subdomain := range subdomains {
		i, subdomain := i, subdomain
		g.Go(func() error {
			min := int(math.Floor(concurrentRequests * minDirectPercentage))
			return checkDistribution(t, clients, subdomain, concurrentRequests, min, []string{targetsExpected[i]})
		})
	}
	if err := g.Wait(); err != nil {
		return errors.Wrap(err, "error checking routing distribution")
	}
	return nil
}

func validateImageDigest(imageName string, imageDigest string) (bool, error) {
	imageDigestRegex := fmt.Sprintf("%s/%s@sha256:[0-9a-f]{64}", test.ServingFlags.DockerRepo, imageName)
	return regexp.MatchString(imageDigestRegex, imageDigest)
}

// sendRequests sends "num" requests to "domain", returning a string for each spoof.Response.Body.
func sendRequests(client spoof.Interface, domain string, num int) ([]string, error) {
	responses := make([]string, num)

	// Launch "num" requests, recording the responses we get in "responses".
	g, _ := errgroup.WithContext(context.Background())
	for i := 0; i < num; i++ {
		// We don't index into "responses" inside the goroutine to avoid a race, see #1545.
		result := &responses[i]
		g.Go(func() error {
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s", domain), nil)
			if err != nil {
				return err
			}

			resp, err := client.Do(req)
			if err != nil {
				return err
			}

			*result = string(resp.Body)
			return nil
		})
	}
	return responses, g.Wait()
}

// checkResponses verifies that each "expectedResponse" is present in "actualResponses" at least "min" times.
func checkResponses(t *testing.T, num int, min int, domain string, expectedResponses []string, actualResponses []string) error {
	// counts maps the expected response body to the number of matching requests we saw.
	counts := make(map[string]int)
	// badCounts maps the unexpected response body to the number of matching requests we saw.
	badCounts := make(map[string]int)

	// counts := eval(
	//   SELECT body, count(*) AS total
	//   FROM $actualResponses
	//   WHERE body IN $expectedResponses
	//   GROUP BY body
	// )
	for _, ar := range actualResponses {
		expected := false
		for _, er := range expectedResponses {
			if strings.Contains(ar, er) {
				counts[er]++
				expected = true
			}
		}
		if !expected {
			badCounts[ar]++
		}
	}

	// Verify that we saw each entry in "expectedResponses" at least "min" times.
	// check(SELECT body FROM $counts WHERE total < $min)
	totalMatches := 0
	for _, er := range expectedResponses {
		count := counts[er]
		if count < min {
			return fmt.Errorf("domain %s failed: want at least %d, got %d for response %q", domain, min, count, er)
		}

		t.Logf("For domain %s: wanted at least %d, got %d requests.", domain, min, count)
		totalMatches += count
	}
	// Verify that the total expected responses match the number of requests made.
	for badResponse, count := range badCounts {
		t.Logf("Saw unexpected response %q %d times.", badResponse, count)
	}
	if totalMatches < num {
		return fmt.Errorf("domain %s: saw expected responses %d times, wanted %d", domain, totalMatches, num)
	}
	// If we made it here, the implementation conforms. Congratulations!
	return nil
}

// checkDistribution sends "num" requests to "domain", then validates that
// we see each body in "expectedResponses" at least "min" times.
func checkDistribution(t *testing.T, clients *test.Clients, domain string, num, min int, expectedResponses []string) error {
	client, err := pkgTest.NewSpoofingClient(clients.KubeClient, t.Logf, domain, test.ServingFlags.ResolvableDomain)
	if err != nil {
		return err
	}

	t.Logf("Performing %d concurrent requests to %s", num, domain)
	actualResponses, err := sendRequests(client, domain, num)
	if err != nil {
		return err
	}

	return checkResponses(t, num, min, domain, expectedResponses, actualResponses)
}
