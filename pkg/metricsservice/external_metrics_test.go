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

package metricsservice

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"k8s.io/metrics/pkg/apis/external_metrics/v1beta1"

	"github.com/kedacore/keda/v2/pkg/metricsservice/api"
)

func TestExternalMetricsProtoRoundTrip(t *testing.T) {
	window := int64(30)
	remaining := int64(2)
	input := &external_metrics.ExternalMetricValueList{
		ListMeta: metav1.ListMeta{
			ResourceVersion:    "12345",
			Continue:           "next-page",
			RemainingItemCount: &remaining,
		},
		Items: []external_metrics.ExternalMetricValue{
			{
				MetricName:    "queue-length",
				MetricLabels:  map[string]string{"queue": "jobs", "region": "eu"},
				Timestamp:     metav1.NewTime(time.Unix(1716900000, 789000000).UTC()),
				WindowSeconds: &window,
				Value:         resource.MustParse("1500m"),
			},
			{
				MetricName:   "queue-depth",
				MetricLabels: nil,
				Value:        resource.MustParse("42"),
			},
		},
	}

	output, err := protoToExternalMetrics(externalMetricsToProto(input))
	require.NoError(t, err)

	require.Equal(t, input.ResourceVersion, output.ResourceVersion)
	require.Equal(t, input.Continue, output.Continue)
	require.Equal(t, input.RemainingItemCount, output.RemainingItemCount)
	require.Len(t, output.Items, len(input.Items))
	require.Equal(t, input.Items[0].MetricName, output.Items[0].MetricName)
	require.Equal(t, input.Items[0].MetricLabels, output.Items[0].MetricLabels)
	require.Equal(t, input.Items[0].Timestamp.Unix(), output.Items[0].Timestamp.Unix())
	require.Equal(t, int64(0), int64(output.Items[0].Timestamp.Nanosecond()))
	require.Equal(t, input.Items[0].WindowSeconds, output.Items[0].WindowSeconds)
	require.Equal(t, input.Items[0].Value.String(), output.Items[0].Value.String())
	require.Equal(t, input.Items[1].MetricName, output.Items[1].MetricName)
	require.Nil(t, output.Items[1].MetricLabels)
	require.Nil(t, output.Items[1].WindowSeconds)
	require.Equal(t, input.Items[1].Value.String(), output.Items[1].Value.String())
}

func TestExternalMetricsProtoMalformedQuantity(t *testing.T) {
	input := &api.ExternalMetricValueList{
		Items: []*api.ExternalMetricValue{
			{
				MetricName: "bad-value",
				Value:      &api.Quantity{String_: "not-a-quantity"},
			},
		},
	}

	_, err := protoToExternalMetrics(input)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not-a-quantity")
}

func TestExternalMetricsProtoWireCompatibleWithKubernetesExternalMetrics(t *testing.T) {
	window := int64(45)
	remaining := int64(3)
	timestamp := metav1.NewTime(time.Unix(1716900100, 0).UTC())
	kubernetesList := &v1beta1.ExternalMetricValueList{
		ListMeta: metav1.ListMeta{
			ResourceVersion:    "67890",
			Continue:           "next",
			RemainingItemCount: &remaining,
		},
		Items: []v1beta1.ExternalMetricValue{
			{
				MetricName:    "requests",
				MetricLabels:  map[string]string{"route": "checkout"},
				Timestamp:     timestamp,
				WindowSeconds: &window,
				Value:         resource.MustParse("42"),
			},
		},
	}

	kubernetesData, err := kubernetesList.Marshal()
	require.NoError(t, err)
	decodedAPIList := &api.ExternalMetricValueList{}
	require.NoError(t, proto.Unmarshal(kubernetesData, decodedAPIList))
	require.Equal(t, kubernetesList.ResourceVersion, decodedAPIList.GetMetadata().GetResourceVersion())
	require.Equal(t, kubernetesList.Continue, decodedAPIList.GetMetadata().GetContinue())
	require.Equal(t, *kubernetesList.RemainingItemCount, decodedAPIList.GetMetadata().GetRemainingItemCount())
	require.Equal(t, kubernetesList.Items[0].MetricName, decodedAPIList.GetItems()[0].GetMetricName())
	require.Equal(t, kubernetesList.Items[0].MetricLabels, decodedAPIList.GetItems()[0].GetMetricLabels())
	require.Equal(t, kubernetesList.Items[0].Timestamp.Unix(), decodedAPIList.GetItems()[0].GetTimestamp().GetSeconds())
	require.Equal(t, *kubernetesList.Items[0].WindowSeconds, decodedAPIList.GetItems()[0].GetWindow())
	require.Equal(t, kubernetesList.Items[0].Value.String(), decodedAPIList.GetItems()[0].GetValue().GetString_())

	apiData, err := proto.Marshal(decodedAPIList)
	require.NoError(t, err)
	decodedKubernetesList := &v1beta1.ExternalMetricValueList{}
	require.NoError(t, decodedKubernetesList.Unmarshal(apiData))
	require.Equal(t, kubernetesList.ResourceVersion, decodedKubernetesList.ResourceVersion)
	require.Equal(t, kubernetesList.Continue, decodedKubernetesList.Continue)
	require.Equal(t, kubernetesList.RemainingItemCount, decodedKubernetesList.RemainingItemCount)
	require.Equal(t, kubernetesList.Items[0].MetricName, decodedKubernetesList.Items[0].MetricName)
	require.Equal(t, kubernetesList.Items[0].MetricLabels, decodedKubernetesList.Items[0].MetricLabels)
	require.Equal(t, kubernetesList.Items[0].Timestamp.Unix(), decodedKubernetesList.Items[0].Timestamp.Unix())
	require.Equal(t, *kubernetesList.Items[0].WindowSeconds, *decodedKubernetesList.Items[0].WindowSeconds)
	require.Equal(t, kubernetesList.Items[0].Value.String(), decodedKubernetesList.Items[0].Value.String())
}
