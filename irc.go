package main

import (
	"strings"
	"time"

	"github.com/spf13/viper"
	irc "github.com/thoj/go-ircevent"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

const server = "irc.freenode.net:6667"

type IRCCLient struct {
	conn       *irc.Connection
	channel    string
	controller cache.Controller
	store      cache.Store
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
		if strings.HasPrefix(i.channel, "#") {
			if e.Message() == "#get pods" {
				pods := i.store.List()
				for _, pod := range pods {
					p := pod.(*v1.Pod)
					i.SendMessage(p.Name)
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
	store, controller := cache.NewInformer(watcher, &v1.Pod{}, time.Second*3, cache.ResourceEventHandlerFuncs{
		AddFunc:    alertFunc("Pod Added: "),
		DeleteFunc: alertFunc("Pod Deleted: "),
	})

	return IRCCLient{
		conn:       conn,
		channel:    channel,
		controller: controller,
		store:      store,
	}
}
