package cadvisor

import (
	"sort"
	"strings"
	"testing"

	m "github.com/gnydick/metric-scraper/metric"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const nodeName = "node-a"

func newTestDataSet() *DataSet {
	return NewDataSet(&v1.Node{ObjectMeta: metav1.ObjectMeta{Name: nodeName}})
}

// machineMetric is a machine_ line as the emitter hands it over: its only tag is the node.
func machineMetric(name string) *m.Metric {
	return &m.Metric{Metric: name, Value: 4, Tags: map[string]string{"node": nodeName}}
}

// nodeLevelMetric is a container_ line for the root cgroup, which the data set files under the
// node as node_<rest>.
func nodeLevelMetric(name string) *m.Metric {
	return &m.Metric{Metric: name, Value: 1, Tags: map[string]string{
		"container_name": "", "id": "/", "name": "", "image": "", "namespace": "", "pod_name": "", "node": nodeName,
	}}
}

// nodeMetricNames returns the names filed under the node, sorted.
func nodeMetricNames(ds *DataSet) string {
	node, ok := (*ds.GetNodes())[nodeName]
	if !ok {
		return "<no node>"
	}
	var names []string
	for name := range *node.GetMetrics() {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, " ")
}

// A line for a pod's own POD container is always kept and renamed pod_<rest>, however Go happens
// to order the tags (#47). Go orders a map differently from run to run, so the same line is
// registered many times; before the fix about half were lost.
func TestRegisterAlwaysKeepsAPodContainerLine(t *testing.T) {
	const tries = 200
	kept := 0
	for i := 0; i < tries; i++ {
		ds := newTestDataSet()
		ds.RegisterMetric(&m.Metric{Metric: "container_cpu_usage", Value: 1, Tags: map[string]string{
			"container_name": "POD", "id": "/kubepods/pod1/abc", "name": "k8s_POD_p", "image": "pause",
			"namespace": "default", "pod_name": "p", "node": nodeName,
		}})
		container, ok := (*ds.GetContainers())["POD"]
		if !ok {
			continue
		}
		if _, ok := (*container.GetMetrics())["pod_cpu_usage"]; ok {
			kept++
		}
	}
	if kept != tries {
		t.Errorf("kept the POD line, renamed pod_cpu_usage, %d times out of %d; want every time", kept, tries)
	}
}

// A machine metric is filed under its node even when it is the first line seen (#23).
func TestRegisterMachineMetricOnAFreshDataSet(t *testing.T) {
	for _, name := range []string{"machine_cpu_cores", "machine_memory_bytes"} {
		t.Run(name, func(t *testing.T) {
			ds := newTestDataSet()
			ds.RegisterMetric(machineMetric(name))
			if got := nodeMetricNames(ds); got != name {
				t.Errorf("node holds %q, want %q", got, name)
			}
		})
	}
}

// Registering does not depend on the order of the lines: both orders file the same two metrics
// under the node (#23).
func TestRegisterGivesTheSameNodeMetricsInEitherOrder(t *testing.T) {
	const want = "machine_cpu_cores node_cpu_usage"

	nodeFirst := newTestDataSet()
	nodeFirst.RegisterMetric(nodeLevelMetric("container_cpu_usage"))
	nodeFirst.RegisterMetric(machineMetric("machine_cpu_cores"))
	if got := nodeMetricNames(nodeFirst); got != want {
		t.Errorf("node line first: node holds %q, want %q", got, want)
	}

	machineFirst := newTestDataSet()
	machineFirst.RegisterMetric(machineMetric("machine_cpu_cores"))
	machineFirst.RegisterMetric(nodeLevelMetric("container_cpu_usage"))
	if got := nodeMetricNames(machineFirst); got != want {
		t.Errorf("machine line first: node holds %q, want %q", got, want)
	}
}
