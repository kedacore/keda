package util

import (
	"strconv"

	"k8s.io/apimachinery/pkg/version"
)

type K8sVersion struct {
	Version       *version.Info
	MinorVersion  int
	PrettyVersion string
	Parsed        bool
}

func NewK8sVersion(version *version.Info) K8sVersion {
	minorTrimmed := ""
	if len(version.Minor) > 2 {
		minorTrimmed = version.Minor[:2]
	}

	parsed := false
	minor, err := strconv.Atoi(minorTrimmed)
	if err == nil {
		parsed = true
	}

	k8sVersion := new(K8sVersion)
	k8sVersion.Parsed = parsed
	k8sVersion.Version = version
	k8sVersion.MinorVersion = minor
	k8sVersion.PrettyVersion = version.Major + "." + version.Minor

	return *k8sVersion
}
