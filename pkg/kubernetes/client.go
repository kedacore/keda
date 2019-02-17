package kubernetes

import (
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	clientset "github.com/Azure/Kore/pkg/client/clientset/versioned"
	"github.com/kelseyhightower/envconfig"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const envconfigPrefix = "KUBE"

type config struct {
	MasterURL      string `envconfig:"MASTER"`
	KubeConfigPath string `envconfig:"CONFIG"`
}

func Config() (*rest.Config, error) {
	c := config{}
	err := envconfig.Process(envconfigPrefix, &c)
	if err != nil {
		return nil, err
	}
	var cfg *rest.Config
	if c.MasterURL == "" && c.KubeConfigPath == "" {
		cfg, err = rest.InClusterConfig()
	} else {
		cfg, err = clientcmd.BuildConfigFromFlags(c.MasterURL, c.KubeConfigPath)
	}
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func GetClients() (*clientset.Clientset, *kubernetes.Clientset, error) {
	cfg, err := Config()
	if err != nil {
		return nil, nil, err
	}

	koreClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	return koreClient, kubeClient, nil
}
