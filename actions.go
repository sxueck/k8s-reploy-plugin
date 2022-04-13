package main

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	rconfig "redeploy/config"
)

var cfg = rconfig.Cfg

func NewInClusterClient() (*kubernetes.Clientset, error) {
	var (
		config *rest.Config
		err    error
	)

	apiServer := cfg.ApiServer
	if len(apiServer) > 0 {

		log.Printf("apiServer Address is : %s", apiServer)

		if cfg.KubeConfig == "" {
			return nil, fmt.Errorf("please configure KUBECONFIG parameters")
		}

		config, err = clientcmd.BuildConfigFromFlags("", cfg.KubeConfig)
		if err != nil {
			return nil, err
		}

	} else {
		//return nil, fmt.Errorf("please define API_SERVER parameter to specify k8s address")
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Printf("failed to obtain config from InClusterConfig: %v", err)
			return nil, err
		}
	}

	return kubernetes.NewForConfig(config)
}
