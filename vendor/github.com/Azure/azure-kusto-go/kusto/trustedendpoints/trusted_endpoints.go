package trustedendpoints

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strings"

	"github.com/Azure/azure-kusto-go/kusto/data/errors"
	"github.com/samber/lo"
)

var (
	Instance = createInstance()
	//go:embed well_known_kusto_endpoints.json
	jsonFile []byte
)

type AllowedEndpoints struct {
	AllowedKustoSuffixes  []string
	AllowedKustoHostnames []string
}

type WellKnownKustoEndpointsDataStruct struct {
	AllowedEndpointsByLogin map[string]AllowedEndpoints
}

func createInstance() *TrustedEndpoints {
	matchers := map[string]*FastSuffixMatcher{}
	wellKnownData := WellKnownKustoEndpointsDataStruct{}

	err := json.Unmarshal(jsonFile, &wellKnownData)
	if err != nil {
		panic(err.Error())
	}

	for key, value := range wellKnownData.AllowedEndpointsByLogin {
		rules := []MatchRule{}
		for _, suf := range value.AllowedKustoSuffixes {
			rules = append(rules, MatchRule{suffix: suf, exact: false})
		}
		for _, host := range value.AllowedKustoHostnames {
			rules = append(rules, MatchRule{suffix: host, exact: true})
		}

		f, err := newFastSuffixMatcher(rules)
		if err != nil {
			panic(err.Error())
		}
		matchers[key] = f
	}

	return &TrustedEndpoints{matchers: matchers}
}

// SetOverridePolicy Set a policy to override all other trusted rules
func (trusted *TrustedEndpoints) SetOverridePolicy(matcher func(string) bool) {
	trusted.overrideMatcher = matcher
}

type TrustedEndpoints struct {
	matchers          map[string]*FastSuffixMatcher
	additionalMatcher *FastSuffixMatcher
	overrideMatcher   func(string) bool
}

type MatchRule struct {
	suffix string
	exact  bool
}

type FastSuffixMatcher struct {
	suffixLength int
	rules        map[string][]MatchRule
}

func tailLowerCase(str string, length int) string {
	if length <= 0 {
		return ""
	}

	if length >= len(str) {
		return strings.ToLower(str)
	}

	return strings.ToLower(str[len(str)-length:])
}

func (matcher *FastSuffixMatcher) isMatch(candidate string) bool {
	if len(candidate) < matcher.suffixLength {
		return false
	}
	if lst, ok := matcher.rules[tailLowerCase(candidate, matcher.suffixLength)]; ok {
		for _, rule := range lst {
			if strings.HasSuffix(strings.ToLower(candidate), rule.suffix) {
				if len(candidate) == len(rule.suffix) || !rule.exact {
					return true
				}
			}
		}
	}

	return false
}

func newFastSuffixMatcher(rules []MatchRule) (*FastSuffixMatcher, error) {
	minSufLen := len(lo.MinBy(rules, func(a MatchRule, cur MatchRule) bool {
		return len(a.suffix) < len(cur.suffix)
	}).suffix)

	if minSufLen == 0 || minSufLen == math.MaxInt32 {
		return nil, errors.ES(
			errors.OpUnknown,
			errors.KClientArgs,
			"FastSuffixMatcher should have at list one rule with at least one character",
		).SetNoRetry()

	}

	processedRules := map[string][]MatchRule{}
	for _, rule := range rules {
		suffix := tailLowerCase(rule.suffix, minSufLen)
		if lst, ok := processedRules[suffix]; !ok {
			processedRules[suffix] = []MatchRule{rule}
		} else {
			processedRules[suffix] = append(lst, rule)
		}
	}

	return &FastSuffixMatcher{
		suffixLength: minSufLen,
		rules:        processedRules,
	}, nil
}

func values[T comparable, R any](m map[T]R) []R {
	l := make([]R, 0, len(m))
	for _, val := range m {
		l = append(l, val)
	}

	return l
}

func createFastSuffixMatcherFromExisting(rules []MatchRule, existing *FastSuffixMatcher) (*FastSuffixMatcher, error) {
	if existing == nil || len(existing.rules) == 0 {
		return newFastSuffixMatcher(rules)
	}

	if rules == nil || len(rules) == 0 {
		return existing, nil
	}

	for _, elem := range existing.rules {
		rules = append(rules, elem...)
	}

	return newFastSuffixMatcher(rules)
}

// AddTrustedHosts Add or set a list of trusted endpoints rules
func (trusted *TrustedEndpoints) AddTrustedHosts(rules []MatchRule, replace bool) error {
	if rules == nil || len(rules) == 0 {
		if replace {
			trusted.additionalMatcher = nil
		}
		return nil
	}

	if replace {
		trusted.additionalMatcher = nil
	}

	matcher, err := createFastSuffixMatcherFromExisting(rules, trusted.additionalMatcher)
	trusted.additionalMatcher = matcher
	return err
}

// ValidateTrustedEndpoint Validates the endpoint uri trusted
func (trusted *TrustedEndpoints) ValidateTrustedEndpoint(endpoint string, loginEndpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil {
		return err
	}

	host := u.Host
	if host == "" {
		host = endpoint
	}

	// Check that target hostname is trusted and can accept security token
	return trusted.validateHostnameIsTrusted(host, loginEndpoint)
}

func isLocalAddress(host string) bool {
	if host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "[::1]" {
		return true
	}

	if strings.HasPrefix(host, "127.") && len(host) <= 15 && len(host) >= 9 {
		for _, c := range host {
			if c != '.' && (c < '0' || c > '9') {
				return false
			}
		}
		return true
	}

	return false
}

func (trusted *TrustedEndpoints) validateHostnameIsTrusted(host string, loginEndpoint string) error {
	// The loopback is unconditionally allowed (since we trust ourselves)
	if isLocalAddress(host) {
		return nil
	}

	// Either check the override matcher OR the matcher:
	override := trusted.overrideMatcher
	if override != nil && override(host) {
		return nil
	} else {
		matcher, ok := trusted.matchers[strings.ToLower(loginEndpoint)]
		if ok && (*matcher).isMatch(host) {
			return nil
		}
	}

	matcher := trusted.additionalMatcher
	if matcher != nil && matcher.isMatch(host) {
		return nil
	}

	return errors.ES(
		errors.OpUnknown,
		errors.KClientArgs,
		fmt.Sprintf("Can't communicate with '%s' as this hostname is currently not trusted; please see https://aka.ms/kustotrustedendpoints.", host),
	).SetNoRetry()
}
