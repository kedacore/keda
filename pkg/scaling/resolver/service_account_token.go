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
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"

	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
)

const ServiceAccountTokenAudiencesEnvVar = "KEDA_SERVICE_ACCOUNT_TOKEN_AUDIENCES"

// ServiceAccountTokenAudience permits an audience for file tokens and optionally
// selects it for token minting for an exact namespace/service account pair.
type ServiceAccountTokenAudience struct {
	Audience           string `json:"audience"`
	Namespace          string `json:"namespace,omitempty"`
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

// LoadServiceAccountTokenAudiences reads audience approvals and minting mappings from YAML or JSON.
func (cfg *Config) LoadServiceAccountTokenAudiences(value string) error {
	var audiences []ServiceAccountTokenAudience
	if strings.TrimSpace(value) != "" {
		reader := k8syaml.NewYAMLReader(bufio.NewReader(strings.NewReader(value)))
		document, err := reader.Read()
		if err != nil {
			return fmt.Errorf("invalid %s: %w", ServiceAccountTokenAudiencesEnvVar, err)
		}
		if _, err = reader.Read(); !errors.Is(err, io.EOF) {
			return fmt.Errorf("%s must contain one YAML or JSON document", ServiceAccountTokenAudiencesEnvVar)
		}
		if err := yaml.UnmarshalStrict(document, &audiences); err != nil {
			return fmt.Errorf("invalid %s: %w", ServiceAccountTokenAudiencesEnvVar, err)
		}
		if audiences == nil {
			return fmt.Errorf("%s must be an array, not null", ServiceAccountTokenAudiencesEnvVar)
		}
	}
	cfg.ServiceAccountTokenAudiences = audiences
	return cfg.validateServiceAccountTokenPolicy()
}

// IsLegacyServiceAccountTokenMode reports whether the audience enforcement is disabled.
func (cfg *Config) IsLegacyServiceAccountTokenMode() bool {
	return cfg.ServiceAccountTokenMode == "legacy"
}

func (cfg *Config) validateServiceAccountTokenPolicy() error {
	switch cfg.ServiceAccountTokenMode {
	case "legacy":
		if len(cfg.ServiceAccountTokenAudiences) != 0 {
			return errors.New("legacy token mode cannot be combined with configured audiences; use enforce-audience or remove the audience settings")
		}
		return nil
	case "", "enforce-audience":
	default:
		return fmt.Errorf("unsupported service-account-token-mode %q: expected enforce-audience or legacy", cfg.ServiceAccountTokenMode)
	}
	seen := make(map[types.NamespacedName]bool)
	for _, entry := range cfg.ServiceAccountTokenAudiences {
		if entry.Audience == "" || entry.Audience != strings.TrimSpace(entry.Audience) {
			return errors.New("service account token audiences must be nonempty and contain no surrounding whitespace")
		}
		if entry.Namespace == "" && entry.ServiceAccountName == "" {
			continue
		}
		if entry.Namespace == "" || entry.ServiceAccountName == "" {
			return fmt.Errorf("%s entries must set both namespace and serviceAccountName for minting, or omit both for audience approval only", ServiceAccountTokenAudiencesEnvVar)
		}
		if len(validation.IsDNS1123Label(entry.Namespace)) != 0 || len(validation.IsDNS1123Subdomain(entry.ServiceAccountName)) != 0 {
			return fmt.Errorf("%s entries require valid, exact Kubernetes namespace and service account names; wildcards are not supported", ServiceAccountTokenAudiencesEnvVar)
		}
		key := types.NamespacedName{Namespace: entry.Namespace, Name: entry.ServiceAccountName}
		if seen[key] {
			return fmt.Errorf("duplicate service account token audience mapping for %s", key)
		}
		seen[key] = true
	}
	return nil
}

// Every entry permits file tokens; the allowed set is shared across integrations.
func (cfg *Config) serviceAccountTokenAllowedAudiences() []string {
	audiences := make([]string, 0, len(cfg.ServiceAccountTokenAudiences))
	for _, entry := range cfg.ServiceAccountTokenAudiences {
		audiences = append(audiences, entry.Audience)
	}
	return audiences
}

func (cfg *Config) serviceAccountTokenAudiences(namespace, name string) ([]string, error) {
	for _, mapping := range cfg.ServiceAccountTokenAudiences {
		if mapping.ServiceAccountName != "" && mapping.Namespace == namespace && mapping.ServiceAccountName == name {
			return []string{mapping.Audience}, nil
		}
	}
	return nil, fmt.Errorf("no configured token audience for service account %s/%s in %s", namespace, name, ServiceAccountTokenAudiencesEnvVar)
}

// GenerateBoundServiceAccountToken mints a token for Vault or BSAT authentication.
// Outside legacy mode, the configured SA audience is required before TokenRequest.
func GenerateBoundServiceAccountToken(ctx context.Context, serviceAccountName, namespace string, acs *authentication.AuthClientSet) (string, error) {
	legacy := globalConfig.IsLegacyServiceAccountTokenMode()
	var audiences []string
	if !legacy {
		var err error
		audiences, err = globalConfig.serviceAccountTokenAudiences(namespace, serviceAccountName)
		if err != nil {
			return "", err
		}
	}
	token, err := acs.CoreV1Interface.ServiceAccounts(namespace).CreateToken(ctx, serviceAccountName,
		&authenticationv1.TokenRequest{Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: new(int64(boundServiceAccountTokenExpiry.Seconds())),
			Audiences:         audiences,
		}}, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create bound service account token for %s/%s: %w", namespace, serviceAccountName, err)
	}
	if token == nil || token.Status.Token == "" {
		return "", fmt.Errorf("token request for %s/%s returned an empty token", namespace, serviceAccountName)
	}
	if legacy {
		log.Info("Warning: minting a legacy service account token without an explicit audience; forwarding it may expose Kubernetes API access. Configure KEDA_SERVICE_ACCOUNT_TOKEN_AUDIENCES and --service-account-token-mode=enforce-audience", "ServiceAccount.Namespace", namespace, "ServiceAccount.Name", serviceAccountName)
	} else if err := validateK8sSATokenAudiences([]byte(token.Status.Token), audiences); err != nil {
		return "", fmt.Errorf("invalid token returned for service account %s/%s: %w", namespace, serviceAccountName, err)
	}
	log.V(1).Info("Bound service account token created successfully", "ServiceAccount.Namespace", namespace, "ServiceAccount.Name", serviceAccountName, "Token.Audiences", audiences)
	return token.Status.Token, nil
}
