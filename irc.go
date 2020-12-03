package main

import (
	"fmt"

	"github.com/spf13/viper"
	irc "github.com/thoj/go-ircevent"
)

type IRCCLient struct {
	connection *irc.Connection
	channel    string
}

func (i *IRCCLient) SendMessage(msg string) {
	i.connection.Privmsg(i.channel, msg)
}

func NewIRCClient() IRCCLient {
	viper.SetConfigName("config")
	viper.AddConfigPath("./")
	err := viper.ReadInConfig()

	if err != nil {
		fmt.Println(err)
		return IRCCLient{}
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

	joinedIn := make(chan struct{})
	conn.AddCallback("366", func(e *irc.Event) {
		conn.Privmsg(channel, "Joined in.\n")
		joinedIn <- struct{}{}
	})
	err = conn.Connect(server)

	if err != nil {
		fmt.Println(err)
		conn.Quit()
		return IRCCLient{}
	}

	conn.Loop()
	<-joinedIn

	return IRCCLient{
		connection: conn,
		channel:    channel,
	}
}
