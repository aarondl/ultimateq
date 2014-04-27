package bot

import (
	"sync"

	"github.com/aarondl/ultimateq/irc"
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
func (c *coreHandler) HandleRaw(w irc.Writer, ev *irc.Event) {
	switch ev.Name {

	case irc.PING:
		w.Send(irc.PONG + " :" + ev.Args[0])

	case irc.CONNECT:
		server := c.getServer(ev.NetworkID)
		server.bot.protectConfig.Lock()
		nick, uname, realname := server.conf.GetNick(),
			server.conf.GetUsername(), server.conf.GetRealname()
		server.bot.protectConfig.Unlock()
		c.protect.Lock()
		c.nickvalue = 0
		c.protect.Unlock()
		w.Send("NICK :", nick)
		w.Sendf("USER %v 0 * :%v", uname, realname)

	case irc.ERR_NICKNAMEINUSE:
		server := c.getServer(ev.NetworkID)

		c.bot.protectConfig.Lock()
		defer c.bot.protectConfig.Unlock()
		nick, altnick := server.conf.GetNick(), server.conf.GetAltnick()

		c.protect.Lock()
		defer c.protect.Unlock()
		if c.nickvalue == 0 && 0 < len(altnick) {
			nick = altnick
			c.nickvalue++
		} else {
			for i := 0; i < c.nickvalue; i++ {
				nick += "_"
			}
			c.nickvalue++
		}
		w.Send("NICK :" + nick)

	case irc.JOIN:
		server := c.getServer(ev.NetworkID)
		server.protectState.RLock()
		defer server.protectState.RUnlock()
		if server.state != nil {
			if ev.Sender == server.state.Self.Host() {
				w.Send("WHO :", ev.Args[0])
				w.Send("MODE :", ev.Args[0])
			}
		}

	case irc.RPL_MYINFO:
		server := c.getServer(ev.NetworkID)
		server.netInfo.ParseMyInfo(ev)
		server.rehashNetworkInfo()

	case irc.RPL_ISUPPORT:
		server := c.getServer(ev.NetworkID)
		server.netInfo.ParseISupport(ev)
		server.rehashNetworkInfo()
	}
}

// getServer is a helper to look up the server based on w.
func (c *coreHandler) getServer(netID string) *Server {
	c.bot.protectServers.RLock()
	defer c.bot.protectServers.RUnlock()
	return c.bot.servers[netID]
}
