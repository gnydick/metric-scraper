package targeting

import (
    c "github.com/gnydick/metric-scraper/config"
    e "github.com/gnydick/metric-scraper/emitters"
    k "github.com/gnydick/metric-scraper/sink"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/tools/clientcmd"
)

type Cadvisor struct {
    configPtr *c.Config
    scheme    string
    sink      k.Sink
}

func NewCadvisor(configPtr *c.Config, scheme string, sink k.Sink) (Cadvisor) {
    svcTarget := Cadvisor{
        configPtr: configPtr,
        scheme:    scheme,
        sink:      sink,
    }
    return svcTarget
}

func (c Cadvisor) GetConfig() (config *c.Config) {
    return
}

// http://k8s.io/client-go/tools/clientcmd.BuildConfigFromFlags()

func (c Cadvisor) getK8sConfig() *rest.Config {
    var config *rest.Config
    var _err error
    if c.configPtr.Mode() == "deployed" {
        config, _err = rest.InClusterConfig()
        if _err != nil {
            panic(_err.Error())
        }

    } else {
        kubeConfigPtr := (*c.configPtr.Optionals())["development"]
        config, _err = clientcmd.BuildConfigFromFlags("", kubeConfigPtr["path"])
    }

    return config

}

func (c Cadvisor) EmitterPtrs() ([]e.Emitter) {
    config := c.getK8sConfig()

    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        panic(err.Error())
    }

    nodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
    if err != nil {
        panic(err.Error())
    }
    emitters := make([]e.Emitter, len(nodes.Items))
    // emitters := make([]e.Emitter, 1)
    for i, node := range nodes.Items {
        newInst := node // have to create a new instance as 'node' gets destroyed in each loop
        // if node.Name == "ip-10-90-8-99.us-west-2.compute.internal" {
        emitter := e.NewCadvisor(c.sink, c.configPtr, &newInst)
        // emitters[0] = emitter
        emitters[i] = emitter
        // }
    }
    return emitters
}
