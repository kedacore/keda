package handler

import (
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/go-logr/logr"
	vaultApi "github.com/hashicorp/vault/api"
	kedav1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
)

type HashicorpVaultHandler struct {
	vault  *kedav1alpha1.HashiCorpVault
	client *vaultApi.Client
	stopCh chan struct{}
}

func NewHashicorpVaultHandler(v *kedav1alpha1.HashiCorpVault) *HashicorpVaultHandler {
	return &HashicorpVaultHandler{
		vault: v,
	}
}

// Initialize the Vault client
func (vh *HashicorpVaultHandler) Initialize(logger logr.Logger) error {
	config := vaultApi.DefaultConfig()
	client, err := vaultApi.NewClient(config)

	err = client.SetAddress(vh.vault.Address)
	if err != nil {
		return err
	}

	token, err := vh.token(client)
	if err != nil {
		return err
	}

	if len(token) > 0 {
		client.SetToken(token)
	}

	lookup, err := client.Auth().Token().LookupSelf()
	//If token is not valid so get out of here early
	if err != nil {
		return err
	}

	renew := lookup.Data["renewable"].(bool)
	if renew == true {
		vh.stopCh = make(chan struct{})
		go vh.renewToken(logger)
	}

	vh.client = client

	return nil
}

func (vh *HashicorpVaultHandler) token(client *vaultApi.Client) (string, error) {
	var token string

	switch vh.vault.Authentication {
	case kedav1alpha1.VaultAuthenticationToken:
		// Got token from VAULT_TOKEN env variable
		if len(client.Token()) > 0 {
			break
		} else if len(vh.vault.Credential.Token) > 0 {
			token = vh.vault.Credential.Token
		} else {
			return token, errors.New("Could not get Vault token")
		}
	case kedav1alpha1.VaultAuthenticationKubernetes:
		if len(vh.vault.Mount) == 0 {
			return token, errors.New("Auth mount not in config")
		}

		if len(vh.vault.Role) == 0 {
			return token, errors.New("K8s role not in config")
		}

		if len(vh.vault.Credential.ServiceAccount) == 0 {
			return token, errors.New("K8s SA file not in config")
		}

		//Get the JWT from POD
		jwt, err := ioutil.ReadFile(vh.vault.Credential.ServiceAccount)
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
		return token, fmt.Errorf("Vault auth method %s is not supported", vh.vault.Authentication)
	}

	return token, nil
}

func (vh *HashicorpVaultHandler) renewToken(logger logr.Logger) {
	secret, err := vh.client.Auth().Token().RenewSelf(0)
	if err != nil {
		logger.Error(err, "Vault renew token: failed to create the payload")
	}

	renewer, err := vh.client.NewRenewer(&vaultApi.RenewerInput{
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

func (vh *HashicorpVaultHandler) Read(path string) (*vaultApi.Secret, error) {
	return vh.client.Logical().Read(path)
}

// Stop is responsible for stoping the renew token proccess
func (vh *HashicorpVaultHandler) Stop() {
	if vh.stopCh != nil {
		vh.stopCh <- struct{}{}
	}
}
