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
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/metricsservice/api"
)

func externalMetricsToProto(in *external_metrics.ExternalMetricValueList) *api.ExternalMetricValueList {
	if in == nil {
		return &api.ExternalMetricValueList{}
	}

	out := &api.ExternalMetricValueList{
		Metadata: listMetaToProto(in.ListMeta),
		Items:    make([]*api.ExternalMetricValue, 0, len(in.Items)),
	}
	for i := range in.Items {
		item := in.Items[i]
		out.Items = append(out.Items, &api.ExternalMetricValue{
			MetricName:   item.MetricName,
			MetricLabels: copyStringMap(item.MetricLabels),
			Timestamp:    timeToProto(item.Timestamp),
			Window:       copyInt64Pointer(item.WindowSeconds),
			Value:        quantityToProto(item.Value),
		})
	}

	return out
}

func protoToExternalMetrics(in *api.ExternalMetricValueList) (*external_metrics.ExternalMetricValueList, error) {
	if in == nil {
		return &external_metrics.ExternalMetricValueList{}, nil
	}

	out := &external_metrics.ExternalMetricValueList{
		ListMeta: listMetaFromProto(in.GetMetadata()),
		Items:    make([]external_metrics.ExternalMetricValue, 0, len(in.GetItems())),
	}
	for _, item := range in.GetItems() {
		if item == nil {
			continue
		}

		value, err := quantityFromProto(item.GetValue())
		if err != nil {
			return nil, fmt.Errorf("error parsing metric value %q for %q: %w", item.GetValue().GetString_(), item.GetMetricName(), err)
		}

		out.Items = append(out.Items, external_metrics.ExternalMetricValue{
			MetricName:    item.GetMetricName(),
			MetricLabels:  copyStringMap(item.GetMetricLabels()),
			Timestamp:     timeFromProto(item.GetTimestamp()),
			WindowSeconds: copyInt64Pointer(item.Window),
			Value:         value,
		})
	}

	return out, nil
}

func listMetaToProto(in metav1.ListMeta) *api.ListMeta {
	return &api.ListMeta{
		ResourceVersion:    in.ResourceVersion,
		Continue:           in.Continue,
		RemainingItemCount: copyInt64Pointer(in.RemainingItemCount),
	}
}

func listMetaFromProto(in *api.ListMeta) metav1.ListMeta {
	if in == nil {
		return metav1.ListMeta{}
	}

	return metav1.ListMeta{
		SelfLink:           in.GetSelfLink(),
		ResourceVersion:    in.GetResourceVersion(),
		Continue:           in.GetContinue(),
		RemainingItemCount: copyInt64Pointer(in.RemainingItemCount),
	}
}

func timeToProto(in metav1.Time) *api.Time {
	if in.IsZero() {
		return nil
	}

	// Kubernetes metav1.Time protobuf serialization stores seconds precision.
	return &api.Time{Seconds: in.Unix()}
}

func timeFromProto(in *api.Time) metav1.Time {
	if in == nil || (in.GetSeconds() == 0 && in.GetNanos() == 0) {
		return metav1.Time{}
	}

	return metav1.NewTime(time.Unix(in.GetSeconds(), 0).UTC())
}

func quantityToProto(in resource.Quantity) *api.Quantity {
	return &api.Quantity{String_: in.String()}
}

func quantityFromProto(in *api.Quantity) (resource.Quantity, error) {
	if in == nil || in.GetString_() == "" {
		return resource.Quantity{}, nil
	}

	return resource.ParseQuantity(in.GetString_())
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}

	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func copyInt64Pointer(in *int64) *int64 {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}
