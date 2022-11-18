// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 The Kubernetes Authors

package versions

import (
	"fmt"
	"regexp"
)

// Platform contains OS & architecture information
// Either may be '*' to indicate a wildcard match.
type Platform struct {
	OS   string
	Arch string
}

// Matches indicates if this platform matches the other platform,
// potentially with wildcard values.
func (p Platform) Matches(other Platform) bool {
	return (p.OS == other.OS || p.OS == "*" || other.OS == "*") &&
		(p.Arch == other.Arch || p.Arch == "*" || other.Arch == "*")
}

// IsWildcard checks if either OS or Arch are set to wildcard values.
func (p Platform) IsWildcard() bool {
	return p.OS == "*" || p.Arch == "*"
}
func (p Platform) String() string {
	return fmt.Sprintf("%s/%s", p.OS, p.Arch)
}

// BaseName returns the base directory name that fully identifies a given
// version and platform.
func (p Platform) BaseName(ver Concrete) string {
	return fmt.Sprintf("%d.%d.%d-%s-%s", ver.Major, ver.Minor, ver.Patch, p.OS, p.Arch)
}

// ArchiveName returns the full archive name for this version and platform.
func (p Platform) ArchiveName(ver Concrete) string {
	return "kubebuilder-tools-" + p.BaseName(ver) + ".tar.gz"
}

// PlatformItem represents a platform with corresponding
// known metadata for its download.
type PlatformItem struct {
	Platform
	MD5 string
}

// Set is a concrete version and all the corresponding platforms that it's available for.
type Set struct {
	Version   Concrete
	Platforms []PlatformItem
}

// ExtractWithPlatform produces a version & platform from the given regular expression
// and string that should match it.  If no match is found, Version will be nil.
//
// The regular expression must have the following capture groups:
// major, minor, patch, prelabel, prenum, os, arch, and must not support wildcard
// versions.
func ExtractWithPlatform(re *regexp.Regexp, name string) (*Concrete, Platform) {
	match := re.FindStringSubmatch(name)
	if match == nil {
		return nil, Platform{}
	}
	verInfo := PatchSelectorFromMatch(match, re)
	if verInfo.AsConcrete() == nil {
		panic(fmt.Sprintf("%v", verInfo))
	}
	// safe to convert, we've ruled out wildcards in our RE
	return verInfo.AsConcrete(), Platform{
		OS:   match[re.SubexpIndex("os")],
		Arch: match[re.SubexpIndex("arch")],
	}
}

var (
	versionPlatformREBase = ConcreteVersionRE.String() + `-(?P<os>\w+)-(?P<arch>\w+)`
	// VersionPlatformRE matches concrete version-platform strings.
	VersionPlatformRE = regexp.MustCompile(`^` + versionPlatformREBase + `$`)
	// ArchiveRE matches concrete version-platform.tar.gz strings.
	ArchiveRE = regexp.MustCompile(`^kubebuilder-tools-` + versionPlatformREBase + `\.tar\.gz$`)
)
