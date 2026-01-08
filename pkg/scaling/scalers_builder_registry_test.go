package scaling

import (
	"context"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kedacore/keda/v2/pkg/scalers"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

func TestTestScalerRegistry(t *testing.T) {
	called := false
	RegisterTestScalerBuilder("test-scaler", func(_ context.Context, _ client.Client, _ string, _ *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		called = true
		return nil, nil
	})

	_, _ = buildScaler(context.Background(), nil, "test-scaler", &scalersconfig.ScalerConfig{})

	if !called {
		t.Error("Test scaler builder was not called")
	}

	delete(testScalerBuilders, "test-scaler")
}

func TestProductionScalerStillWorks(t *testing.T) {
	// Verify that a production scaler like "cron" still returns an error with proper message
	config := &scalersconfig.ScalerConfig{}
	_, err := buildScaler(context.Background(), nil, "cron", config)

	if err != nil && err.Error() == "no scaler found for type: cron" {
		t.Error("Production scaler 'cron' was not found in buildScaler")
	}
}
