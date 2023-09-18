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

package modifiers

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/antonmedv/expr"
	"github.com/go-logr/logr"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scaling/cache"
)

// apply defined ScalingModifiers structure (formula) and simply return
// calculated metrics
func HandleScalingModifiers(so *kedav1alpha1.ScaledObject, metrics []external_metrics.ExternalMetricValue, metricTriggerList map[string]string, fallbackActive bool, cacheObj *cache.ScalersCache, log logr.Logger) []external_metrics.ExternalMetricValue {
	var err error
	if !fallbackActive && so != nil && so.Spec.Advanced != nil && !reflect.DeepEqual(so.Spec.Advanced.ScalingModifiers, kedav1alpha1.ScalingModifiers{}) {
		sm := so.Spec.Advanced.ScalingModifiers

		// apply formula if defined
		metrics, err = applyComplexLogicFormula(sm, metrics, metricTriggerList, cacheObj)
		if err != nil {
			log.Error(err, "error applying custom compositeScaler formula")
		}
		log.V(1).Info("returned metrics after formula is applied", "metrics", metrics)
	}
	return metrics
}

// help function to determine whether or not metricName is the correct one.
// standard function will be array of one element if it matches or none if it doesnt
// that is given from getTrueMetricArray().
// In case of compositeScaler, cycle through all external metric names
func ArrayContainsElement(el string, arr []string) bool {
	for _, item := range arr {
		if strings.EqualFold(item, el) {
			return true
		}
	}
	return false
}

// if given right conditions, try to apply the given custom formula in SO
func applyComplexLogicFormula(sm kedav1alpha1.ScalingModifiers, metrics []external_metrics.ExternalMetricValue, pairList map[string]string, cacheObj *cache.ScalersCache) ([]external_metrics.ExternalMetricValue, error) {
	if sm.Formula != "" {
		metrics, err := calculateComplexLogicFormula(metrics, cacheObj, pairList)
		return metrics, err
	}
	return metrics, nil
}

// calculate custom formula to metrics and return calculated and finalized metric
func calculateComplexLogicFormula(list []external_metrics.ExternalMetricValue, cacheObj *cache.ScalersCache, pairList map[string]string) ([]external_metrics.ExternalMetricValue, error) {
	var ret external_metrics.ExternalMetricValue
	var out float64
	ret.MetricName = "composite-metric-name"
	ret.Timestamp = v1.Now()

	// using https://github.com/antonmedv/expr to evaluate formula expression
	data := make(map[string]float64)
	for _, v := range list {
		data[pairList[v.MetricName]] = v.Value.AsApproximateFloat64()
	}

	if cacheObj.CompiledFormula == nil {
		return nil, fmt.Errorf("cached compiled formula is nil during its calculation")
	}

	tmp, err := expr.Run(cacheObj.CompiledFormula, data)
	if err != nil {
		return nil, fmt.Errorf("error trying to run custom formula: %w", err)
	}

	out = tmp.(float64)
	ret.Value.SetMilli(int64(out * 1000))
	return []external_metrics.ExternalMetricValue{ret}, nil
}

// Add pair trigger-metric to the triggers-metrics list for custom formula. Trigger name is used in
// formula itself (in SO) and metric name is used for its value internally.
func AddPairTriggerAndMetric(list map[string]string, so *kedav1alpha1.ScaledObject, metric string, trigger string) (map[string]string, error) {
	if so.Spec.Advanced != nil && so.Spec.Advanced.ScalingModifiers.Formula != "" {
		if trigger == "" {
			return list, fmt.Errorf("trigger name not given with compositeScaler for metric %s", metric)
		}

		triggerHasMetrics := 0
		// count number of metrics per trigger
		for _, t := range list {
			if strings.HasPrefix(t, trigger) {
				triggerHasMetrics++
			}
		}

		// if trigger doesnt have a pair yet
		if triggerHasMetrics == 0 {
			list[metric] = trigger
		} else {
			// if trigger has a pair add a number
			list[metric] = fmt.Sprintf("%s%02d", trigger, triggerHasMetrics)
		}

		return list, nil
	}
	return map[string]string{}, nil
}
