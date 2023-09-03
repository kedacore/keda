package util

import (
	"fmt"
	"os"

	"sigs.k8s.io/controller-runtime/pkg/cache"
)

// GetWatchNamespaces returns the namespaces the operator should be watching for changes
func GetWatchNamespaces() (map[string]cache.Config, error) {
	const WatchNamespaceEnvVar = "WATCH_NAMESPACE"
	ns, found := os.LookupEnv(WatchNamespaceEnvVar)
	if !found {
		return map[string]cache.Config{}, fmt.Errorf("%s must be set", WatchNamespaceEnvVar)
	}

	if ns == "" {
		return map[string]cache.Config{}, nil
	}

	return map[string]cache.Config{
		ns: {},
	}, nil
}
