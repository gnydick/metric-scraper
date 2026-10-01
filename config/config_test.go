package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gnydick/metric-scraper/util/testsupport"
)

// A complete config loads, and each accessor gives back what the file states.
func TestFileBuildLoadsACompleteConfig(t *testing.T) {
	path := testsupport.WriteConfigFile(t, map[string]interface{}{
		"debug": true, "kind": "service", "disco": "d", "ident": "i", "deploymentId": "dep",
		"interval": "90s", "orch": "o", "metric": "m", "mode": "deployed",
		"optionals": map[string]interface{}{"development": map[string]interface{}{"path": "/kube/config"}},
	})

	cfg, err := FileBuild(path)
	if err != nil {
		t.Fatalf("FileBuild err = %v, want nil", err)
	}
	if !cfg.Debug() || cfg.Kind() != KindService || cfg.Disco() != "d" || cfg.Ident() != "i" ||
		cfg.DeploymentId() != "dep" || cfg.Orch() != "o" || cfg.Metric() != "m" ||
		cfg.Sink() != SinkOpentsdb || cfg.Mode() != ModeDeployed {
		t.Errorf("loaded config %+v does not match the file", cfg)
	}
	if cfg.Interval() != 90*time.Second {
		t.Errorf("Interval() = %v, want 1m30s", cfg.Interval())
	}
	if got := (*cfg.Optionals())["development"]["path"]; got != "/kube/config" {
		t.Errorf("optionals development path = %q, want /kube/config", got)
	}
}

// The two sample configs in the repo still load: validation must not refuse a config that works
// today.
func TestFileBuildLoadsTheSampleConfigs(t *testing.T) {
	for _, name := range []string{"cadvisor.json", "ksm.json"} {
		if _, err := FileBuild(filepath.Join("..", name)); err != nil {
			t.Errorf("FileBuild(%s) err = %v, want nil", name, err)
		}
	}
}

// The cadvisor kind does not read disco, so an empty one is not a bad config.
func TestFileBuildAllowsAnEmptyDiscoForTheCadvisorKind(t *testing.T) {
	path := testsupport.WriteConfigFile(t, map[string]interface{}{"kind": "cadvisor", "disco": ""})
	if _, err := FileBuild(path); err != nil {
		t.Errorf("FileBuild err = %v, want nil", err)
	}
}

// The optionals block may be left out.
func TestFileBuildAllowsMissingOptionals(t *testing.T) {
	path := testsupport.WriteConfigFile(t, map[string]interface{}{"optionals": nil})
	if _, err := FileBuild(path); err != nil {
		t.Errorf("FileBuild err = %v, want nil", err)
	}
}

// A bad config is refused with a validation error that names the field. It never panics and never
// returns a usable config (docs/dictated-specs/config.md, Bad config at startup; #17).
func TestFileBuildRefusesABadConfig(t *testing.T) {
	cases := []struct {
		name    string
		changes map[string]interface{}
		field   string
	}{
		{"debug missing", map[string]interface{}{"debug": nil}, "debug"},
		{"kind missing", map[string]interface{}{"kind": nil}, "kind"},
		{"disco missing", map[string]interface{}{"disco": nil}, "disco"},
		{"ident missing", map[string]interface{}{"ident": nil}, "ident"},
		{"deploymentId missing", map[string]interface{}{"deploymentId": nil}, "deploymentId"},
		{"interval missing", map[string]interface{}{"interval": nil}, "interval"},
		{"orch missing", map[string]interface{}{"orch": nil}, "orch"},
		{"metric missing", map[string]interface{}{"metric": nil}, "metric"},
		{"sink missing", map[string]interface{}{"sink": nil}, "sink"},
		{"mode missing", map[string]interface{}{"mode": nil}, "mode"},
		{"debug is not a boolean", map[string]interface{}{"debug": "yes"}, "debug"},
		{"kind is not a string", map[string]interface{}{"kind": 3}, "kind"},
		{"unknown kind", map[string]interface{}{"kind": "nodes"}, "kind"},
		{"unknown sink", map[string]interface{}{"sink": "influx"}, "sink"},
		{"unknown mode", map[string]interface{}{"mode": "deploy"}, "mode"},
		{"interval with no unit", map[string]interface{}{"interval": "5"}, "interval"},
		{"interval of zero", map[string]interface{}{"interval": "0s"}, "interval"},
		{"negative interval", map[string]interface{}{"interval": "-5s"}, "interval"},
		{"optionals of the wrong shape", map[string]interface{}{"optionals": "none"}, "optionals"},
		// A field may be empty only when nothing reads it: the service kind looks its target up
		// by disco, and the opentsdb sink looks OpenTSDB up by metric.
		{"service kind with an empty disco", map[string]interface{}{"kind": "service", "disco": ""}, "disco"},
		{"opentsdb sink with an empty metric", map[string]interface{}{"metric": ""}, "metric"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := testsupport.WriteConfigFile(t, tc.changes)
			cfg, err := FileBuild(path)
			if err == nil {
				t.Fatal("FileBuild err = nil, want a validation error")
			}
			// No usable config comes back: every field is at its zero value.
			if cfg.Kind() != "" || cfg.Sink() != "" || cfg.Mode() != "" || cfg.Interval() != 0 {
				t.Errorf("FileBuild returned a filled config %+v alongside the error", cfg)
			}
			if !strings.Contains(err.Error(), `"`+tc.field+`"`) {
				t.Errorf("error %q does not name the field %q", err, tc.field)
			}
			if !strings.Contains(err.Error(), path) {
				t.Errorf("error %q does not name the config file", err)
			}
		})
	}
}

// A config file that is missing or is not JSON is refused with an error that names the file.
func TestFileBuildRefusesAnUnreadableFile(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "nope.json")
	notJSON := filepath.Join(dir, "broken.json")
	if err := os.WriteFile(notJSON, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Valid JSON, but a list where the config object should be.
	notAnObject := filepath.Join(dir, "list.json")
	if err := os.WriteFile(notAnObject, []byte(`["cadvisor"]`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{missing, notJSON, notAnObject} {
		_, err := FileBuild(path)
		if err == nil {
			t.Errorf("FileBuild(%s) err = nil, want an error", path)
			continue
		}
		if !strings.Contains(err.Error(), path) {
			t.Errorf("error %q does not name the config file %s", err, path)
		}
	}
}
