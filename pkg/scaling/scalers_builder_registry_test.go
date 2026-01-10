package scaling

import (
	"context"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kedacore/keda/v2/pkg/scalers"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

func TestScalerRegistry(t *testing.T) {
	// Test registering a custom scaler
	called := false
	RegisterScalerBuilder("test-scaler", func(_ context.Context, _ client.Client, _ *scalersconfig.ScalerConfig) (scalers.Scaler, error) {
		called = true
		return nil, nil
	})

	_, _ = buildScaler(context.Background(), nil, "test-scaler", &scalersconfig.ScalerConfig{})

	if !called {
		t.Error("Test scaler builder was not called")
	}

	// Clean up
	delete(scalerBuilders, "test-scaler")
}

func TestProductionScalersRegistered(t *testing.T) {
	// Verify that production scalers are registered in the init() function
	testCases := []string{
		"cron",
		"prometheus",
		"kafka",
		"cpu",
		"memory",
	}

	for _, scalerType := range testCases {
		if _, ok := scalerBuilders[scalerType]; !ok {
			t.Errorf("Production scaler '%s' was not registered", scalerType)
		}
	}
}

func TestUnknownScalerReturnsError(t *testing.T) {
	config := &scalersconfig.ScalerConfig{}
	_, err := buildScaler(context.Background(), nil, "non-existent-scaler", config)

	if err == nil {
		t.Error("Expected error for unknown scaler type, got nil")
	}

	expectedMsg := "no scaler found for type: non-existent-scaler"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}
