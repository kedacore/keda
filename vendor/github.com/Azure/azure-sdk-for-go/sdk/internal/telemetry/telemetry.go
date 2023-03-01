// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package telemetry

import (
	"fmt"
	"os"
	"runtime"
)

// Format creates the properly formatted SDK component for a User-Agent string.
// Ex: azsdk-go-azservicebus/v1.0.0 (go1.19.3; linux)
// comp - the package name for a component (ex: azservicebus)
// ver - the version of the component (ex: v1.0.0)
func Format(comp, ver string) string {
	// ex: azsdk-go-azservicebus/v1.0.0 (go1.19.3; windows)
	return fmt.Sprintf("azsdk-go-%s/%s %s", comp, ver, platformInfo)
}

// platformInfo is the Go version and OS, formatted properly for insertion
// into a User-Agent string. (ex: '(go1.19.3; windows')
// NOTE: the ONLY function that should write to this variable is this func
var platformInfo = func() string {
	operatingSystem := runtime.GOOS // Default OS string
	switch operatingSystem {
	case "windows":
		operatingSystem = os.Getenv("OS") // Get more specific OS information
	case "linux": // accept default OS info
	case "freebsd": //  accept default OS info
	}
	return fmt.Sprintf("(%s; %s)", runtime.Version(), operatingSystem)
}()
