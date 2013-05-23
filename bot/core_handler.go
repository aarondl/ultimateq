package bot

import (
	"fmt"
	"github.com/aarondl/ultimateq/irc"
	"sync"
)

// coreHandler is the bot's main handling struct. As such it has access directly
// to the bot itself. It's used to deal with mission critical events such as
// pings, connects, disconnects etc.
type coreHandler struct {
	// The bot this core handler belongs to.
	bot *Bot

	// How many nicks have been sent.
	nickvalue int

	// Protect access to core Handler
	protect sync.RWMutex
}

// HandleRaw implements the dispatch.EventHandler interface so the bot can
// deal with all irc messages coming in.
func (c *coreHandler) HandleRaw(msg *irc.IrcMessage, sender irc.Sender) {
	switch {
	case msg.Name == irc.PING:
		sender.Writeln(irc.PONG + " :" + msg.Args[0])

	case msg.Name == irc.CONNECT:
		c.protect.RLock()
		server := c.getServer(sender)
		c.protect.RUnlock()
		sender.Writeln("NICK :" + server.conf.GetNick())
		sender.Writeln(fmt.Sprintf(
			"USER %v 0 * :%v",
			server.conf.GetUsername(),
			server.conf.GetRealname(),
		))

	case msg.Name == irc.ERR_NICKNAMEINUSE:
		c.protect.Lock()
		server := c.getServer(sender)
		var nick string
		if c.nickvalue == 0 && 0 < len(server.conf.GetAltnick()) {
			nick = server.conf.GetAltnick()
			c.nickvalue += 1
		} else {
			nick = server.conf.GetNick()
			for i := 0; i < c.nickvalue; i++ {
				nick += "_"
			}
			c.nickvalue += 1
		}
		c.protect.Unlock()
		sender.Writeln("NICK :" + nick)
	}
}

// getServer is a helper to look up the server based on sender key.
func (c *coreHandler) getServer(sender irc.Sender) *Server {
	c.bot.serversProtect.RLock()
	defer c.bot.serversProtect.RUnlock()
	return c.bot.servers[sender.GetKey()]
}
