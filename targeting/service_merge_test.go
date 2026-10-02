//go:build merge

package targeting

import (
	"context"
	"strings"
	"testing"
	"time"

	c "github.com/gnydick/metric-scraper/config"
	"github.com/gnydick/metric-scraper/util/testsupport"
)

// A service whose address cannot be looked up makes EmitterPtrs return an error that names the
// address, so the round is skipped and logged. It does not panic (#17). The .invalid top-level
// domain never resolves.
func TestServiceEmitterPtrsReturnsAnErrorWhenTheLookupFails(t *testing.T) {
	const address = "_http._tcp.some-service.invalid"
	cfg, err := c.FileBuild(testsupport.WriteConfigFile(t, map[string]interface{}{
		"kind": "service", "disco": address,
	}))
	if err != nil {
		t.Fatal(err)
	}
	target := NewService(&cfg, "http", nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	emitters, err := target.EmitterPtrs(ctx)
	if err == nil {
		t.Fatalf("EmitterPtrs err = nil for an address that does not resolve (%d emitters), want an error", len(emitters))
	}
	if len(emitters) != 0 {
		t.Errorf("EmitterPtrs returned %d emitters alongside the error, want none", len(emitters))
	}
	if !strings.Contains(err.Error(), address) {
		t.Errorf("error %q does not name the address %q", err, address)
	}
}
