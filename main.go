package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
	irc "github.com/thoj/go-ircevent"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

const server = "irc.freenode.net:6667"

func main() {
	viper.SetDefault("nick", "kubeyirc")
	viper.SetDefault("fullname", "Kubernetes IRC bot")
	viper.SetDefault("channel", "#testing-kubeyirc")

	conn := irc.IRC(viper.GetString("nick"), viper.GetString("fullname"))
	fmt.Println("in the channel")
	defer conn.Quit()

	channel := viper.GetString("channel")

	conn.VerboseCallbackHandler = true
	conn.Debug = false
	conn.AddCallback("001", func(e *irc.Event) { conn.Join(channel) })
	joinedIn := make(chan struct{})
	conn.AddCallback("366", func(e *irc.Event) {
		conn.Privmsg(channel, "Joined in.\n")
		joinedIn <- struct{}{}
	})
	err := conn.Connect(server)

	if err != nil {
		fmt.Println(err)
		conn.Quit()
		return
	}

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
			conn.Privmsg(channel, logString+pod.Name)
		}
	}

	watcher := cache.NewListWatchFromClient(cs.CoreV1().RESTClient(), "pods", "", fields.Everything())
	_, controller := cache.NewInformer(watcher, &v1.Pod{}, time.Second*3, cache.ResourceEventHandlerFuncs{
		AddFunc:    alertFunc("Pod Added: "),
		DeleteFunc: alertFunc("Pod Deleted:"),
	})

	go conn.Loop()
	<-joinedIn
	stop := make(chan struct{})
	controller.Run(stop)
}
