package targeting

import (
	"context"
	"fmt"

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
	clientset kubernetes.Interface
}

// NewCadvisor builds the Kubernetes client once, from the config. A client config that cannot be
// built is an error for the caller to report at startup, never a panic (#17).
func NewCadvisor(configPtr *c.Config, scheme string, sink k.Sink) (Cadvisor, error) {
	restConfig, err := k8sConfig(configPtr)
	if err != nil {
		return Cadvisor{}, fmt.Errorf("kubernetes client config for mode %q: %w", configPtr.Mode(), err)
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return Cadvisor{}, fmt.Errorf("kubernetes client: %w", err)
	}
	return Cadvisor{
		configPtr: configPtr,
		scheme:    scheme,
		sink:      sink,
		clientset: clientset,
	}, nil
}

func (c Cadvisor) GetConfig() (config *c.Config) {
	return
}

// http://k8s.io/client-go/tools/clientcmd.BuildConfigFromFlags()

func k8sConfig(configPtr *c.Config) (*rest.Config, error) {
	switch configPtr.Mode() {
	case c.ModeDeployed:
		return rest.InClusterConfig()
	case c.ModeDevelopment:
		kubeConfigPtr := (*configPtr.Optionals())["development"]
		return clientcmd.BuildConfigFromFlags("", kubeConfigPtr["path"])
	default:
		return nil, fmt.Errorf("no client config is built for mode %q", configPtr.Mode())
	}
}

// EmitterPtrs lists the nodes and returns one emitter per node. The node List ends when ctx does,
// and a failed List is returned to the caller, never panicked (#7).
func (c Cadvisor) EmitterPtrs(ctx context.Context) ([]e.Emitter, error) {
	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
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
	return emitters, nil
}
