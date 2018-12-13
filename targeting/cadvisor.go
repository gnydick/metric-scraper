package targeting

import (
	"fmt"

	c "github.com/gnydick/metric-scraper/config"
	e "github.com/gnydick/metric-scraper/emitters"
	k "github.com/gnydick/metric-scraper/sink"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Cadvisor struct {
	configPtr *c.Config
	scheme    string
	sink   k.Sink
}

func NewCadvisor(configPtr *c.Config, scheme string, sinkPtr k.Sink) (Cadvisor) {
	svcTarget := Cadvisor{
		configPtr: configPtr,
		scheme:    scheme,
	}
	return svcTarget
}

func (c Cadvisor) GetConfig() (config *c.Config) {
	return
}

func (c Cadvisor) EmitterPtrs() ([]e.Emitter) {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	nodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	emitters := make([]e.Emitter, len(nodes.Items))
	for i, node := range nodes.Items {
		emitter := e.NewCadvisor(c.sink, c.configPtr,
			fmt.Sprintf("http://%s:%s/metrics/cadvisor", node.ObjectMeta.Name, "10255"),
				"node="+node.Name)
		emitters[i] = emitter
	}
	return emitters
}

// processHttp(fmt.Sprintf("http://%s:%s/metrics", node.ObjectMeta.Name, port)) TODO make as separate type of emitter
