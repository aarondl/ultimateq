package commander

import (
	"bytes"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/dispatch"
	"github.com/aarondl/ultimateq/irc"
	"strings"
	"sync"
	. "testing"
)

type basicCommands struct {
	called  bool
	msg     *irc.Message
	end     irc.Endpoint
	user    *data.User
	channel *data.Channel
}

func (b *basicCommands) HandleCommand(msg *irc.Message,
	end *data.DataEndpoint, user *data.User, ch *data.Channel) {

	b.called = true
	b.msg = msg
	b.end = end
	b.user = user
	b.channel = ch
}

type authCommands struct {
	*basicCommands
	access *data.UserAccess
}

func (a *authCommands) HandleAuthedCommand(msg *irc.Message,
	end *data.DataEndpoint, user *data.User, access *data.UserAccess,
	ch *data.Channel) {

	a.HandleCommand(msg, end, user, ch)
	a.access = access
}

var (
	server  = "irc.test.net"
	cmd     = "cmd"
	channel = "#chan"
	nick    = "nick"
)

func setup() (store *data.Store, user *data.UserAccess) {
	var err error
	store, err = data.CreateStore(data.MemStoreProvider)
	if err != nil {
		panic(err)
	}
	user, err = data.CreateUserAccess("user", "pass", "*!*@host")
	if err != nil {
		panic(err)
	}
	err = store.AddUser(user)
	if err != nil {
		panic(err)
	}
	_, err = store.AuthUser(server, "nick!user@host", "user", "pass")
	if err != nil {
		panic(err)
	}

	return
}

var core, _ = dispatch.CreateDispatchCore(irc.CreateProtoCaps())
var prefix = '.'

func TestCommander(t *T) {
	c := CreateCommander(prefix, core)
	if c == nil {
		t.Fatal("Commander should not be nil.")
	}
	if c.prefix != prefix {
		t.Error("Prefix not set correctly.")
	}
	if c.commands == nil {
		t.Error("Globals should have been instantiated.")
	}
}

func TestCommander_Register(t *T) {
	c := CreateCommander(prefix, core)

	handler := &basicCommands{}

	var success bool
	var err error
	err = c.Register(GLOBAL, cmd, nil, ALL, ALL)
	if err == nil {
		t.Error("Should not register nil event handlers.")
	}

	err = c.Register(GLOBAL, cmd, handler, ALL, ALL)
	if err != nil {
		t.Error("Registration failed:", err)
	}

	err = c.Register(GLOBAL, cmd, handler, ALL, ALL)
	if err == nil {
		t.Error("Registration of an existing command should fail.")
	}

	success = c.Unregister(GLOBAL, cmd)
	if !success {
		t.Error("Should have unregistered successfully.")
	}
	success = c.Unregister(GLOBAL, cmd)
	if success {
		t.Error("Should not be able to double unregister.")
	}
}

func TestCommander_RegisterProtected(t *T) {
	c := CreateCommander(prefix, core)

	handler := &basicCommands{}
	var success bool
	var err error
	err = c.RegisterAuthed(GLOBAL, cmd, nil, ALL, ALL, 100, "ab")
	if err == nil {
		t.Error("Should not register nil event handlers.")
	}

	err = c.Register(GLOBAL, cmd, handler, ALL, ALL)
	if err != nil {
		t.Error("Registration failed:", err)
	}

	err = c.Register(GLOBAL, cmd, handler, ALL, ALL)
	if err == nil {
		t.Error("Registration of an existing command should fail.")
	}

	success = c.Unregister(GLOBAL, cmd)
	if !success {
		t.Error("Should have unregistered successfully.")
	}
	success = c.Unregister(GLOBAL, cmd)
	if success {
		t.Error("Should not be able to double unregister.")
	}
}

