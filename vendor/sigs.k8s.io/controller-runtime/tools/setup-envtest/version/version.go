package version

import "runtime/debug"

// Version to be set using ldflags:
// -ldflags "-X sigs.k8s.io/controller-runtime/tools/setup-envtest/version.version=v1.0.0"
// falls back to module information is unse
var version = ""

// Version returns the version of the main module
func Version() string {
	if version != "" {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok || info == nil || info.Main.Version == "" {
		// binary has not been built with module support or doesn't contain a version.
		return "(unknown)"
	}
	return info.Main.Version
}
