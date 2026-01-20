package connectionpool

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Start -> Missing -> Create -> Update
func TestGlobalPoolConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "pool-config.yaml")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	InitGlobalPoolConfig(ctx, configFile)
	if val := LookupConfigValue("postgres.db"); val != "" {
		t.Errorf("Expected empty string for missing file, got '%s'", val)
	}

	err := os.WriteFile(configFile, []byte(`postgres.db: "10"`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	loadConfig()

	if val := LookupConfigValue("postgres.db"); val != "10" {
		t.Errorf("Expected '10', got '%s'", val)
	}

	err = os.WriteFile(configFile, []byte(`postgres.db: "50"`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	success := false
	for i := 0; i < 20; i++ { // Try for 2 seconds
		if LookupConfigValue("postgres.db") == "50" {
			success = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if !success {
		t.Errorf("Watcher failed to reload config. Expected '50', got '%s'", LookupConfigValue("postgres.db"))
	}
}
