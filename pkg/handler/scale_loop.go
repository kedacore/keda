package handler

import (
	"context"
	"time"

	kedav1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
)

// HandleScaleLoop blocks forever and checks the scaledObject based on its pollingInterval
func (h *ScaleHandler) HandleScaleLoop(ctx context.Context, scaledObject *kedav1alpha1.ScaledObject) {
	h.logger = h.logger.WithValues("ScaledObject.Namespace", scaledObject.Namespace, "ScaledObject.Name", scaledObject.Name)

	h.handleScaledObject(ctx, scaledObject)

	var pollingInterval time.Duration
	if scaledObject.Spec.PollingInterval != nil {
		pollingInterval = time.Second * time.Duration(*scaledObject.Spec.PollingInterval)
	} else {
		pollingInterval = time.Second * time.Duration(defaultPollingInterval)
	}

	h.logger.V(1).Info("Watching scaledObject with pollingInterval", "ScaledObject.PollingInterval", pollingInterval)

	for {
		select {
		case <-time.After(pollingInterval):
			h.handleScaledObject(ctx, scaledObject)
		case <-ctx.Done():
			h.logger.V(1).Info("Context for scaledObject canceled")
			return
		}
	}
}

// TODO refactor
func (h *ScaleHandler) handleScaleJob(ctx context.Context, scaledObject *kedav1alpha1.ScaledObject) {
	//TODO: need to actually handle the scale here
	h.logger.V(1).Info("Handle Scale Job called")
	scalers, err := h.getJobScalers(scaledObject)

	if err != nil {
		h.logger.Error(err, "Error getting scalers")
		return
	}

	isScaledObjectActive := false
	h.logger.Info("Scalers count", "Count", len(scalers))
	var queueLength int64
	var maxValue int64

	for _, scaler := range scalers {
		scalerLogger := h.logger.WithValues("Scaler", scaler)

		isTriggerActive, err := scaler.IsActive(ctx)
		scalerLogger.Info("Active trigger", "isTriggerActive", isTriggerActive)
		metricSpecs := scaler.GetMetricSpecForScaling()

		var metricValue int64
		for _, metric := range metricSpecs {
			metricValue, _ = metric.External.Target.AverageValue.AsInt64()
			maxValue += metricValue
		}
		scalerLogger.Info("Scaler max value", "MaxValue", maxValue)

		metrics, _ := scaler.GetMetrics(ctx, "queueLength", nil)

		for _, m := range metrics {
			if m.MetricName == "queueLength" {
				metricValue, _ = m.Value.AsInt64()
				queueLength += metricValue
			}
		}
		scalerLogger.Info("QueueLength Metric value", "queueLength", queueLength)

		if err != nil {
			scalerLogger.V(1).Info("Error getting scale decision, but continue", "Error", err)
			continue
		} else if isTriggerActive {
			isScaledObjectActive = true
			scalerLogger.Info("Scaler is active")
		}
		scaler.Close()
	}

	h.scaleJobs(scaledObject, isScaledObjectActive, queueLength, maxValue)
}

// handleScaledObject contains the main logic for the ScaleHandler scaling logic.
// It'll check each trigger active status then call scaleScaledObject
func (h *ScaleHandler) handleScaledObject(ctx context.Context, scaledObject *kedav1alpha1.ScaledObject) {
	scalers, err := h.GetScaledObjectScalers(scaledObject)
	if err != nil {
		h.logger.Error(err, "Error getting scalers")
		return
	}

	isScaledObjectActive := false

	for _, scaler := range scalers {
		defer scaler.Close()
		isTriggerActive, err := scaler.IsActive(ctx)

		if err != nil {
			h.logger.V(1).Info("Error getting scale decision", "Error", err)
			continue
		} else if isTriggerActive {
			isScaledObjectActive = true
			h.logger.V(1).Info("Scaler for scaledObject is active", "Scaler", scaler)
		}
	}

	h.scaleScaledObject(scaledObject, isScaledObjectActive)
}
