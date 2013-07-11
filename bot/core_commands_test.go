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
	server   = "irc.test.net"
	bothost  = "bot!botuser@bothost"
	botnick  = "bot"
	u1host   = "nick1!user1@host1"
	u1nick   = "nick1"
	u1user   = "user"
	u2host   = "nick2!user2@host2"
	u2nick   = "nick2"
	u2user   = "user2"
	channel  = "#chan"
	password = "password"
	prefix   = "."
)

var (
	channelKinds = data.CreateChannelModeKinds("a", "b", "c", "d")
	userKinds, _ = data.CreateUserModeKinds("(ov)@+")
	rgxCreator   = strings.NewReplacer(
		`(`, `\(`, `)`, `\)`, `]`, `\]`, `[`,
		`\[`, `\`, `\\`, `/`, `\/`, `%v`, `.*`,
	)
)

type tSetup struct {
	b      *Bot
	ep     *data.DataEndpoint
	state  *data.State
	store  *data.Store
	buffer *bytes.Buffer
	t      *T
}

func commandsSetup(t *T) *tSetup {
	conf := Configure().Nick("nobody").Altnick("nobody1").Username("nobody").
		Userhost("host.com").Realname("ultimateq").NoReconnect(true).
		Ssl(true).Prefix(prefix).Server(serverId)

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
		Sender: u1host, Name: irc.JOIN,
		Args: []string{channel},
	})
	srv.state.Update(&irc.IrcMessage{
		Sender: u2host, Name: irc.PRIVMSG,
		Args: []string{botnick, "hithere"},
	})

	return &tSetup{b, srv.endpoint.DataEndpoint, srv.state, b.store, buf, t}
}

func commandsTeardown(s *tSetup, t *T) {
	if s.store != nil {
		s.store.Close()
	}
	s.b.coreCommands.unregisterCoreCommands()
}

func pubRspChk(ts *tSetup, expected, sender string, args ...string) error {
	return prvRspChk(ts, expected, channel, sender, args...)
}

func rspChk(ts *tSetup, expected, sender string, args ...string) error {
	return prvRspChk(ts, expected, botnick, sender, args...)
}

func prvRspChk(ts *tSetup, expected, to, sender string, args ...string) error {
	ts.buffer.Reset()
	err := ts.b.commander.Dispatch(serverId, &irc.IrcMessage{
		Name: irc.PRIVMSG, Sender: sender,
		Args: []string{to, strings.Join(args, " ")},
	}, ts.ep)
	ts.b.commander.WaitForHandlers()

	if s := ts.buffer.String(); len(s) == 0 {
		if err != nil {
			return fmt.Errorf("Buffer not full and error returned: %v", err)
		}
		return fmt.Errorf("Everything should generate a response.")
	} else {
		rgx := `^NOTICE [A-Za-z0-9]+ :` + rgxCreator.Replace(expected) + `$`
		match, err := regexp.MatchString(rgx, s)
		if err != nil {
			return fmt.Errorf("Error making pattern: \n\t%s\n\t%s",
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

	commandsTeardown(&tSetup{b: b}, t)
}

func TestCoreCommands_Register(t *T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)

	var err error

	if ts.store.GetAuthedUser(serverId, u1user) != nil {
		t.Error("Somehow user was authed already.")
	}

	err = rspChk(ts, registerSuccessFirst, u1host, register, password, u1user)
	if err != nil {
		t.Error(err)
	}

	access := ts.store.GetAuthedUser(serverId, u1host)
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

	err = rspChk(ts, registerSuccess, u2host, register, passwd)
	if err != nil {
		t.Error(err)
	}

	access = ts.store.GetAuthedUser(serverId, u2host)
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

	ts.store.Logout(serverId, u2host)
	err = rspChk(ts, errMsgAuthed, u1host, register, passwd, u1user)
	if err != nil {
		t.Error(err)
	}
}

func TestCoreCommands_Auth(t *T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)

	var err error

	err = rspChk(ts, registerSuccessFirst, u1host, register, password)
	if err != nil {
		t.Error(err)
	}
	access := ts.store.GetAuthedUser(serverId, u1host)
	if access == nil {
		t.Error("User was not authenticated.")
	}
	err = rspChk(ts, logoutSuccess, u1host, logout)
	if err != nil {
		t.Error(err)
	}
	access = ts.store.GetAuthedUser(serverId, u1host)
	if access != nil {
		t.Error("User was not logged out.")
	}
	err = rspChk(ts, authSuccess, u1host, auth, password)
	if err != nil {
		t.Error(err)
	}
	err = rspChk(ts, errMsgAuthed, u1host, auth, password)
	if err != nil {
		t.Error(err)
	}

	access = ts.store.GetAuthedUser(serverId, u1host)
	if access == nil {
		t.Error("User was not authenticated.")
	}
}

func TestCoreCommands_Logout(t *T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)

	var err error

	err = rspChk(ts, registerSuccessFirst, u1host, register, password)
	if err != nil {
		t.Error(err)
	}
	err = rspChk(ts, logoutSuccess, u1host, logout)
	if err != nil {
		t.Error(err)
	}
	err = rspChk(ts, ".*not authenticated.*", u1host, logout)
	if err != nil {
		t.Error(err)
	}

	access := ts.store.GetAuthedUser(serverId, u1host)
	if access != nil {
		t.Error("User was not logged out.")
	}
}

func TestCoreCommands_Access(t *T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)
	var err error

	err = rspChk(ts, registerSuccessFirst, u1host, register, password, u1user)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, registerSuccess, u2host, register, password)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, accessSuccess, u1host, access)
	if err != nil {
		t.Error(err)
	}

	err = pubRspChk(ts, accessSuccess, u1host, prefix+access)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, accessSuccess, u1host, access, "*"+u2user)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, accessSuccess, u1host, access, u2nick)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, logoutSuccess, u2host, logout)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, accessSuccess, u1host, access, "*"+u2user)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, ".*not authenticated.*", u1host, access, u2nick)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, ".*Username must follow.*", u1host, access, "*")
	if err != nil {
		t.Error(err)
	}
}

func TestCoreCommands_Deluser(t *T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)

	var err error

	err = rspChk(ts, registerSuccessFirst, u1host, register, password, u1user)
	if err != nil {
		t.Error(err)
	}
	err = rspChk(ts, registerSuccess, u2host, register, password)
	if err != nil {
		t.Error(err)
	}

	access1 := ts.store.GetAuthedUser(serverId, u1host)
	access2 := ts.store.GetAuthedUser(serverId, u1host)
	if access1 == nil || access2 == nil {
		t.Error("User's were not authenticated.")
	}

	err = rspChk(ts, ".*[A] flag(s) required.*", u2host, deluser, u1user)
	if err != nil {
		t.Error(err)
	}
	err = rspChk(ts, deluserSuccess, u1host, deluser, u2nick)
	if err != nil {
		t.Error(err)
	}

	access2 = ts.store.GetAuthedUser(serverId, u2host)
	if access2 != nil {
		t.Error("User was not logged out.")
	}
	access2, err = ts.store.FindUser(u2user)
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if access2 != nil {
		t.Error("User was not deleted.")
	}

	err = rspChk(ts, registerSuccess, u2host, register, password)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, deluserSuccess, u1host, deluser, "*"+u2user)
	if err != nil {
		t.Error(err)
	}

	access2 = ts.store.GetAuthedUser(serverId, u2host)
	if access2 != nil {
		t.Error("User was not logged out.")
	}
	access2, err = ts.store.FindUser(u2user)
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if access2 != nil {
		t.Error("User was not deleted.")
	}

	err = rspChk(ts, ".*could not be found.*", u1host, deluser, "noexist")
	if err != nil {
		t.Error(err)
	}
	err = rspChk(ts, ".*is not registered.*", u1host, deluser, "*noexist")
	if err != nil {
		t.Error(err)
	}
	err = rspChk(ts, ".*Username must follow.*", u1host, deluser, "*")
	if err != nil {
		t.Error(err)
	}
}

func TestCoreCommands_Delme(t *T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)

	var err error

	err = rspChk(ts, registerSuccessFirst, u1host, register, password, u1user)
	if err != nil {
		t.Error(err)
	}

	access := ts.store.GetAuthedUser(serverId, u1host)
	if access == nil {
		t.Error("User was not authenticated.")
	}

	err = rspChk(ts, delmeSuccess, u1host, delme)
	if err != nil {
		t.Error(err)
	}

	access = ts.store.GetAuthedUser(serverId, u1host)
	if access != nil {
		t.Error("User was not logged out.")
	}
	access, err = ts.store.FindUser(u1user)
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if access != nil {
		t.Error("User was not deleted.")
	}

	err = rspChk(ts, ".*not authenticated.*", u1host, delme)
	if err != nil {
		t.Error(err)
	}
}

func TestCoreCommands_Passwd(t *T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)

	var err error
	var access *data.UserAccess

	newpasswd := "newpasswd"

	err = rspChk(ts, ".*not authenticated.*", u1host, passwd, password,
		newpasswd)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, registerSuccessFirst, u1host, register, password, u1user)
	if err != nil {
		t.Error(err)
	}

	access = ts.store.GetAuthedUser(serverId, u1host)
	if access == nil {
		t.Error("User was not authenticatd.")
	}
	oldPwd := access.Password

	err = rspChk(ts, passwdSuccess, u1host, passwd, password,
		newpasswd)
	if err != nil {
		t.Error(err)
	}

	access = ts.store.GetAuthedUser(serverId, u1host)
	if access == nil {
		t.Error("User was not authenticatd.")
	}
	if bytes.Compare(access.Password, oldPwd) == 0 {
		t.Error("Password was not changed.")
	}

	err = rspChk(ts, passwdFailure, u1host, passwd, password, newpasswd)
	if err != nil {
		t.Error(err)
	}
}

func TestCoreCommands_Masks(t *T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)

	var err error
	var access *data.UserAccess

	err = rspChk(ts, ".*not authenticated.*", u1host, addmask, u1host)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, registerSuccessFirst, u1host, register, password, u1user)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, addmaskSuccess, u1host, addmask, u1host)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, addmaskFailure, u1host, addmask, u1host)
	if err != nil {
		t.Error(err)
	}

	access = ts.store.GetAuthedUser(serverId, u1host)
	if access == nil {
		t.Fatal("User was not authed.")
	}
	if len(access.Masks) != 1 || access.Masks[0] != u1host {
		t.Error("Mask not set correctly.")
	}

	err = rspChk(ts, "Host [.*] does not match.*", u2host, auth, password,
		u1user)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, ".*"+u1host+".*", u1host, masks)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, delmaskSuccess, u1host, delmask, u1host)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, delmaskFailure, u1host, delmask, u1host)
	if err != nil {
		t.Error(err)
	}

	access = ts.store.GetAuthedUser(serverId, u1host)
	if access == nil {
		t.Fatal("User was not authed.")
	}
	if len(access.Masks) != 0 {
		t.Error("Mask not removed correctly.")
	}

	err = rspChk(ts, masksFailure, u1host, masks)
	if err != nil {
		t.Error(err)
	}
}
