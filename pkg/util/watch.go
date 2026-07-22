package util

import (
	"fmt"
	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

// WatchLabelSelectorEnvVar scopes which ScaledObjects, ScaledJobs,
// TriggerAuthentications and ClusterTriggerAuthentications this operator
// reconciles, based on their labels. Empty (default) means watch everything,
// which preserves the previous behaviour.
//
// Mirrors WATCH_NAMESPACE, but filters by label instead of by namespace.
//
// Example: "environment=production" reconciles only resources labelled
// environment=production. Set notation is also supported (e.g.
// "tier in (gold,silver)", "!canary"). See
// k8s.io/apimachinery/pkg/apis/meta/v1.ParseToLabelSelector.
const WatchLabelSelectorEnvVar = "WATCH_LABEL_SELECTOR"

// GetWatchLabelSelector returns a parsed label selector from WATCH_LABEL_SELECTOR
// or nil if unset or empty (meaning: no label-based filtering).
func GetWatchLabelSelector() (*metav1.LabelSelector, error) {
	raw, found := os.LookupEnv(WatchLabelSelectorEnvVar)
	if !found || raw == "" || raw == "\"\"" {
		return nil, nil
	}
	selector, err := metav1.ParseToLabelSelector(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid %s %q: %w", WatchLabelSelectorEnvVar, raw, err)
	}
	return selector, nil
}

// WatchLabelSelectorPredicate returns a controller-runtime predicate that
// filters events to only those whose labels match WATCH_LABEL_SELECTOR.
// If the env var is unset or empty, all events pass (no filtering).
func WatchLabelSelectorPredicate() (predicate.Predicate, error) {
	ls, err := GetWatchLabelSelector()
	if err != nil {
		return nil, err
	}
	if ls == nil {
		// Zero-value Funcs returns true for every event, equivalent to "no filter".
		return predicate.Funcs{}, nil
	}
	return predicate.LabelSelectorPredicate(*ls)
}

// WatchLabelSelectorByObject returns cache.ByObject entries that apply
// WATCH_LABEL_SELECTOR to the given object types at the API server's
// list/watch level. Non-matching objects never enter the informer cache,
// so Owns()-triggered reconciles for them are also dropped (the Get from
// the cache returns NotFound).
//
// Returns nil if WATCH_LABEL_SELECTOR is unset/empty (no filtering).
func WatchLabelSelectorByObject(objs ...client.Object) (map[client.Object]cache.ByObject, error) {
	ls, err := GetWatchLabelSelector()
	if err != nil {
		return nil, err
	}
	if ls == nil {
		return nil, nil
	}
	selector, err := metav1.LabelSelectorAsSelector(ls)
	if err != nil {
		return nil, fmt.Errorf("invalid WATCH_LABEL_SELECTOR: %w", err)
	}
	out := make(map[client.Object]cache.ByObject, len(objs))
	for _, o := range objs {
		out[o] = cache.ByObject{Label: selector}
	}
	return out, nil
}
