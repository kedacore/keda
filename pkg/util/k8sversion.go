/*
Copyright 2021 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"strconv"

	"k8s.io/apimachinery/pkg/version"
	ctrl "sigs.k8s.io/controller-runtime"
)

var versionLog = ctrl.Log.WithName("k8s_version")

// K8sVersion holds parsed data from a K8s version
type K8sVersion struct {
	Version       *version.Info
	MinorVersion  int
	PrettyVersion string
	Parsed        bool
}

// NewK8sVersion will parse a version info and return a struct
func NewK8sVersion(version *version.Info) K8sVersion {
	minorTrimmed := ""
	if len(version.Minor) >= 2 {
		minorTrimmed = version.Minor[:2]
	} else {
		minorTrimmed = version.Minor
	}

	parsed := false
	minor, err := strconv.Atoi(minorTrimmed)
	if err == nil {
		parsed = true
	} else {
		versionLog.Error(err, "Could not parse Kubernetes minor version",
			"minorTrimmed", minorTrimmed, "rawMinor", version.Minor)
	}

	k8sVersion := new(K8sVersion)
	k8sVersion.Parsed = parsed
	k8sVersion.Version = version
	k8sVersion.MinorVersion = minor
	k8sVersion.PrettyVersion = version.Major + "." + version.Minor

	return *k8sVersion
}
