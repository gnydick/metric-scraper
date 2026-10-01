package sink

import (
	"strings"
	"testing"

	"github.com/gnydick/metric-scraper/util/testsupport"
)

// rejectionLines returns the logged lines that report an OpenTSDB rejection.
func rejectionLines(logged string) []string {
	var lines []string
	for _, line := range strings.Split(logged, "\n") {
		if strings.Contains(line, "OpenTSDB rejected a metric: ") {
			lines = append(lines, line)
		}
	}
	return lines
}

// Each line OpenTSDB writes back is a rejected put. Each one is logged at ERROR, as written (#15).
// Two reply lines and one blank line in, two log lines out. The default LogLevel is in force.
func TestLogRepliesLogsEachRejectionAtError(t *testing.T) {
	replies := "put: illegal argument: Invalid metric name (\"cpu%d\"): illegal character: %\n" +
		"\n" +
		"put: unknown metric: No such name for 'metrics': 'cpu'\r\n"

	logged := testsupport.CaptureStdout(t, func() { logReplies(strings.NewReader(replies)) })

	got := rejectionLines(logged)
	if len(got) != 2 {
		t.Fatalf("logged %d rejection lines, want 2: %q", len(got), logged)
	}
	wantEnds := []string{
		"OpenTSDB rejected a metric: put: illegal argument: Invalid metric name (\"cpu%d\"): illegal character: %",
		"OpenTSDB rejected a metric: put: unknown metric: No such name for 'metrics': 'cpu'",
	}
	for i, want := range wantEnds {
		if !strings.HasSuffix(got[i], want) {
			t.Errorf("rejection line %d = %q, want it to end with %q", i, got[i], want)
		}
		if !strings.Contains(got[i], "ERROR") {
			t.Errorf("rejection line %d = %q is not at ERROR", i, got[i])
		}
	}
}

// A reply line longer than the read buffer must not stop the reader: the reply after it is still
// logged. One over-long line and one short line in, two log lines out.
func TestLogRepliesKeepsReadingAfterAnOverlongLine(t *testing.T) {
	replies := "put: " + strings.Repeat("x", 200000) + "\n" + "put: second\n"

	logged := testsupport.CaptureStdout(t, func() { logReplies(strings.NewReader(replies)) })

	got := rejectionLines(logged)
	if len(got) != 2 {
		t.Fatalf("logged %d rejection lines, want 2", len(got))
	}
	if !strings.HasSuffix(got[1], "OpenTSDB rejected a metric: put: second") {
		t.Errorf("second rejection line = %q, want it to end with the second reply", got[1])
	}
	// The over-long reply is cut at 1024 bytes. The log line adds its prefix, timestamp and level.
	if len(got[0]) > 1024+256 {
		t.Errorf("the over-long reply was logged as %d bytes, want at most 1024 of it", len(got[0]))
	}
}
