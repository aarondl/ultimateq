package commander

import (
	"bytes"
	"fmt"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/dispatch"
	"github.com/aarondl/ultimateq/irc"
	"regexp"
	"strings"
	"sync"
	. "testing"
)

func init() {
	data.UserAccessPwdCost = 4 // See constant for bcrypt.MinCost
}

type commandHandler struct {
	called          bool
	cmd             string
	msg             *irc.Message
	end             irc.Endpoint
	user            *data.User
	channel         *data.Channel
	targChan        *data.Channel
	usrchanmodes    *data.UserModes
	access          *data.UserAccess
	targUsers       map[string]*data.User
	targVarUsers    []*data.User
	targUserAccs    map[string]*data.UserAccess
	targVarUserAccs []*data.UserAccess
	args            map[string]string
	state           *data.State
	store           *data.Store
}

func (b *commandHandler) Command(cmd string, msg *irc.Message,
	end *data.DataEndpoint, cmdata *CommandData) (err error) {

	b.called = true
	b.cmd = cmd
	b.msg = msg
	b.end = end
	b.user = cmdata.User
	b.access = cmdata.UserAccess
	b.usrchanmodes = cmdata.UserChannelModes
	b.channel = cmdata.Channel
	b.targChan = cmdata.TargetChannel
	b.targUsers = cmdata.TargetUsers
	b.targUserAccs = cmdata.TargetUserAccess
	b.targVarUsers = cmdata.TargetVarUsers
	b.targVarUserAccs = cmdata.TargetVarUserAccess
	b.args = cmdata.args
	b.state = cmdata.State
	b.store = cmdata.Store

	// Test Coverage obviously will work.
	for k, v := range cmdata.args {
		if cmdata.GetArg(k) != v {
			panic("Something is incredibly wrong.")
		}
	}

	return
}

type errorHandler struct {
	Error error
}

func (e *errorHandler) Command(_ string, _ *irc.Message, _ *data.DataEndpoint,
	_ *CommandData) error {

	return e.Error
}

const (
	ext     = "extension"
	dsc     = "description"
	server  = "irc.test.net"
	host    = "nick!user@host"
	self    = "self!self@self.com"
	cmd     = "cmd"
	channel = "#chan"
	nick    = "nick"
)

