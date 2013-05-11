package bot

import (
	"fmt"
	"github.com/aarondl/ultimateq/irc"
)

// coreHandler is the bot's main handling struct. As such it has access directly
// to the bot itself. It's used to deal with mission critical events such as
// pings, connects, disconnects etc.
type coreHandler struct {
	bot *Bot
}

// HandleRaw implements the dispatch.EventHandler interface so the bot can
// deal with all irc messages coming in.
func (c coreHandler) HandleRaw(msg *irc.IrcMessage, sender irc.Sender) {
	switch {
	case msg.Name == "PING":
		sender.Writeln("PONG :" + msg.Args[0])

	case msg.Name == irc.CONNECT:
		server := c.bot.servers[sender.GetKey()]
		sender.Writeln("NICK :" + server.conf.GetNick())
		sender.Writeln(fmt.Sprintf(
			"USER %v 0 * :%v",
			server.conf.GetUsername(),
			server.conf.GetRealname(),
		))
	}
}
