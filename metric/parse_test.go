package metric

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type unmarshal func(millis int64, line string) (*Metric, error)

// A line in the metric format becomes exactly the metric it states (#19). The expected name, tags
// and value are read off the line itself.
func TestUnmarshalParsesAMetricLine(t *testing.T) {
	const millis = 1000
	cases := []struct {
		name  string
		parse unmarshal
		line  string
		want  Metric
	}{
		{"cadvisor tagged", CadvUnmarshal, `container_cpu{id="/",image="x"} 12.5`,
			Metric{Metric: "container_cpu", Tags: map[string]string{"id": "/", "image": "x"}, Value: 12.5, Time: millis}},
		{"service tagged", SvcUnmarshal, `http_requests{code="200"} 3`,
			Metric{Metric: "http_requests", Tags: map[string]string{"code": "200"}, Value: 3, Time: millis}},
		{"cadvisor machine line has no tags", CadvUnmarshal, `machine_cpu_cores 4`,
			Metric{Metric: "machine_cpu_cores", Tags: map[string]string{}, Value: 4, Time: millis}},
		// A comma after the last tag is legal in the format; some clients write one on every line.
		{"cadvisor trailing comma", CadvUnmarshal, `jvm_threads{state="runnable",} 7.0`,
			Metric{Metric: "jvm_threads", Tags: map[string]string{"state": "runnable"}, Value: 7, Time: millis}},
		{"service trailing comma", SvcUnmarshal, `jvm_threads{state="runnable",} 7.0`,
			Metric{Metric: "jvm_threads", Tags: map[string]string{"state": "runnable"}, Value: 7, Time: millis}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.parse(millis, tc.line)
			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if got == nil {
				t.Fatal("got no metric and no error")
			}
			if !reflect.DeepEqual(*got, tc.want) {
				t.Errorf("got %+v, want %+v", *got, tc.want)
			}
		})
	}
}

// A line that is not in the metric format never becomes a metric: it gives ErrNotAMetric (#19).
// The two sample lines are the two in data/cadvisor.txt that today's pattern does not match.
func TestUnmarshalRefusesALineThatIsNotAMetric(t *testing.T) {
	cases := []struct {
		name  string
		parse unmarshal
		line  string
	}{
		{"cadvisor untagged, not a machine metric", CadvUnmarshal, `container_scrape_error 0`},
		{"cadvisor tag value with a space", CadvUnmarshal, `cadvisor_version_info{osVersion="Amazon Linux 2"} 1`},
		{"service untagged", SvcUnmarshal, `go_goroutines 12`},
		{"service does not take machine lines", SvcUnmarshal, `machine_cpu_cores 4`},
		{"empty", CadvUnmarshal, ``},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.parse(1000, tc.line)
			if !errors.Is(err, ErrNotAMetric) {
				t.Errorf("err = %v, want ErrNotAMetric", err)
			}
			if got != nil {
				t.Errorf("got a metric %+v from a line that is not one", *got)
			}
		})
	}
}

// A line in the metric format with a bad value or tag never becomes a metric and is not reported as
// "not a metric": it gives its own error, which the emitters log at ERROR (#19).
func TestUnmarshalRefusesAMetricLineWithABadValueOrTag(t *testing.T) {
	lines := []struct {
		name string
		line string
	}{
		{"value with two dots", `foo{a="b"} 1.2.3`},
		{"tag with no equals sign", `foo{a="b",c} 1`},
		{"tag value holding an equals sign", `foo{a="x=y"} 1`},
		{"tag value holding a comma", `foo{a="x,y"} 1`},
		// Only a comma after the last tag is accepted; an empty tag elsewhere is still bad.
		{"empty tag in the middle", `foo{a="b",,c="d"} 1`},
		{"nothing but a comma", `foo{,} 1`},
		{"two commas after the last tag", `foo{a="b",,} 1`},
		{"comma before the first tag", `foo{,a="b"} 1`},
	}
	parsers := []struct {
		name  string
		parse unmarshal
	}{
		{"cadvisor", CadvUnmarshal},
		{"service", SvcUnmarshal},
	}
	for _, p := range parsers {
		for _, tc := range lines {
			t.Run(p.name+"/"+tc.name, func(t *testing.T) {
				got, err := p.parse(1000, tc.line)
				if err == nil {
					t.Fatalf("err = nil, want an error (got %+v)", got)
				}
				if errors.Is(err, ErrNotAMetric) {
					t.Errorf("err = %v, want an error that is not ErrNotAMetric", err)
				}
				if got != nil {
					t.Errorf("got a metric %+v from a bad line", *got)
				}
			})
		}
	}
	// A bad value on a machine line is cadvisor-only.
	if got, err := CadvUnmarshal(1000, `machine_cpu_cores 1.2.3`); err == nil || errors.Is(err, ErrNotAMetric) || got != nil {
		t.Errorf("machine line with a bad value: got %v, err %v; want no metric and an error that is not ErrNotAMetric", got, err)
	}
}

// Regression pin, not coverage: the counts were measured on 2026-10-01 by running the patterns as
// they stood before #19 over data/cadvisor.txt (3905 tagged lines, 2 machine lines, 2 lines that
// matched nothing, no bad value, no bad tag). The parser must still sort that file the same way.
func TestCadvUnmarshalSortsTheSampleFileAsBefore(t *testing.T) {
	file, err := os.Open(filepath.Join("..", "data", "cadvisor.txt"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	var metrics, notMetrics, bad int
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		_, err := CadvUnmarshal(1000, line)
		switch {
		case err == nil:
			metrics++
		case errors.Is(err, ErrNotAMetric):
			notMetrics++
		default:
			bad++
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if metrics != 3907 || notMetrics != 2 || bad != 0 {
		t.Errorf("metrics=%d notMetrics=%d bad=%d, want 3907, 2, 0", metrics, notMetrics, bad)
	}
}
