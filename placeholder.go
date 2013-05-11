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

func (h Handler) HandleRaw(m *irc.IrcMessage, sender irc.Sender) {
	if strings.Split(m.Sender, "!")[0] == "Aaron" {
		sender.Writeln(m.Args[1])
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
			Server("irc.gamesurge.net"),
	)
	if err != nil {
		log.Println(err)
	}

	b.Register(irc.PRIVMSG, Handler{})

	ers := b.Connect()
	if len(ers) != 0 {
		log.Println(ers)
	}
	b.Start()
	b.WaitForShutdown()
}
