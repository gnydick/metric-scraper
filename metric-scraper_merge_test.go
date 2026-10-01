//go:build merge

package main

import (
	"strings"
	"testing"

	"github.com/gnydick/metric-scraper/util/testsupport"
)

// A valid config whose OpenTSDB address cannot be looked up makes startup fail with an error that
// names the address. It does not panic (#17). The .invalid top-level domain never resolves.
func TestStartupFailsWhenTheSinkCannotBeLookedUp(t *testing.T) {
	const address = "_telnet._tcp.opentsdb.invalid"
	path := testsupport.WriteConfigFile(t, map[string]interface{}{"metric": address})

	scraper, err := startup(path)
	if err == nil {
		t.Fatal("startup err = nil for an OpenTSDB address that does not resolve, want an error")
	}
	if scraper != nil {
		t.Error("startup returned a scraper with no sink")
	}
	if !strings.Contains(err.Error(), address) {
		t.Errorf("error %q does not name the address %q", err, address)
	}
}
