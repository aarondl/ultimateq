package bot

import (
	"strings"
	"sync"
	"time"

	"github.com/aarondl/ultimateq/config"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/irc"
)

// coreHandler is the bot's main handling struct. As such it has access directly
// to the bot itself. It's used to deal with mission critical events such as
// pings, connects, disconnects etc.
type coreHandler struct {
	// The bot this core handler belongs to.
	bot *Bot

	// How many nicks have been sent.
	nickvalue      int
	untilJoinScale time.Duration

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

		c.protect.Lock()
		c.nickvalue = 0
		c.protect.Unlock()

		cfg := server.conf.Network(ev.NetworkID)
		nick, _ := cfg.Nick()
		uname, _ := cfg.Username()
		realname, _ := cfg.Realname()
		noautojoin, _ := cfg.NoAutoJoin()
		joindelay, _ := cfg.JoinDelay()

		if password, ok := cfg.Password(); ok {
			w.Send("PASS :", password)
		}

		w.Send("NICK :", nick)
		w.Sendf("USER %s 0 * :%s", uname, realname)

		if noautojoin {
			break
		}

		if chs, ok := cfg.Channels(); ok {
			<-time.After(c.untilJoinScale * time.Duration(joindelay))
			for _, ch := range chs {
				if len(ch.Password) > 0 {
					w.Sendf("JOIN %s %s", ch.Name, ch.Password)
				} else {
					w.Sendf("JOIN %s", ch.Name)
				}
			}
		}
	case irc.KICK, irc.ERR_BANNEDFROMCHAN:
		server := c.getServer(ev.NetworkID)
		cfg := server.conf.Network(ev.NetworkID)
		noautojoin, _ := cfg.NoAutoJoin()
		joindelay, _ := cfg.JoinDelay()

		if noautojoin {
			break
		}

		var chs []config.Channel
		var ok bool
		if chs, ok = cfg.Channels(); !ok {
			break
		}

		var nick, channel, curNick string
		if ev.Name == irc.KICK {
			channel = strings.ToLower(ev.Args[0])
			nick = strings.ToLower(ev.Args[1])
		} else {
			nick = strings.ToLower(ev.Args[0])
			channel = strings.ToLower(ev.Args[1])
		}

		c.bot.UsingState(ev.NetworkID, func(st *data.State) {
			curNick = strings.ToLower(st.Self.Nick())
		})

		if len(curNick) == 0 || nick != curNick {
			break
		}

		for _, ch := range chs {
			if strings.ToLower(ch.Name) != channel {
				continue
			}

			if ev.Name == irc.ERR_BANNEDFROMCHAN {
				<-time.After(c.untilJoinScale * time.Duration(joindelay))
			}
			if len(ch.Password) > 0 {
				w.Sendf("JOIN %s %s", ch.Name, ch.Password)
			} else {
				w.Sendf("JOIN %s", ch.Name)
			}
		}

	case irc.ERR_NICKNAMEINUSE:
		server := c.getServer(ev.NetworkID)

		cfg := server.conf.Network(ev.NetworkID)
		nick, _ := cfg.Nick()
		altnick, _ := cfg.Altnick()

		c.protect.Lock()
		defer c.protect.Unlock()
		if c.nickvalue == 0 && len(altnick) > 0 {
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
