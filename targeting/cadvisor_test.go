package targeting

import (
	"path/filepath"
	"strings"
	"testing"

	c "github.com/gnydick/metric-scraper/config"
	"github.com/gnydick/metric-scraper/util/testsupport"
)

// A kubeconfig that cannot be loaded makes NewCadvisor return an error. It does not panic, at
// startup or in a later round (#17).
func TestNewCadvisorRefusesAKubeconfigThatCannotBeLoaded(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-kubeconfig.yaml")
	cfg, err := c.FileBuild(testsupport.WriteConfigFile(t, map[string]interface{}{
		"optionals": map[string]interface{}{"development": map[string]interface{}{"path": missing}},
	}))
	if err != nil {
		t.Fatal(err)
	}

	// The helper's config runs in development mode, the mode that reads a kubeconfig file.
	if cfg.Mode() != c.ModeDevelopment {
		t.Fatalf("test config has mode %q, want development", cfg.Mode())
	}

	_, err = NewCadvisor(&cfg, "http", nil, nil)
	if err == nil {
		t.Fatal("NewCadvisor err = nil for a kubeconfig file that does not exist, want an error")
	}
	// The error is about that file, not about something else.
	if !strings.Contains(err.Error(), missing) {
		t.Errorf("error %q does not name the kubeconfig file %s", err, missing)
	}
}
