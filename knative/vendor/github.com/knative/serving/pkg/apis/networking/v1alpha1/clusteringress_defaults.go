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

package v1alpha1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// DefaultTimeout will be set if timeout not specified.
	DefaultTimeout = 10 * time.Minute
	// DefaultRetryCount will be set if Attempts not specified.
	DefaultRetryCount = 3
)

func (c *ClusterIngress) SetDefaults() {
	c.Spec.SetDefaults()
}

func (c *IngressSpec) SetDefaults() {
	for i := range c.TLS {
		c.TLS[i].SetDefaults()
	}
	for i := range c.Rules {
		c.Rules[i].SetDefaults()
	}
	if c.Visibility == "" {
		c.Visibility = IngressVisibilityExternalIP
	}
}

func (t *ClusterIngressTLS) SetDefaults() {
	// Default Secret key for ServerCertificate is `tls.crt`.
	if t.ServerCertificate == "" {
		t.ServerCertificate = "tls.crt"
	}
	// Default Secret key for PrivateKey is `tls.key`.
	if t.PrivateKey == "" {
		t.PrivateKey = "tls.key"
	}
}

func (r *ClusterIngressRule) SetDefaults() {
	r.HTTP.SetDefaults()
}

func (r *HTTPClusterIngressRuleValue) SetDefaults() {
	for i := range r.Paths {
		r.Paths[i].SetDefaults()
	}
}

func (p *HTTPClusterIngressPath) SetDefaults() {
	// If only one split is specified, we default to 100.
	if len(p.Splits) == 1 && p.Splits[0].Percent == 0 {
		p.Splits[0].Percent = 100
	}

	if p.Timeout == nil {
		p.Timeout = &metav1.Duration{Duration: DefaultTimeout}
	}

	if p.Retries == nil {
		p.Retries = &HTTPRetry{
			PerTryTimeout: &metav1.Duration{Duration: DefaultTimeout},
			Attempts:      DefaultRetryCount,
		}
	}
	if p.Retries.PerTryTimeout == nil {
		p.Retries.PerTryTimeout = &metav1.Duration{Duration: DefaultTimeout}
	}
}
