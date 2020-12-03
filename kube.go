package main

import (
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewKubeClient() *kubernetes.Clientset {
	conf, err := clientcmd.BuildConfigFromFlags("", os.Getenv("HOME")+"/.kube/config")
	if err != nil {
		panic(err)
	}

	cs, err := kubernetes.NewForConfig(conf)
	if err != nil {
		panic(err)
	}
	return cs
}
