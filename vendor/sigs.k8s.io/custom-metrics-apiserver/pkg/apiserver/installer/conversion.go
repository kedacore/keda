/*
Copyright 2020 The Kubernetes Authors.

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

package installer

import (
	"net/url"

	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	cmv1beta1 "k8s.io/metrics/pkg/apis/custom_metrics/v1beta1"
	cmv1beta2 "k8s.io/metrics/pkg/apis/custom_metrics/v1beta2"
)

func ConvertURLValuesToV1beta1MetricListOptions(in *url.Values, out *cmv1beta1.MetricListOptions, s conversion.Scope) error {
	if values, ok := map[string][]string(*in)["labelSelector"]; ok && len(values) > 0 {
		if err := runtime.Convert_Slice_string_To_string(&values, &out.LabelSelector, s); err != nil {
			return err
		}
	} else {
		out.LabelSelector = ""
	}
	if values, ok := map[string][]string(*in)["metricLabelSelector"]; ok && len(values) > 0 {
		if err := runtime.Convert_Slice_string_To_string(&values, &out.MetricLabelSelector, s); err != nil {
			return err
		}
	} else {
		out.MetricLabelSelector = ""
	}
	return nil
}

func ConvertURLValuesToV1beta2MetricListOptions(in *url.Values, out *cmv1beta2.MetricListOptions, s conversion.Scope) error {
	if values, ok := map[string][]string(*in)["labelSelector"]; ok && len(values) > 0 {
		if err := runtime.Convert_Slice_string_To_string(&values, &out.LabelSelector, s); err != nil {
			return err
		}
	} else {
		out.LabelSelector = ""
	}
	if values, ok := map[string][]string(*in)["metricLabelSelector"]; ok && len(values) > 0 {
		if err := runtime.Convert_Slice_string_To_string(&values, &out.MetricLabelSelector, s); err != nil {
			return err
		}
	} else {
		out.MetricLabelSelector = ""
	}
	return nil
}

// RegisterConversions adds conversion functions to the given scheme.
func RegisterConversions(s *runtime.Scheme) error {
	if err := s.AddConversionFunc((*url.Values)(nil), (*cmv1beta1.MetricListOptions)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return ConvertURLValuesToV1beta1MetricListOptions(a.(*url.Values), b.(*cmv1beta1.MetricListOptions), scope)
	}); err != nil {
		return err
	}
	if err := s.AddConversionFunc((*url.Values)(nil), (*cmv1beta2.MetricListOptions)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return ConvertURLValuesToV1beta2MetricListOptions(a.(*url.Values), b.(*cmv1beta2.MetricListOptions), scope)
	}); err != nil {
		return err
	}
	return nil
}
