package connectionpool

import (
	"context"
	"fmt"
	"os"

	"sync"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	globalOverrides sync.Map
	configPath      string
	logger          = log.Log.WithName("connectionpool")
)

// InitGlobalPoolConfig loads the YAML config and starts a watcher for live reloads.
func InitGlobalPoolConfig(ctx context.Context, path string) {
	configPath = path
	loadConfig()
	go startConfigWatcher(ctx)
}

// loadConfig parses the YAML and updates globalOverrides safely.
func loadConfig() {
	data, err := os.ReadFile(configPath)
	if err != nil {
		logger.V(1).Info("No pool config found;", "path", configPath, "err", err)
		return
	}

	var parsed map[string]int32
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		logger.Error(err, "Invalid pool config format", "path", configPath)
		return
	}

	// clear existing map before writing new values
	globalOverrides.Range(func(key, value any) bool {
		globalOverrides.Delete(key)
		return true
	})

	// store new values
	for key, val := range parsed {
		globalOverrides.Store(key, val)
	}

	logger.Info("Loaded global pool configuration", "entries", len(parsed))
}

// startConfigWatcher watches for ConfigMap updates and reloads on file change.
func startConfigWatcher(ctx context.Context) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error(err, "Failed to start global connection pool configuration file watcher")
		return
	}
	defer watcher.Close()

	_ = watcher.Add(configPath)
	logger.Info("Started watching global connection pool configuration", "path", configPath)

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				logger.Info("Detected pool config change; reloading")
				loadConfig()
			}
		case err := <-watcher.Errors:
			if err != nil {
				logger.Error(err, "Watcher error")
			}
		case <-ctx.Done():
			logger.Info("Stopping pool config watcher")
			return
		}
	}
}

// LookupMaxConns returns max connections for a scaler/resource identifier.
// Keys are structured as <scaler>.<identifier>, e.g., "postgres.db1.analytics".
func LookupMaxConns(scalerType, identifier string) int32 {
	key := fmt.Sprintf("%s.%s", scalerType, identifier)
	if val, ok := globalOverrides.Load(key); ok {
		return val.(int32)
	}
	return 0
}
