/*
Copyright 2021 The KEDA Authors

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
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-logr/logr"
	vaultapi "github.com/hashicorp/vault/api"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
)

const (
	// serviceAccountTokenFile is the operator's Kubernetes API token path, used only for the legacy Vault authentication fallback. Remove when legacy mode is retired.
	serviceAccountTokenFile             = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	DefaultVaultKubernetesAuthTokenFile = "/var/run/secrets/keda-vault/token"
)

// HashicorpVaultHandler is a specification of HashiCorp Vault
type HashicorpVaultHandler struct {
	vault     *kedav1alpha1.HashiCorpVault
	client    *vaultapi.Client
	acs       *authentication.AuthClientSet
	namespace string
	stopCh    chan struct{}
}

// NewHashicorpVaultHandler creates a HashicorpVaultHandler object
func NewHashicorpVaultHandler(v *kedav1alpha1.HashiCorpVault, acs *authentication.AuthClientSet, namespace string) *HashicorpVaultHandler {
	return &HashicorpVaultHandler{
		vault:     v,
		acs:       acs,
		namespace: namespace,
	}
}

// Initialize the Vault client
func (vh *HashicorpVaultHandler) Initialize(logger logr.Logger) error {
	if err := vh.validateAddress(logger); err != nil {
		return err
	}

	config := vaultapi.DefaultConfig()
	client, err := vaultapi.NewClient(config)
	if err != nil {
		return err
	}

	err = client.SetAddress(vh.vault.Address)
	if err != nil {
		return err
	}

	if len(vh.vault.Namespace) > 0 {
		client.SetNamespace(vh.vault.Namespace)
	}

	token, err := vh.token(client, logger)
	if err != nil {
		return err
	}

	if len(token) > 0 {
		client.SetToken(token)
	}

	lookup, err := client.Auth().Token().LookupSelf()
	// If token is not valid so get out of here early
	if err != nil {
		return err
	}

	vh.client = client

	if renew, ok := lookup.Data["renewable"].(bool); ok && renew {
		vh.stopCh = make(chan struct{})
		go vh.renewToken(logger)
	}

	return nil
}

// token Extract a vault token from the Authentication method
func (vh *HashicorpVaultHandler) token(client *vaultapi.Client, logger logr.Logger) (string, error) {
	var token string

	switch vh.vault.Authentication {
	case kedav1alpha1.VaultAuthenticationToken:
		// Got token from VAULT_TOKEN env variable
		switch {
		case len(client.Token()) > 0:
			break
		case vh.vault.Credential != nil && len(vh.vault.Credential.Token) > 0:
			token = vh.vault.Credential.Token
		default:
			return token, errors.New("could not get Vault token")
		}
	case kedav1alpha1.VaultAuthenticationKubernetes:
		if len(vh.vault.Mount) == 0 {
			return token, errors.New("auth mount not in config")
		}

		if len(vh.vault.Role) == 0 {
			return token, errors.New("k8s role not in config")
		}

		jwt, err := vh.kubernetesToken(context.Background())
		if err != nil {
			return token, err
		}
		if globalConfig.IsLegacyServiceAccountTokenMode() {
			logger.Info("Warning: Vault Kubernetes authentication is using a legacy service account token; a leaked token may grant Kubernetes API access. Configure approved audiences and --service-account-token-mode=enforce-audience", "vaultOrigin", vh.origin())
		}

		data := map[string]any{"jwt": string(jwt), "role": vh.vault.Role}
		secret, err := client.Logical().Write(fmt.Sprintf("auth/%s/login", vh.vault.Mount), data)
		if err != nil {
			return token, err
		}
		if secret == nil || secret.Auth == nil || secret.Auth.ClientToken == "" {
			return token, errors.New("vault Kubernetes login returned no client token")
		}
		token = secret.Auth.ClientToken

	default:
		return token, fmt.Errorf("vault auth method %s is not supported", vh.vault.Authentication)
	}

	return token, nil
}

// kubernetesToken mints a named-SA token or reads a token file on each login.
func (vh *HashicorpVaultHandler) kubernetesToken(ctx context.Context) ([]byte, error) {
	legacy := globalConfig.IsLegacyServiceAccountTokenMode()
	path := globalConfig.VaultKubernetesAuthTokenFile
	if legacy {
		path = serviceAccountTokenFile
	}
	if credential := vh.vault.Credential; credential != nil {
		if credential.ServiceAccountName != "" {
			token, err := GenerateBoundServiceAccountToken(ctx, credential.ServiceAccountName, vh.namespace, vh.acs)
			return []byte(token), err
		}
		path = credential.ServiceAccount
	}

	var allowed []string
	if !legacy {
		allowed = globalConfig.serviceAccountTokenAllowedAudiences()
		if len(allowed) == 0 {
			return nil, fmt.Errorf("no approved service account token audiences; configure %s", ServiceAccountTokenAudiencesEnvVar)
		}
	}
	if path == "" {
		return nil, errors.New("k8s SA file not in config or serviceAccountName not supplied")
	}
	token, err := readKubernetesServiceAccountProjectedToken(path)
	if err != nil {
		return nil, err
	}
	if legacy {
		return token, nil
	}
	if err := validateK8sSATokenAudiences(token, allowed); err != nil {
		return nil, err
	}
	return token, nil
}

// validateAddress checks the Vault destination before accessing credentials.
func (vh *HashicorpVaultHandler) validateAddress(logger logr.Logger) error {
	policy := globalConfig.OutboundEndpointPolicy
	switch policy {
	case "", outboundPolicyOff:
		return nil
	case outboundPolicyWarn, outboundPolicyEnforce:
	default:
		return fmt.Errorf("unsupported %s.hashiCorpVault.mode %q", OutboundFilterEnvVar, policy)
	}
	allowed := globalConfig.AllowedOutboundEndpoints
	for _, endpoint := range allowed {
		if vaultAddressesEqual(endpoint, vh.vault.Address) {
			return nil
		}
	}
	if policy == outboundPolicyWarn {
		logger.Info("Warning: Vault endpoint is outside the allowlist; credentials may be sent to an untrusted destination. Configure KEDA_OUTBOUND_FILTER.hashiCorpVault with mode enforce and trusted allowedEndpoints", "vaultOrigin", vh.origin())
		return nil
	}
	if len(allowed) == 0 {
		return errors.New("no outbound endpoint allowlist is configured in enforce mode (KEDA_OUTBOUND_FILTER.hashiCorpVault.allowedEndpoints)")
	}
	return fmt.Errorf("vault endpoint %q is not in the configured allowlist (KEDA_OUTBOUND_FILTER.hashiCorpVault.allowedEndpoints)", vh.origin())
}

func (vh *HashicorpVaultHandler) origin() string {
	address, err := url.Parse(vh.vault.Address)
	if err != nil {
		return "<invalid endpoint>"
	}
	return (&url.URL{Scheme: address.Scheme, Host: address.Host}).String()
}

// vaultAddressesEqual compares two Vault addresses by scheme+host+port, ignoring path and trailing
// slashes, to avoid trivial allowlist bypasses via string prefixing.
func vaultAddressesEqual(a, b string) bool {
	ua, erra := url.Parse(strings.TrimSpace(a))
	ub, errb := url.Parse(strings.TrimSpace(b))
	if erra != nil || errb != nil {
		return false
	}
	if ua.User != nil || ub.User != nil || ua.Hostname() == "" || ub.Hostname() == "" {
		return false
	}
	if ua.Scheme != "http" && ua.Scheme != "https" || ub.Scheme != "http" && ub.Scheme != "https" {
		return false
	}
	return strings.EqualFold(ua.Scheme, ub.Scheme) && strings.EqualFold(ua.Host, ub.Host)
}

// renewToken takes charge of renewing the vault token
func (vh *HashicorpVaultHandler) renewToken(logger logr.Logger) {
	secret, err := vh.client.Auth().Token().RenewSelf(0)
	if err != nil {
		logger.Error(err, "Vault renew token: failed to create the payload")
		return
	}

	renewer, err := vh.client.NewLifetimeWatcher(&vaultapi.RenewerInput{
		Secret: secret,
		//Grace: time.Duration(15 * time.Second),
		//Increment: 60,
	})
	if err != nil {
		logger.Error(err, "Vault renew token: cannot create the renewer")
		return
	}

	go renewer.Renew()
	defer renewer.Stop()

RenewWatcherLoop:
	for {
		select {
		case <-vh.stopCh:
			break RenewWatcherLoop
		case err := <-renewer.DoneCh():
			if err != nil {
				logger.Error(err, "error renewing token")
			}
			break RenewWatcherLoop
		}
	}
}

// Read is used to get a secret from vault Read api. (e.g., secret)
func (vh *HashicorpVaultHandler) Read(path string) (*vaultapi.Secret, error) {
	return vh.client.Logical().Read(path)
}

// Write is used to get a secret from vault that needs to pass along data and uses the vault Write api. (e.g., pki)
func (vh *HashicorpVaultHandler) Write(path string, data map[string]any) (*vaultapi.Secret, error) {
	return vh.client.Logical().Write(path, data)
}

// Stop is responsible for stopping the renewal token process
func (vh *HashicorpVaultHandler) Stop() {
	if vh.stopCh != nil {
		close(vh.stopCh)
	}
}

// getPkiRequest format the pkiData in a format that the vault sdk understands.
func (vh *HashicorpVaultHandler) getPkiRequest(pkiData *kedav1alpha1.VaultPkiData) map[string]any {
	data := make(map[string]any)
	if pkiData.CommonName != "" {
		data["common_name"] = pkiData.CommonName
	}
	if pkiData.AltNames != "" {
		data["alt_names"] = pkiData.AltNames
	}
	if pkiData.IPSans != "" {
		data["ip_sans"] = pkiData.IPSans
	}
	if pkiData.URISans != "" {
		data["uri_sans"] = pkiData.URISans
	}
	if pkiData.OtherSans != "" {
		data["other_sans"] = pkiData.OtherSans
	}
	if pkiData.TTL != "" {
		data["ttl"] = pkiData.TTL
	}
	if pkiData.Format != "" {
		data["format"] = pkiData.Format
	}

	return data
}

// getSecretValue extract the secret value from the vault api response. As the vault api returns us a map[string]interface{},
// specific handling might be needed for some secret type.
func (vh *HashicorpVaultHandler) getSecretValue(secret *kedav1alpha1.VaultSecret, vaultSecret *vaultapi.Secret) (string, error) {
	if secret.Type == kedav1alpha1.VaultSecretTypeGeneric {
		if _, ok := vaultSecret.Data["data"]; ok {
			// Probably a v2 secret
			secret.Type = kedav1alpha1.VaultSecretTypeSecretV2
		} else {
			secret.Type = kedav1alpha1.VaultSecretTypeSecret
		}
	}
	switch secret.Type {
	case kedav1alpha1.VaultSecretTypePki:
		if vData, ok := vaultSecret.Data[secret.Key]; ok {
			if secret.Key == "ca_chain" {
				// Cast the secret to []interface{}
				if ai, ok := vData.([]any); ok {
					// Cast the secret to []string
					stringSlice := make([]string, len(ai))
					for i, v := range ai {
						stringSlice[i] = v.(string)
					}
					return strings.Join(stringSlice, "\n"), nil
				}
				err := fmt.Errorf("key '%s' is not castable to []interface{}", secret.Key)
				return "", err
			}
			if s, ok := vData.(string); ok {
				return s, nil
			}
			// If this happens, bad data from vault
			err := fmt.Errorf("key '%s' is not castable to string", secret.Key)
			return "", err
		}
		err := fmt.Errorf("key '%s' not found", secret.Key)
		return "", err
	case kedav1alpha1.VaultSecretTypeSecret:
		if vData, ok := vaultSecret.Data[secret.Key]; ok {
			if s, ok := vData.(string); ok {
				return s, nil
			}
			err := fmt.Errorf("key '%s' is not castable to string", secret.Key)
			return "", err
		}
		err := fmt.Errorf("key '%s' not found", secret.Key)
		return "", err
	case kedav1alpha1.VaultSecretTypeSecretV2:
		if v2Data, ok := vaultSecret.Data["data"].(map[string]any); ok {
			if value, ok := v2Data[secret.Key]; ok {
				if s, ok := value.(string); ok {
					return s, nil
				}
				err := fmt.Errorf("key '%s' is not castable to string", secret.Key)
				return "", err
			}
			err := fmt.Errorf("key '%s' not found", secret.Key)
			return "", err
		}
		// Unreachable
		return "", nil
	default:
		err := fmt.Errorf("unsupported vault secret type %s", secret.Type)
		return "", err
	}
}

// SecretGroup is used to group secret together by path, secretType and vaultPkiData.
type SecretGroup struct {
	path         string
	secretType   kedav1alpha1.VaultSecretType
	vaultPkiData kedav1alpha1.VaultPkiData
}

// fetchSecret returns the vaultSecret at a given vault path. If the secret is a pki, then the secret will use the
// vault Write method and will send the pkiData along
func (vh *HashicorpVaultHandler) fetchSecret(secretType kedav1alpha1.VaultSecretType, path string, vaultPkiData *kedav1alpha1.VaultPkiData) (*vaultapi.Secret, error) {
	var vaultSecret *vaultapi.Secret
	var err error
	switch secretType {
	case kedav1alpha1.VaultSecretTypePki:
		data := vh.getPkiRequest(vaultPkiData)
		vaultSecret, err = vh.Write(path, data)
		if err != nil {
			return nil, err
		}
	case kedav1alpha1.VaultSecretTypeSecret, kedav1alpha1.VaultSecretTypeSecretV2, kedav1alpha1.VaultSecretTypeGeneric:
		vaultSecret, err = vh.Read(path)
		if err != nil {
			return nil, err
		}
	default:
		err = fmt.Errorf("unsupported vault secret type %s", secretType)
		return nil, err
	}
	return vaultSecret, nil
}

// ResolveSecrets allows resolving a slice of secrets by vault. The function returns the list of secrets with the value updated.
// If multiple secrets refer to the same SecretGroup, the secret will be fetched only once.
func (vh *HashicorpVaultHandler) ResolveSecrets(secrets []kedav1alpha1.VaultSecret) ([]kedav1alpha1.VaultSecret, error) {
	// Group secret by path and type, this allows to fetch a path only once. This is useful for dynamic credentials
	grouped := make(map[SecretGroup][]kedav1alpha1.VaultSecret)
	vaultSecrets := make(map[SecretGroup]*vaultapi.Secret)
	for _, e := range secrets {
		group := SecretGroup{secretType: e.Type, path: e.Path, vaultPkiData: e.PkiData}
		if _, ok := grouped[group]; !ok {
			grouped[group] = make([]kedav1alpha1.VaultSecret, 0)
		}
		grouped[group] = append(grouped[group], e)
	}
	// For each group fetch the secret from vault
	for group := range grouped {
		vaultSecret, err := vh.fetchSecret(group.secretType, group.path, &group.vaultPkiData)
		if err != nil {
			// could not fetch secret, skipping group
			continue
		}
		vaultSecrets[group] = vaultSecret
	}
	// For each secret in each group, fetch the value and add to out
	out := make([]kedav1alpha1.VaultSecret, 0)
	for group, unFetchedSecrets := range grouped {
		vaultSecret := vaultSecrets[group]
		for _, secret := range unFetchedSecrets {
			if vaultSecret == nil {
				// This happens if we were not able to fetch the secret from vault
				secret.Value = ""
			} else {
				value, err := vh.getSecretValue(&secret, vaultSecret)
				if err != nil {
					secret.Value = ""
				} else {
					secret.Value = value
				}
			}
			out = append(out, secret)
		}
	}
	return out, nil
}
