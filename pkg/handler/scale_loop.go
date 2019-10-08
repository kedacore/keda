package handler

import (
	"context"
	"time"

	keda_v1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	log "github.com/sirupsen/logrus"
)

// This method blocks forever and checks the scaledObject based on its pollingInterval
// if isDue is set to true, the method will check the scaledObject right away. Otherwise
// it'll wait for pollingInterval then check.
func (h *ScaleHandler) handleScaleLoop(ctx context.Context, scaledObject *keda_v1alpha1.ScaledObject, isDue bool) {
	h.handleScale(ctx, scaledObject)
	var pollingInterval time.Duration
	if scaledObject.Spec.PollingInterval != nil {
		pollingInterval = time.Second * time.Duration(*scaledObject.Spec.PollingInterval)
	} else {
		pollingInterval = time.Second * time.Duration(defaultPollingInterval)
	}

	getPollingInterval := func() time.Duration {
		if isDue {
			isDue = false
			return 0
		}
		return pollingInterval
	}

	log.Debugf("watching scaledObject (%s/%s) with pollingInterval: %d", scaledObject.GetNamespace(), scaledObject.GetName(), pollingInterval)

	for {
		select {
		case <-time.After(getPollingInterval()):
			switch scaledObject.Spec.ScaleType {
			case keda_v1alpha1.ScaleTypeJob:
				h.createOrUpdateJobInformerForScaledObject(scaledObject)
			default:
				h.createHPAWithRetry(scaledObject, false)
			}
			h.handleScale(ctx, scaledObject)
		case <-ctx.Done():
			log.Debugf("context for scaledObject (%s/%s) canceled", scaledObject.GetNamespace(), scaledObject.GetName())
			return
		}
	}
}

// handleScale contains the main logic for the ScaleHandler scaling logic.
// It'll check each trigger active status then call scaleDeployment
func (h *ScaleHandler) handleScale(ctx context.Context, scaledObject *keda_v1alpha1.ScaledObject) {
	switch scaledObject.Spec.ScaleType {
	case keda_v1alpha1.ScaleTypeJob:
		h.handleScaleJob(ctx, scaledObject)
		break
	default:
		h.handleScaleDeployment(ctx, scaledObject)

	}
	return
}

func (h *ScaleHandler) handleScaleJob(ctx context.Context, scaledObject *keda_v1alpha1.ScaledObject) {
	//TODO: need to actually handle the scale here
	log.Println("Handle Scale Job called")
	scalers, err := h.getJobScalers(scaledObject)

	if err != nil {
		log.Errorf("Error getting scalers: %s", err)
		return
	}

	isScaledObjectActive := false
	log.Printf("Scalers: %d", len(scalers))
	var queueLength int64
	var maxValue int64

	for _, scaler := range scalers {
		isTriggerActive, err := scaler.IsActive(ctx)
		log.Printf("IsTriggerActive: %t", isTriggerActive)
		metricSpecs := scaler.GetMetricSpecForScaling()

		var metricValue int64
		for _, metric := range metricSpecs {
			metricValue, _ = metric.External.TargetAverageValue.AsInt64()
			maxValue += metricValue
		}
		log.Printf("Max value: %d", maxValue)

		metrics, _ := scaler.GetMetrics(ctx, "queueLength", nil)

		for _, m := range metrics {
			if m.MetricName == "queueLength" {
				metricValue, _ = m.Value.AsInt64()
				queueLength += metricValue
			}
		}
		log.Printf("QueueLength Metric value: %d", queueLength)

		if err != nil {
			log.Debugf("Error getting scale decision: %s", err)
			continue
		} else if isTriggerActive {
			isScaledObjectActive = true
			log.Printf("Scaler %s for scaledObject %s/%s is active", scaler, scaledObject.GetNamespace(), scaledObject.GetName())
		}
		scaler.Close()
	}

	h.scaleJobs(scaledObject, isScaledObjectActive, queueLength, maxValue)

}

func (h *ScaleHandler) handleScaleDeployment(ctx context.Context, scaledObject *keda_v1alpha1.ScaledObject) {
	scalers, deployment, err := h.getDeploymentScalers(scaledObject)

	if deployment == nil {
		return
	}
	if err != nil {
		log.Errorf("Error getting scalers: %s", err)
		return
	}

	isScaledObjectActive := false

	for _, scaler := range scalers {
		defer scaler.Close()
		isTriggerActive, err := scaler.IsActive(ctx)

		if err != nil {
			log.Debugf("Error getting scale decision: %s", err)
			continue
		} else if isTriggerActive {
			isScaledObjectActive = true
			log.Debugf("Scaler %T for scaledObject %s/%s is active", scaler, scaledObject.GetNamespace(), scaledObject.GetName())
		}
	}

	h.scaleDeployment(deployment, scaledObject, isScaledObjectActive)

	return
}
