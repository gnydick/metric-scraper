package targeting

import (
	"path/filepath"
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

	_, err = NewCadvisor(&cfg, "http", nil)
	if err == nil {
		t.Fatal("NewCadvisor err = nil for a kubeconfig file that does not exist, want an error")
	}
}
