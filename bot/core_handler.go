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
func (c *coreHandler) HandleRaw(msg *irc.IrcMessage, endpoint irc.Endpoint) {
	switch msg.Name {

	case irc.PING:
		endpoint.Send(irc.PONG + " :" + msg.Args[0])

	case irc.CONNECT:
		server := c.getServer(endpoint)
		c.protect.Lock()
		c.nickvalue = 0
		c.protect.Unlock()
		endpoint.Send("NICK :" + server.conf.GetNick())
		endpoint.Send(fmt.Sprintf(
			"USER %v 0 * :%v",
			server.conf.GetUsername(),
			server.conf.GetRealname(),
		))

	case irc.ERR_NICKNAMEINUSE:
		server := c.getServer(endpoint)
		c.protect.Lock()
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
		endpoint.Send("NICK :" + nick)

	case irc.JOIN:
		server := c.getServer(endpoint)
		server.protectState.RLock()
		defer server.protectState.RUnlock()
		if server.state != nil {
			if msg.Sender == server.state.Self.GetFullhost() {
				endpoint.Send("WHO :", msg.Args[0])
				endpoint.Send("MODE :", msg.Args[0])
			}
		}

	case irc.RPL_MYINFO:
		server := c.getServer(endpoint)
		server.caps.ParseMyInfo(msg)
		server.rehashProtocaps()

	case irc.RPL_ISUPPORT:
		server := c.getServer(endpoint)
		server.caps.ParseISupport(msg)
		server.rehashProtocaps()
	}
}

// getServer is a helper to look up the server based on endpoint.
func (c *coreHandler) getServer(endpoint irc.Endpoint) *Server {
	s, ok := endpoint.(*ServerEndpoint)
	if ok {
		return s.server
	}

	c.bot.protectServers.RLock()
	defer c.bot.protectServers.RUnlock()
	return c.bot.servers[endpoint.GetKey()]
}
