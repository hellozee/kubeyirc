package main

import (
	"context"
	"strings"
	"time"

	"github.com/spf13/viper"
	irc "github.com/thoj/go-ircevent"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const server = "irc.freenode.net:6667"

type IRCCLient struct {
	conn       *irc.Connection
	channel    string
	controller cache.Controller
	client     *kubernetes.Clientset
}

func (i *IRCCLient) SendMessage(msg string) {
	i.conn.Privmsg(i.channel, msg)
}

func (i *IRCCLient) Start() {
	joinedIn := make(chan struct{})
	i.conn.AddCallback("366", func(e *irc.Event) {
		i.conn.Privmsg(i.channel, "Joined in.\n")
		joinedIn <- struct{}{}
	})

	i.setupCallBacks()

	i.conn.Loop()
	<-joinedIn

	stop := make(chan struct{})
	i.controller.Run(stop)
}

func (i *IRCCLient) setupCallBacks() {
	i.conn.AddCallback("PRIVMSG", func(e *irc.Event) {
		if strings.HasPrefix(e.Arguments[0], "#") {

			if e.Message() == "#get pods" {
				pods, err := i.client.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
				if err != nil {
					panic(err)
				}
				for _, p := range pods.Items {
					i.SendMessage(p.Name)
				}
			}

			if e.Message() == "#get deployments" {
				deployments, err := i.client.AppsV1().Deployments("").List(context.Background(), metav1.ListOptions{})
				if err != nil {
					panic(err)
				}
				for _, d := range deployments.Items {
					i.SendMessage(d.Name)
				}
			}

			if e.Message() == "#get nodes" {
				nodes, err := i.client.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
				if err != nil {
					panic(err)
				}
				for _, n := range nodes.Items {
					i.SendMessage(n.Name)
				}
			}

			if e.Message() == "#get services" {
				services, err := i.client.CoreV1().Services("").List(context.Background(), metav1.ListOptions{})
				if err != nil {
					panic(err)
				}
				for _, s := range services.Items {
					i.SendMessage(s.Name)
				}
			}
		}
	})
}

func NewIRCClient() IRCCLient {
	viper.SetConfigName("config")
	viper.AddConfigPath("./")
	err := viper.ReadInConfig()

	if err != nil {
		panic(err)
	}

	viper.SetDefault("nick", "kubeyirc")
	viper.SetDefault("fullname", "Kubernetes IRC bot")
	viper.SetDefault("channel", "#testing-kubeyirc")

	conn := irc.IRC(viper.GetString("nick"), viper.GetString("fullname"))
	defer conn.Quit()

	channel := viper.GetString("channel")

	conn.VerboseCallbackHandler = true
	conn.Debug = false

	conn.AddCallback("001", func(e *irc.Event) { conn.Join(channel) })

	err = conn.Connect(server)

	if err != nil {
		conn.Quit()
		panic(err)
	}

	alertFunc := func(logString string) func(obj interface{}) {
		return func(obj interface{}) {
			pod := obj.(*v1.Pod)
			conn.Privmsg(channel, logString+pod.Name)
		}
	}

	cs := NewKubeClient()

	watcher := cache.NewListWatchFromClient(cs.CoreV1().RESTClient(), "pods", "", fields.Everything())
	_, controller := cache.NewInformer(watcher, &v1.Pod{}, time.Second*3, cache.ResourceEventHandlerFuncs{
		AddFunc:    alertFunc("Pod Added: "),
		DeleteFunc: alertFunc("Pod Deleted: "),
	})

	return IRCCLient{
		conn:       conn,
		channel:    channel,
		controller: controller,
		client:     cs,
	}
}
