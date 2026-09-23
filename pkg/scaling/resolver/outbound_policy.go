/*
Copyright 2026 The KEDA Authors

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

package resolver

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"
)

const OutboundFilterEnvVar = "KEDA_OUTBOUND_FILTER"

type endpointPolicy struct {
	Mode             *string  `json:"mode"`
	AllowedEndpoints []string `json:"allowedEndpoints"`
}

// LoadOutboundFilter configures HashiCorp Vault destination filtering.
// The default is off; warn logs unlisted endpoints and enforce rejects them.
func (cfg *Config) LoadOutboundFilter(value string) error {
	defaultMode := outboundPolicyOff
	filter := &struct {
		HashiCorpVault *endpointPolicy `json:"hashiCorpVault"`
	}{HashiCorpVault: &endpointPolicy{Mode: &defaultMode}}
	if strings.TrimSpace(value) != "" {
		reader := k8syaml.NewYAMLReader(bufio.NewReader(strings.NewReader(value)))
		document, err := reader.Read()
		if err != nil {
			return fmt.Errorf("invalid %s: %w", OutboundFilterEnvVar, err)
		}
		if _, err = reader.Read(); !errors.Is(err, io.EOF) {
			return fmt.Errorf("%s must contain one YAML or JSON document", OutboundFilterEnvVar)
		}
		if err := yaml.UnmarshalStrict(document, &filter); err != nil {
			return fmt.Errorf("invalid %s: %w", OutboundFilterEnvVar, err)
		}
	}
	if filter == nil || filter.HashiCorpVault == nil {
		return fmt.Errorf("%s and its hashiCorpVault section must be objects, not null", OutboundFilterEnvVar)
	}
	if filter.HashiCorpVault.Mode == nil || *filter.HashiCorpVault.Mode == "" {
		return fmt.Errorf("%s.hashiCorpVault.mode must be off, warn, or enforce", OutboundFilterEnvVar)
	}
	cfg.OutboundEndpointPolicy = *filter.HashiCorpVault.Mode
	cfg.AllowedOutboundEndpoints = filter.HashiCorpVault.AllowedEndpoints
	return cfg.Validate()
}