func TestCommander_Dispatch(t *T) {
	dcore, err := dispatch.CreateDispatchCore(irc.CreateProtoCaps())
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	c := CreateCommander(prefix, dcore)
	c.AddChannels(channel)
	if c == nil {
		t.Error("Commander should not be nil.")
	}

	var buffer = &bytes.Buffer{}
	var storeMutex sync.RWMutex
	store, _ := setup()
	var dataEndpoint = data.CreateDataEndpoint(server, buffer, nil, store,
		nil, &storeMutex)

	handler := &basicCommands{}
	cmsg := []string{channel, string(c.prefix) + cmd}
	badcmsg := []string{"#otherchan", string(c.prefix) + cmd}
	umsg := []string{nick, cmd}
	uargmsg := []string{nick, cmd, "arg1", "arg2"}
	uargvargs := []string{nick, cmd, "arg1", "arg2", "arg3", "arg4"}

	arg1req := []string{"arg1"}
	arg1opt := []string{"[opt1]"}
	argvar := []string{"opts..."}
	arg1req1opt := []string{"arg", "[opt]"}
	argreq1var := []string{"arg1", "opts..."}

	var table = []struct {
		CmdArgs []string
		MsgType int
		Scope   int
		Name    string
		MsgArgs []string
		Called  bool
		ErrMsg  string
	}{
		// Args
		{nil, ALL, ALL, irc.PRIVMSG, umsg, true, ""},
		{arg1opt, ALL, ALL, irc.PRIVMSG, umsg, true, ""},
		{arg1req, ALL, ALL, irc.PRIVMSG, umsg, false, "arguments"},

		{arg1req, ALL, ALL, irc.PRIVMSG, uargmsg, false, "arguments"},
		{arg1opt, ALL, ALL, irc.PRIVMSG, uargmsg, false, "arguments"},
		{arg1req1opt, ALL, ALL, irc.PRIVMSG, uargmsg, true, ""},

		{argreq1var, ALL, ALL, irc.PRIVMSG, umsg, false, "arguments"},
		{argreq1var, ALL, ALL, irc.PRIVMSG, uargvargs, true, ""},
		{argvar, ALL, ALL, irc.PRIVMSG, uargvargs, true, ""},

		// Bad message
		{nil, ALL, ALL, irc.RPL_WHOREPLY, cmsg, false, ""},
		// Message to wrong channel
		{nil, ALL, ALL, irc.PRIVMSG, badcmsg, false, ""},

		// Msgtype All + Scope
		{nil, ALL, ALL, irc.PRIVMSG, cmsg, true, ""},
		{nil, ALL, PRIVATE, irc.PRIVMSG, umsg, true, ""},
		{nil, ALL, PRIVATE, irc.PRIVMSG, cmsg, false, ""},
		{nil, ALL, PUBLIC, irc.PRIVMSG, umsg, false, ""},
		{nil, ALL, PUBLIC, irc.PRIVMSG, cmsg, true, ""},

		// Msgtype Privmsg + Scope
		{nil, PRIVMSG, ALL, irc.PRIVMSG, cmsg, true, ""},
		{nil, PRIVMSG, PRIVATE, irc.PRIVMSG, umsg, true, ""},
		{nil, PRIVMSG, PRIVATE, irc.PRIVMSG, cmsg, false, ""},
		{nil, PRIVMSG, PUBLIC, irc.PRIVMSG, umsg, false, ""},
		{nil, PRIVMSG, PUBLIC, irc.PRIVMSG, cmsg, true, ""},
		{nil, PRIVMSG, ALL, irc.NOTICE, cmsg, false, ""},

		// Msgtype Notice + Scope
		{nil, NOTICE, ALL, irc.NOTICE, cmsg, true, ""},
		{nil, NOTICE, PRIVATE, irc.NOTICE, umsg, true, ""},
		{nil, NOTICE, PRIVATE, irc.NOTICE, cmsg, false, ""},
		{nil, NOTICE, PUBLIC, irc.NOTICE, umsg, false, ""},
		{nil, NOTICE, PUBLIC, irc.NOTICE, cmsg, true, ""},
		{nil, NOTICE, ALL, irc.PRIVMSG, cmsg, false, ""},
	}

	for _, test := range table {
		err := c.Register(GLOBAL, cmd, handler, test.MsgType, test.Scope,
			test.CmdArgs...)
		if err != nil {
			t.Errorf("Failed to register test: [%v]\n(%v)", err, test)
			continue
		}

		handler.called = false
		msg := &irc.IrcMessage{Name: test.Name, Args: test.MsgArgs}
		err = c.Dispatch(msg, dataEndpoint)
		if handler.called != test.Called {
			if handler.called {
				t.Errorf("Test erroneously called: %v", test)
			} else {
				t.Errorf("Test erroneously skipped: %v", test)
			}
		}

		if err != nil {
			if test.ErrMsg == "" {
				t.Errorf("Unexpected User Error: %v\n%v", err, test)
			} else if !strings.Contains(err.Error(), test.ErrMsg) {
				t.Errorf("Expected: %v but got: %v\n%v", test.ErrMsg, err, test)
			}
		} else if test.ErrMsg != "" {
			t.Errorf("Expected user Error matching '%v' but none occurred.\n%v",
				test.ErrMsg, test)
		}

		success := c.Unregister(GLOBAL, cmd)
		if !success {
			t.Errorf("Failed to unregister test: %v", test)
		}
	}
}

func TestCommander_DispatchAuthed(t *T) {
	c := CreateCommander(prefix, core)

	handler := &authCommands{&basicCommands{}, nil}

	var table = []struct {
		Sender   string
		LevelReq uint8
		Flags    string
		Called   bool
		ErrMsg   string
	}{
		{"nick!user@host", 250, "a", false, "Access Denied: Level"},
		{"nick!user@host", 100, "ab", false, "Access Denied: ab flag(s)"},
		{"nick!user@diffhost", 100, "ab", false, "not authenticated"},
		{"nick!user@host", 100, "a", true, ""},
	}

	var buffer = &bytes.Buffer{}
	var storeMutex sync.RWMutex
	store, user := setup()
	user.GrantGlobal(100, "a")

	var dataEndpoint = data.CreateDataEndpoint(server, buffer, nil, store,
		nil, &storeMutex)

	for _, test := range table {
		err := c.RegisterAuthed(GLOBAL, cmd, handler, ALL, ALL,
			test.LevelReq, test.Flags)
		if err != nil {
			t.Errorf("Failed to register test: [%v]\n(%v)", err, test)
			continue
		}

		handler.called = false
		msg := &irc.IrcMessage{
			Sender: test.Sender,
			Name:   irc.PRIVMSG,
			Args:   []string{channel, string(prefix) + cmd},
		}

		err = c.Dispatch(msg, dataEndpoint)
		if handler.called != test.Called {
			if handler.called {
				t.Errorf("Test erroneously called: %v", test)
			} else {
				t.Errorf("Test erroneously skipped: %v", test)
			}
		}

		if err != nil {
			if test.ErrMsg == "" {
				t.Errorf("Unexpected User Error: %v\n%v", err, test)
			} else if !strings.Contains(err.Error(), test.ErrMsg) {
				t.Errorf("Expected: %v but got: %v\n%v", test.ErrMsg, err, test)
			}
		} else if test.ErrMsg != "" {
			t.Errorf("Expected user Error matching '%v' but none occurred.\n%v",
				test.ErrMsg, test)
		}

		success := c.Unregister(GLOBAL, cmd)
		if !success {
			t.Errorf("Failed to unregister test: %v", test)
		}
	}
}
