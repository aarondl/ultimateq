// The ultimateq bot framework.
package main

import (
	"github.com/aarondl/ultimateq/bot"
	"github.com/aarondl/ultimateq/config"
	"github.com/aarondl/ultimateq/irc"
	"log"
	"os"
	"strings"
	"time"
)

type Handler struct {
}

func (h Handler) PrivmsgUser(m *irc.Message, sender irc.Sender) {
	if strings.Split(m.Raw.Sender, "!")[0] == "Aaron" {
		sender.Writeln(m.Message())
	}
}

func (h Handler) PrivmsgChannel(m *irc.Message, sender irc.Sender) {
	if m.Message() == "hello" {
		sender.Writeln("PRIVMSG " + m.Target() + " :Hello to you too!")
	}
}

func conf(c *config.Config) *config.Config {
	c. // Defaults
		Nick("nobody__").
		Altnick("nobody_").
		Realname("there").
		Username("guy").
		Userhost("friend")

	c. // First server
		Server("irc.gamesurge.net1").
		Host("localhost").
		Nick("nobody1")

	c. // Second Server
		Server("irc.gamesurge.net2").
		Host("localhost").
		Nick("nobody2")

	return c
}

func main() {
	log.SetOutput(os.Stdout)

	b, err := bot.CreateBot(bot.ConfigureFunction(conf))
	if err != nil {
		log.Println(err)
	}

	b.Register(irc.PRIVMSG, Handler{})

	ers := b.Connect()
	if len(ers) != 0 {
		log.Println(ers)
		return
	}
	b.Start()

	server1 := "irc.gamesurge.net1"
	<-time.After(30 * time.Second)
	b.StopServer(server1)
	b.DisconnectServer(server1)
	log.Println("Server Disconnected... Waiting 10s")
	<-time.After(10 * time.Second)
	log.Println("Reconnecting")
	_, err = b.ConnectServer(server1)
	if err != nil {
		log.Println("Could not connect again:", err)
	} else {
		b.StartServer(server1)
	}

	b.WaitForHalt()
	b.Stop()
	b.Disconnect()
	<-time.After(10 * time.Second)
}
