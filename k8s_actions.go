package main

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
)

func ClusterAPIServerHost() (string, bool) {
	apiServerAddr := os.Getenv("API_SERVER")
	if apiServerAddr != "" {
		return apiServerAddr, true
	}
	return "", false
}

func NewInClusterClient() (*kubernetes.Clientset, error) {
	var (
		config *rest.Config
		err    error
	)

	if apiServerAddr, override := ClusterAPIServerHost(); override {

		log.Printf("apiServer Address is : %s", apiServerAddr)

		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			return nil, fmt.Errorf("please configure KUBECONFIG parameters")
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
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
