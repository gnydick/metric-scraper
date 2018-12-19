package cadvisor

import (
    m "github.com/gnydick/metric-scraper/metric"
    "regexp"
)

type Node struct {
    name string
    metrics  map[string]*m.Metric
}

type Pod struct {
    podName    string
    containers map[string]*Container
}

type Container struct {
    containerName string
    pod           *Pod
    image         string
    name          string
    metrics       map[string]*m.Metric
}

func (c Container) GetMetrics() (*map[string]*m.Metric) {
    return &c.metrics
}

func (n Node) GetMetrics() (*map[string]*m.Metric) {
    return &n.metrics
}

type DataSet struct {
    containers map[string]*Container
    pods       map[string]*Pod
    nodes      map[string]*Node
}

func NewDataSet() *DataSet {
    ds := DataSet{
        containers: make(map[string]*Container),
        pods:       make(map[string]*Pod),
        nodes:      make(map[string]*Node),
    }
    return &ds
}

func (ds *DataSet) RegisterMetric(metric *m.Metric) {
    tags := &metric.Tags
    for k, v := range *tags {
        switch key := k; key {
        case "container_name":
            containerName := v
            if len(containerName) > 0 {
                switch cName := containerName; cName {
                case "POD":
                    metricName := (*metric).Metric
                    re := regexp.MustCompile(`(?P<container>container)_(?P<theRest>.*)`)
                    matches := re.FindStringSubmatchIndex(metricName)
                    if matches != nil {
                        var newMetricNameBytes []byte
                        (*metric).Metric = string(re.ExpandString(newMetricNameBytes, "pod_${theRest}", metricName, matches))
                    }

                }
                container := (*ds).getOrCreateContainer(&containerName)
                (*ds).fixUpContainer(container, metric)
            } else {
                if ds.hasTagKey("id", metric) && ds.getTagValue("id", metric) == "/" {
                    if ds.hasTagKey("name", metric) && ds.getTagValue("name", metric) == "" {
                        if ds.hasTagKey("image", metric) && ds.getTagValue("image", metric) == "" {
                            if ds.hasTagKey("namespace", metric) && ds.getTagValue("namespace", metric) == "" {
                                if ds.hasTagKey("pod_name", metric) && ds.getTagValue("pod_name", metric) == "" {
                                    if ds.hasTagKey("node", metric) && len(ds.getTagValue("node", metric)) > 0 {
                                        metricName := (*metric).Metric
                                        re := regexp.MustCompile(`(?P<container>container)_(?P<theRest>.*)`)
                                        matches := re.FindStringSubmatchIndex(metricName)
                                        if matches != nil {
                                            var newMetricNameBytes []byte
                                            (*metric).Metric = string(re.ExpandString(newMetricNameBytes, "node_${theRest}", metricName, matches))
                                            nodeName := (*ds).getTagValue("node", metric)
                                            node := (*ds).getOrCreateNode(&nodeName)
                                            (*ds).fixUpNode(node, metric)
                                            (*ds).nodes[nodeName] = node
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }

        case "pod_name":
            podName := v
            if len(podName) > 0 {
                pod := (*ds).getOrCreatePod(&podName)
                (*ds).fixUpPod(pod, metric)
            }
        }
    }
}

func (ds *DataSet) hasTagKey(key string, metric *m.Metric) bool {
    for k := range (*metric).Tags {
        if k == key {
            return true
        }
    }
    return false
}

func (ds *DataSet) getTagValue(key string, metric *m.Metric) string {
    tags := &(*metric).Tags
    return (*tags)[key]

}

func (ds *DataSet) fixUpNode(node *Node, metric *m.Metric) {
    delete((*metric).Tags, "id")
    delete((*metric).Tags, "name")
    delete((*metric).Tags, "image")
    delete((*metric).Tags, "namespace")
    delete((*metric).Tags, "pod_name")
    delete((*metric).Tags, "container_name")
    node.metrics[(*metric).Metric] = metric
}


func (ds *DataSet) fixUpContainer(container *Container, metric *m.Metric) {
    container.metrics[metric.Metric] = metric
    if (*ds).hasTagKey("pod_name", metric) && container.pod == nil {
        tags := &(*metric).Tags
        podName := (*tags)["pod_name"]
        pod := (*ds).getOrCreatePod(&podName)
        container.pod = pod
        (*pod).containers[container.containerName] = container
    }

    if (*ds).hasTagKey("image", metric) && len(container.image) == 0 {
        container.image = (*ds).getTagValue("image", metric)
    }

    if (*ds).hasTagKey("name", metric) && len(container.image) == 0 {
        container.name = (*ds).getTagValue("name", metric)
    }

    if (*ds).hasTagKey("container_name", metric) && (*ds).getTagValue("container_name", metric) == "POD" {
        delete((*metric).Tags, "container_name")
    }

    if (*ds).hasTagKey("id", metric) {
        delete((*metric).Tags, "id")
    }

}

func (ds *DataSet) fixUpPod(pod *Pod, metric *m.Metric) {
    if (*ds).hasTagKey("container_name", metric) && len((*ds).getTagValue("container_name", metric)) > 0 {
        if (*ds).hasTagKey("pod_name", metric) {
            pod.podName = (*ds).getTagValue("pod_name", metric)
        }
    }

    if (*ds).hasTagKey("container_name", metric) && (*ds).getTagValue("container_name", metric) == "POD" {
        delete((*metric).Tags, "container_name")
    }
}

func (ds *DataSet) getOrCreatePod(name *string) *Pod {
    for k, v := range (*ds).pods {
        if k == *name {
            return v
        }
    }
    newPptr := (*ds).newPod(*name)
    (*ds).pods[*name] = newPptr
    return newPptr
}

func (ds *DataSet) getOrCreateContainer(name *string) *Container {
    for k, v := range (*ds).containers {
        if k == *name {
            return v
        }
    }
    newCptr := (*ds).newContainer(*name)
    (*ds).containers[*name] = newCptr
    return newCptr
}

func (ds *DataSet) podExists(name *string) bool {
    for k := range (*ds).pods {
        if k == *name {
            return true
        }
    }
    return false
}

func (ds *DataSet) newContainer(containerName string) *Container {
    cntr := Container{
        containerName: containerName,
        metrics:       make(map[string]*m.Metric),
    }
    return &cntr
}

func (ds *DataSet) newPod(podName string) *Pod {
    pod := Pod{
        podName:    podName,
        containers: make(map[string]*Container),
    }
    return &pod
}

func (ds DataSet) GetContainers() *map[string]*Container {
    return &ds.containers
}


func (ds DataSet) GetNodes() *map[string]*Node {
    return &ds.nodes
}

func (ds *DataSet) getOrCreateNode(nodeName *string) *Node {
    for k, v := range (*ds).nodes {
        if k == *nodeName {
            return v
        }

    }
    newNodePtr := (*ds).newNode(*nodeName)
    (*ds).nodes[*nodeName] = newNodePtr
    return newNodePtr
}
func (ds *DataSet) newNode(nodeName string) *Node {
    node := Node{
        name: nodeName,
        metrics:  make(map[string]*m.Metric),
    }
    return &node
}
