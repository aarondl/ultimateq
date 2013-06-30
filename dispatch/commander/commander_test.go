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

type commandHandler struct {
	called       bool
	cmd          string
	msg          *irc.Message
	end          irc.Endpoint
	user         *data.User
	channel      *data.Channel
	usrchanmodes *data.UserModes
	access       *data.UserAccess
}

func (b *commandHandler) Command(cmd string, msg *irc.Message,
	end *data.DataEndpoint, data *CommandData) {

	b.called = true
	b.cmd = cmd
	b.msg = msg
	b.end = end
	b.user = data.User
	b.channel = data.Channel
	b.access = data.UserAccess
	b.usrchanmodes = data.UserChannelModes
}

var (
	server  = "irc.test.net"
	host    = "nick!user@host"
	self    = "self!self@self.com"
	cmd     = "cmd"
	channel = "#chan"
	nick    = "nick"
)

func setup() (state *data.State, store *data.Store) {
	var err error
	state, err = data.CreateState(irc.CreateProtoCaps())
	if err != nil {
		panic(err)
	}

	state.Update(&irc.IrcMessage{
		Sender: server, Name: irc.RPL_WELCOME,
		Args: []string{"welcome", self},
	})
	state.Update(&irc.IrcMessage{
		Sender: self, Name: irc.JOIN,
		Args: []string{channel},
	})
	state.Update(&irc.IrcMessage{
		Sender: host, Name: irc.JOIN,
		Args: []string{channel},
	})
	return
}

func setupForAuth() (state *data.State, store *data.Store,
	user *data.UserAccess) {

	var err error
	state, _ = setup()
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

	var handler = &commandHandler{}

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

	handler := &commandHandler{}
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
	var stateMutex, storeMutex sync.RWMutex
	state, store := setup()
	var dataEndpoint = data.CreateDataEndpoint(server, buffer, state, store,
		&stateMutex, &storeMutex)

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
		handler := &commandHandler{}
		err := c.Register(GLOBAL, cmd, handler, test.MsgType, test.Scope,
			test.CmdArgs...)
		if err != nil {
			t.Errorf("Failed to register test: [%v]\n(%v)", err, test)
			continue
		}

		msg := &irc.IrcMessage{
			Sender: host,
			Name:   test.Name,
			Args:   test.MsgArgs,
		}
		err = c.Dispatch(msg, dataEndpoint)
		c.WaitForHandlers()
		if handler.called != test.Called {
			if handler.called {
				t.Errorf("Test erroneously called: %v", test)
			} else {
				t.Errorf("Test erroneously skipped: %v", test)
			}
		}

		if handler.called {
			if handler.cmd == "" {
				t.Error("The command was not passed to the handler.")
			}
			if handler.user == nil {
				t.Error("The sender was not passed to the handler.")
			}
			if handler.access != nil {
				t.Error("Permless commands should not verify access.")
			}
			if test.MsgArgs[0][0] == '#' {
				if handler.channel == nil {
					t.Error("The channel was not passed to the handler.")
				}
				if handler.usrchanmodes == nil {
					t.Error("The user modes were not passed to the handler.")
				}
			}
			if handler.msg == nil {
				t.Error("The message was not passed to the handler.")
			}
			if handler.end == nil {
				t.Error("The endpoint was not passed to the handler.")
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

	var table = []struct {
		Sender   string
		LevelReq uint8
		Flags    string
		Called   bool
		ErrMsg   string
	}{
		{host, 250, "a", false, "Access Denied: Level"},
		{host, 100, "ab", false, "Access Denied: ab flag(s)"},
		{"nick!user@diffhost", 100, "ab", false, "not authenticated"},
		{host, 100, "a", true, ""},
	}

	var buffer = &bytes.Buffer{}
	var stateMutex, storeMutex sync.RWMutex
	state, store, user := setupForAuth()
	user.GrantGlobal(100, "a")

	var dataEndpoint = data.CreateDataEndpoint(server, buffer, state, store,
		&stateMutex, &storeMutex)

	for _, test := range table {
		handler := &commandHandler{}

		err := c.RegisterAuthed(GLOBAL, cmd, handler, ALL, ALL,
			test.LevelReq, test.Flags)
		if err != nil {
			t.Errorf("Failed to register test: [%v]\n(%v)", err, test)
			continue
		}

		msg := &irc.IrcMessage{
			Sender: test.Sender,
			Name:   irc.PRIVMSG,
			Args:   []string{channel, string(prefix) + cmd},
		}

		err = c.Dispatch(msg, dataEndpoint)
		c.WaitForHandlers()
		if handler.called != test.Called {
			if handler.called {
				t.Errorf("Test erroneously called: %v", test)
			} else {
				t.Errorf("Test erroneously skipped: %v", test)
			}
		}

		if handler.called {
			if handler.cmd == "" {
				t.Error("The command was not passed to the handler.")
			}
			if handler.user == nil {
				t.Error("The sender was not passed to the handler.")
			}
			if handler.usrchanmodes == nil {
				t.Error("The user modes were not passed to the handler.")
			}
			if handler.access == nil {
				t.Error("The access was not passed to the handler.")
			}
			if handler.channel == nil {
				t.Error("The channel was not passed to the handler.")
			}
			if handler.msg == nil {
				t.Error("The message was not passed to the handler.")
			}
			if handler.end == nil {
				t.Error("The endpoint was not passed to the handler.")
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