var (
	rgxCreator = strings.NewReplacer(
		`(`, `\(`, `)`, `\)`, `]`, `\]`, `[`,
		`\[`, `\`, `\\`, `/`, `\/`, `%v`, `.*`,
		`*`, `\*`,
	)
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

func chkErr(err error, pattern string) error {
	if err == nil {
		return fmt.Errorf("Error was nil but expected: %s", pattern)
	}
	return chkStr(err.Error(), pattern)
}

func chkStr(msg, pattern string) error {
	pattern = `^` + rgxCreator.Replace(pattern) + `$`
	match, err := regexp.MatchString(pattern, msg)
	if err != nil {
		return fmt.Errorf("Error making pattern: \n\t%s\n\t%s", msg, pattern)
	}
	if !match {
		return fmt.Errorf("Unexpected: \n\t%s\n\t%s", msg, pattern)
	}
	return nil
}

func TestCommander_Register(t *T) {
	c := CreateCommander(prefix, core)

	var handler = &commandHandler{}

	var success bool
	var err error
	err = c.Register(GLOBAL, MkCmd(ext, dsc, cmd, nil, ALL, ALL))
	err = chkErr(err, errMsgHandlerRequired)
	if err != nil {
		t.Error(err)
	}

	helper := func(args ...string) *Command {
		return MkCmd(ext, dsc, cmd, handler, ALL, ALL, args...)
	}

	brokenCmd := helper()
	brokenCmd.Cmd = ""
	err = c.Register(GLOBAL, brokenCmd)
	err = chkErr(err, errMsgCmdRequired)
	if err != nil {
		t.Error(err)
	}

	brokenCmd = helper()
	brokenCmd.Extension = ""
	err = c.Register(GLOBAL, brokenCmd)
	err = chkErr(err, errMsgExtRequired)
	if err != nil {
		t.Error(err)
	}

	brokenCmd = helper()
	brokenCmd.Description = ""
	err = c.Register(GLOBAL, brokenCmd)
	err = chkErr(err, errMsgDescRequired)
	if err != nil {
		t.Error(err)
	}

	err = c.Register(GLOBAL, helper("!!!"))
	err = chkErr(err, errFmtArgumentForm)
	if err != nil {
		t.Error(err)
	}

	err = c.Register(GLOBAL, helper("~#badarg"))
	err = chkErr(err, errFmtArgumentForm)
	if err != nil {
		t.Error(err)
	}

	err = c.Register(GLOBAL, helper("#*badarg"))
	err = chkErr(err, errFmtArgumentForm)
	if err != nil {
		t.Error(err)
	}

	err = c.Register(GLOBAL, helper("[opt]", "req"))
	err = chkErr(err, errFmtArgumentOrderReq)
	if err != nil {
		t.Error(err)
	}

	err = c.Register(GLOBAL, helper("req...", "[opt]"))
	err = chkErr(err, errFmtArgumentOrderOpt)
	if err != nil {
		t.Error(err)
	}

	err = c.Register(GLOBAL, helper("name", "[name]"))
	err = chkErr(err, errFmtArgumentDupName)
	if err != nil {
		t.Error(err)
	}

	err = c.Register(GLOBAL, helper("vrgs...", "vrgs2..."))
	err = chkErr(err, errFmtArgumentDupVargs)
	if err != nil {
		t.Error(err)
	}

	err = c.Register(GLOBAL, helper("[opt]", "#chan1"))
	err = chkErr(err, errFmtArgumentOrderChan)
	if err != nil {
		t.Error(err)
	}

	err = c.Register(GLOBAL, helper("vargs...", "#chan1"))
	err = chkErr(err, errFmtArgumentOrderChan)
	if err != nil {
		t.Error(err)
	}

	err = c.Register(GLOBAL, helper("req", "#chan1"))
	err = chkErr(err, errFmtArgumentOrderChan)
	if err != nil {
		t.Error(err)
	}

	err = c.Register(GLOBAL, helper("#chan1", "#chan2"))
	err = chkErr(err, errFmtArgumentDupChan)
	if err != nil {
		t.Error(err)
	}

	err = c.Register(GLOBAL, helper())
	if err != nil {
		t.Error("Registration failed:", err)
	}

	err = c.Register(GLOBAL, helper())
	err = chkErr(err, errFmtDuplicateCommand)
	if err != nil {
		t.Error(err)
	}

	err = c.Register("otherserv", helper())
	err = chkErr(err, errFmtDuplicateCommand)
	if err != nil {
		t.Error(err)
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
	err = c.Register(GLOBAL,
		MkAuthCmd(ext, dsc, cmd, handler, ALL, ALL, 100, "ab"))
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	success = c.Unregister(GLOBAL, cmd)
	if !success {
		t.Error("Should have unregistered successfully.")
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

	ccmd := string(c.prefix) + cmd
	cmsg := []string{channel, ccmd}
	notcmd := []string{nick, "not a command"}
	badcmsg := []string{"#otherchan", string(c.prefix) + cmd}
	umsg := []string{nick, cmd}
	uargmsg := []string{nick, "cmd arg1 arg2"}
	uargvargs := []string{nick, "cmd arg1 arg2 arg3 arg4"}
	cmsgarg := []string{channel, ccmd + " arg1"}
	cmsgargchan := []string{channel, ccmd + " arg1 " + channel}
	cmsgchanarg := []string{channel, ccmd + " " + channel + " arg1"}
	umsgarg := []string{nick, cmd + " arg1"}
	umsgargchan := []string{nick, cmd + " arg1 " + channel}
	umsgchanarg := []string{nick, cmd + " " + channel + " arg1"}
	unil := []string{nick, ""}

	arg1req := []string{"arg"}
	arg1opt := []string{"[opt]"}
	arg1opt1var := []string{"[opt]", "opts..."}
	arg1var := []string{"opts..."}
	arg1req1opt := []string{"arg", "[opt]"}
	arg1req1var := []string{"arg", "opts..."}
	arg1chan1req := []string{"#chan", "arg"}
	arg1chan1req1opt := []string{"#chan", "arg", "[opt]"}

	argErr := errFmtNArguments
	chanErr := errFmtArgumentNotChannel

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
		{nil, ALL, ALL, irc.PRIVMSG, unil, false, ""},
		{arg1opt, ALL, ALL, irc.PRIVMSG, unil, false, ""},
		{arg1opt1var, ALL, ALL, irc.PRIVMSG, unil, false, ""},

		{nil, ALL, ALL, irc.PRIVMSG, umsg, true, ""},
		{nil, ALL, ALL, irc.PRIVMSG, notcmd, false, ""},
		{nil, ALL, ALL, irc.PRIVMSG, uargmsg, false, errMsgUnexpectedArgument},
		{arg1opt, ALL, ALL, irc.PRIVMSG, umsg, true, ""},
		{arg1opt1var, ALL, ALL, irc.PRIVMSG, uargvargs, true, ""},
		{arg1req, ALL, ALL, irc.PRIVMSG, umsg, false, argErr},
		{arg1req1opt, ALL, ALL, irc.PRIVMSG, umsg, false, argErr},

		{arg1req, ALL, ALL, irc.PRIVMSG, uargmsg, false, argErr},
		{arg1opt, ALL, ALL, irc.PRIVMSG, uargmsg, false, argErr},
		{arg1req1opt, ALL, ALL, irc.PRIVMSG, uargmsg, true, ""},

		{arg1req1var, ALL, ALL, irc.PRIVMSG, umsg, false, argErr},
		{arg1req1var, ALL, ALL, irc.PRIVMSG, uargvargs, true, ""},
		{arg1var, ALL, ALL, irc.PRIVMSG, uargvargs, true, ""},

		// Channel Arguments
		{arg1chan1req, ALL, ALL, irc.PRIVMSG, cmsgarg, true, ""},
		{arg1chan1req, ALL, ALL, irc.PRIVMSG, cmsgargchan, false, argErr},
		{arg1chan1req, ALL, ALL, irc.PRIVMSG, cmsgchanarg, true, ""},
		{arg1chan1req, ALL, ALL, irc.PRIVMSG, umsgarg, false, chanErr},
		{arg1chan1req, ALL, ALL, irc.PRIVMSG, umsgargchan, false, chanErr},
		{arg1chan1req, ALL, ALL, irc.PRIVMSG, umsgchanarg, true, ""},

		{arg1chan1req1opt, ALL, ALL, irc.PRIVMSG, cmsgarg, true, ""},
		{arg1chan1req1opt, ALL, ALL, irc.PRIVMSG, umsgarg, false, chanErr},

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
		buffer.Reset()
		handler := &commandHandler{}
		err := c.Register(GLOBAL, MkCmd(ext, dsc, cmd, handler,
			test.MsgType, test.Scope, test.CmdArgs...))
		if err != nil {
			t.Errorf("Failed to register test: [%v]\n(%v)", err, test)
			continue
		}

		msg := &irc.IrcMessage{
			Sender: host,
			Name:   test.Name,
			Args:   test.MsgArgs,
		}
		err = c.Dispatch(server, msg, dataEndpoint)
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
			if len(test.CmdArgs) != 0 {
				if handler.args == nil {
					t.Errorf("No arguments passed in to the handler\n%v",
						test)
				} else {
					for i, arg := range test.CmdArgs {
						req := !strings.ContainsAny(arg, ".[")
						arg = strings.Trim(arg, argStripChars)
						if handler.args[arg] == "" {
							if req || i < len(test.MsgArgs)-2 {
								t.Errorf("The argument was not present: %v\n%v",
									arg, test)
							}
						}
					}
				}
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
			if handler.state == nil {
				t.Error("The state was not set in the command data.")
			}
			if handler.store != nil {
				t.Error("The store was set to not nil, but it doesn't exist!")
			}
		}

		if err != nil {
			if test.ErrMsg == "" {
				t.Errorf("Unexpected User Error: %v\n%v", err, test)
			} else if e := chkErr(err, test.ErrMsg); e != nil {
				t.Error(e)
			}

			if e := chkStr(string(buffer.Bytes()),
				"NOTICE nick :"+test.ErrMsg); e != nil {
				t.Error(e)
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
		{host, 250, "a", false, errFmtInsuffLevel},
		{host, 100, "ab", false, errFmtInsuffFlags},
		{"nick!user@diffhost", 100, "ab", false, errMsgNotAuthed},
		{"nick!user@diffhost", 0, "", false, errMsgNotAuthed},
		{host, 100, "a", true, ""},
	}

	var buffer = &bytes.Buffer{}
	var stateMutex, storeMutex sync.RWMutex
	state, store, user := setupForAuth()
	user.GrantGlobal(100, "a")

	var dataEndpoint = data.CreateDataEndpoint(server, buffer, state, store,
		&stateMutex, &storeMutex)

	for _, test := range table {
		buffer.Reset()
		handler := &commandHandler{}

		err := c.Register(GLOBAL, MkAuthCmd(ext, dsc, cmd, handler, ALL, ALL,
			test.LevelReq, test.Flags))
		if err != nil {
			t.Errorf("Failed to register test: [%v]\n(%v)", err, test)
			continue
		}

		msg := &irc.IrcMessage{
			Sender: test.Sender,
			Name:   irc.PRIVMSG,
			Args:   []string{channel, string(prefix) + cmd},
		}

		err = c.Dispatch(server, msg, dataEndpoint)
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
			if handler.state == nil {
				t.Error("The state was not set in the command data.")
			}
			if handler.store == nil {
				t.Error("The store was not set in the command data.")
			}
		}

		if err != nil {
			if test.ErrMsg == "" {
				t.Errorf("Unexpected User Error: %v\n%v", err, test)
			} else if e := chkErr(err, test.ErrMsg); e != nil {
				t.Error(e)
			}

			if e := chkStr(string(buffer.Bytes()),
				"NOTICE nick :"+test.ErrMsg); e != nil {
				t.Error(e)
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

func TestCommander_DispatchNils(t *T) {
	c := CreateCommander(prefix, core)
	var buffer = &bytes.Buffer{}
	var stateMutex, storeMutex sync.RWMutex

	var dataEndpoint = data.CreateDataEndpoint(server, buffer, nil, nil,
		&stateMutex, &storeMutex)

	msg := &irc.IrcMessage{
		Sender: host,
		Name:   irc.PRIVMSG,
		Args:   []string{channel, string(prefix) + cmd},
	}

	handler := &commandHandler{}

	err := c.Register(GLOBAL,
		MkAuthCmd(ext, dsc, cmd, handler, ALL, ALL, 100, "a"))
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	err = c.Dispatch(server, msg, dataEndpoint)
	c.WaitForHandlers()
	err = chkErr(err, errMsgStoreDisabled)
	if err != nil {
		t.Error(err)
	}
	if !c.Unregister(GLOBAL, cmd) {
		t.Error("Unregistration failed.")
	}

	err = c.Register(GLOBAL, MkCmd(ext, dsc, cmd, handler, ALL, ALL))
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	err = c.Dispatch(server, msg, dataEndpoint)
	c.WaitForHandlers()
	if handler.channel != nil {
		t.Error("Channel should just be nil when state is disabled.")
	}
	if handler.user != nil {
		t.Error("User should be nil when state is disabled.")
	}
	if handler.usrchanmodes != nil {
		t.Error("User chan modes should just be nil when state is disabled.")
	}
	if !c.Unregister(GLOBAL, cmd) {
		t.Error("Unregistration failed.")
	}
}

func TestCommander_DispatchReturns(t *T) {
	c := CreateCommander(prefix, core)
	var buffer = &bytes.Buffer{}
	var stateMutex, storeMutex sync.RWMutex

	var dataEndpoint = data.CreateDataEndpoint(server, buffer, nil, nil,
		&stateMutex, &storeMutex)

	msg := &irc.IrcMessage{
		Sender: host,
		Name:   irc.PRIVMSG,
		Args:   []string{channel, string(prefix) + cmd},
	}

	handler := &errorHandler{}

	err := c.Register(GLOBAL, MkCmd(ext, dsc, cmd, handler, ALL, ALL))
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	var errors = []struct {
		Error    error
		ErrorMsg string
	}{
		{MakeLevelError(100), errFmtInsuffLevel},
		{MakeFlagsError("a"), errFmtInsuffFlags},
		{MakeUserNotAuthedError("user"), errFmtUserNotAuthed},
		{MakeUserNotFoundError("user"), errFmtUserNotFound},
	}

	for _, test := range errors {
		buffer.Reset()
		handler.Error = test.Error
		err = c.Dispatch(server, msg, dataEndpoint)
		c.WaitForHandlers()
		err = chkStr(string(buffer.Bytes()), `NOTICE nick :`+test.ErrorMsg)
		if err != nil {
			t.Error("Failed test:", test)
			t.Error(err)
		}
	}

	success := c.Unregister(GLOBAL, cmd)
	if !success {
		t.Error("Handler could not be unregistered.")
	}
}

func TestCommander_DispatchChannel(t *T) {
	c := CreateCommander(prefix, core)
	var buffer = &bytes.Buffer{}
	var stateMutex, storeMutex sync.RWMutex

	state, store := setup()

	var dataEndpoint = data.CreateDataEndpoint(server, buffer, state, store,
		&stateMutex, &storeMutex)

	msg := &irc.IrcMessage{
		Sender: host, Name: irc.PRIVMSG,
	}

	handler := &commandHandler{}

	err := c.Register(GLOBAL,
		MkCmd(ext, dsc, cmd, handler, ALL, ALL, "#channelArg"))
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	msg.Args = []string{channel, string(prefix) + cmd}
	err = c.Dispatch(server, msg, dataEndpoint)
	c.WaitForHandlers()
	if err != nil {
		t.Error("There was an unexpected error:", err)
	}
	if handler.targChan == nil {
		t.Error("The target channel was not set.")
	}
	if handler.args["channelArg"] != channel {
		t.Error("The channel argument was not set.")
	}

	handler.targChan = nil
	handler.args = nil
	msg.Args = []string{channel, string(prefix) + cmd + " " + channel}
	err = c.Dispatch(server, msg, dataEndpoint)
	c.WaitForHandlers()
	if err != nil {
		t.Error("There was an unexpected error:", err)
	}
	if handler.targChan == nil {
		t.Error("The target channel was not set.")
	}
	if handler.args["channelArg"] != channel {
		t.Error("The channel argument was not set.")
	}

	handler.targChan = nil
	handler.args = nil
	msg.Args = []string{nick, cmd}
	err = c.Dispatch(server, msg, dataEndpoint)
	c.WaitForHandlers()
	err = chkErr(err, errFmtNArguments)
	if err != nil {
		t.Error(err)
	}

	msg.Args = []string{nick, cmd + " " + channel}
	noState := data.CreateDataEndpoint(server, buffer, nil, store,
		&stateMutex, &storeMutex)
	err = c.Dispatch(server, msg, noState)
	c.WaitForHandlers()
	err = chkErr(err, errMsgStateDisabled)
	if err != nil {
		t.Error(err)
	}

	handler.targChan = nil
	handler.args = nil
	msg.Args = []string{nick, cmd + " " + channel}
	err = c.Dispatch(server, msg, dataEndpoint)
	c.WaitForHandlers()
	if err != nil {
		t.Error("There was an unexpected error:", err)
	}
	if handler.targChan == nil {
		t.Error("The target channel was not set.")
	}
	if handler.args["channelArg"] != channel {
		t.Error("The channel argument was not set.")
	}

	success := c.Unregister(GLOBAL, cmd)
	if !success {
		t.Error("Handler could not be unregistered.")
	}
}

func TestCommander_DispatchUsers(t *T) {
	c := CreateCommander(prefix, core)
	var buffer = &bytes.Buffer{}
	var stateMutex, storeMutex sync.RWMutex

	state, store, _ := setupForAuth()
	var dataEndpoint = data.CreateDataEndpoint(server, buffer, state, store,
		&stateMutex, &storeMutex)

	msg := &irc.IrcMessage{
		Sender: host, Name: irc.PRIVMSG,
	}

	handler := &commandHandler{}

	err := c.Register(GLOBAL, MkCmd(ext, dsc, cmd, handler, ALL, ALL,
		"*user1", "~user2", "[*user3]", "~users..."),
	)
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	msg.Args = []string{nick, cmd + " nick nick"}
	err = c.Dispatch(server, msg, dataEndpoint)
	c.WaitForHandlers()
	if err != nil {
		t.Error("There was an unexpected error:", err)
	}

	if u := handler.targUsers["user1"]; u == nil {
		t.Error("User1 was not set.")
	}
	if u := handler.targUserAccs["user1"]; u == nil {
		t.Error("User1 was not set.")
	}
	if u := handler.targUsers["user2"]; u == nil {
		t.Error("User2 was not set.")
	}

	msg.Args = []string{nick, cmd + " *user nick"}
	err = c.Dispatch(server, msg, dataEndpoint)
	c.WaitForHandlers()
	if err != nil {
		t.Error("There was an unexpected error:", err)
	}

	if u := handler.targUserAccs["user1"]; u == nil {
		t.Error("User1 was not set.")
	}
	if u := handler.targUsers["user2"]; u == nil {
		t.Error("User2 was not set.")
	}

	msg.Args = []string{nick, cmd + " *user nick *user"}
	err = c.Dispatch(server, msg, dataEndpoint)
	c.WaitForHandlers()
	if err != nil {
		t.Error("There was an unexpected error:", err)
	}

	if u := handler.targUserAccs["user1"]; u == nil {
		t.Error("User1 was not set.")
	}
	if u := handler.targUsers["user2"]; u == nil {
		t.Error("User2 was not set.")
	}
	if u := handler.targUserAccs["user3"]; u == nil {
		t.Error("User3 was not set.")
	}

	msg.Args = []string{nick, cmd + " *user nick *user nick nick"}
	err = c.Dispatch(server, msg, dataEndpoint)
	c.WaitForHandlers()
	if err != nil {
		t.Error("There was an unexpected error:", err)
	}

	if u := handler.targUserAccs["user1"]; u == nil {
		t.Error("User1 was not set.")
	}
	if u := handler.targUsers["user2"]; u == nil {
		t.Error("User2 was not set.")
	}
	if u := handler.targUserAccs["user3"]; u == nil {
		t.Error("User3 was not set.")
	}
	if u := handler.targVarUsers; u == nil {
		t.Error("User var args was not set or empty.")
	} else if len(u) != 2 {
		t.Error("Unexpected number of users:", len(u))
	}

	success := c.Unregister(GLOBAL, cmd)
	if !success {
		t.Error("Handler could not be unregistered.")
	}
}

func TestCommander_DispatchErrors(t *T) {
	c := CreateCommander(prefix, core)
	var buffer = &bytes.Buffer{}
	var stateMutex, storeMutex sync.RWMutex

	state, store, _ := setupForAuth()
	var dataEndpoint = data.CreateDataEndpoint(server, buffer, state, store,
		&stateMutex, &storeMutex)

	msg := &irc.IrcMessage{
		Sender: host, Name: irc.PRIVMSG,
	}

	handler := &commandHandler{}
	err := c.Register(GLOBAL, MkCmd(ext, dsc, cmd, handler, ALL, ALL,
		"*user1", "~user2", "[*user3]", "~users..."),
	)
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	msg.Args = []string{nick, cmd + " *baduser nick"}
	err = c.Dispatch(server, msg, dataEndpoint)
	c.WaitForHandlers()
	err = chkErr(err, fmt.Sprintf(errFmtUserNotRegistered, "baduser"))
	if err != nil {
		t.Error(err)
	}

	msg.Args = []string{nick, cmd + " * nick"}
	err = c.Dispatch(server, msg, dataEndpoint)
	c.WaitForHandlers()
	err = chkErr(err, errMsgMissingUsername)
	if err != nil {
		t.Error(err)
	}

	msg.Args = []string{nick, cmd + " self nick"}
	err = c.Dispatch(server, msg, dataEndpoint)
	c.WaitForHandlers()
	err = chkErr(err, fmt.Sprintf(errFmtUserNotAuthed, "self"))
	if err != nil {
		t.Error(err)
	}

	msg.Args = []string{nick, cmd + " nick badnick"}
	err = c.Dispatch(server, msg, dataEndpoint)
	c.WaitForHandlers()
	err = chkErr(err, fmt.Sprintf(errFmtUserNotFound, "badnick"))
	if err != nil {
		t.Error(err)
	}

	msg.Args = []string{nick, cmd + " nick nick nick badnick"}
	err = c.Dispatch(server, msg, dataEndpoint)
	c.WaitForHandlers()
	err = chkErr(err, fmt.Sprintf(errFmtUserNotFound, "badnick"))
	if err != nil {
		t.Error(err)
	}

	disableStoreEp := data.CreateDataEndpoint(server, buffer, state, nil,
		&stateMutex, &storeMutex)
	msg.Args = []string{nick, cmd + " *user nick"}
	err = c.Dispatch(server, msg, disableStoreEp)
	c.WaitForHandlers()
	err = chkErr(err, errMsgStoreDisabled)
	if err != nil {
		t.Error(err)
	}

	disableStateEp := data.CreateDataEndpoint(server, buffer, nil, store,
		&stateMutex, &storeMutex)
	msg.Args = []string{nick, cmd + " nick nick"}
	err = c.Dispatch(server, msg, disableStateEp)
	c.WaitForHandlers()
	err = chkErr(err, errMsgStateDisabled)
	if err != nil {
		t.Error(err)
	}

	success := c.Unregister(GLOBAL, cmd)
	if !success {
		t.Error("Handler could not be unregistered.")
	}

	err = c.Register(GLOBAL, MkCmd(ext, dsc, cmd, handler, ALL, ALL, "~user1"))
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	msg.Args = []string{nick, cmd + " nick"}
	err = c.Dispatch(server, msg, disableStateEp)
	c.WaitForHandlers()
	err = chkErr(err, errMsgStateDisabled)
	if err != nil {
		t.Error(err)
	}

	success = c.Unregister(GLOBAL, cmd)
	if !success {
		t.Error("Handler could not be unregistered.")
	}
}

func TestCommander_DispatchVariadicUsers(t *T) {
	c := CreateCommander(prefix, core)
	var buffer = &bytes.Buffer{}
	var stateMutex, storeMutex sync.RWMutex

	state, store, _ := setupForAuth()
	var dataEndpoint = data.CreateDataEndpoint(server, buffer, state, store,
		&stateMutex, &storeMutex)

	msg := &irc.IrcMessage{
		Sender: host, Name: irc.PRIVMSG,
	}

	handler := &commandHandler{}
	var err error
	err = c.Register(GLOBAL, MkCmd(ext, dsc, cmd, handler, ALL, ALL,
		"*users..."),
	)
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	msg.Args = []string{nick, cmd + " *user nick"}
	err = c.Dispatch(server, msg, dataEndpoint)
	c.WaitForHandlers()
	if err != nil {
		t.Error("There was an unexpected error:", err)
	}

	if u := handler.targVarUsers; u == nil {
		t.Error("User var args was not set or empty.")
	} else if len(u) != 2 {
		t.Error("Unexpected number of users:", len(u))
	} else {
		if u[0] != nil {
			t.Error("Username lookup populated users array.")
		}
		if u[1] == nil {
			t.Error("User lookup did not populate user array.")
		}
	}

	if u := handler.targVarUserAccs; u == nil {
		t.Error("UserAccess var args was not set or empty.")
	} else if len(u) != 2 {
		t.Error("Unexpected number of users:", len(u))
	} else {
		if u[0] == nil {
			t.Error("Username lookup did not populate access array.")
		}
		if u[1] == nil {
			t.Error("Nickname lookup did not populate access array.")
		}
	}

	msg.Args = []string{nick, cmd + " nick nick badnick"}
	err = c.Dispatch(server, msg, dataEndpoint)
	c.WaitForHandlers()
	err = chkErr(err, fmt.Sprintf(errFmtUserNotFound, "badnick"))
	if err != nil {
		t.Error(err)
	}

	msg.Args = []string{nick, cmd + " nick nick self"}
	err = c.Dispatch(server, msg, dataEndpoint)
	c.WaitForHandlers()
	err = chkErr(err, fmt.Sprintf(errFmtUserNotAuthed, "self"))
	if err != nil {
		t.Error(err)
	}

	msg.Args = []string{nick, cmd + " nick nick *badusername"}
	err = c.Dispatch(server, msg, dataEndpoint)
	c.WaitForHandlers()
	err = chkErr(err, fmt.Sprintf(errFmtUserNotRegistered, "badusername"))
	if err != nil {
		t.Error(err)
	}

	success := c.Unregister(GLOBAL, cmd)
	if !success {
		t.Error("Handler could not be unregistered.")
	}
}

func TestCommander_DispatchMixUserAndChan(t *T) {
	c := CreateCommander(prefix, core)
	var buffer = &bytes.Buffer{}
	var stateMutex, storeMutex sync.RWMutex

	state, store, _ := setupForAuth()
	var dataEndpoint = data.CreateDataEndpoint(server, buffer, state, store,
		&stateMutex, &storeMutex)

	msg := &irc.IrcMessage{
		Sender: host, Name: irc.PRIVMSG,
	}

	handler := &commandHandler{}
	var err error
	err = c.Register(GLOBAL, MkCmd(ext, dsc, cmd, handler, ALL, ALL,
		"#chan", "~user"),
	)
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	msg.Args = []string{nick, cmd + " " + channel + " nick"}
	err = c.Dispatch(server, msg, dataEndpoint)
	c.WaitForHandlers()
	if err != nil {
		t.Error(err)
	}

	if handler.targUsers["user"] == nil {
		t.Error("The user argument was nil.")
	}

	success := c.Unregister(GLOBAL, cmd)
	if !success {
		t.Error("Handler could not be unregistered.")
	}
}
