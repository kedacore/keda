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

// ******************************* DESCRIPTION ****************************** \\
// modifiers package describes functions that handle scaling modifiers. This
// file contains main functionality and supporting functions. The parent
// function is HandleScalingModifiers() that is called from scale_handler.
// If fallback is active or the struct scalingModifiers in SO is not defined,
// input metrics are simply returned without change, otherwise apply formula if
// conditions are met.
// ************************************************************************** \\

package modifiers

import (
	"fmt"
	"strings"

	"github.com/antonmedv/expr"
	"github.com/go-logr/logr"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scaling/cache"
)

// HandleScalingModifiers is the parent function for scalingModifiers structure.
// If the structure is defined and conditions are met, apply the formula to
// manipulate the metrics and return them
func HandleScalingModifiers(so *kedav1alpha1.ScaledObject, metrics []external_metrics.ExternalMetricValue, metricTriggerList map[string]string, fallbackActive bool, cacheObj *cache.ScalersCache, log logr.Logger) []external_metrics.ExternalMetricValue {
	var err error
	// dont manipulate with metrics if fallback is currently active or structure isnt defined
	if !fallbackActive && so != nil && so.IsUsingModifiers() {
		sm := so.Spec.Advanced.ScalingModifiers

		// apply formula if defined
		metrics, err = applyScalingModifiersFormula(sm, metrics, metricTriggerList, cacheObj)
		if err != nil {
			log.Error(err, "error applying custom scalingModifiers.Formula")
		}
		log.V(1).Info("returned metrics after formula is applied", "metrics", metrics)
	}
	return metrics
}

// ArrayContainsElement determines whether array 'arr' contains element 'el'
func ArrayContainsElement(el string, arr []string) bool {
	for _, item := range arr {
		if strings.EqualFold(item, el) {
			return true
		}
	}
	return false
}

// applyScalingModifiersFormula applies formula if formula is defined, otherwise
// skip
func applyScalingModifiersFormula(sm kedav1alpha1.ScalingModifiers, metrics []external_metrics.ExternalMetricValue, pairList map[string]string, cacheObj *cache.ScalersCache) ([]external_metrics.ExternalMetricValue, error) {
	if sm.Formula != "" {
		metrics, err := calculateScalingModifiersFormula(metrics, cacheObj, pairList)
		return metrics, err
	}
	return metrics, nil
}

// calculateScalingModifiersFormula creates custom composite metric & calculates
// custom formula and returns this finalized metric
func calculateScalingModifiersFormula(list []external_metrics.ExternalMetricValue, cacheObj *cache.ScalersCache, pairList map[string]string) ([]external_metrics.ExternalMetricValue, error) {
	var ret external_metrics.ExternalMetricValue
	var out float64
	ret.MetricName = kedav1alpha1.CompositeMetricName
	ret.Timestamp = v1.Now()

	// using https://github.com/antonmedv/expr to evaluate formula expression
	data := make(map[string]float64)
	for _, v := range list {
		data[pairList[v.MetricName]] = v.Value.AsApproximateFloat64()
	}

	if cacheObj.CompiledFormula == nil {
		return nil, fmt.Errorf("cached compiled formula is nil during its calculation")
	}

	// run expression with precompiled formula and real data
	tmp, err := expr.Run(cacheObj.CompiledFormula, data)
	if err != nil {
		return nil, fmt.Errorf("error trying to run custom formula: %w", err)
	}

	// return values to known format for externalMetricValue struct
	out = tmp.(float64)
	ret.Value.SetMilli(int64(out * 1000))
	return []external_metrics.ExternalMetricValue{ret}, nil
}

// AddPairTriggerAndMetric adds new pair of trigger-metric to the list for
// scalingModifiers formula list thats needed to map the metric value to
// trigger name. This is only ran if scalingModifiers.Formula is defined in SO.
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
