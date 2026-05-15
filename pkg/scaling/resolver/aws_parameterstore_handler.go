/*
Copyright 2024 The KEDA Authors

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
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	awsutils "github.com/kedacore/keda/v2/pkg/scalers/aws"
)

type AwsParameterStoreHandler struct {
	parameterStore *kedav1alpha1.AwsParameterStore
	session        *ssm.Client
	awsMetadata    awsutils.AuthorizationMetadata
}

func NewAwsParameterStoreHandler(a *kedav1alpha1.AwsParameterStore) *AwsParameterStoreHandler {
	return &AwsParameterStoreHandler{
		parameterStore: a,
	}
}

func (apsh *AwsParameterStoreHandler) Read(ctx context.Context, logger logr.Logger, parameterName string, withDecryption *bool) (string, error) {
	decrypt := true
	if withDecryption != nil {
		decrypt = *withDecryption
	}

	input := &ssm.GetParameterInput{
		Name:           aws.String(parameterName),
		WithDecryption: aws.Bool(decrypt),
	}

	result, err := apsh.session.GetParameter(ctx, input)
	if err != nil {
		logger.Error(err, "Error getting parameter from Parameter Store")
		return "", err
	}

	if result.Parameter == nil || result.Parameter.Value == nil {
		logger.Error(nil, "Parameter value is nil")
		return "", fmt.Errorf("parameter value is nil")
	}

	return *result.Parameter.Value, nil
}

func (apsh *AwsParameterStoreHandler) Initialize(ctx context.Context, client client.Client, logger logr.Logger, triggerNamespace string, secretsLister corev1listers.SecretLister, podSpec *corev1.PodSpec) error {
	metadata, err := initializeAwsMetadata(ctx, client, logger, triggerNamespace, secretsLister, podSpec,
		"aws-parameter-store", apsh.parameterStore.Region, apsh.parameterStore.PodIdentity, apsh.parameterStore.Credentials)
	if err != nil {
		return err
	}
	apsh.awsMetadata = metadata

	config, err := awsutils.GetAwsConfig(ctx, apsh.awsMetadata)
	if err != nil {
		logger.Error(err, "Error getting credentials")
		return err
	}
	apsh.session = ssm.NewFromConfig(*config)
	return nil
}

func (apsh *AwsParameterStoreHandler) Stop() {
	awsutils.ClearAwsConfig(apsh.awsMetadata)
}
