//go:build aix
// +build aix

package filewatcher

import (
	"fmt"
	"runtime"
)

type Event struct {
	PkgPath string
	Debug   bool
}

func Watch(dirs []string, run func(Event) error) error {
	return fmt.Errorf("file watching is not supported on %v/%v", runtime.GOOS, runtime.GOARCH)
}
