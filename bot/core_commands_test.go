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
		Ssl(true).Prefix(prefix).Server(serverID)

	b, err := createBot(conf, nil, func(_ string) (*data.Store, error) {
		return data.CreateStore(data.MemStoreProvider)
	}, true, true)

	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	srv := b.servers[serverID]
	buf := &bytes.Buffer{}
	srv.endpoint.Writer = buf

	srv.state.Update(&irc.Message{
		Sender: serverID, Name: irc.RPL_WELCOME,
		Args: []string{"Welcome", bothost},
	})
	srv.state.Update(&irc.Message{
		Sender: bothost, Name: irc.JOIN,
		Args: []string{channel},
	})
	srv.state.Update(&irc.Message{
		Sender: u1host, Name: irc.JOIN,
		Args: []string{channel},
	})
	srv.state.Update(&irc.Message{
		Sender: u2host, Name: irc.PRIVMSG,
		Args: []string{botnick, "hithere"},
	})

	return &tSetup{b, srv.endpoint.DataEndpoint, srv.state, b.store, buf, t}
}

func commandsTeardown(s *tSetup, t *T) {
	if s.store != nil {
		s.store.Close()
	}
	s.b.coreCommands.unregisterCoreCmds()
}

func pubRspChk(ts *tSetup, expected, sender string, args ...string) error {
	return prvRspChk(ts, expected, channel, sender, args...)
}

func rspChk(ts *tSetup, expected, sender string, args ...string) error {
	return prvRspChk(ts, expected, botnick, sender, args...)
}

func prvRspChk(ts *tSetup, expected, to, sender string, args ...string) error {
	ts.buffer.Reset()
	err := ts.b.cmds.Dispatch(serverID, 0, &irc.Message{
		Name: irc.PRIVMSG, Sender: sender,
		Args: []string{to, strings.Join(args, " ")},
	}, ts.ep)
	ts.b.cmds.WaitForHandlers()

	s := ts.buffer.String()
	if len(s) == 0 {
		if err != nil {
			return fmt.Errorf("Buffer not full and error returned: %v", err)
		}
		return fmt.Errorf("Everything should generate a response.")
	}

	rgx := `^NOTICE [A-Za-z0-9]+ :` + rgxCreator.Replace(expected) + `$`
	match, err := regexp.MatchString(rgx, s)
	if err != nil {
		return fmt.Errorf("Error making pattern: \n\t%s\n\t%s", expected, rgx)
	}
	if !match {
		return fmt.Errorf("\nUnexpected Response: \n\t%s\n\t%s", s, rgx)
	}
	return nil
}

