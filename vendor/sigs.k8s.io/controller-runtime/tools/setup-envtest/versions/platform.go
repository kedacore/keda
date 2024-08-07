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
// useGCS is deprecated and will be removed when the remote.GCSClient is removed.
func (p Platform) ArchiveName(useGCS bool, ver Concrete) string {
	if useGCS {
		return "kubebuilder-tools-" + p.BaseName(ver) + ".tar.gz"
	}
	return "envtest-v" + p.BaseName(ver) + ".tar.gz"
}

// PlatformItem represents a platform with corresponding
// known metadata for its download.
type PlatformItem struct {
	Platform

	*Hash
}

// Hash of an archive with envtest binaries.
type Hash struct {
	// Type of the hash.
	// GCS uses MD5HashType, controller-tools uses SHA512HashType.
	Type HashType

	// Encoding of the hash value.
	// GCS uses Base64HashEncoding, controller-tools uses HexHashEncoding.
	Encoding HashEncoding

	// Value of the hash.
	Value string
}

// HashType is the type of a hash.
type HashType string

const (
	// SHA512HashType represents a sha512 hash
	SHA512HashType HashType = "sha512"

	// MD5HashType represents a md5 hash
	MD5HashType HashType = "md5"
)

// HashEncoding is the encoding of a hash
type HashEncoding string

const (
	// Base64HashEncoding represents base64 encoding
	Base64HashEncoding HashEncoding = "base64"

	// HexHashEncoding represents hex encoding
	HexHashEncoding HashEncoding = "hex"
)

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
	// The archives published to GCS by kubebuilder use the "kubebuilder-tools-" prefix (e.g. "kubebuilder-tools-1.30.0-darwin-amd64.tar.gz").
	// The archives published to GitHub releases by controller-tools use the "envtest-v" prefix (e.g. "envtest-v1.30.0-darwin-amd64.tar.gz").
	ArchiveRE = regexp.MustCompile(`^(kubebuilder-tools-|envtest-v)` + versionPlatformREBase + `\.tar\.gz$`)
)
