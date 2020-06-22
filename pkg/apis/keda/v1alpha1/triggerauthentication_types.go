package v1alpha1

import (
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/go-logr/logr"
	vaultApi "github.com/hashicorp/vault/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TriggerAuthenticationSpec defines the various ways to authenticate
// +k8s:openapi-gen=true
type TriggerAuthenticationSpec struct {
	// +optional
	PodIdentity AuthPodIdentity `json:"podIdentity"`

	// +optional
	// +listType
	SecretTargetRef []AuthSecretTargetRef `json:"secretTargetRef"`

	// +optional
	// +listType
	Env []AuthEnvironment `json:"env"`

	// +optional
	Vault Vault `json:"vault"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TriggerAuthentication defines how a trigger can authenticate
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=triggerauthentications,scope=Namespaced
type TriggerAuthentication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec TriggerAuthenticationSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TriggerAuthenticationList contains a list of TriggerAuthentication
type TriggerAuthenticationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []TriggerAuthentication `json:"items"`
}

// PodIdentityProvider contains the list of providers
type PodIdentityProvider string

const (
	PodIdentityProviderNone    PodIdentityProvider = "none"
	PodIdentityProviderAzure                       = "azure"
	PodIdentityProviderGCP                         = "gcp"
	PodIdentityProviderSpiffe                      = "spiffe"
	PodIdentityProviderAwsEKS                      = "aws-eks"
	PodIdentityProviderAwsKiam                     = "aws-kiam"
)

const (
	PodIdentityAnnotationEKS  = "eks.amazonaws.com/role-arn"
	PodIdentityAnnotationKiam = "iam.amazonaws.com/role"
)

// AuthPodIdentity allows users to select the platform native identity
// mechanism
// +k8s:openapi-gen=true
type AuthPodIdentity struct {
	Provider PodIdentityProvider `json:"provider"`
}

// AuthSecretTargetRef is used to authenticate using a reference to a secret
// +k8s:openapi-gen=true
type AuthSecretTargetRef struct {
	Parameter string `json:"parameter"`
	Name      string `json:"name"`
	Key       string `json:"key"`
}

// AuthEnvironment is used to authenticate using environment variables
// in the destination deployment spec
// +k8s:openapi-gen=true
type AuthEnvironment struct {
	Parameter string `json:"parameter"`
	Name      string `json:"name"`

	// +optional
	ContainerName string `json:"containerName"`
}

// Vault is used to authenticate using Hashicorp Vault
// +k8s:openapi-gen=true
type Vault struct {
	Address        string              `json:"address"`
	Authentication VaultAuthentication `json:"authentication"`

	// +listType
	Secrets []VaultSecret `json:"secrets"`

	// +optional
	Credential Credential `json:"credetial"`

	// +optional
	Role string `json:"role"`

	// +optional
	Mount string `json:"mount"`
}

// Credential defines the Hashicorp Vault credentials depending on the authentication method
// +k8s:openapi-gen=true
type Credential struct {
	// +optional
	Token string `json:"token"`

	// +optional
	ServiceAccount string `json:"serviceAccount"`
}

// VaultAuthentication contains the list of Hashicorp Vault authentication methods
type VaultAuthentication string

// Client authenticating to Vault
const (
	VaultAuthenticationToken      VaultAuthentication = "token"
	VaultAuthenticationKubernetes                     = "kubernetes"
	// VaultAuthenticationAWS                            = "aws"
)

// VaultSecret defines the mapping between the path of the secret in Vault to the parameter
// +k8s:openapi-gen=true
type VaultSecret struct {
	Parameter string `json:"parameter"`
	Path      string `json:"path"`
	Key       string `json:"key"`
}

func init() {
	SchemeBuilder.Register(&TriggerAuthentication{}, &TriggerAuthenticationList{})
}

// Authenticate returns a client connected to Vault
func (v *Vault) Authenticate(logger logr.Logger) (*vaultApi.Client, error) {
	config := vaultApi.DefaultConfig()
	client, err := vaultApi.NewClient(config)

	err = client.SetAddress(v.Address)
	if err != nil {
		return nil, err
	}

	token, err := v.token(client)
	if err != nil {
		return nil, err
	}
	client.SetToken(token)

	lookup, err := client.Auth().Token().LookupSelf()
	//If token is not valid so get out of here early
	if err != nil {
		return nil, err
	}

	renew := lookup.Data["renewable"].(bool)
	if renew == true {
		go v.renewToken(client, logger)
	}

	return client, nil
}

func (v *Vault) token(client *vaultApi.Client) (string, error) {
	var token string

	switch v.Authentication {
	case VaultAuthenticationToken:
		// Got token from VAULT_TOKEN env variable
		if len(client.Token()) > 0 {
			break
		} else if len(v.Credential.Token) > 0 {
			token = v.Credential.Token
		} else {
			return token, errors.New("Could not get Vault token")
		}
	case VaultAuthenticationKubernetes:
		if len(v.Mount) == 0 {
			return token, errors.New("Auth mount not in config")
		}

		if len(v.Role) == 0 {
			return token, errors.New("K8s role not in config")
		}

		if len(v.Credential.ServiceAccount) == 0 {
			return token, errors.New("K8s SA file not in config")
		}

		//Get the JWT from POD
		jwt, err := ioutil.ReadFile(v.Credential.ServiceAccount)
		if err != nil {
			return token, err
		}

		data := map[string]interface{}{"jwt": string(jwt), "role": v.Role}
		secret, err := client.Logical().Write(fmt.Sprintf("auth/%s/login", v.Mount), data)
		if err != nil {
			return token, err
		}

		token = secret.Auth.ClientToken
	default:
		return token, fmt.Errorf("Vault auth method %s is not supported", v.Authentication)
	}

	return token, nil
}

func (v *Vault) renewToken(client *vaultApi.Client, logger logr.Logger) {
	secret, err := client.Auth().Token().RenewSelf(0)
	if err != nil {
		logger.Error(err, "Vault renew token: failed to create the payload")
	}

	renewer, err := client.NewRenewer(&vaultApi.RenewerInput{
		Secret: secret,
		//Grace:  time.Duration(15 * time.Second),
		//Increment: 60,
	})
	if err != nil {
		logger.Error(err, "Vault renew token: cannot create the renewer")
	}

	go renewer.Renew()
	defer renewer.Stop()

	for {
		select {
		case err := <-renewer.DoneCh():
			if err != nil {
				logger.Error(err, "error renewing token")
			}
		}
	}
}