func TestCoreCommands(t *T) {
	conf := Configure().Nick("nobody").Altnick("nobody1").Username("nobody").
		Userhost("bitforge.ca").Realname("ultimateq").NoReconnect(true).
		Ssl(true).Server(serverID)

	b, err := createBot(conf, nil, func(_ string) (*data.Store, error) {
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

	if ts.store.GetAuthedUser(serverID, u1user) != nil {
		t.Error("Somehow user was authed already.")
	}

	err = rspChk(ts, registerSuccessFirst, u1host, register, password, u1user)
	if err != nil {
		t.Error(err)
	}

	access := ts.store.GetAuthedUser(serverID, u1host)
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

	access = ts.store.GetAuthedUser(serverID, u2host)
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

	ts.store.Logout(serverID, u2host)
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
	access := ts.store.GetAuthedUser(serverID, u1host)
	if access == nil {
		t.Error("User was not authenticated.")
	}
	err = rspChk(ts, logoutSuccess, u1host, logout)
	if err != nil {
		t.Error(err)
	}
	access = ts.store.GetAuthedUser(serverID, u1host)
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

	access = ts.store.GetAuthedUser(serverID, u1host)
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
	err = rspChk(ts, registerSuccess, u2host, register, password)
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

	access := ts.store.GetAuthedUser(serverID, u1host)
	if access != nil {
		t.Error("User was not logged out.")
	}

	err = rspChk(ts, authSuccess, u1host, auth, password)
	if err != nil {
		t.Error(err)
	}
	err = rspChk(ts, ".*(G) global flag(s) required.*", u2host, logout, u1nick)
	if err != nil {
		t.Error(err)
	}
	err = rspChk(ts, logoutSuccess, u1host, logout, u2nick)
	if err != nil {
		t.Error(err)
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

	access1 := ts.store.GetAuthedUser(serverID, u1host)
	access2 := ts.store.GetAuthedUser(serverID, u1host)
	if access1 == nil || access2 == nil {
		t.Error("User's were not authenticated.")
	}

	err = rspChk(ts, ".*(G) global flag(s) required.*", u2host,
		deluser, "*"+u1user)
	if err != nil {
		t.Error(err)
	}
	err = rspChk(ts, deluserSuccess, u1host, deluser, u2nick)
	if err != nil {
		t.Error(err)
	}

	access2 = ts.store.GetAuthedUser(serverID, u2host)
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

	access2 = ts.store.GetAuthedUser(serverID, u2host)
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

	access := ts.store.GetAuthedUser(serverID, u1host)
	if access == nil {
		t.Error("User was not authenticated.")
	}

	err = rspChk(ts, delmeSuccess, u1host, delme)
	if err != nil {
		t.Error(err)
	}

	access = ts.store.GetAuthedUser(serverID, u1host)
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

	access = ts.store.GetAuthedUser(serverID, u1host)
	if access == nil {
		t.Error("User was not authenticatd.")
	}
	oldPwd := access.Password

	err = rspChk(ts, passwdSuccess, u1host, passwd, password,
		newpasswd)
	if err != nil {
		t.Error(err)
	}

	access = ts.store.GetAuthedUser(serverID, u1host)
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

	other := "other!other@other"
	ts.state.Update(&irc.Message{
		Name: irc.PRIVMSG, Sender: other,
		Args: []string{botnick}},
	)

	var err error
	var access *data.UserAccess

	err = rspChk(ts, registerSuccessFirst, u1host, register, password, u1user)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, registerSuccess, u2host, register, password)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, addmaskSuccess, u1host, addmask, u1host)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, ".*(G) global flag(s) required.*", u2host, addmask,
		u1host, u1nick)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, addmaskSuccess, u1host, addmask, u2host, u2nick)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, addmaskFailure, u1host, addmask, u1host)
	if err != nil {
		t.Error(err)
	}

	access = ts.store.GetAuthedUser(serverID, u1host)
	if access == nil {
		t.Fatal("User was not authed.")
	}
	if len(access.Masks) != 1 || access.Masks[0] != u1host {
		t.Error("Mask not set correctly.")
	}

	err = rspChk(ts, "Host [.*] does not match.*", "other!other@other",
		auth, password, u1user)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, ".*(G) global flag(s) required.*", u2host, masks, u1nick)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, ".*"+u2host+".*", u1host, masks, u2nick)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, ".*"+u1host+".*", u1host, masks)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, ".*"+u1host+".*", u1host, masks)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, ".*(G) global flag(s) required.*", u2host, delmask,
		u1host, u1nick)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, delmaskSuccess, u1host, delmask, u2host, u2nick)
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

	access = ts.store.GetAuthedUser(serverID, u1host)
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

func TestCoreCommands_Resetpasswd(t *T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)
	var err error
	var access *data.UserAccess

	err = rspChk(ts, registerSuccessFirst, u1host, register, password, u1user)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, registerSuccess, u2host, register, password)
	if err != nil {
		t.Error(err)
	}

	access = ts.store.GetAuthedUser(serverID, u2host)
	if access == nil {
		t.Fatal("User was not authenticatd.")
	}
	oldPwd := access.Password

	doubleMessage := resetpasswdSuccess + "NOTICE " +
		u2nick + " :" + resetpasswdSuccessTarget
	err = rspChk(ts, doubleMessage, u1host, resetpasswd, u2nick, "*"+u2user)
	if err != nil {
		t.Error(err)
	}

	access = ts.store.GetAuthedUser(serverID, u2host)
	if access == nil {
		t.Fatal("User was not authenticatd.")
	}
	if bytes.Compare(access.Password, oldPwd) == 0 {
		t.Error("Password was not changed.")
	}
}

