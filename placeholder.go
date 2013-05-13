// The ultimateq bot framework.
package main

import (
	"github.com/aarondl/ultimateq/bot"
	"github.com/aarondl/ultimateq/irc"
	"log"
	"os"
	"strings"
)

type Handler struct {
}

func (h Handler) PrivmsgUser(m *irc.Message, sender irc.Sender) {
	if strings.Split(m.Raw.Sender, "!")[0] == "Aaron" {
		sender.Writeln(m.Message())
	}
}

func main() {
	log.SetOutput(os.Stdout)

	b, err := bot.CreateBot(
		bot.Configure().
			Nick("nobody__").
			Altnick("nobody_").
			Realname("there").
			Username("guy").
			Userhost("friend").
			Server("irc.gamesurge.net1").
			Host("irc.gamesurge.net").
			Nick("nobody1").
			Server("irc.gamesurge.net2").
			Host("irc.gamesurge.net").
			Nick("nobody2"),
	)
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
	b.WaitForShutdown()
}
