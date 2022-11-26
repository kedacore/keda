// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 The Kubernetes Authors

package versions

import (
	"fmt"
	"regexp"
	"strconv"
)

var (
	// baseVersionRE is a semver-ish version -- either X.Y.Z, X.Y, or X.Y.{*|x}.
	baseVersionRE = `(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)(?:\.(?P<patch>0|[1-9]\d*|x|\*))?`
	// versionExprRe matches valid version input for FromExpr.
	versionExprRE = regexp.MustCompile(`^(?P<sel><|~|<=)?` + baseVersionRE + `(?P<latest>!)?$`)

	// ConcreteVersionRE matches a concrete version anywhere in the string.
	ConcreteVersionRE = regexp.MustCompile(`(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)`)
	// OnlyConcreteVersionRE matches a string that's just a concrete version.
	OnlyConcreteVersionRE = regexp.MustCompile(`^` + ConcreteVersionRE.String() + `$`)
)

// FromExpr extracts a version from a string in the form of a semver version,
// where X, Y, and Z may also be wildcards ('*', 'x'),
// and pre-release names & numbers may also be wildcards.  The prerelease section is slightly
// restricted to match what k8s does.
// The whole string is a version selector as follows:
//   - X.Y.Z matches version X.Y.Z where x, y, and z are
//     are ints >= 0, and Z may be '*' or 'x'
//   - X.Y is equivalent to X.Y.*
//   - ~X.Y.Z means >= X.Y.Z && < X.Y+1.0
//   - <X.Y.Z means older than the X.Y.Z (mainly useful for cleanup (similarly for <=)
//   - an '!' at the end means force checking API server for the latest versions
//     instead of settling for local matches.
//
// ^[^~]?SEMVER(!)??$ .
func FromExpr(expr string) (Spec, error) {
	match := versionExprRE.FindStringSubmatch(expr)
	if match == nil {
		return Spec{}, fmt.Errorf("unable to parse %q as a version string"+
			"should be X.Y.Z, where Z may be '*','x', or left off entirely "+
			"to denote a wildcard, optionally prefixed by ~|<|<=, and optionally"+
			"followed by ! (latest remote version)", expr)
	}
	verInfo := PatchSelectorFromMatch(match, versionExprRE)
	latest := match[versionExprRE.SubexpIndex("latest")] == "!"
	sel := match[versionExprRE.SubexpIndex("sel")]
	spec := Spec{
		CheckLatest: latest,
	}
	if sel == "" {
		spec.Selector = verInfo
		return spec, nil
	}

	switch sel {
	case "<", "<=":
		spec.Selector = LessThanSelector{PatchSelector: verInfo, OrEquals: sel == "<="}
	case "~":
		// since patch & preNum are >= comparisons, if we use
		// wildcards with a selector we can just set them to zero.
		if verInfo.Patch == AnyPoint {
			verInfo.Patch = PointVersion(0)
		}
		baseVer := *verInfo.AsConcrete()
		spec.Selector = TildeSelector{Concrete: baseVer}
	default:
		panic("unreachable: mismatch between FromExpr and its RE in selector")
	}

	return spec, nil
}

// PointVersionFromValidString extracts a point version
// from the corresponding string representation, which may
// be a number >= 0, or x|* (AnyPoint).
//
// Anything else will cause a panic (use this on strings
// extracted from regexes).
func PointVersionFromValidString(str string) PointVersion {
	switch str {
	case "*", "x":
		return AnyPoint
	default:
		ver, err := strconv.Atoi(str)
		if err != nil {
			panic(err)
		}
		return PointVersion(ver)
	}
}

// PatchSelectorFromMatch constructs a simple selector according to the
// ParseExpr rules out of pre-validated sections.
//
// re must include name captures for major, minor, patch, prenum, and prelabel
//
// Any bad input may cause a panic.  Use with when you got the parts from an RE match.
func PatchSelectorFromMatch(match []string, re *regexp.Regexp) PatchSelector {
	// already parsed via RE, should be fine to ignore errors unless it's a
	// *huge* number
	major, err := strconv.Atoi(match[re.SubexpIndex("major")])
	if err != nil {
		panic("invalid input passed as patch selector (invalid state)")
	}
	minor, err := strconv.Atoi(match[re.SubexpIndex("minor")])
	if err != nil {
		panic("invalid input passed as patch selector (invalid state)")
	}

	// patch is optional, means wilcard if left off
	patch := AnyPoint
	if patchRaw := match[re.SubexpIndex("patch")]; patchRaw != "" {
		patch = PointVersionFromValidString(patchRaw)
	}
	return PatchSelector{
		Major: major,
		Minor: minor,
		Patch: patch,
	}
}
