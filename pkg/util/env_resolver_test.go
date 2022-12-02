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
