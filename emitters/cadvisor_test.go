package emitters

import (
	"context"
	"strings"
	"testing"

	"github.com/gnydick/metric-scraper/util/testsupport"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// A failed scan logs its error text as is, even when the text holds a format verb (#13). Port 0
// cannot be dialled, so the GET fails at once and the error names the URL, which holds "%20".
func TestScanLogsFetchErrorTextExactly(t *testing.T) {
	const url = "http://127.0.0.1:0/a%20b"
	emitter := Cadvisor{
		url:  url,
		node: &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-a"}},
	}

	logged := testsupport.CaptureStdout(t, func() { emitter.Scan(context.Background()) })

	// The observer is alive only if an error line was printed at all. Off Windows the library wraps
	// the level in colour codes, so the brackets are not next to it.
	if !strings.Contains(logged, "ERROR") {
		t.Fatalf("no error line was printed: %q", logged)
	}
	if !strings.Contains(logged, url) {
		t.Errorf("error line %q does not hold the URL %q as written", logged, url)
	}
}
