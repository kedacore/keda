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

// WatchLabelSelectorEnvVar scopes which ScaledObjects and ScaledJobs (and
// their derived HorizontalPodAutoscalers) this operator reconciles, based
// on their labels. Empty (default) means watch everything, which preserves
// the previous behaviour.
//
// Mirrors WATCH_NAMESPACE, but filters by label instead of by namespace.
//
// TriggerAuthentications and ClusterTriggerAuthentications are scoped via
// WATCH_LABEL_SELECTOR_FOR_TRIGGERAUTH instead, so that cluster-scoped
// CTAs can be shared across all reconciling operators by default.
//
// Example: "environment=production" reconciles only resources labelled
// environment=production. Set notation is also supported (e.g.
// "tier in (gold,silver)", "!canary"). See
// k8s.io/apimachinery/pkg/apis/meta/v1.ParseToLabelSelector.
const WatchLabelSelectorEnvVar = "WATCH_LABEL_SELECTOR"

// WatchLabelSelectorForTriggerAuthEnvVar scopes which
// TriggerAuthentications and ClusterTriggerAuthentications this operator
// reconciles, based on their labels. Empty (default) means watch every
// TriggerAuthentication and ClusterTriggerAuthentication.
//
// This is intentionally decoupled from WATCH_LABEL_SELECTOR: when the
// operator is scoped to a subset of ScaledObjects, the (Cluster)TriggerAuthentication
// resources those SOs reference typically still need to be visible in
// the operator's cache. Splitting the selector lets a single CTA be
// shared cluster-wide while still filtering the workload resources.
//
// Accepts the same syntax as WATCH_LABEL_SELECTOR.
const WatchLabelSelectorForTriggerAuthEnvVar = "WATCH_LABEL_SELECTOR_FOR_TRIGGERAUTH"

// getLabelSelectorFromEnv reads and parses a label selector from the given
// env var. Returns (nil, nil) if the env var is unset or empty, meaning
// "no label-based filtering".
func getLabelSelectorFromEnv(envVar string) (*metav1.LabelSelector, error) {
	raw, found := os.LookupEnv(envVar)
	if !found || raw == "" || raw == "\"\"" {
		return nil, nil
	}
	selector, err := metav1.ParseToLabelSelector(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid %s %q: %w", envVar, raw, err)
	}
	return selector, nil
}

// labelSelectorPredicate turns a parsed label selector into a controller-runtime
// predicate. A nil selector means "no filter" and is represented by the
// zero-value Funcs{}, which returns true for every event.
func labelSelectorPredicate(ls *metav1.LabelSelector) (predicate.Predicate, error) {
	if ls == nil {
		return predicate.Funcs{}, nil
	}
	return predicate.LabelSelectorPredicate(*ls)
}

// labelSelectorByObject builds cache.ByObject entries applying the given
// selector to each object type at the API server's list/watch level.
// Returns nil if the selector is nil (no filtering).
func labelSelectorByObject(ls *metav1.LabelSelector, envVar string, objs ...client.Object) (map[client.Object]cache.ByObject, error) {
	if ls == nil {
		return nil, nil
	}
	selector, err := metav1.LabelSelectorAsSelector(ls)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", envVar, err)
	}
	out := make(map[client.Object]cache.ByObject, len(objs))
	for _, o := range objs {
		out[o] = cache.ByObject{Label: selector}
	}
	return out, nil
}

// GetWatchLabelSelector returns a parsed label selector from WATCH_LABEL_SELECTOR
// or nil if unset or empty (meaning: no label-based filtering).
func GetWatchLabelSelector() (*metav1.LabelSelector, error) {
	return getLabelSelectorFromEnv(WatchLabelSelectorEnvVar)
}

// WatchLabelSelectorPredicate returns a controller-runtime predicate that
// filters events to only those whose labels match WATCH_LABEL_SELECTOR.
// If the env var is unset or empty, all events pass (no filtering).
func WatchLabelSelectorPredicate() (predicate.Predicate, error) {
	ls, err := GetWatchLabelSelector()
	if err != nil {
		return nil, err
	}
	return labelSelectorPredicate(ls)
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
	return labelSelectorByObject(ls, WatchLabelSelectorEnvVar, objs...)
}

// GetWatchLabelSelectorForTriggerAuth returns a parsed label selector from
// WATCH_LABEL_SELECTOR_FOR_TRIGGERAUTH or nil if unset or empty (meaning:
// no label-based filtering on TriggerAuthentications).
func GetWatchLabelSelectorForTriggerAuth() (*metav1.LabelSelector, error) {
	return getLabelSelectorFromEnv(WatchLabelSelectorForTriggerAuthEnvVar)
}

// WatchLabelSelectorForTriggerAuthPredicate returns a controller-runtime
// predicate that filters events to only TriggerAuthentications and
// ClusterTriggerAuthentications whose labels match
// WATCH_LABEL_SELECTOR_FOR_TRIGGERAUTH. If the env var is unset or empty,
// all events pass.
func WatchLabelSelectorForTriggerAuthPredicate() (predicate.Predicate, error) {
	ls, err := GetWatchLabelSelectorForTriggerAuth()
	if err != nil {
		return nil, err
	}
	return labelSelectorPredicate(ls)
}

// WatchLabelSelectorForTriggerAuthByObject returns cache.ByObject entries
// that apply WATCH_LABEL_SELECTOR_FOR_TRIGGERAUTH to the given object types.
// Returns nil if the env var is unset/empty (no filtering).
func WatchLabelSelectorForTriggerAuthByObject(objs ...client.Object) (map[client.Object]cache.ByObject, error) {
	ls, err := GetWatchLabelSelectorForTriggerAuth()
	if err != nil {
		return nil, err
	}
	return labelSelectorByObject(ls, WatchLabelSelectorForTriggerAuthEnvVar, objs...)
}
