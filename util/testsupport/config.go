package testsupport

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// WriteConfigFile writes a complete, valid scraper config file into a temp directory and returns
// its path. Each entry of changes replaces that field; a nil value removes the field.
func WriteConfigFile(t *testing.T, changes map[string]interface{}) string {
	t.Helper()
	fields := map[string]interface{}{
		"debug":        false,
		"kind":         "cadvisor",
		"disco":        "",
		"ident":        "node",
		"deploymentId": "test",
		"interval":     "5s",
		"orch":         "",
		"metric":       "_telnet._tcp.opentsdb.test.invalid",
		"sink":         "opentsdb",
		"mode":         "development",
		"optionals":    map[string]interface{}{"development": map[string]interface{}{"path": ""}},
	}
	for name, value := range changes {
		if value == nil {
			delete(fields, name)
		} else {
			fields[name] = value
		}
	}
	text, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, text, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
