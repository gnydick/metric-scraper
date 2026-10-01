//go:build merge

package sink

import (
	"io"
	"net"
	"strings"
	"testing"
	"time"

	m "github.com/gnydick/metric-scraper/metric"
	"github.com/gnydick/metric-scraper/util"
	"github.com/gnydick/metric-scraper/util/testsupport"
)

// Send writes the marshalled metric to OpenTSDB byte for byte, and its debug lines print it as is,
// even when the text holds a format verb (#13). The expected text comes from the put line layout
// "put <metric> <time> <value> <tags>\n" and the inputs, not from a run.
func TestSendWritesAndLogsMetricTextExactly(t *testing.T) {
	savedLevel := util.LogLevel
	util.LogLevel = util.DEBUG
	defer func() { util.LogLevel = savedLevel }()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	received := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			received <- "accept failed: " + err.Error()
			return
		}
		defer conn.Close()
		got, _ := io.ReadAll(conn)
		received <- string(got)
	}()

	metrics := make(chan *m.Metric)
	sink := &Opentsdb{endpoint: ln.Addr().String(), receiver: &metrics}

	// container_name=adminserver is the tag that turns the two debug lines on.
	metric := &m.Metric{
		Metric: "cpu%d",
		Time:   1000,
		Value:  1.5,
		Tags:   map[string]string{"container_name": "adminserver"},
	}
	const wantSent = "put cpu%d 1000 1.500000 container_name=adminserver\n"

	logged := testsupport.CaptureStdout(t, func() {
		sent := make(chan struct{})
		go func() {
			sink.Send()
			close(sent)
		}()
		metrics <- metric
		close(metrics)
		select {
		case <-sent:
		case <-time.After(30 * time.Second):
			// Generous on purpose: this only catches a hang.
			t.Error("Send did not return 30s after its channel closed")
		}
	})

	select {
	case got := <-received:
		if got != wantSent {
			t.Errorf("OpenTSDB received %q, want %q", got, wantSent)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("the stand-in OpenTSDB received nothing within 30s")
	}

	// The observer is alive only if the debug lines printed at all. Off Windows the library wraps
	// the level in colour codes, so the brackets are not next to it.
	if !strings.Contains(logged, "DEBUG") {
		t.Fatalf("no debug line was printed: %q", logged)
	}
	for _, want := range []string{
		"cpu%d map[container_name:adminserver]\n",
		wantSent,
	} {
		if !strings.Contains(logged, want) {
			t.Errorf("debug output %q does not hold %q", logged, want)
		}
	}
}

// Send reads what OpenTSDB writes back while it sends (#15). The stand-in writes far more reply
// text than a TCP connection can hold unread, so its writes finish only if Send is reading. The
// metric still arrives byte for byte, so reading does not get in the way of sending.
func TestSendReadsRepliesWhileSending(t *testing.T) {
	// Nothing is printed: this test is about the bytes on the wire, and the reply text is large.
	savedLevel := util.LogLevel
	util.LogLevel = util.FATAL
	defer func() { util.LogLevel = savedLevel }()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	// 64 MiB of reply lines: well past any loopback socket buffer.
	replyLine := []byte("put: illegal argument: " + strings.Repeat("x", 1000) + "\n")
	const replyLines = 64 * 1024

	repliesWritten := make(chan error, 1)
	received := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			repliesWritten <- err
			received <- ""
			return
		}
		defer conn.Close()
		go func() {
			for i := 0; i < replyLines; i++ {
				if _, err := conn.Write(replyLine); err != nil {
					repliesWritten <- err
					return
				}
			}
			repliesWritten <- nil
		}()
		got, _ := io.ReadAll(conn)
		received <- string(got)
	}()

	metrics := make(chan *m.Metric)
	sink := &Opentsdb{endpoint: ln.Addr().String(), receiver: &metrics}
	sent := make(chan struct{})
	go func() {
		sink.Send()
		close(sent)
	}()

	metrics <- &m.Metric{Metric: "cpu", Time: 1000, Value: 1.5, Tags: map[string]string{"host": "a"}}

	select {
	case err := <-repliesWritten:
		if err != nil {
			t.Fatalf("the stand-in OpenTSDB could not write its replies: %v", err)
		}
	case <-time.After(60 * time.Second):
		// Generous on purpose: an unread connection blocks the writer for good.
		t.Fatal("the stand-in OpenTSDB was still blocked writing replies after 60s: Send is not reading them")
	}

	close(metrics)
	select {
	case <-sent:
	case <-time.After(30 * time.Second):
		t.Fatal("Send did not return 30s after its channel closed")
	}

	select {
	case got := <-received:
		const want = "put cpu 1000 1.500000 host=a\n"
		if got != want {
			t.Errorf("OpenTSDB received %q, want %q", got, want)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("the stand-in OpenTSDB received nothing within 30s")
	}
}