func TestCoreCommands_GiveTakeGlobal(t *T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)
	var err error
	var a *data.UserAccess

	err = rspChk(ts, registerSuccessFirst, u1host, register, password, u1user)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, registerSuccess, u2host, register, password)
	if err != nil {
		t.Error(err)
	}

	a, err = ts.store.FindUser(u2user)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, ggiveSuccess, u1host, ggive, u2nick, "100", "h")
	if err != nil {
		t.Error(err)
	}

	if !a.HasGlobalFlag('h') || !a.HasGlobalLevel(100) {
		t.Error("Global access not granted correctly.")
	}

	err = rspChk(ts, ggiveSuccess, u1host, gtake, u2nick)
	if err != nil {
		t.Error(err)
	}

	if a.HasGlobalLevel(100) {
		t.Error("Global access not taken correctly.")
	}

	err = rspChk(ts, ggiveSuccess, u1host, gtake, u2nick, "h")
	if err != nil {
		t.Error(err)
	}

	if a.HasGlobalFlag('h') {
		t.Error("Global access not taken correctly.")
	}

	a.GrantGlobal(100, "h")
	err = rspChk(ts, ggiveSuccess, u1host, gtake, u2nick, "all")
	if err != nil {
		t.Error(err)
	}

	if a.HasGlobalLevel(100) || a.HasGlobalFlag('h') {
		t.Error("Global access not taken correctly.")
	}

	err = rspChk(ts, takeFailureNo, u1host, gtake, u2nick, "h")
	if err != nil {
		t.Error(err)
	}
}

func TestCoreCommands_GiveTakeServer(t *T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)
	var err error
	var a *data.UserAccess

	err = rspChk(ts, registerSuccessFirst, u1host, register, password, u1user)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, registerSuccess, u2host, register, password)
	if err != nil {
		t.Error(err)
	}

	a, err = ts.store.FindUser(u2user)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, sgiveSuccess, u1host, sgive, u2nick, "100", "h")
	if err != nil {
		t.Error(err)
	}

	if !a.HasServerFlag(serverID, 'h') || !a.HasServerLevel(serverID, 100) {
		t.Error("Server access not granted correctly.")
	}

	err = rspChk(ts, sgiveSuccess, u1host, stake, u2nick)
	if err != nil {
		t.Error(err)
	}

	if a.HasServerLevel(serverID, 100) {
		t.Error("Server access not taken correctly.")
	}

	err = rspChk(ts, sgiveSuccess, u1host, stake, u2nick, "h")
	if err != nil {
		t.Error(err)
	}

	if a.HasServerFlag(serverID, 'h') {
		t.Error("Server access not taken correctly.")
	}

	a.GrantServer(serverID, 100, "h")
	err = rspChk(ts, sgiveSuccess, u1host, stake, u2nick, "all")
	if err != nil {
		t.Error(err)
	}

	if a.HasServerLevel(serverID, 100) || a.HasServerFlag(serverID, 'h') {
		t.Error("Server access not taken correctly.")
	}

	err = rspChk(ts, takeFailureNo, u1host, stake, u2nick, "h")
	if err != nil {
		t.Error(err)
	}
}

func TestCoreCommands_GiveTakeChannel(t *T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)
	var err error
	var a *data.UserAccess

	err = rspChk(ts, registerSuccessFirst, u1host, register, password, u1user)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, registerSuccess, u2host, register, password)
	if err != nil {
		t.Error(err)
	}

	a, err = ts.store.FindUser(u2user)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, giveSuccess, u1host, give, channel, u2nick, "100", "h")
	if err != nil {
		t.Error(err)
	}

	if !a.HasChannelFlag(serverID, channel, 'h') ||
		!a.HasChannelLevel(serverID, channel, 100) {
		t.Error("Channel access not granted correctly.")
	}

	err = rspChk(ts, giveSuccess, u1host, take, channel, u2nick)
	if err != nil {
		t.Error(err)
	}

	if a.HasChannelLevel(serverID, channel, 100) {
		t.Error("Channel access not taken correctly.")
	}

	err = rspChk(ts, giveSuccess, u1host, take, channel, u2nick, "h")
	if err != nil {
		t.Error(err)
	}

	if a.HasChannelFlag(serverID, channel, 'h') {
		t.Error("Channel access not taken correctly.")
	}

	a.GrantChannel(serverID, channel, 100, "h")
	err = rspChk(ts, giveSuccess, u1host, take, channel, u2nick, "all")
	if err != nil {
		t.Error(err)
	}

	if a.HasChannelFlag(serverID, channel, 'h') ||
		a.HasChannelLevel(serverID, channel, 100) {
		t.Error("Channel access not taken correctly.")
	}

	err = rspChk(ts, takeFailureNo, u1host, take, channel, u2nick, "h")
	if err != nil {
		t.Error(err)
	}
}

