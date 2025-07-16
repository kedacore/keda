/*
Copyright 2023 The KEDA Authors

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
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"os"

	"cloud.google.com/go/compute/metadata"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/go-logr/logr"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

type GCPSecretManagerHandler struct {
	gcpSecretsManager       *kedav1alpha1.GCPSecretManager
	gcpSecretsManagerClient *secretmanager.Client
	gcpProjectID            string
}

// NewGCPSecretManagerHandler creates a GCPSecretManagerHandler object
func NewGCPSecretManagerHandler(v *kedav1alpha1.GCPSecretManager) *GCPSecretManagerHandler {
	return &GCPSecretManagerHandler{
		gcpSecretsManager: v,
	}
}

// Initialize the GCP Secret Manager client
func (vh *GCPSecretManagerHandler) Initialize(ctx context.Context, client client.Client, logger logr.Logger, triggerNamespace string, secretsLister corev1listers.SecretLister) error {
	var err error

	podIdentity := vh.gcpSecretsManager.PodIdentity
	if podIdentity == nil {
		podIdentity = &kedav1alpha1.AuthPodIdentity{}
	}

	switch podIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		missingErr := fmt.Errorf("clientSecret is expected when not using a pod identity provider")
		if vh.gcpSecretsManager.Credentials == nil {
			return missingErr
		}

		clientSecretName := vh.gcpSecretsManager.Credentials.ClientSecret.ValueFrom.SecretKeyRef.Name
		clientSecretKey := vh.gcpSecretsManager.Credentials.ClientSecret.ValueFrom.SecretKeyRef.Key
		clientSecret := resolveAuthSecret(ctx, client, logger, clientSecretName, triggerNamespace, clientSecretKey, secretsLister)

		if clientSecret == "" {
			return missingErr
		}

		gcpCredentials, err := google.CredentialsFromJSON(ctx, []byte(clientSecret), secretmanager.DefaultAuthScopes()...)
		if err != nil {
			return fmt.Errorf("failed to get credentials from json, %w", err)
		}

		vh.gcpSecretsManagerClient, err = secretmanager.NewClient(ctx, option.WithCredentials(gcpCredentials))
		if err != nil {
			return fmt.Errorf("failed to create secretmanager client, %w", err)
		}

		gcpCredentialsMap := make(map[string]interface{})
		if err := json.Unmarshal([]byte(clientSecret), &gcpCredentialsMap); err != nil {
			return fmt.Errorf("failed to unmarshal gcp credentials key into a map, %w", err)
		}

		project, ok := gcpCredentialsMap["project_id"]
		if !ok {
			return fmt.Errorf("project_id field is missing in the json credentials")
		}

		vh.gcpProjectID = project.(string)

	case kedav1alpha1.PodIdentityProviderGCP:
		if vh.gcpSecretsManagerClient, err = secretmanager.NewClient(ctx); err != nil {
			return fmt.Errorf("failed to create secretmanager client: %w", err)
		}

		// Running workload identity outside GKE, we can't use the metadata api
		// and we need to use the env that it's provided from the hook
		project, found := os.LookupEnv("CLOUDSDK_CORE_PROJECT")
		if !found {
			if project, err = metadata.NewClient(&http.Client{}).ProjectIDWithContext(ctx); err != nil {
				return fmt.Errorf("failed to fetch gcp project id: %w", err)
			}
		}

		vh.gcpProjectID = project

	default:
		return fmt.Errorf("gcp secret manager does not support pod identity provider - %v", podIdentity.Provider)
	}

	return nil
}

func (vh *GCPSecretManagerHandler) Read(ctx context.Context, secretID, secretVersion string) (string, error) {
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/%s", vh.gcpProjectID, secretID, secretVersion),
	}

	result, err := vh.gcpSecretsManagerClient.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to access the secret %s version %s, %w", secretID, secretVersion, err)
	}

	if result == nil || result.Payload == nil {
		return "", errors.New("received empty result payload upon fetching the secret version")
	}

	crc32c := crc32.MakeTable(crc32.Castagnoli)
	checksum := int64(crc32.Checksum(result.Payload.Data, crc32c))
	if result.Payload.DataCrc32C != nil && checksum != *result.Payload.DataCrc32C {
		return "", errors.New("secret payload data corruption detected")
	}

	return string(result.Payload.Data), nil
}
