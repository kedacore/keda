package util

import (
	"fmt"
	"os"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// GetWatchNamespaces returns the namespaces the operator should be watching for changes
func GetWatchNamespaces() (map[string]cache.Config, error) {
	const WatchNamespaceEnvVar = "WATCH_NAMESPACE"
	ns, found := os.LookupEnv(WatchNamespaceEnvVar)
	if !found {
		return map[string]cache.Config{}, fmt.Errorf("%s must be set", WatchNamespaceEnvVar)
	}

	if ns == "" || ns == "\"\"" {
		return map[string]cache.Config{}, nil
	}
	nss := strings.Split(ns, ",")
	nssMap := make(map[string]cache.Config)
	for _, n := range nss {
		nssMap[n] = cache.Config{}
	}

	return nssMap, nil
}

// IgnoreOtherNamespaces returns the predicate for watched events that will filter out those that are not coming
// from a watched namespace (empty namespace or unset env var denotes all)
func IgnoreOtherNamespaces() predicate.Predicate {
	nss, err := GetWatchNamespaces()
	return predicate.Funcs{
		GenericFunc: func(e event.GenericEvent) bool {
			if len(nss) == 0 || err != nil {
				return true
			}
			_, ok := nss[e.Object.GetNamespace()]
			return ok
		},
	}
}
