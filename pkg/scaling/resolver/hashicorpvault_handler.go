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
	"strings"

	"github.com/go-logr/logr"
	vaultapi "github.com/hashicorp/vault/api"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
)

const (
	serviceAccountTokenFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"
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

	token, err := vh.token(client)
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

	if renew, ok := lookup.Data["renewable"].(bool); ok && renew {
		vh.stopCh = make(chan struct{})
		go vh.renewToken(logger)
	}

	vh.client = client

	return nil
}

// token Extract a vault token from the Authentication method
func (vh *HashicorpVaultHandler) token(client *vaultapi.Client) (string, error) {
	var token string
	var jwt []byte
	var err error

	switch vh.vault.Authentication {
	case kedav1alpha1.VaultAuthenticationToken:
		// Got token from VAULT_TOKEN env variable
		switch {
		case len(client.Token()) > 0:
			break
		case len(vh.vault.Credential.Token) > 0:
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

		if vh.vault.Credential == nil {
			defaultCred := kedav1alpha1.Credential{
				ServiceAccount: serviceAccountTokenFile,
			}
			vh.vault.Credential = &defaultCred
		}

		if vh.vault.Credential.ServiceAccountName == "" && vh.vault.Credential.ServiceAccount == "" {
			return token, errors.New("k8s SA file not in config or serviceAccountName not supplied")
		}

		if vh.vault.Credential.ServiceAccountName != "" {
			jwt = []byte(GenerateBoundServiceAccountToken(context.Background(), vh.vault.Credential.ServiceAccountName, vh.namespace, vh.acs))
		} else if len(vh.vault.Credential.ServiceAccount) != 0 {
			// Get the JWT from POD
			jwt, err = readKubernetesServiceAccountProjectedToken(vh.vault.Credential.ServiceAccount)
			if err != nil {
				return token, err
			}
		}

		data := map[string]interface{}{"jwt": string(jwt), "role": vh.vault.Role}
		secret, err := client.Logical().Write(fmt.Sprintf("auth/%s/login", vh.vault.Mount), data)
		if err != nil {
			return token, err
		}
		token = secret.Auth.ClientToken

	default:
		return token, fmt.Errorf("vault auth method %s is not supported", vh.vault.Authentication)
	}

	return token, nil
}

// renewToken takes charge of renewing the vault token
func (vh *HashicorpVaultHandler) renewToken(logger logr.Logger) {
	secret, err := vh.client.Auth().Token().RenewSelf(0)
	if err != nil {
		logger.Error(err, "Vault renew token: failed to create the payload")
	}

	renewer, err := vh.client.NewLifetimeWatcher(&vaultapi.RenewerInput{
		Secret: secret,
		//Grace: time.Duration(15 * time.Second),
		//Increment: 60,
	})
	if err != nil {
		logger.Error(err, "Vault renew token: cannot create the renewer")
	}

	go renewer.Renew()
	defer func() {
		renewer.Stop()
		close(vh.stopCh)
	}()

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
func (vh *HashicorpVaultHandler) Write(path string, data map[string]interface{}) (*vaultapi.Secret, error) {
	return vh.client.Logical().Write(path, data)
}

// Stop is responsible for stopping the renewal token process
func (vh *HashicorpVaultHandler) Stop() {
	if vh.stopCh != nil {
		vh.stopCh <- struct{}{}
	}
}

// getPkiRequest format the pkiData in a format that the vault sdk understands.
func (vh *HashicorpVaultHandler) getPkiRequest(pkiData *kedav1alpha1.VaultPkiData) map[string]interface{} {
	data := make(map[string]interface{})
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
				if ai, ok := vData.([]interface{}); ok {
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
		if v2Data, ok := vaultSecret.Data["data"].(map[string]interface{}); ok {
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
