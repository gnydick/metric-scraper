// Package testsupport holds helpers shared by this module's tests.
package testsupport

import (
	"io"
	"os"
	"testing"
)

// CaptureStdout runs fn and returns what it wrote to os.Stdout. Unknwon/log prints with fmt.Printf,
// so stdout is the path the user's log output takes.
func CaptureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	saved := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = saved }()
	fn()
	w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}
