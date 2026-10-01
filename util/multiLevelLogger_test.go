package util

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
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

// A message prints when its level is at or above LogLevel (#9). The expectation comes from the
// order of the Level constants, not from a run.
func TestLogFuncsPrintAtOrAboveLogLevel(t *testing.T) {
	savedLevel := LogLevel
	defer func() { LogLevel = savedLevel }()

	loggers := []struct {
		name  string
		level Level
		log   func(string, ...interface{})
	}{
		{"DebugLog", DEBUG, DebugLog},
		{"InfoLog", INFO, InfoLog},
		{"WarningLog", WARNING, WarningLog},
		{"ErrorLog", ERROR, ErrorLog},
	}
	for _, logLevel := range []Level{DEBUG, INFO, WARNING, ERROR, FATAL} {
		for _, lg := range loggers {
			want := lg.level >= logLevel
			t.Run(fmt.Sprintf("LogLevel=%d/%s", logLevel, lg.name), func(t *testing.T) {
				LogLevel = logLevel
				out := captureStdout(t, func() { lg.log("marker") })
				got := strings.Contains(out, "marker\n")
				if got != want {
					t.Errorf("%s at LogLevel %d: printed=%v, want %v (output %q)", lg.name, logLevel, got, want, out)
				}
			})
		}
	}
}

// fatalChildEnv names the LogLevel the re-run test binary sets before it calls FatalLog.
const fatalChildEnv = "METRIC_SCRAPER_FATALLOG_CHILD_LEVEL"

// FatalLog logs and exits whenever it is called (#9). It calls os.Exit, so the call runs in a
// re-run of this test binary. INFO is the default LogLevel; DEBUG is the only other one main sets.
func TestFatalLogLogsAndExits(t *testing.T) {
	if child := os.Getenv(fatalChildEnv); child != "" {
		if child == "DEBUG" {
			LogLevel = DEBUG
		}
		FatalLog("boom=%d", 7)
		// Reached only if FatalLog did not exit.
		fmt.Println("survived FatalLog")
		return
	}

	for _, level := range []string{"INFO", "DEBUG"} {
		t.Run(level, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestFatalLogLogsAndExits$")
			// Only the child switch: nothing inherited can steer the child elsewhere.
			cmd.Env = []string{fatalChildEnv + "=" + level}
			out, err := cmd.CombinedOutput()

			// The observer is alive only if the child logged the message.
			if !strings.Contains(string(out), "boom=7\n") {
				t.Errorf("child printed %q, want a line ending in \"boom=7\"", out)
			}
			if strings.Contains(string(out), "survived FatalLog") {
				t.Errorf("child carried on after FatalLog: %q", out)
			}
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) {
				t.Fatalf("child did not exit non-zero: err=%v, output %q", err, out)
			}
		})
	}
}
