package testsupport

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Page GETs path from handler and returns the body. It fails the test unless the answer is 200.
// Tests of the metrics page read it this way, the same path a Prometheus server takes.
func Page(t *testing.T, handler http.Handler, path string) string {
	t.Helper()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, want 200", path, rec.Code)
	}
	return rec.Body.String()
}

// HasLine says whether text holds line as one whole line.
func HasLine(text string, line string) bool {
	for _, got := range strings.Split(text, "\n") {
		if got == line {
			return true
		}
	}
	return false
}
