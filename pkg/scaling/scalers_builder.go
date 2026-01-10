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

package scaling

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/common/message"
	"github.com/kedacore/keda/v2/pkg/eventreason"
	"github.com/kedacore/keda/v2/pkg/scalers"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	"github.com/kedacore/keda/v2/pkg/scaling/cache"
	"github.com/kedacore/keda/v2/pkg/scaling/resolver"
)

/// --------------------------------------------------------------------------- ///
/// ----------            Scaler-Building related methods             --------- ///
/// --------------------------------------------------------------------------- ///

// buildScalers returns list of Scalers for the specified triggers
func (h *scaleHandler) buildScalers(ctx context.Context, withTriggers *kedav1alpha1.WithTriggers, podTemplateSpec *corev1.PodTemplateSpec, containerName string, asMetricSource bool) ([]cache.ScalerBuilder, error) {
	logger := log.WithValues("type", withTriggers.Kind, "namespace", withTriggers.Namespace, "name", withTriggers.Name)
	var err error
	resolvedEnv := make(map[string]string)
	result := make([]cache.ScalerBuilder, 0, len(withTriggers.Spec.Triggers))

	for i, t := range withTriggers.Spec.Triggers {
		triggerIndex, trigger := i, t

		factory := func() (scalers.Scaler, *scalersconfig.ScalerConfig, error) {
			if podTemplateSpec != nil {
				resolvedEnv, err = resolver.ResolveContainerEnv(ctx, h.client, logger, &podTemplateSpec.Spec, containerName, withTriggers.Namespace, h.authClientSet.SecretLister)
				if err != nil {
					return nil, nil, fmt.Errorf("error resolving secrets for ScaleTarget: %w", err)
				}
			}
			config := &scalersconfig.ScalerConfig{
				ScalableObjectName:      withTriggers.Name,
				ScalableObjectNamespace: withTriggers.Namespace,
				ScalableObjectType:      withTriggers.Kind,
				TriggerName:             trigger.Name,
				TriggerMetadata:         trigger.Metadata,
				TriggerType:             trigger.Type,
				TriggerUseCachedMetrics: trigger.UseCachedMetrics,
				ResolvedEnv:             resolvedEnv,
				AuthParams:              make(map[string]string),
				GlobalHTTPTimeout:       h.globalHTTPTimeout,
				TriggerIndex:            triggerIndex,
				MetricType:              trigger.MetricType,
				AsMetricSource:          asMetricSource,
				ScaledObject:            withTriggers,
				Recorder:                h.recorder,
				TriggerUniqueKey:        fmt.Sprintf("%s-%s-%s-%d", withTriggers.Kind, withTriggers.Namespace, withTriggers.Name, triggerIndex),
			}

			authParams, podIdentity, err := resolver.ResolveAuthRefAndPodIdentity(ctx, h.client, logger, trigger.AuthenticationRef, podTemplateSpec, withTriggers.Namespace, h.authClientSet)
			switch podIdentity.Provider {
			case kedav1alpha1.PodIdentityProviderAwsEKS:
				// FIXME: Delete this for v3
				logger.Info("WARNING: AWS EKS Identity has been deprecated in favor of AWS Identity and will be removed from KEDA on v3")
			default:
			}

			if err != nil {
				return nil, nil, err
			}
			config.AuthParams = authParams
			config.PodIdentity = podIdentity
			scaler, err := buildScaler(ctx, h.client, trigger.Type, config)
			return scaler, config, err
		}

		// nosemgrep: invalid-usage-of-modified-variable
		scaler, config, err := factory()
		if err != nil {
			h.recorder.Event(withTriggers, corev1.EventTypeWarning, eventreason.KEDAScalerFailed, err.Error())
			logger.Error(err, "error resolving auth params", "triggerIndex", triggerIndex)
			if scaler != nil {
				if closeErr := scaler.Close(ctx); closeErr != nil {
					logger.Error(closeErr, "failed to close scaler")
				}
			}
			for _, builder := range result {
				if closeErr := builder.Scaler.Close(ctx); closeErr != nil {
					logger.Error(closeErr, "failed to close scaler")
				}
			}
			return nil, err
		}
		msg := fmt.Sprintf(message.ScalerIsBuiltMsg, trigger.Type)
		h.recorder.Event(withTriggers, corev1.EventTypeNormal, eventreason.KEDAScalersStarted, msg)

		result = append(result, cache.ScalerBuilder{
			Scaler:       scaler,
			ScalerConfig: *config,
			Factory:      factory,
		})
	}

	return result, nil
}

// buildScaler builds a scaler from input config and trigger type
func buildScaler(ctx context.Context, client client.Client, triggerType string, config *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
	builder, ok := scalerBuilders[triggerType]
	if !ok {
		return nil, fmt.Errorf("no scaler found for type: %s", triggerType)
	}
	return builder(ctx, client, config)
}
