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
	"errors"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	vaultapi "github.com/hashicorp/vault/api"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

// HashicorpVaultHandler is specification of Hashi Corp Vault
type HashicorpVaultHandler struct {
	vault  *kedav1alpha1.HashiCorpVault
	client *vaultapi.Client
	stopCh chan struct{}
}

// NewHashicorpVaultHandler creates a HashicorpVaultHandler object
func NewHashicorpVaultHandler(v *kedav1alpha1.HashiCorpVault) *HashicorpVaultHandler {
	return &HashicorpVaultHandler{
		vault: v,
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

	renew := lookup.Data["renewable"].(bool)
	if renew {
		vh.stopCh = make(chan struct{})
		go vh.renewToken(logger)
	}

	vh.client = client

	return nil
}

func (vh *HashicorpVaultHandler) token(client *vaultapi.Client) (string, error) {
	var token string

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

		if len(vh.vault.Credential.ServiceAccount) == 0 {
			return token, errors.New("k8s SA file not in config")
		}

		// Get the JWT from POD
		jwt, err := os.ReadFile(vh.vault.Credential.ServiceAccount)
		if err != nil {
			return token, err
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

func (vh *HashicorpVaultHandler) renewToken(logger logr.Logger) {
	secret, err := vh.client.Auth().Token().RenewSelf(0)
	if err != nil {
		logger.Error(err, "Vault renew token: failed to create the payload")
	}

	renewer, err := vh.client.NewLifetimeWatcher(&vaultapi.RenewerInput{
		Secret: secret,
		//Grace:  time.Duration(15 * time.Second),
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

func (vh *HashicorpVaultHandler) Read(path string) (*vaultapi.Secret, error) {
	return vh.client.Logical().Read(path)
}

// Stop is responsible for stoping the renew token process
func (vh *HashicorpVaultHandler) Stop() {
	if vh.stopCh != nil {
		vh.stopCh <- struct{}{}
	}
}
