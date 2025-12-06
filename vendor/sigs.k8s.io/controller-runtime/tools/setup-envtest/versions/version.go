// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 The Kubernetes Authors

package versions

import (
	"fmt"
	"strconv"
)

// NB(directxman12): much of this is custom instead of using a library because
// a) none of the standard libraries have hashable version types (for valid reasons,
//    but we can use a restricted subset for our usecase)
// b) everybody has their own definition of how selectors work anyway

// NB(directxman12): pre-release support is... complicated with selectors
// if we end up needing it, think carefully about what a wildcard prerelease
// type means (does it include "not a prerelease"?), and what <=1.17.3-x.x means.

// Concrete is a concrete Kubernetes-style semver version.
type Concrete struct {
	Major, Minor, Patch int
}

// AsConcrete returns this version.
func (c Concrete) AsConcrete() *Concrete {
	return &c
}

// NewerThan checks if the given other version is newer than this one.
func (c Concrete) NewerThan(other Concrete) bool {
	if c.Major != other.Major {
		return c.Major > other.Major
	}
	if c.Minor != other.Minor {
		return c.Minor > other.Minor
	}
	return c.Patch > other.Patch
}

// Matches checks if this version is equal to the other one.
func (c Concrete) Matches(other Concrete) bool {
	return c == other
}

func (c Concrete) String() string {
	return fmt.Sprintf("%d.%d.%d", c.Major, c.Minor, c.Patch)
}

// PatchSelector selects a set of versions where the patch is a wildcard.
type PatchSelector struct {
	Major, Minor int
	Patch        PointVersion
}

func (s PatchSelector) String() string {
	return fmt.Sprintf("%d.%d.%s", s.Major, s.Minor, s.Patch)
}

// Matches checks if the given version matches this selector.
func (s PatchSelector) Matches(ver Concrete) bool {
	return s.Major == ver.Major && s.Minor == ver.Minor && s.Patch.Matches(ver.Patch)
}

// AsConcrete returns nil if there are wildcards in this selector,
// and the concrete version that this selects otherwise.
func (s PatchSelector) AsConcrete() *Concrete {
	if s.Patch == AnyPoint {
		return nil
	}

	return &Concrete{
		Major: s.Major,
		Minor: s.Minor,
		Patch: int(s.Patch), // safe to cast, we've just checked wilcards above
	}
}

// TildeSelector selects [X.Y.Z, X.Y+1.0).
type TildeSelector struct {
	Concrete
}

// Matches checks if the given version matches this selector.
func (s TildeSelector) Matches(ver Concrete) bool {
	if s.Concrete.Matches(ver) {
		// easy, "exact" match
		return true
	}
	return ver.Major == s.Major && ver.Minor == s.Minor && ver.Patch >= s.Patch
}
func (s TildeSelector) String() string {
	return "~" + s.Concrete.String()
}

// AsConcrete returns nil (this is never a concrete version).
func (s TildeSelector) AsConcrete() *Concrete {
	return nil
}

// LessThanSelector selects versions older than the given one
// (mainly useful for cleaning up).
type LessThanSelector struct {
	PatchSelector
	OrEquals bool
}

// Matches checks if the given version matches this selector.
func (s LessThanSelector) Matches(ver Concrete) bool {
	if s.Major != ver.Major {
		return s.Major > ver.Major
	}
	if s.Minor != ver.Minor {
		return s.Minor > ver.Minor
	}
	if !s.Patch.Matches(ver.Patch) {
		// matches rules out a wildcard, so it's fine to compare as normal numbers
		return int(s.Patch) > ver.Patch
	}
	return s.OrEquals
}
func (s LessThanSelector) String() string {
	if s.OrEquals {
		return "<=" + s.PatchSelector.String()
	}
	return "<" + s.PatchSelector.String()
}

// AsConcrete returns nil (this is never a concrete version).
func (s LessThanSelector) AsConcrete() *Concrete {
	return nil
}

// AnySelector matches any version at all.
type AnySelector struct{}

// Matches checks if the given version matches this selector.
func (AnySelector) Matches(_ Concrete) bool { return true }

// AsConcrete returns nil (this is never a concrete version).
func (AnySelector) AsConcrete() *Concrete { return nil }
func (AnySelector) String() string        { return "*" }

// Selector selects some concrete version or range of versions.
type Selector interface {
	// AsConcrete tries to return this selector as a concrete version.
	// If the selector would only match a single version, it'll return
	// that, otherwise it'll return nil.
	AsConcrete() *Concrete
	// Matches checks if this selector matches the given concrete version.
	Matches(ver Concrete) bool
	String() string
}

// Spec matches some version or range of versions, and tells us how to deal with local and
// remote when selecting a version.
type Spec struct {
	Selector

	// CheckLatest tells us to check the remote server for the latest
	// version that matches our selector, instead of just relying on
	// matching local versions.
	CheckLatest bool
}

// MakeConcrete replaces the contents of this spec with one that
// matches the given concrete version (without checking latest
// from the server).
func (s *Spec) MakeConcrete(ver Concrete) {
	s.Selector = ver
	s.CheckLatest = false
}

// AsConcrete returns the underlying selector as a concrete version, if
// possible.
func (s Spec) AsConcrete() *Concrete {
	return s.Selector.AsConcrete()
}

// Matches checks if the underlying selector matches the given version.
func (s Spec) Matches(ver Concrete) bool {
	return s.Selector.Matches(ver)
}

func (s Spec) String() string {
	res := s.Selector.String()
	if s.CheckLatest {
		res += "!"
	}
	return res
}

// PointVersion represents a wildcard (patch) version
// or concrete number.
type PointVersion int

const (
	// AnyPoint matches any point version.
	AnyPoint PointVersion = -1
)

// Matches checks if a point version is compatible
// with a concrete point version.
// Two point versions are compatible if they are
// a) both concrete
// b) one is a wildcard.
func (p PointVersion) Matches(other int) bool {
	switch p {
	case AnyPoint:
		return true
	default:
		return int(p) == other
	}
}
func (p PointVersion) String() string {
	switch p {
	case AnyPoint:
		return "*"
	default:
		return strconv.Itoa(int(p))
	}
}

var (
	// LatestVersion matches the most recent version on the remote server.
	LatestVersion = Spec{
		Selector:    AnySelector{},
		CheckLatest: true,
	}
	// AnyVersion matches any local or remote version.
	AnyVersion = Spec{
		Selector: AnySelector{},
	}
)
