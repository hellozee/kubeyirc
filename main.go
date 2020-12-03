package main

import (
	"os"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

const server = "irc.freenode.net:6667"

func main() {

	conn := NewIRCClient()
	conf, err := clientcmd.BuildConfigFromFlags("", os.Getenv("HOME")+"/.kube/config")
	if err != nil {
		panic(err)
	}

	cs, err := kubernetes.NewForConfig(conf)
	if err != nil {
		panic(err)
	}

	alertFunc := func(logString string) func(obj interface{}) {
		return func(obj interface{}) {
			pod := obj.(*v1.Pod)
			conn.SendMessage(logString + pod.Name)
		}
	}

	watcher := cache.NewListWatchFromClient(cs.CoreV1().RESTClient(), "pods", "", fields.Everything())
	_, controller := cache.NewInformer(watcher, &v1.Pod{}, time.Second*3, cache.ResourceEventHandlerFuncs{
		AddFunc:    alertFunc("Pod Added: "),
		DeleteFunc: alertFunc("Pod Deleted:"),
	})

	stop := make(chan struct{})
	controller.Run(stop)
}
