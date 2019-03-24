package bot

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/aarondl/ultimateq/config"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/irc"
)

const (
	bothost   = "bot!botuser@bothost"
	botnick   = "bot"
	u1host    = "nick1!user1@host1"
	u1nick    = "nick1"
	u1user    = "user"
	u2host    = "nick2!user2@host2"
	u2nick    = "nick2"
	u2user    = "user2"
	u2userArg = "*user2"
	channel   = "#chan"
	password  = "password"
	prefix    = '.'
)

var (
	rgxCreator = strings.NewReplacer(
		`(`, `\(`, `)`, `\)`, `]`, `\]`, `[`,
		`\[`, `\`, `\\`, `/`, `\/`, `%v`, `.*`,
	)
	netInfo = irc.NewNetworkInfo()
)

type tSetup struct {
	b        *Bot
	provider data.Provider
	writer   irc.Writer
	state    *data.State
	store    *data.Store
	buffer   *bytes.Buffer
	t        *testing.T
}

func commandsSetup(t *testing.T) *tSetup {
	conf := config.New()
	conf.Network("").SetNick("nobody").SetAltnick("nobody1").
		SetUsername("nobody").SetRealname("ultimateq").
		SetNoReconnect(true).SetTLS(true).SetPrefix(prefix)
	conf.NewNetwork(netID)

	b, err := createBot(conf, nil, func(_ string) (*data.Store, error) {
		return data.NewStore(data.MemStoreProvider)
	}, devNull, true, true)

	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	srv := b.servers[netID]
	buf := &bytes.Buffer{}
	srv.writer = irc.Helper{Writer: buf}

	srv.state.Update(
		irc.NewEvent(netID, netInfo, irc.RPL_WELCOME, "", "Hi", bothost),
	)
	srv.state.Update(
		irc.NewEvent(netID, netInfo, irc.JOIN, bothost, channel),
	)
	srv.state.Update(
		irc.NewEvent(netID, netInfo, irc.JOIN, u1host, channel),
	)
	srv.state.Update(
		irc.NewEvent(netID, netInfo, irc.PRIVMSG, u2host, botnick, "hi"),
	)

	return &tSetup{b, b, srv.writer, srv.state, b.store, buf, t}
}

func commandsTeardown(s *tSetup, t *testing.T) {
	if s.store != nil {
		s.store.Close()
	}
	// One day I hope to know why we needed this?
	// s.b.coreCommands.unregisterCoreCmds()
}

func pubRspChk(ts *tSetup, expected, sender string, args ...string) error {
	return prvRspChk(ts, expected, channel, sender, args...)
}

func rspChk(ts *tSetup, expected, sender string, args ...string) error {
	return prvRspChk(ts, expected, botnick, sender, args...)
}

func prvRspChk(ts *tSetup, expected, to, sender string, args ...string) error {
	ts.t.Helper()
	ts.buffer.Reset()
	err := ts.b.cmds.Dispatch(ts.writer, irc.NewEvent(
		netID, netInfo, irc.PRIVMSG, sender, to, strings.Join(args, " ")),
		ts.provider,
	)
	ts.b.cmds.WaitForHandlers()

	s := ts.buffer.String()
	if len(s) == 0 {
		if err != nil {
			return fmt.Errorf("buffer not full and error returned: %v", err)
		}
		return fmt.Errorf("everything should generate a response")
	}

	rgx := `^NOTICE [A-Za-z0-9]+ :` + rgxCreator.Replace(expected) + `$`
	match, err := regexp.MatchString(rgx, s)
	if err != nil {
		return fmt.Errorf("error making pattern: \n\t%s\n\t%s", expected, rgx)
	}
	if !match {
		return fmt.Errorf("\nunexpected response: \n\t%s\n\t%s", s, rgx)
	}
	return nil
}

func TestCoreCommands(t *testing.T) {
	conf := config.New()
	conf.Network("").SetNick("nobody").SetAltnick("nobody1").
		SetUsername("nobody").SetRealname("ultimateq").SetNoReconnect(true).
		SetTLS(true)
	conf.NewNetwork(netID)

	b, err := createBot(conf, nil, func(_ string) (*data.Store, error) {
		return data.NewStore(data.MemStoreProvider)
	}, devNull, true, true)

	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if b.coreCommands == nil {
		t.Error("Core commands should have been attached.")
	}

	commandsTeardown(&tSetup{b: b}, t)
}

func TestCoreCommands_Register(t *testing.T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)

	var err error

	if ts.store.AuthedUser(netID, u1user) != nil {
		t.Error("Somehow user was authed already.")
	}

	err = rspChk(ts, registerSuccessFirst, u1host, register, password, u1user)
	if err != nil {
		t.Error(err)
	}

	access := ts.store.AuthedUser(netID, u1host)
	if access == nil {
		t.Error("User was not authenticated.")
	} else {
		accessObj := ignoreOK(access.GetAccess("", ""))
		if accessObj.Level != ^uint8(0) {
			t.Error("Level not granted.")
		}
		if accessObj.Flags != ^uint64(0) {
			t.Error("Flags not granted.")
		}
	}

	err = rspChk(ts, registerSuccess, u2host, register, passwd)
	if err != nil {
		t.Error(err)
	}

	access = ts.store.AuthedUser(netID, u2host)
	if access == nil {
		t.Error("User was not authenticated.")
	} else if accessObj, ok := access.GetAccess("", ""); ok {
		if accessObj.Level != 0 {
			t.Error("Level granted by mistake.")
		}
		if accessObj.Flags != 0 {
			t.Error("Flags granted by mistake.")
		}
	}

	ts.store.Logout(netID, u1host)
	err = rspChk(ts, registerFailure, u1host, register, passwd, u1user)
	if err != nil {
		t.Error(err)
	}
}

func TestCoreCommands_Auth(t *testing.T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)

	var err error

	err = rspChk(ts, registerSuccessFirst, u1host, register, password)
	if err != nil {
		t.Error(err)
	}
	access := ts.store.AuthedUser(netID, u1host)
	if access == nil {
		t.Error("User was not authenticated.")
	}
	err = rspChk(ts, logoutSuccess, u1host, logout)
	if err != nil {
		t.Error(err)
	}
	access = ts.store.AuthedUser(netID, u1host)
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

	access = ts.store.AuthedUser(netID, u1host)
	if access == nil {
		t.Error("User was not authenticated.")
	}
}

func TestCoreCommands_Logout(t *testing.T) {
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

	access := ts.store.AuthedUser(netID, u1host)
	if access != nil {
		t.Error("User was not logged out.")
	}

	err = rspChk(ts, authSuccess, u1host, auth, password)
	if err != nil {
		t.Error(err)
	}
	err = rspChk(ts, ".*(G) global flag(s) required.*", u2host, logout, u2userArg)
	if err != nil {
		t.Error(err)
	}
	err = rspChk(ts, logoutSuccess, u1host, logout, u2userArg)
	if err != nil {
		t.Error(err)
	}
}

func TestCoreCommands_Access(t *testing.T) {
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

	err = pubRspChk(ts, accessSuccess, u1host, string(prefix)+access)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, accessSuccess, u1host, access, u2userArg)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, accessSuccess, u1host, access, u2userArg)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, logoutSuccess, u1host, logout)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, accessSuccess, u2host, access, "*"+u1user)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, ".*not authenticated.*", u2host, access, u1nick)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, ".*Username must follow.*", u2host, access, "*")
	if err != nil {
		t.Error(err)
	}
}

func TestCoreCommands_Deluser(t *testing.T) {
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

	access1 := ts.store.AuthedUser(netID, u1host)
	access2 := ts.store.AuthedUser(netID, u1host)
	if access1 == nil || access2 == nil {
		t.Error("Users were not authenticated.")
	}

	err = rspChk(ts, ".*(G) global flag(s) required.*", u2host,
		deluser, "*"+u1user)
	if err != nil {
		t.Error(err)
	}
	err = rspChk(ts, deluserSuccess, u1host, deluser, u2userArg)
	if err != nil {
		t.Error(err)
	}

	access2 = ts.store.AuthedUser(netID, u2host)
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

	err = rspChk(ts, deluserSuccess, u1host, deluser, u2userArg)
	if err != nil {
		t.Error(err)
	}

	access2 = ts.store.AuthedUser(netID, u2host)
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

func TestCoreCommands_Delme(t *testing.T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)

	var err error

	err = rspChk(ts, registerSuccessFirst, u1host, register, password, u1user)
	if err != nil {
		t.Error(err)
	}

	access := ts.store.AuthedUser(netID, u1host)
	if access == nil {
		t.Error("User was not authenticated.")
	}

	err = rspChk(ts, delmeSuccess, u1host, delme)
	if err != nil {
		t.Error(err)
	}

	access = ts.store.AuthedUser(netID, u1host)
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

func TestCoreCommands_Passwd(t *testing.T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)

	var err error
	var access *data.StoredUser

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

	access = ts.store.AuthedUser(netID, u1host)
	if access == nil {
		t.Error("User was not authenticatd.")
	}
	oldPwd := access.Password

	err = rspChk(ts, passwdSuccess, u1host, passwd, password,
		newpasswd)
	if err != nil {
		t.Error(err)
	}

	access = ts.store.AuthedUser(netID, u1host)
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

func TestCoreCommands_Masks(t *testing.T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)

	other := "other!other@other"
	ts.state.Update(
		irc.NewEvent(netID, netInfo, irc.PRIVMSG, other, botnick),
	)

	var err error
	var access *data.StoredUser

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

	err = rspChk(ts, addmaskSuccess, u1host, addmask, u2host, u2userArg)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, addmaskFailure, u1host, addmask, u1host)
	if err != nil {
		t.Error(err)
	}

	access = ts.store.AuthedUser(netID, u1host)
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

	err = rspChk(ts, ".*"+u2host+".*", u1host, masks, u2userArg)
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

	err = rspChk(ts, delmaskSuccess, u1host, delmask, u2host, u2userArg)
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

	access = ts.store.AuthedUser(netID, u1host)
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

func TestCoreCommands_Resetpasswd(t *testing.T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)
	var err error
	var access *data.StoredUser

	err = rspChk(ts, registerSuccessFirst, u1host, register, password, u1user)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, registerSuccess, u2host, register, password)
	if err != nil {
		t.Error(err)
	}

	access = ts.store.AuthedUser(netID, u2host)
	if access == nil {
		t.Fatal("User was not authenticatd.")
	}
	oldPwd := access.Password

	doubleMessage := resetpasswdSuccess + "NOTICE " +
		u2nick + " :" + resetpasswdSuccessTarget
	err = rspChk(ts, doubleMessage, u1host, resetpasswd, u2nick, u2userArg)
	if err != nil {
		t.Error(err)
	}

	access = ts.store.AuthedUser(netID, u2host)
	if access == nil {
		t.Fatal("User was not authenticatd.")
	}
	if bytes.Compare(access.Password, oldPwd) == 0 {
		t.Error("Password was not changed.")
	}
}

func TestCoreCommands_GiveTakeGlobal(t *testing.T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)
	var err error
	var a *data.StoredUser

	err = rspChk(ts, registerSuccessFirst, u1host, register, password, u1user)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, registerSuccess, u2host, register, password)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, ggiveSuccess, u1host, ggive, u2userArg, "100", "h")
	if err != nil {
		t.Error(err)
	}

	a = testGetUser(ts.store.FindUser(u2user))
	if !a.HasFlags("", "", "h") || !a.HasLevel("", "", 100) {
		t.Error("Global access not granted correctly.")
	}

	err = rspChk(ts, giveFailureHas, u1host, ggive, u2userArg, "h")
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, ggiveSuccess, u1host, gtake, u2userArg)
	if err != nil {
		t.Error(err)
	}

	a = testGetUser(ts.store.FindUser(u2user))
	if a.HasLevel("", "", 100) {
		t.Error("Global access not taken correctly.")
	}

	err = rspChk(ts, ggiveSuccess, u1host, gtake, u2userArg, "h")
	if err != nil {
		t.Error(err)
	}

	a = testGetUser(ts.store.FindUser(u2user))
	if a.HasFlags("", "", "h") {
		t.Error("Global access not taken correctly.")
	}

	a.Grant("", "", 100, "h")
	if err = ts.store.SaveUser(a); err != nil {
		t.Error(err)
	}

	err = rspChk(ts, ggiveSuccess, u1host, gtake, u2userArg, "all")
	if err != nil {
		t.Error(err)
	}

	a = testGetUser(ts.store.FindUser(u2user))
	if a.HasLevel("", "", 100) || a.HasFlags("", "", "h") {
		t.Error("Global access not taken correctly.")
	}

	err = rspChk(ts, takeFailureNo, u1host, gtake, u2userArg, "h")
	if err != nil {
		t.Error(err)
	}
}

func TestCoreCommands_GiveTakeNetwork(t *testing.T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)
	var err error
	var a *data.StoredUser

	err = rspChk(ts, registerSuccessFirst, u1host, register, password, u1user)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, registerSuccess, u2host, register, password)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, sgiveSuccess, u1host, sgive, u2userArg, "100", "h")
	if err != nil {
		t.Error(err)
	}

	a = testGetUser(ts.store.FindUser(u2user))
	if !a.HasFlags(netID, "", "h") || !a.HasLevel(netID, "", 100) {
		t.Error("Network access not granted correctly.")
	}

	err = rspChk(ts, giveFailureHas, u1host, sgive, u2userArg, "h")
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, sgiveSuccess, u1host, stake, u2userArg)
	if err != nil {
		t.Error(err)
	}

	a = testGetUser(ts.store.FindUser(u2user))
	if a.HasLevel(netID, "", 100) {
		t.Error("Network access not taken correctly.")
	}

	err = rspChk(ts, sgiveSuccess, u1host, stake, u2userArg, "h")
	if err != nil {
		t.Error(err)
	}

	a = testGetUser(ts.store.FindUser(u2user))
	if a.HasFlags(netID, "", "h") {
		t.Error("Network access not taken correctly.")
	}

	a.Grant(netID, "", 100, "h")
	if err = ts.store.SaveUser(a); err != nil {
		t.Error(err)
	}

	err = rspChk(ts, sgiveSuccess, u1host, stake, u2userArg, "all")
	if err != nil {
		t.Error(err)
	}

	a = testGetUser(ts.store.FindUser(u2user))
	if a.HasLevel(netID, "", 100) || a.HasFlags(netID, "", "h") {
		t.Error("Network access not taken correctly.")
	}

	err = rspChk(ts, takeFailureNo, u1host, stake, u2userArg, "h")
	if err != nil {
		t.Error(err)
	}
}

func TestCoreCommands_GiveTakeChannel(t *testing.T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)
	var err error
	var a *data.StoredUser

	err = rspChk(ts, registerSuccessFirst, u1host, register, password, u1user)
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, registerSuccess, u2host, register, password)
	if err != nil {
		t.Error(err)
	}

	a = testGetUser(ts.store.FindUser(u2user))
	err = rspChk(ts, giveSuccess, u1host, give, channel, u2userArg, "100", "h")
	if err != nil {
		t.Error(err)
	}

	a = testGetUser(ts.store.FindUser(u2user))
	if !a.HasFlags(netID, channel, "h") || !a.HasLevel(netID, channel, 100) {
		t.Error("Channel access not granted correctly.")
	}

	err = rspChk(ts, giveFailureHas, u1host, give, channel, u2userArg, "h")
	if err != nil {
		t.Error(err)
	}

	err = rspChk(ts, giveSuccess, u1host, take, channel, u2userArg)
	if err != nil {
		t.Error(err)
	}

	a = testGetUser(ts.store.FindUser(u2user))
	if a.HasLevel(netID, channel, 100) {
		t.Error("Channel access not taken correctly.")
	}

	err = rspChk(ts, giveSuccess, u1host, take, channel, u2userArg, "h")
	if err != nil {
		t.Error(err)
	}

	a = testGetUser(ts.store.FindUser(u2user))
	if a.HasFlags(netID, channel, "h") {
		t.Error("Channel access not taken correctly.")
	}

	a.Grant(netID, channel, 100, "h")
	if err = ts.store.SaveUser(a); err != nil {
		t.Error(err)
	}

	err = rspChk(ts, giveSuccess, u1host, take, channel, u2userArg, "all")
	if err != nil {
		t.Error(err)
	}

	a = testGetUser(ts.store.FindUser(u2user))
	if a.HasFlags(netID, channel, "h") || a.HasLevel(netID, channel, 100) {
		t.Error("Channel access not taken correctly.")
	}

	err = rspChk(ts, takeFailureNo, u1host, take, channel, u2userArg, "h")
	if err != nil {
		t.Error(err)
	}
}

func TestCoreCommands_Help(t *testing.T) {
	t.Skip("currently totally broken")
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

	check = fmt.Sprintf("%s %s.%sNOTICE .* :%sNOTICE .* :%s",
		helpSuccess, extension, register, registerDesc,
		fmt.Sprintf(helpSuccessUsage, register,
			strings.Join(commands[0].Args, " ")))
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

func TestCoreCommands_Gusers(t *testing.T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)
	var a1, a2 *data.StoredUser
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
	a2.Grant("", "", 100, "abc")
	if err = ts.store.SaveUser(a2); err != nil {
		t.Fatal("Could not save user.")
	}

	check := gusersHead +
		`NOTICE .* :` + usersListHeadUser + `.*` + usersListHeadAccess +
		`NOTICE .* :` + u1user + `.*` + a1.String("", "") +
		`NOTICE .* :` + u2user + `.*` + a2.String("", "")
	err = rspChk(ts, check, u1host, gusers)
	if err != nil {
		t.Error(err)
	}
}

func TestCoreCommands_Susers(t *testing.T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)
	var a1, a2 *data.StoredUser
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
	a1.Grant(netID, "", 2, "b")
	a2, err = ts.store.FindUser(u2user)
	if err != nil {
		t.Fatal("Could not find user1.")
	}
	a2.Grant(netID, "", 100, "abc")

	if err = ts.store.SaveUser(a1); err != nil {
		t.Fatal("Could not save user.")
	}
	if err = ts.store.SaveUser(a2); err != nil {
		t.Fatal("Could not save user.")
	}

	check := susersHead +
		`NOTICE .* :` + usersListHeadUser + `.*` + usersListHeadAccess +
		`NOTICE .* :` + u1user + `.*` + a1.String(netID, "") +
		`NOTICE .* :` + u2user + `.*` + a2.String(netID, "")
	err = rspChk(ts, check, u1host, susers)
	if err != nil {
		t.Error(err)
	}
}

func TestCoreCommands_Users(t *testing.T) {
	ts := commandsSetup(t)
	defer commandsTeardown(ts, t)
	var a1, a2 *data.StoredUser
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
	a1.Grant(netID, channel, 3, "c")
	a2, err = ts.store.FindUser(u2user)
	if err != nil {
		t.Fatal("Could not find user1.")
	}
	a2.Grant(netID, channel, 100, "abc")

	if err = ts.store.SaveUser(a1); err != nil {
		t.Fatal("Could not save user.")
	}
	if err = ts.store.SaveUser(a2); err != nil {
		t.Fatal("Could not save user.")
	}

	check := usersHead +
		`NOTICE .* :` + usersListHeadUser + `.*` + usersListHeadAccess +
		`NOTICE .* :` + u1user + `.*` + a1.String(netID, channel) +
		`NOTICE .* :` + u2user + `.*` + a2.String(netID, channel)
	err = rspChk(ts, check, u1host, users, channel)
	if err != nil {
		t.Error(err)
	}
}

func testGetUser(u *data.StoredUser, err error) *data.StoredUser {
	if err == nil {
		return u
	}

	panic(fmt.Sprintf("Failed to get user: %v", err))
}
