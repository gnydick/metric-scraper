//go:build merge

package targeting

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	c "github.com/gnydick/metric-scraper/config"
	"github.com/gnydick/metric-scraper/util/testsupport"
)

// stubCadvisor returns a Cadvisor target whose kubeconfig points at apiServerURL. It goes through
// config.FileBuild and the development kubeconfig path, the same way main builds one.
func stubCadvisor(t *testing.T, apiServerURL string) Cadvisor {
	t.Helper()
	dir := t.TempDir()

	kubeConfigPath := filepath.Join(dir, "kubeconfig.yaml")
	kubeConfig := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- name: stub
  cluster:
    server: %s
contexts:
- name: stub
  context:
    cluster: stub
    user: stub
current-context: stub
users:
- name: stub
  user: {}
`, apiServerURL)
	if err := os.WriteFile(kubeConfigPath, []byte(kubeConfig), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := c.FileBuild(testsupport.WriteConfigFile(t, map[string]interface{}{
		"optionals": map[string]interface{}{"development": map[string]interface{}{"path": kubeConfigPath}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	target, err := NewCadvisor(&cfg, "http", nil)
	if err != nil {
		t.Fatal(err)
	}
	return target
}

// An API server that never answers the node List must not hold EmitterPtrs past the caller's
// deadline (#7).
func TestEmitterPtrsEndsAtDeadlineWhenNodeListHangs(t *testing.T) {
	listed := make(chan struct{}, 1)
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case listed <- struct{}{}:
		default:
		}
		// Never answer: hold the request until the client gives up or the test ends.
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(func() {
		close(release)
		srv.Close()
	})

	target := stubCadvisor(t, srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	type result struct {
		count int
		err   error
	}
	done := make(chan result, 1)
	go func() {
		emitters, err := target.EmitterPtrs(ctx)
		done <- result{len(emitters), err}
	}()

	select {
	case got := <-done:
		// The observer is alive only if the request reached the server and hung there.
		select {
		case <-listed:
		default:
			t.Fatalf("EmitterPtrs returned (%d emitters, err=%v) without the node List reaching the server", got.count, got.err)
		}
		if !errors.Is(got.err, context.DeadlineExceeded) {
			t.Errorf("EmitterPtrs err = %v, want one wrapping context.DeadlineExceeded", got.err)
		}
		if got.count != 0 {
			t.Errorf("EmitterPtrs returned %d emitters from a List that never answered, want 0", got.count)
		}
	case <-time.After(30 * time.Second):
		// Generous on purpose: this bound only catches the hang, it is not the deadline under test.
		t.Fatal("EmitterPtrs still blocked 30s after a 300ms deadline")
	}
}

// Positive control for the stub wiring: the same path returns one emitter per listed node when the
// API server answers. Two nodes in, two emitters out.
func TestEmitterPtrsReturnsOneEmitterPerNode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/nodes" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"kind":"NodeList","apiVersion":"v1","metadata":{},"items":[`+
			`{"metadata":{"name":"node-a"}},{"metadata":{"name":"node-b"}}]}`)
	}))
	t.Cleanup(srv.Close)

	target := stubCadvisor(t, srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	emitters, err := target.EmitterPtrs(ctx)
	if err != nil {
		t.Fatalf("EmitterPtrs err = %v, want nil", err)
	}
	if len(emitters) != 2 {
		t.Fatalf("EmitterPtrs returned %d emitters, want 2", len(emitters))
	}
	names := []string{emitters[0].GetName(), emitters[1].GetName()}
	if names[0] != "node-a" || names[1] != "node-b" {
		t.Errorf("emitter names = %v, want [node-a node-b]", names)
	}
}
