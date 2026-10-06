package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResolveMissingOsEnvDuration(t *testing.T) {
	actual, err := ResolveOsEnvDuration("missing_duration")
	assert.Nil(t, actual)
	assert.Nil(t, err)

	t.Setenv("empty_duration", "")
	actual, err = ResolveOsEnvDuration("empty_duration")
	assert.Nil(t, actual)
	assert.Nil(t, err)
}

func TestResolveInvalidOsEnvDuration(t *testing.T) {
	t.Setenv("blank_duration", "    ")
	actual, err := ResolveOsEnvDuration("blank_duration")
	assert.Equal(t, time.Duration(0), *actual)
	assert.NotNil(t, err)

	t.Setenv("invalid_duration", "deux heures")
	actual, err = ResolveOsEnvDuration("invalid_duration")
	assert.Equal(t, time.Duration(0), *actual)
	assert.NotNil(t, err)
}

func TestResolveValidOsEnvDuration(t *testing.T) {
	t.Setenv("valid_duration_seconds", "8s")
	actual, err := ResolveOsEnvDuration("valid_duration_seconds")
	assert.Equal(t, time.Duration(8)*time.Second, *actual)
	assert.Nil(t, err)

	t.Setenv("valid_duration_minutes", "30m")
	actual, err = ResolveOsEnvDuration("valid_duration_minutes")
	assert.Equal(t, time.Duration(30)*time.Minute, *actual)
	assert.Nil(t, err)
}

func TestResolveScaleLoopJitterMax(t *testing.T) {
	// 1. Neither flag nor env var set -> returns default (0)
	jitter, err := ResolveScaleLoopJitterMax(0, false)
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), jitter)

	// 2. Flag explicitly set to a positive duration -> returns flag value
	jitter, err = ResolveScaleLoopJitterMax(10*time.Second, true)
	assert.NoError(t, err)
	assert.Equal(t, 10*time.Second, jitter)

	// 3. Flag explicitly set to 0 -> returns 0 (even if env var is set)
	t.Setenv(ScaleLoopJitterMaxEnvVar, "30s")
	jitter, err = ResolveScaleLoopJitterMax(0, true)
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), jitter)

	// 4. Flag explicitly set to a negative duration -> returns error
	_, err = ResolveScaleLoopJitterMax(-5*time.Second, true)
	assert.Error(t, err)

	// 5. Flag not explicitly set, env var set to valid duration -> returns env var value
	t.Setenv(ScaleLoopJitterMaxEnvVar, "15s")
	jitter, err = ResolveScaleLoopJitterMax(0, false)
	assert.NoError(t, err)
	assert.Equal(t, 15*time.Second, jitter)

	// 6. Flag not explicitly set, env var set to negative duration -> returns error
	t.Setenv(ScaleLoopJitterMaxEnvVar, "-10s")
	_, err = ResolveScaleLoopJitterMax(0, false)
	assert.Error(t, err)

	// 7. Flag not explicitly set, env var invalid duration string -> returns error
	t.Setenv(ScaleLoopJitterMaxEnvVar, "not-a-duration")
	_, err = ResolveScaleLoopJitterMax(0, false)
	assert.Error(t, err)

	// 8. Flag not explicitly set, env var empty -> returns flag value (default 0)
	t.Setenv(ScaleLoopJitterMaxEnvVar, "")
	jitter, err = ResolveScaleLoopJitterMax(0, false)
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), jitter)
}
