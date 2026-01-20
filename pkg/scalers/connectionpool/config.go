package connectionpool

import (
	"context"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/atomic"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	configStore atomic.Value
	configPath  string
	logger      = log.Log.WithName("connectionpool")
)

// InitGlobalPoolConfig loads the YAML config and starts a watcher for live reloads.
func InitGlobalPoolConfig(ctx context.Context, path string) {
	configPath = path
	configStore.Store(make(map[string]string))
	loadConfig()
	go startConfigWatcher(ctx)
}

// loadConfig parses the YAML and updates globalOverrides safely.
func loadConfig() {
	data, err := os.ReadFile(configPath)
	if err != nil {
		logger.V(1).Info("Pool config file not found", "path", configPath, "err", err)
		configStore.Store(make(map[string]string))
		return
	}

	var parsed map[string]string
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		logger.Error(err, "Invalid pool config format", "path", configPath)
		return
	}

	// clear existing map before writing new values
	configStore.Store(parsed)

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

	configDir := filepath.Dir(configPath)
	if err := watcher.Add(configDir); err != nil {
		logger.Error(err, "Failed to add config file to watcher", "path", configPath)
		return
	}
	logger.Info("Started watching global connection pool configuration", "path", configPath)

	for {
		select {
		case event := <-watcher.Events:
			if filepath.Base(event.Name) == filepath.Base(configPath) && event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
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

func LookupConfigValue(key string) string {

	configMap := configStore.Load().(map[string]string)

	return configMap[key]
}