func TestCoreCommands_Help(t *T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)
	var err error

	err = rspChk(ts, registerSuccessFirst, u1host, register, password, u1user)
	if err != nil {
		t.Error(err)
	}

	check := `core:NOTICE .* (access|addmask|auth){3}.*`
	err = rspChk(ts, check, u1host, help)
	if err != nil {
		t.Error(err)
	}

	check = helpSuccess + " " + extension + "." + register +
		`NOTICE .* :` + registerDesc +
		`NOTICE .* :` + helpSuccessUsage + strings.Join(commands[0].Args, " ")
	err = rspChk(ts, check, u1host, help, register)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, check, u1host, help, "core."+register)
	if err != nil {
		t.Error(err)
	}

	check = helpSuccess + " " + extension + "." + delme +
		`NOTICE .* :` + delmeDesc
	err = rspChk(ts, check, u1host, help, delme)
	if err != nil {
		t.Error(err)
	}

	check = `core:NOTICE .* (give|ggive|sgive|register){4}`
	err = rspChk(ts, check, u1host, help, "gi")
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, helpFailure, u1host, help, "badsearch")
	if err != nil {
		t.Error(err)
	}
}

func TestCoreCommands_Gusers(t *T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)
	var a1, a2 *data.UserAccess
	var err error

	err = rspChk(ts, gusersNoUsers, u1host, gusers)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, registerSuccessFirst, u1host, register, password, u1user)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, registerSuccess, u2host, register, password)
	if err != nil {
		t.Error(err)
	}

	a1, err = ts.store.FindUser(u1user)
	if err != nil {
		t.Fatal("Could not find user1.")
	}
	a2, err = ts.store.FindUser(u2user)
	if err != nil {
		t.Fatal("Could not find user1.")
	}
	a2.GrantGlobal(100, "abc")
	if err = ts.store.AddUser(a2); err != nil {
		t.Fatal("Could not save user.")
	}

	check := gusersHead +
		`NOTICE .* :` + usersListHeadUser + `.*` + usersListHeadAccess +
		`NOTICE .* :` + u1user + `.*` + a1.Global.String() +
		`NOTICE .* :` + u2user + `.*` + a2.Global.String()
	err = rspChk(ts, check, u1host, gusers)
	if err != nil {
		t.Error(err)
	}
}

func TestCoreCommands_Susers(t *T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)
	var a1, a2 *data.UserAccess
	var err error

	err = rspChk(ts, susersNoUsers, u1host, susers)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, registerSuccessFirst, u1host, register, password, u1user)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, registerSuccess, u2host, register, password)
	if err != nil {
		t.Error(err)
	}

	a1, err = ts.store.FindUser(u1user)
	if err != nil {
		t.Fatal("Could not find user1.")
	}
	a1.GrantServer(serverID, 2, "b")
	a2, err = ts.store.FindUser(u2user)
	if err != nil {
		t.Fatal("Could not find user1.")
	}
	a2.GrantServer(serverID, 100, "abc")

	if err = ts.store.AddUser(a1); err != nil {
		t.Fatal("Could not save user.")
	}
	if err = ts.store.AddUser(a2); err != nil {
		t.Fatal("Could not save user.")
	}

	check := susersHead +
		`NOTICE .* :` + usersListHeadUser + `.*` + usersListHeadAccess +
		`NOTICE .* :` + u1user + `.*` + a1.GetServer(serverID).String() +
		`NOTICE .* :` + u2user + `.*` + a2.GetServer(serverID).String()
	err = rspChk(ts, check, u1host, susers)
	if err != nil {
		t.Error(err)
	}
}

func TestCoreCommands_Users(t *T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)
	var a1, a2 *data.UserAccess
	var err error

	err = rspChk(ts, gusersNoUsers, u1host, gusers)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, registerSuccessFirst, u1host, register, password, u1user)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, registerSuccess, u2host, register, password)
	if err != nil {
		t.Error(err)
	}

	a1, err = ts.store.FindUser(u1user)
	if err != nil {
		t.Fatal("Could not find user1.")
	}
	a1.GrantChannel(serverID, channel, 3, "c")
	a2, err = ts.store.FindUser(u2user)
	if err != nil {
		t.Fatal("Could not find user1.")
	}
	a2.GrantChannel(serverID, channel, 100, "abc")

	if err = ts.store.AddUser(a1); err != nil {
		t.Fatal("Could not save user.")
	}
	if err = ts.store.AddUser(a2); err != nil {
		t.Fatal("Could not save user.")
	}

	check := usersHead +
		`NOTICE .* :` + usersListHeadUser + `.*` + usersListHeadAccess +
		`NOTICE .* :` + u1user + `.*` + a1.GetChannel(serverID, channel).String() +
		`NOTICE .* :` + u2user + `.*` + a2.GetChannel(serverID, channel).String()
	err = rspChk(ts, check, u1host, users, channel)
	if err != nil {
		t.Error(err)
	}
}
