package cadvisor

import (
	m "github.com/gnydick/metric-scraper/metric"
)

type Pod struct {
	podName    string
	id         string
	nannyName  string
	nannyImage string
	containers map[string]*Container
}

type Container struct {
	containerName string
	pod           *Pod
	id            string
	image         string
	name          string
}

type DataSet struct {
	containers map[string]*Container
	pods       map[string]*Pod
	metrics    []*m.Metric
}

func NewDataSet() *DataSet {
	ds := DataSet{}
	return &ds
}

func (ds DataSet) RegisterMetric(metric *m.Metric) {
	tags := &metric.Tags
	for k, v := range *tags {
		switch key := k; key {
		case "container_name":
			containerName := v
			if len(containerName) > 0 {
				container := ds.getOrCreateContainer(&containerName)
				ds.fixUpContainer(container, metric)
			}

		case "pod_name":
			podName := v
			if len(podName) > 0 {
				pod := ds.getOrCreatePod(&podName)
				ds.fixUpPod(pod, metric)
			}
		}
	}
}

func (ds DataSet) fixUpContainer(container *Container, metric *m.Metric) {

}

func (ds DataSet) fixUpPod(pod *Pod, metric *m.Metric) {

}

func (ds DataSet) getOrCreatePod(name *string) *Pod {
	for k, v := range ds.pods {
		if k == *name {
			return v
		}
	}
	newPptr := ds.newPod(*name)
	ds.pods[*name] = newPptr
	return newPptr
}

func (ds DataSet) getOrCreateContainer(name *string) *Container {
	for k, v := range ds.containers {
		if k == *name {
			return v
		}
	}
	newCptr := ds.newContainer(*name)
	ds.containers[*name] = newCptr
	return newCptr
}

func (ds DataSet) podExists(name *string) bool {
	for k := range ds.pods {
		if k == *name {
			return true
		}
	}
	return false
}

func (ds DataSet) newContainer(containerName string) *Container {
	cntr := Container{
		containerName: containerName,
	}
	return &cntr
}

func (ds DataSet) newPod(podName string) *Pod {
	pod := Pod{
		podName: podName,
	}
	return &pod
}
