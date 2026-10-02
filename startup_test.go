package main

import (
	"strings"
	"testing"

	"github.com/gnydick/metric-scraper/util/testsupport"
)

// A bad config makes startup fail with a validation error that names the field. It does not panic
// (docs/dictated-specs/config.md, Bad config at startup; #17).
func TestStartupFailsOnABadConfig(t *testing.T) {
	path := testsupport.WriteConfigFile(t, map[string]interface{}{"kind": "nodes"})

	scraper, _, err := startup(path)
	if err == nil {
		t.Fatal("startup err = nil for a config with an unknown kind, want a validation error")
	}
	if scraper != nil {
		t.Error("startup returned a scraper for a bad config")
	}
	if !strings.Contains(err.Error(), `"kind"`) || !strings.Contains(err.Error(), path) {
		t.Errorf("error %q does not name the field and the config file", err)
	}
}
