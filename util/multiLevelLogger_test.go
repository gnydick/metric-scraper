package util

import (
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout runs fn and returns what it wrote to os.Stdout. Unknwon/log prints with fmt.Printf,
// so stdout is the path the user's log output takes.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
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

// Each logger forwards its args to the format string. The expected text comes from the format and
// the argument, not from a run: "count=%d" with 5 is "count=5".
func TestLogFuncsForwardFormatArgs(t *testing.T) {
	savedLevel := LogLevel
	defer func() { LogLevel = savedLevel }()

	cases := []struct {
		name  string
		level Level
		log   func(string, ...interface{})
	}{
		{"DebugLog", DEBUG, DebugLog},
		{"InfoLog", INFO, InfoLog},
		{"WarningLog", WARNING, WarningLog},
		{"ErrorLog", ERROR, ErrorLog},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			LogLevel = tc.level
			out := captureStdout(t, func() { tc.log("count=%d", 5) })
			// The observer is alive only if the line was printed at all.
			if out == "" {
				t.Fatalf("%s printed nothing at LogLevel %d", tc.name, tc.level)
			}
			if !strings.Contains(out, "count=5\n") {
				t.Errorf("%s printed %q, want a line ending in \"count=5\"", tc.name, out)
			}
		})
	}
}
