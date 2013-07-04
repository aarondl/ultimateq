package bot

import (
	"bytes"
	"fmt"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/irc"
	"strings"
	. "testing"
)

const (
	server    = "irc.test.net"
	bothost   = "bot!botuser@bothost"
	botnick   = "bot"
	user1host = "nick1!user1@host1"
	user2host = "nick2!user2@host2"
	user1user = "user"
	channel   = "#chan"
)

var (
	channelKinds = data.CreateChannelModeKinds("a", "b", "c", "d")
	userKinds, _ = data.CreateUserModeKinds("(ov)@+")
)

func commandsSetup(t *T) (*Bot, *data.DataEndpoint, *data.State, *data.Store,
	*bytes.Buffer) {

	conf := Configure().Nick("nobody").Altnick("nobody1").Username("nobody").
		Userhost("bitforge.ca").Realname("ultimateq").NoReconnect(true).
		Ssl(true).Server(serverId)

	b, err := createBot(conf, nil, nil, func(_ string) (*data.Store, error) {
		return data.CreateStore(data.MemStoreProvider)
	}, true, true)

	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	srv := b.servers[serverId]
	buf := &bytes.Buffer{}
	srv.endpoint.Writer = buf

	srv.state.Update(&irc.IrcMessage{
		Sender: server, Name: irc.RPL_WELCOME,
		Args: []string{"Welcome", bothost},
	})
	srv.state.Update(&irc.IrcMessage{
		Sender: bothost, Name: irc.JOIN,
		Args: []string{channel},
	})
	srv.state.Update(&irc.IrcMessage{
		Sender: user1host, Name: irc.JOIN,
		Args: []string{channel},
	})
	srv.state.Update(&irc.IrcMessage{
		Sender: user2host, Name: irc.PRIVMSG,
		Args: []string{botnick, "hithere"},
	})

	return b, srv.endpoint.DataEndpoint, srv.state, b.store, buf
}

func commandsTeardown(b *Bot, t *T) {
	b.coreCommands.unregisterCoreCommands()
}

func TestCoreCommands(t *T) {
	conf := Configure().Nick("nobody").Altnick("nobody1").Username("nobody").
		Userhost("bitforge.ca").Realname("ultimateq").NoReconnect(true).
		Ssl(true).Server(serverId)

	b, err := createBot(conf, nil, nil, func(_ string) (*data.Store, error) {
		return data.CreateStore(data.MemStoreProvider)
	}, true, true)

	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if b.coreCommands == nil {
		t.Error("Core commands should have been attached.")
	}

	commandsTeardown(b, t)
}

func TestCoreCommands_Register(t *T) {
	bot, ep, _, store, buffer := commandsSetup(t)
	defer commandsTeardown(bot, t)

	var err error

	if store.GetAuthedUser(serverId, user1user) != nil {
		t.Error("Somehow user was authed already.")
	}

	msg1 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, "register password " + user1user},
	}
	msg2 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user2host,
		Args:   []string{botnick, "register password"},
	}

	err = dispatchResultCheck(msg1, "registered [user] suc", bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}

	access := store.GetAuthedUser(serverId, user1host)
	if access == nil {
		t.Error("User was not authenticated.")
	} else {
		if access.Global.Level != ^uint8(0) {
			t.Error("Level not granted.")
		}
		if access.Global.Flags != ^uint64(0) {
			t.Error("Flags not granted.")
		}
	}

	dispatchResultCheck(msg2, "Registered [user2] success", bot, ep, buffer, t)

	access = store.GetAuthedUser(serverId, user2host)
	if access == nil {
		t.Error("User was not authenticated.")
	} else if access.Global != nil {
		if access.Global.Level != 0 {
			t.Error("Level granted by mistake.")
		}
		if access.Global.Flags != 0 {
			t.Error("Flags granted by mistake.")
		}
	}

	store.Logout(serverId, user2host)
	dispatchResultCheck(msg2, "username [user2] is already registered",
		bot, ep, buffer, t)
}

func TestCoreCommands_Auth(t *T) {
	bot, ep, _, store, buffer := commandsSetup(t)
	defer commandsTeardown(bot, t)

	var err error

	msg1 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, "register password"},
	}
	msg2 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, "logout"},
	}
	msg3 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, "auth password"},
	}

	err = dispatchResultCheck(msg1, "registered", bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}
	access := store.GetAuthedUser(serverId, user1host)
	if access == nil {
		t.Error("User was not authenticated.")
	}
	err = dispatchResultCheck(msg2, "logged out", bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}
	access = store.GetAuthedUser(serverId, user1host)
	if access != nil {
		t.Error("User was not logged out.")
	}
	err = dispatchResultCheck(msg3, "Successfully auth", bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}
	err = dispatchResultCheck(msg3, "already authenticated", bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}

	access = store.GetAuthedUser(serverId, user1host)
	if access == nil {
		t.Error("User was not authenticated.")
	}
}

func TestCoreCommands_Logout(t *T) {
	bot, ep, _, store, buffer := commandsSetup(t)
	defer commandsTeardown(bot, t)

	var err error

	msg1 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, "register password"},
	}
	msg2 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, "logout"},
	}

	err = dispatchResultCheck(msg1, "registered", bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}
	err = dispatchResultCheck(msg2, "logged out", bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}

	access := store.GetAuthedUser(serverId, user1host)
	if access != nil {
		t.Error("User was not logged out.")
	}
}

func dispatchResultCheck(msg *irc.IrcMessage, expected string,
	bot *Bot, ep *data.DataEndpoint, buffer *bytes.Buffer, t *T) error {

	buffer.Reset()
	err := bot.commander.Dispatch(serverId, msg, ep)
	bot.commander.WaitForHandlers()

	if err != nil {
		return fmt.Errorf("Error from dispatch:", err)
	}
	if s := buffer.String(); len(s) == 0 {
		return fmt.Errorf("Everything should generate a response.")
	} else if !strings.Contains(s, expected) {
		return fmt.Errorf("\nUnexpected Response: \n\t%s\n\t%s", s, expected)
	}
	return nil
}
