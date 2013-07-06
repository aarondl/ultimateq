package bot

import (
	"bytes"
	"fmt"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/irc"
	"regexp"
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
	user2user = "user2"
	user2nick = "nick2"
	channel   = "#chan"
	password  = "password"
)

var (
	channelKinds = data.CreateChannelModeKinds("a", "b", "c", "d")
	userKinds, _ = data.CreateUserModeKinds("(ov)@+")
	rgxCreator   = strings.NewReplacer(
		`(`, `\(`, `)`, `\)`, `]`, `\]`, `[`,
		`\[`, `\`, `\\`, `/`, `\/`, `%v`, `.*`,
	)
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

func dispatchResultCheck(msg *irc.IrcMessage, expected string,
	bot *Bot, ep *data.DataEndpoint, buffer *bytes.Buffer, t *T) error {

	buffer.Reset()
	err := bot.commander.Dispatch(serverId, msg, ep)
	bot.commander.WaitForHandlers()

	if s := buffer.String(); len(s) == 0 {
		if err != nil {
			return fmt.Errorf("Buffer not full and error returned: %v", err)
		}
		return fmt.Errorf("Everything should generate a response.")
	} else {
		rgx := `^NOTICE [A-Za-z0-9]+ :` + rgxCreator.Replace(expected) + `$`
		match, err := regexp.MatchString(rgx, s)
		if err != nil {
			t.Fatalf("Error making pattern: \n\t%s\n\t%s",
				expected, rgx,
			)
		}
		if !match {
			return fmt.Errorf("\nUnexpected Response: \n\t%s\n\t%s",
				s, rgx,
			)
		}
	}
	return nil
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
		Args:   []string{botnick, register + " " + password + " " + user1user},
	}
	msg2 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user2host,
		Args:   []string{botnick, register + " " + password},
	}

	err = dispatchResultCheck(msg1, registerSuccessFirst, bot, ep, buffer, t)
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

	dispatchResultCheck(msg2, registerSuccess, bot, ep, buffer, t)

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
	dispatchResultCheck(msg2, registerFailure, bot, ep, buffer, t)
}

func TestCoreCommands_Auth(t *T) {
	bot, ep, _, store, buffer := commandsSetup(t)
	defer commandsTeardown(bot, t)

	var err error

	msg1 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, register + " " + password},
	}
	msg2 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, logout},
	}
	msg3 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, auth + " " + password},
	}

	err = dispatchResultCheck(msg1, registerSuccessFirst, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}
	access := store.GetAuthedUser(serverId, user1host)
	if access == nil {
		t.Error("User was not authenticated.")
	}
	err = dispatchResultCheck(msg2, logoutSuccess, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}
	access = store.GetAuthedUser(serverId, user1host)
	if access != nil {
		t.Error("User was not logged out.")
	}
	err = dispatchResultCheck(msg3, authSuccess, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}
	err = dispatchResultCheck(msg3, errMsgAuthed, bot, ep, buffer, t)
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
		Args:   []string{botnick, register + " " + password},
	}
	msg2 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, logout},
	}

	err = dispatchResultCheck(msg1, registerSuccessFirst, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}
	err = dispatchResultCheck(msg2, logoutSuccess, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}
	err = dispatchResultCheck(msg2, ".*not authenticated.*", bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}

	access := store.GetAuthedUser(serverId, user1host)
	if access != nil {
		t.Error("User was not logged out.")
	}
}

func TestCoreCommands_Access(t *T) {
	bot, ep, _, _, buffer := commandsSetup(t)
	defer commandsTeardown(bot, t)
	var err error

	msg1 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, register + " " + password + " " + user1user},
	}
	msg2 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user2host,
		Args:   []string{botnick, register + " " + password},
	}
	msg3 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, access},
	}
	msg4 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, access + " " + user2user},
	}
	msg5 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, access + " " + user2nick},
	}
	msg6 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user2host,
		Args:   []string{botnick, access},
	}

	err = dispatchResultCheck(msg1, registerSuccessFirst, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}

	err = dispatchResultCheck(msg2, registerSuccess, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}

	err = dispatchResultCheck(msg3, registerSuccess, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}

	err = dispatchResultCheck(msg4, registerSuccess, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}

	err = dispatchResultCheck(msg5, registerSuccess, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}

	err = dispatchResultCheck(msg6, registerSuccess, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}
}

func TestCoreCommands_Deluser(t *T) {
	bot, ep, _, store, buffer := commandsSetup(t)
	defer commandsTeardown(bot, t)

	var err error

	msg1 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, register + " " + password + " " + user1user},
	}
	msg2 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user2host,
		Args:   []string{botnick, register + " " + password},
	}
	msg3 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user2host,
		Args:   []string{botnick, deluser + " " + user1user},
	}
	msg4 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, deluser + " " + user2nick},
	}
	msg5 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, deluser + " *" + user2user},
	}
	msg6 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, deluser + " noexist"},
	}
	msg7 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, deluser + " *noexist"},
	}
	msg8 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, deluser + " *"},
	}

	err = dispatchResultCheck(msg1, registerSuccessFirst, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}
	err = dispatchResultCheck(msg2, registerSuccess, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}

	access1 := store.GetAuthedUser(serverId, user1host)
	access2 := store.GetAuthedUser(serverId, user1host)
	if access1 == nil || access2 == nil {
		t.Error("User's were not authenticated.")
	}

	err = dispatchResultCheck(msg3, ".*[A] flag(s) required.*", bot, ep,
		buffer, t)
	if err != nil {
		t.Error(err)
	}
	err = dispatchResultCheck(msg4, deluserSuccess, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}

	access2 = store.GetAuthedUser(serverId, user2host)
	if access2 != nil {
		t.Error("User was not logged out.")
	}
	access2, err = store.FindUser(user2user)
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if access2 != nil {
		t.Error("User was not deleted.")
	}

	err = dispatchResultCheck(msg2, registerSuccess, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}

	err = dispatchResultCheck(msg5, deluserSuccess, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}

	access2 = store.GetAuthedUser(serverId, user2host)
	if access2 != nil {
		t.Error("User was not logged out.")
	}
	access2, err = store.FindUser(user2user)
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if access2 != nil {
		t.Error("User was not deleted.")
	}

	err = dispatchResultCheck(msg6, errFmtUserNotFound, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}
	err = dispatchResultCheck(msg7, deluserFailure, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}
	err = dispatchResultCheck(msg8, errFmtUserNotFound, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}
}

func TestCoreCommands_Delme(t *T) {
	bot, ep, _, store, buffer := commandsSetup(t)
	defer commandsTeardown(bot, t)

	var err error

	msg1 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, register + " " + password + " " + user1user},
	}
	msg2 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, delme},
	}

	err = dispatchResultCheck(msg1, registerSuccessFirst, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}

	access := store.GetAuthedUser(serverId, user1host)
	if access == nil {
		t.Error("User was not authenticated.")
	}

	err = dispatchResultCheck(msg2, delmeSuccess, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}

	access = store.GetAuthedUser(serverId, user1host)
	if access != nil {
		t.Error("User was not logged out.")
	}
	access, err = store.FindUser(user1user)
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if access != nil {
		t.Error("User was not deleted.")
	}

	err = dispatchResultCheck(msg2, ".*not authenticated.*", bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}
}

func TestCoreCommands_Passwd(t *T) {
	bot, ep, _, store, buffer := commandsSetup(t)
	defer commandsTeardown(bot, t)

	var err error
	var access *data.UserAccess

	newpasswd := "newpasswd"

	msg1 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, register + " " + password + " " + user1user},
	}
	msg2 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, passwd + " " + password + " " + newpasswd},
	}

	err = dispatchResultCheck(msg2, ".*not authenticated.*", bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}

	err = dispatchResultCheck(msg1, registerSuccessFirst, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}

	access = store.GetAuthedUser(serverId, user1host)
	if access == nil {
		t.Error("User was not authenticatd.")
	}
	oldPwd := access.Password

	err = dispatchResultCheck(msg2, passwdSuccess, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}

	access = store.GetAuthedUser(serverId, user1host)
	if access == nil {
		t.Error("User was not authenticatd.")
	}
	if bytes.Compare(access.Password, oldPwd) == 0 {
		t.Error("Password was not changed.")
	}

	err = dispatchResultCheck(msg2, passwdFailure, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}
}

func TestCoreCommands_Masks(t *T) {
	bot, ep, _, store, buffer := commandsSetup(t)
	defer commandsTeardown(bot, t)

	var err error
	var access *data.UserAccess

	msg1 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, register + " " + password + " " + user1user},
	}
	msg2 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, addmask + " " + user1host},
	}
	msg3 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user2host,
		Args:   []string{botnick, auth + " " + password + " " + user1user},
	}
	msg4 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, delmask + " " + user1host},
	}
	msg5 := &irc.IrcMessage{Name: irc.PRIVMSG,
		Sender: user1host,
		Args:   []string{botnick, masks},
	}

	err = dispatchResultCheck(msg2, ".*not authenticated.*", bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}

	err = dispatchResultCheck(msg1, registerSuccessFirst, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}

	err = dispatchResultCheck(msg2, addmaskSuccess, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}

	err = dispatchResultCheck(msg2, addmaskFailure, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}

	access = store.GetAuthedUser(serverId, user1host)
	if access == nil {
		t.Fatal("User was not authed.")
	}
	if len(access.Masks) != 1 || access.Masks[0] != user1host {
		t.Error("Mask not set correctly.")
	}

	err = dispatchResultCheck(msg3, "Host [.*] does not match.*",
		bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}

	err = dispatchResultCheck(msg5, ".*"+user1host+".*",
		bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}

	err = dispatchResultCheck(msg4, delmaskSuccess, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}

	err = dispatchResultCheck(msg4, delmaskFailure, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}

	access = store.GetAuthedUser(serverId, user1host)
	if access == nil {
		t.Fatal("User was not authed.")
	}
	if len(access.Masks) != 0 {
		t.Error("Mask not removed correctly.")
	}

	err = dispatchResultCheck(msg5, masksFailure, bot, ep, buffer, t)
	if err != nil {
		t.Error(err)
	}
}
