package gocql

import "runtime/debug"

const (
	mainModule = "github.com/gocql/gocql"
)

var driverName string

var driverVersion string

func init() {
	buildInfo, ok := debug.ReadBuildInfo()
	if ok {
		for _, d := range buildInfo.Deps {
			if d.Path == mainModule {
				driverName = mainModule
				driverVersion = d.Version
				if d.Replace != nil {
					driverName = d.Replace.Path
					driverVersion = d.Replace.Version
				}
				break
			}
		}
	}
}
