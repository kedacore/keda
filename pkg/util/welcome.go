/*
Copyright 2023 The KEDA Authors

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
	"fmt"
	"runtime"

	"github.com/go-logr/logr"

	"github.com/kedacore/keda/v2/version"
)

const (
	minSupportedVersion = 32
	maxSupportedVersion = 35
)

func PrintWelcome(logger logr.Logger, kubeVersion K8sVersion, component string) {
	logger.Info(fmt.Sprintf("Starting %s", component))
	logger.Info(fmt.Sprintf("KEDA Version: %s", version.Version))
	logger.Info(fmt.Sprintf("Git Commit: %s", version.GitCommit))
	logger.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	logger.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	logger.Info(fmt.Sprintf("Running on Kubernetes %s", kubeVersion.PrettyVersion), "version", kubeVersion.Version)

	if kubeVersion.MinorVersion < minSupportedVersion ||
		kubeVersion.MinorVersion > maxSupportedVersion {
		logger.Info(fmt.Sprintf("WARNING: KEDA %s hasn't been tested on Kubernetes %s", version.Version, kubeVersion.Version))
		logger.Info("You can check recommended versions on https://keda.sh")
	}
}
