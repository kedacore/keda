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

package gcp

import (
	"context"
	"errors"
	"net"
	"testing"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

func TestGetGCPAuthorization(t *testing.T) {
	tests := []struct {
		name    string
		config  *scalersconfig.ScalerConfig
		wantErr error
		check   func(t *testing.T, meta *AuthorizationMetadata)
	}{
		{
			name: "PodIdentity GCP enables PodIdentityProviderEnabled",
			config: &scalersconfig.ScalerConfig{
				PodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderGCP},
			},
			check: func(t *testing.T, meta *AuthorizationMetadata) {
				assert.True(t, meta.PodIdentityProviderEnabled)
				assert.Empty(t, meta.GoogleApplicationCredentials)
				assert.Empty(t, meta.GoogleApplicationCredentialsFile)
			},
		},
		{
			name: "GoogleApplicationCredentials in AuthParams sets credentials",
			config: &scalersconfig.ScalerConfig{
				PodIdentity: kedav1alpha1.AuthPodIdentity{},
				AuthParams:  map[string]string{"GoogleApplicationCredentials": `{"type":"service_account"}`},
			},
			check: func(t *testing.T, meta *AuthorizationMetadata) {
				assert.False(t, meta.PodIdentityProviderEnabled)
				assert.Equal(t, `{"type":"service_account"}`, meta.GoogleApplicationCredentials)
			},
		},
		{
			name: "credentialsFromEnv in TriggerMetadata resolves from ResolvedEnv",
			config: &scalersconfig.ScalerConfig{
				PodIdentity:     kedav1alpha1.AuthPodIdentity{},
				TriggerMetadata: map[string]string{"credentialsFromEnv": "GCP_CREDS"},
				ResolvedEnv:     map[string]string{"GCP_CREDS": `{"type":"service_account","resolved":true}`},
			},
			check: func(t *testing.T, meta *AuthorizationMetadata) {
				assert.Equal(t, `{"type":"service_account","resolved":true}`, meta.GoogleApplicationCredentials)
				assert.Empty(t, meta.GoogleApplicationCredentialsFile)
			},
		},
		{
			name: "credentialsFromEnvFile in TriggerMetadata sets GoogleApplicationCredentialsFile",
			config: &scalersconfig.ScalerConfig{
				PodIdentity:     kedav1alpha1.AuthPodIdentity{},
				TriggerMetadata: map[string]string{"credentialsFromEnvFile": "GCP_CREDS_FILE"},
				ResolvedEnv:     map[string]string{"GCP_CREDS_FILE": "/var/secrets/gcp/key.json"},
			},
			check: func(t *testing.T, meta *AuthorizationMetadata) {
				assert.Equal(t, "/var/secrets/gcp/key.json", meta.GoogleApplicationCredentialsFile)
				assert.Empty(t, meta.GoogleApplicationCredentials)
			},
		},
		{
			name: "PodIdentity GCP takes precedence over AuthParams credentials",
			config: &scalersconfig.ScalerConfig{
				PodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderGCP},
				AuthParams:  map[string]string{"GoogleApplicationCredentials": `{"type":"service_account"}`},
			},
			check: func(t *testing.T, meta *AuthorizationMetadata) {
				assert.True(t, meta.PodIdentityProviderEnabled)
				assert.Empty(t, meta.GoogleApplicationCredentials)
			},
		},
		{
			name: "AuthParams credentials take precedence over credentialsFromEnv",
			config: &scalersconfig.ScalerConfig{
				PodIdentity:     kedav1alpha1.AuthPodIdentity{},
				AuthParams:      map[string]string{"GoogleApplicationCredentials": `{"source":"authParams"}`},
				TriggerMetadata: map[string]string{"credentialsFromEnv": "GCP_CREDS"},
				ResolvedEnv:     map[string]string{"GCP_CREDS": `{"source":"env"}`},
			},
			check: func(t *testing.T, meta *AuthorizationMetadata) {
				assert.Equal(t, `{"source":"authParams"}`, meta.GoogleApplicationCredentials)
			},
		},
		{
			name: "credentialsFromEnv takes precedence over credentialsFromEnvFile",
			config: &scalersconfig.ScalerConfig{
				PodIdentity: kedav1alpha1.AuthPodIdentity{},
				TriggerMetadata: map[string]string{
					"credentialsFromEnv":     "GCP_CREDS",
					"credentialsFromEnvFile": "GCP_CREDS_FILE",
				},
				ResolvedEnv: map[string]string{
					"GCP_CREDS":      `{"type":"service_account"}`,
					"GCP_CREDS_FILE": "/var/secrets/gcp/key.json",
				},
			},
			check: func(t *testing.T, meta *AuthorizationMetadata) {
				assert.Equal(t, `{"type":"service_account"}`, meta.GoogleApplicationCredentials)
				assert.Empty(t, meta.GoogleApplicationCredentialsFile)
			},
		},
		{
			name: "no credential source set returns error",
			config: &scalersconfig.ScalerConfig{
				PodIdentity:     kedav1alpha1.AuthPodIdentity{},
				TriggerMetadata: map[string]string{},
				AuthParams:      map[string]string{},
				ResolvedEnv:     map[string]string{},
			},
			wantErr: ErrGoogleApplicationCrendentialsNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetGCPAuthorization(tt.config)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr))
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

type fakeMetricServer struct {
	monitoringpb.UnimplementedMetricServiceServer
}

func (fakeMetricServer) ListTimeSeries(context.Context, *monitoringpb.ListTimeSeriesRequest) (*monitoringpb.ListTimeSeriesResponse, error) {
	return &monitoringpb.ListTimeSeriesResponse{}, nil
}

func TestGRPCClientOptionsDisablesGoogleTelemetry(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	monitoringpb.RegisterMetricServiceServer(server, fakeMetricServer{})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)

	reader := sdkmetric.NewManualReader()
	previous := otel.GetMeterProvider()
	otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	t.Cleanup(func() { otel.SetMeterProvider(previous) })

	opts := append(GRPCClientOptions(),
		option.WithEndpoint(listener.Addr().String()),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	client, err := monitoring.NewMetricClient(t.Context(), opts...)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	_, err = client.ListTimeSeries(t.Context(), &monitoringpb.ListTimeSeriesRequest{Name: "projects/test"}).Next()
	require.ErrorIs(t, err, iterator.Done)

	var got metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &got))
	for _, scope := range got.ScopeMetrics {
		assert.NotEqual(t, otelgrpc.ScopeName, scope.Scope.Name, "Google's default otelgrpc handler must be disabled")
	}
}
