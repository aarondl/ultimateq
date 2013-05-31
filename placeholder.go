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
		Userhost("friend").
		NoReconnect(true)

	c. // First server
		Server("irc.gamesurge.net1").
		Host("localhost").
		Nick("Aaron").
		Altnick("nobody1").
		ReconnectTimeout(5)

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

	<-time.After(20 * time.Second)
	c := conf(config.CreateConfig())
	c.RemoveServer("irc.gamesurge.net1")
	c.GlobalContext().
		Channels("#test").
		Server("irc.gamesurge.net3").
		Host("localhost").
		Nick("super").
		ServerContext("irc.gamesurge.net2").
		Nick("heythere").
		Channels("#hithere")
	if !c.IsValid() {
		c.DisplayErrors()
		log.Fatal("Config error")
	}
	b.ReplaceConfig(c)

	b.WaitForHalt()
	b.Stop()
	b.Disconnect()
	<-time.After(10 * time.Second)
}
