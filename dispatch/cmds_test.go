package dispatch

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/dispatch/cmd"
	"github.com/aarondl/ultimateq/irc"
	"gopkg.in/inconshreveable/log15.v2"
)

var (
	netID  = "irc.test.net"
	prefix = '.'
	pfxer  = func(_, _ string) rune {
		return '.'
	}
	core = NewCore(nil)
)

func init() {
	data.StoredUserPwdCost = 4 // See constant for bcrypt.MinCost
}

func newWriter() (*bytes.Buffer, irc.Writer) {
	b := &bytes.Buffer{}
	return b, irc.Helper{b}
}

type testProvider struct {
	state *data.State
	store *data.Store
}

func (t testProvider) State(network string) *data.State {
	return t.state
}

func (t testProvider) Store() *data.Store {
	return t.store
}

type commandHandler struct {
	called          bool
	command         string
	w               irc.Writer
	ev              *irc.Event
	user            *data.User
	channel         *data.Channel
	targChan        *data.Channel
	usrchanmodes    *data.UserModes
	access          *data.StoredUser
	targUsers       map[string]*data.User
	targVarUsers    []*data.User
	targUserAccs    map[string]*data.StoredUser
	targVarUserAccs []*data.StoredUser
	args            map[string]string
}

func (b *commandHandler) Cmd(command string,
	w irc.Writer, ev *cmd.Event) (err error) {

	b.called = true
	b.command = command
	b.w = w
	b.ev = ev.Event
	b.user = ev.User
	b.access = ev.StoredUser
	b.usrchanmodes = ev.UserChannelModes
	b.channel = ev.Channel
	b.targChan = ev.TargetChannel
	b.targUsers = ev.TargetUsers
	b.targUserAccs = ev.TargetStoredUser
	b.targVarUsers = ev.TargetVarUsers
	b.targVarUserAccs = ev.TargetVarStoredUser
	b.args = ev.Args

	// Test Coverage obviously will work.
	for k, v := range ev.Args {
		for _, arg := range ev.SplitArg(k) {
			if !strings.Contains(v, arg) {
				return fmt.Errorf("The argument was not accessbile by SplitArg")
			}
		}
	}

	return
}

type errorHandler struct {
	Error error
}

func (e *errorHandler) Cmd(_ string, _ irc.Writer, _ *cmd.Event) error {

	return e.Error
}

type reflectCmdHandler struct {
	Called    bool
	CalledBad bool
	Error     error
}

func (b *reflectCmdHandler) Cmd(command string, w irc.Writer,
	ev *cmd.Event) (err error) {
	b.CalledBad = true
	return
}

func (b *reflectCmdHandler) Reflect(_ irc.Writer, _ *cmd.Event) (err error) {
	b.Called = true
	return b.Error
}

func (b *reflectCmdHandler) Badargnum(_ irc.Writer, _ irc.Writer) (err error) {
	b.Called = true
	return
}

func (b *reflectCmdHandler) Noreturn(_ irc.Writer, _ irc.Writer) {
	b.Called = true
	return
}

func (b *reflectCmdHandler) Badargs(_ irc.Writer, _ irc.Writer) (err error) {
	b.Called = true
	return
}

type panicHandler struct {
	PanicMessage string
}

func (p panicHandler) Cmd(command string, w irc.Writer, ev *cmd.Event) error {
	panic(p.PanicMessage)
	return nil
}

const (
	ext     = "extension"
	dsc     = "description"
	server  = "irc.test.net"
	host    = "nick!user@host"
	self    = "self!self@self.com"
	command = "command"
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
	state, err = data.NewState(netInfo)
	if err != nil {
		panic(err)
	}

	state.Update(&irc.Event{
		Sender: server, Name: irc.RPL_WELCOME,
		Args:        []string{"welcome", self},
		NetworkID:   netID,
		NetworkInfo: netInfo,
	})
	state.Update(&irc.Event{
		Sender: self, Name: irc.JOIN,
		Args:        []string{channel},
		NetworkID:   netID,
		NetworkInfo: netInfo,
	})
	state.Update(&irc.Event{
		Sender: host, Name: irc.JOIN,
		Args:        []string{channel},
		NetworkID:   netID,
		NetworkInfo: netInfo,
	})
	return
}

func setupForAuth() (state *data.State, store *data.Store,
	user *data.StoredUser) {

	var err error
	state, _ = setup()
	store, err = data.NewStore(data.MemStoreProvider)
	if err != nil {
		panic(err)
	}
	user, err = data.NewStoredUser("user", "pass", "*!*@host")
	if err != nil {
		panic(err)
	}
	err = store.SaveUser(user)
	if err != nil {
		panic(err)
	}
	_, err = store.AuthUserPerma(server, "nick!user@host", "user", "pass")
	if err != nil {
		panic(err)
	}

	return
}

func TestCmds(t *testing.T) {
	t.Parallel()

	c := NewCommandDispatcher(pfxer, NewCore(nil))
	if c == nil {
		t.Fatal("Cmds should not be nil.")
	}
	if c.fetcher == nil {
		t.Error("Prefix fetcher not set correctly.")
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

func TestCmds_Register(t *testing.T) {
	t.Parallel()

	c := NewCommandDispatcher(pfxer, NewCore(nil))

	var handler = &commandHandler{}

	var success bool
	_, err := c.Register("", "", cmd.New(ext, command, dsc, nil, cmd.AnyKind, cmd.AnyScope))
	err = chkErr(err, errMsgHandlerRequired)
	if err != nil {
		t.Error(err)
	}

	helper := func(args ...string) *cmd.Command {
		return cmd.New(ext, dsc, command, handler, cmd.AnyKind, cmd.AnyScope, args...)
	}

	brokenCmd := helper()
	brokenCmd.Name = ""
	_, err = c.Register("", "", brokenCmd)
	err = chkErr(err, errMsgCmdRequired)
	if err != nil {
		t.Error(err)
	}

	brokenCmd = helper()
	brokenCmd.Extension = ""
	_, err = c.Register("", "", brokenCmd)
	err = chkErr(err, errMsgExtRequired)
	if err != nil {
		t.Error(err)
	}

	brokenCmd = helper()
	brokenCmd.Description = ""
	_, err = c.Register("", "", brokenCmd)
	err = chkErr(err, errMsgDescRequired)
	if err != nil {
		t.Error(err)
	}

	id, err := c.Register("", "", helper())
	if err != nil {
		t.Error(err)
	}

	_, err = c.Register("", "", helper())
	err = chkErr(err, errFmtDuplicateCmd)
	if err != nil {
		t.Error(err)
	}

	success = c.Unregister(id)
	if !success {
		t.Error("Should have unregistered successfully.")
	}
	success = c.Unregister(id)
	if success {
		t.Error("Should not be able to double unregister.")
	}
}

func TestCmds_RegisterAuthed(t *testing.T) {
	t.Parallel()

	c := NewCommandDispatcher(pfxer, NewCore(nil))

	handler := &commandHandler{}
	var success bool
	id, err := c.Register("", "",
		cmd.NewAuthed(ext, command, dsc, handler, cmd.AnyKind, cmd.AnyScope, 100, "ab"))
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	success = c.Unregister(id)
	if !success {
		t.Error("Should have unregistered successfully.")
	}
}

func TestCmds_Dispatch(t *testing.T) {
	t.Parallel()

	c := NewCommandDispatcher(pfxer, NewCore(nil))
	if c == nil {
		t.Error("Cmds should not be nil.")
	}

	buffer, writer := newWriter()
	state, store := setup()
	provider := testProvider{state, store}

	ccmd := string(prefix) + command
	cmsg := []string{channel, ccmd}
	//notcmd := []string{nick, "not a command"}
	//cnotcmd := []string{channel, string(prefix) + "ext.not a command"}
	umsg := []string{nick, command}
	uargmsg := []string{nick, "command arg1 arg2"}
	uargvargs := []string{nick, "command arg1 arg2 arg3 arg4"}
	cmsgarg := []string{channel, ccmd + " arg1"}
	cmsgargchan := []string{channel, ccmd + " arg1 " + channel}
	cmsgchanarg := []string{channel, ccmd + " " + channel + " arg1"}
	umsgarg := []string{nick, command + " arg1"}
	umsgargchan := []string{nick, command + " arg1 " + channel}
	umsgchanarg := []string{nick, command + " " + channel + " arg1"}
	unil := []string{nick, ""}

	arg1req := []string{"arg"}
	arg1opt := []string{"[opt]"}
	arg1opt1var := []string{"[opt]", "opts..."}
	arg1var := []string{"opts..."}
	arg1req1opt := []string{"arg", "[opt]"}
	arg1req1var := []string{"arg", "opts..."}
	arg1chan1req := []string{"#chan", "arg"}
	arg1chan1req1opt := []string{"#chan", "arg", "[opt]"}

	// Samefully copied from command package. There's a lot of cross package
	// testing happening. #legacyproblems
	argErr := "Error: Expected %v %v arguments. (%v)"
	chanErr := "Error: Expected a valid channel. (given: %v)"
	atLeastOneArgErr := fmt.Sprintf(argErr, "at least", 1, "%v")
	unexpectedArgument := "Error: No arguments expected."
	argStripChars := `#~*[].`

	var table = []struct {
		CmdArgs []string
		Kind    cmd.Kind
		Scope   cmd.Scope
		Name    string
		MsgArgs []string
		Called  bool
		ErrMsg  string
	}{
		// Args
		{nil, 0, 0, irc.PRIVMSG, unil, false, ""},
		{arg1opt, 0, 0, irc.PRIVMSG, unil, false, ""},
		{arg1opt1var, 0, 0, irc.PRIVMSG, unil, false, ""},

		{nil, 0, 0, irc.PRIVMSG, umsg, true, ""},
		//{nil, 0, 0, irc.PRIVMSG, notcmd, false, errFmtCmdNotFound},
		//{nil, 0, 0, irc.PRIVMSG, cnotcmd, false, errFmtCmdNotFound},
		{nil, 0, 0, irc.PRIVMSG, uargmsg, false, unexpectedArgument},
		{arg1opt, 0, 0, irc.PRIVMSG, umsg, true, ""},
		{arg1opt1var, 0, 0, irc.PRIVMSG, uargvargs, true, ""},
		{arg1req, 0, 0, irc.PRIVMSG, umsg, false, argErr},
		{arg1req1opt, 0, 0, irc.PRIVMSG, umsg, false, argErr},

		{arg1req, 0, 0, irc.PRIVMSG, uargmsg, false, argErr},
		{arg1opt, 0, 0, irc.PRIVMSG, uargmsg, false, argErr},
		{arg1req1opt, 0, 0, irc.PRIVMSG, uargmsg, true, ""},

		{arg1req1var, 0, 0, irc.PRIVMSG, umsg, false, argErr},
		{arg1req1var, 0, 0, irc.PRIVMSG, uargvargs, true, ""},
		{arg1var, 0, 0, irc.PRIVMSG, uargvargs, true, ""},

		// Channel Arguments
		{arg1chan1req, 0, 0, irc.PRIVMSG, cmsgarg, true, ""},
		{arg1chan1req, 0, 0, irc.PRIVMSG, cmsgargchan, false, argErr},
		{arg1chan1req, 0, 0, irc.PRIVMSG, cmsg, false, atLeastOneArgErr},
		{arg1chan1req, 0, 0, irc.PRIVMSG, cmsgchanarg, true, ""},
		{arg1chan1req, 0, 0, irc.PRIVMSG, umsgarg, false, chanErr},
		{arg1chan1req, 0, 0, irc.PRIVMSG, umsgargchan, false, chanErr},
		{arg1chan1req, 0, 0, irc.PRIVMSG, umsgchanarg, true, ""},

		{arg1chan1req1opt, 0, 0, irc.PRIVMSG, cmsgarg, true, ""},
		{arg1chan1req1opt, 0, 0, irc.PRIVMSG, umsgarg, false, chanErr},

		// Bad message
		{nil, 0, 0, irc.RPL_WHOREPLY, cmsg, false, ""},

		// Msgtype All + Scope
		{nil, 0, 0, irc.PRIVMSG, cmsg, true, ""},
		{nil, 0, cmd.Private, irc.PRIVMSG, umsg, true, ""},
		{nil, 0, cmd.Private, irc.PRIVMSG, cmsg, false, ""},
		{nil, 0, cmd.Public, irc.PRIVMSG, umsg, false, ""},
		{nil, 0, cmd.Public, irc.PRIVMSG, cmsg, true, ""},

		// Msgtype Privmsg + Scope
		{nil, cmd.Privmsg, 0, irc.PRIVMSG, cmsg, true, ""},
		{nil, cmd.Privmsg, cmd.Private, irc.PRIVMSG, umsg, true, ""},
		{nil, cmd.Privmsg, cmd.Private, irc.PRIVMSG, cmsg, false, ""},
		{nil, cmd.Privmsg, cmd.Public, irc.PRIVMSG, umsg, false, ""},
		{nil, cmd.Privmsg, cmd.Public, irc.PRIVMSG, cmsg, true, ""},
		{nil, cmd.Privmsg, 0, irc.NOTICE, cmsg, false, ""},

		// Msgtype Notice + Scope
		{nil, cmd.Notice, 0, irc.NOTICE, cmsg, true, ""},
		{nil, cmd.Notice, cmd.Private, irc.NOTICE, umsg, true, ""},
		{nil, cmd.Notice, cmd.Private, irc.NOTICE, cmsg, false, ""},
		{nil, cmd.Notice, cmd.Public, irc.NOTICE, umsg, false, ""},
		{nil, cmd.Notice, cmd.Public, irc.NOTICE, cmsg, true, ""},
		{nil, cmd.Notice, 0, irc.PRIVMSG, cmsg, false, ""},

		// Uppercase
		{nil, 0, 0, irc.PRIVMSG, []string{"nick", "COMMAND"}, true, ""},
	}

	for _, test := range table {
		buffer.Reset()
		handler := &commandHandler{}
		if test.Kind == 0 {
			test.Kind = cmd.AnyKind
		}
		if test.Scope == 0 {
			test.Scope = cmd.AnyScope
		}
		id, err := c.Register("", "", cmd.New(ext, command, dsc, handler,
			test.Kind, test.Scope, test.CmdArgs...))
		if err != nil {
			t.Errorf("Failed to register test: [%v]\n(%v)", err, test)
			continue
		}

		ev := &irc.Event{
			Sender:      host,
			Name:        test.Name,
			Args:        test.MsgArgs,
			NetworkID:   netID,
			NetworkInfo: netInfo,
		}
		err = c.Dispatch(writer, ev, provider)
		c.WaitForHandlers()
		if handler.called != test.Called {
			if handler.called {
				t.Errorf("Test erroneously called: %v", test)
			} else {
				t.Errorf("Test erroneously skipped: %v", test)
			}
		}

		if handler.called {
			if handler.command == "" {
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
			if handler.w == nil {
				t.Error("The writer was not passed to the handler.")
			}
			if handler.ev == nil {
				t.Error("The event was not passed to the handler.")
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

		success := c.Unregister(id)
		if !success {
			t.Errorf("Failed to unregister test: %v", test)
		}
	}
}

func TestCmds_DispatchAuthed(t *testing.T) {
	t.Parallel()

	c := NewCommandDispatcher(pfxer, NewCore(nil))

	var table = []struct {
		Sender   string
		LevelReq uint8
		Flags    string
		Called   bool
		ErrMsg   string
	}{
		{host, 250, "a", false, errFmtInsuffLevel},
		{host, 100, "ab", true, ""},
		{host, 100, "bc", false, errFmtInsuffFlags},
		{"nick!user@diffhost", 100, "ab", false, errMsgNotAuthed},
		{"nick!user@diffhost", 0, "", false, errMsgNotAuthed},
		{host, 100, "a", true, ""},
	}

	buffer, writer := newWriter()
	state, store, user := setupForAuth()
	provider := testProvider{state, store}
	user.Grant("", "", 100, "ab")
	if err := store.SaveUser(user); err != nil {
		t.Fatal(err)
	}

	for _, test := range table {
		buffer.Reset()
		handler := &commandHandler{}

		id, err := c.Register("", "", cmd.NewAuthed(ext, command, dsc, handler, cmd.AnyKind, cmd.AnyScope,
			test.LevelReq, test.Flags))
		if err != nil {
			t.Errorf("Failed to register test: [%v]\n(%v)", err, test)
			continue
		}

		ev := &irc.Event{
			Sender:      test.Sender,
			Name:        irc.PRIVMSG,
			Args:        []string{channel, string(prefix) + command},
			NetworkID:   netID,
			NetworkInfo: netInfo,
		}

		err = c.Dispatch(writer, ev, provider)
		c.WaitForHandlers()
		if handler.called != test.Called {
			if handler.called {
				t.Errorf("Test erroneously called: %v", test)
			} else {
				t.Errorf("Test erroneously skipped: %v", test)
			}
		}

		if handler.called {
			if handler.command == "" {
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
			if handler.ev == nil {
				t.Error("The event was not passed to the handler.")
			}
			if handler.w == nil {
				t.Error("The writer was not passed to the handler.")
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

		success := c.Unregister(id)
		if !success {
			t.Errorf("Failed to unregister test: %v", test)
		}
	}
}

func TestCmds_DispatchNils(t *testing.T) {
	t.Parallel()

	c := NewCommandDispatcher(pfxer, NewCore(nil))
	_, writer := newWriter()
	provider := testProvider{}

	ev := &irc.Event{
		Sender:      host,
		Name:        irc.PRIVMSG,
		Args:        []string{channel, string(prefix) + command},
		NetworkInfo: netInfo,
		NetworkID:   netID,
	}

	handler := &commandHandler{}

	id, err := c.Register("", "",
		cmd.NewAuthed(ext, command, dsc, handler, cmd.AnyKind, cmd.AnyScope, 100, "a"))
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	err = c.Dispatch(writer, ev, provider)
	c.WaitForHandlers()
	err = chkErr(err, errMsgStoreDisabled)
	if err != nil {
		t.Error(err)
	}
	if !c.Unregister(id) {
		t.Error("Unregistration failed.")
	}

	id, err = c.Register("", "", cmd.New(ext, command, dsc, handler, cmd.AnyKind, cmd.AnyScope))
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	err = c.Dispatch(writer, ev, provider)
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
	if !c.Unregister(id) {
		t.Error("Unregistration failed.")
	}
}

func TestCmds_DispatchReturns(t *testing.T) {
	t.Parallel()

	c := NewCommandDispatcher(pfxer, NewCore(nil))
	buffer, writer := newWriter()
	provider := testProvider{}

	ev := &irc.Event{
		Sender:      host,
		Name:        irc.PRIVMSG,
		Args:        []string{channel, string(prefix) + command},
		NetworkInfo: netInfo,
	}

	handler := &errorHandler{}

	id, err := c.Register("", "", cmd.New(ext, command, dsc, handler, cmd.AnyKind, cmd.AnyScope))
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	var errors = []struct {
		Error    error
		ErrorMsg string
	}{
		{MakeLevelError(100), errFmtInsuffLevel},
		{MakeGlobalLevelError(100), errFmtInsuffGlobalLevel},
		{MakeServerLevelError(100), errFmtInsuffServerLevel},
		{MakeChannelLevelError(100), errFmtInsuffChannelLevel},
		{MakeFlagsError("a"), errFmtInsuffFlags},
		{MakeGlobalFlagsError("a"), errFmtInsuffGlobalFlags},
		{MakeServerFlagsError("a"), errFmtInsuffServerFlags},
		{MakeChannelFlagsError("a"), errFmtInsuffChannelFlags},
		{MakeUserNotAuthedError("user"), errFmtUserNotAuthed},
		{MakeUserNotFoundError("user"), errFmtUserNotFound},
		{MakeUserNotRegisteredError("user"), errFmtUserNotRegistered},
	}

	for _, test := range errors {
		buffer.Reset()
		handler.Error = test.Error
		err = c.Dispatch(writer, ev, provider)
		c.WaitForHandlers()
		err = chkStr(string(buffer.Bytes()), `NOTICE nick :`+test.ErrorMsg)
		if err != nil {
			t.Error("Failed test:", test)
			t.Error(err)
		}
	}

	success := c.Unregister(id)
	if !success {
		t.Error("Handler could not be unregistered.")
	}
}

func TestCmds_DispatchChannel(t *testing.T) {
	t.Parallel()

	c := NewCommandDispatcher(pfxer, NewCore(nil))

	_, writer := newWriter()
	state, store := setup()
	provider := testProvider{state, store}

	ev := &irc.Event{
		Sender: host, Name: irc.PRIVMSG,
		NetworkInfo: netInfo,
	}

	handler := &commandHandler{}

	id, err := c.Register("", "",
		cmd.New(ext, command, dsc, handler, cmd.AnyKind, cmd.AnyScope, "#channelArg"))
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	ev.Args = []string{channel, string(prefix) + command}
	err = c.Dispatch(writer, ev, provider)
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
	ev.Args = []string{channel, string(prefix) + command + " " + channel}
	err = c.Dispatch(writer, ev, provider)
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
	ev.Args = []string{nick, command}
	err = c.Dispatch(writer, ev, provider)
	c.WaitForHandlers()
	if err == nil {
		t.Error("should have been an argument error")
	}

	ev.Args = []string{nick, command + " " + channel}
	err = c.Dispatch(writer, ev, testProvider{nil, store})
	c.WaitForHandlers()
	err = chkErr(err, errMsgStateDisabled)
	if err != nil {
		t.Error(err)
	}

	handler.targChan = nil
	handler.args = nil
	ev.Args = []string{nick, command + " " + channel}
	err = c.Dispatch(writer, ev, provider)
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

	ev.Args = []string{channel, string(prefix) + command + " " + channel + " arg"}
	err = c.Dispatch(writer, ev, provider)
	c.WaitForHandlers()
	if err == nil {
		t.Error("should have been an argument error")
	}

	ev.Args = []string{nick, command}
	err = c.Dispatch(writer, ev, provider)
	c.WaitForHandlers()
	if err == nil {
		t.Error("should have been an argument error")
	}

	success := c.Unregister(id)
	if !success {
		t.Error("Handler could not be unregistered.")
	}
}

func TestCmds_DispatchUsers(t *testing.T) {
	t.Parallel()

	c := NewCommandDispatcher(pfxer, NewCore(nil))

	_, writer := newWriter()
	state, store, _ := setupForAuth()
	provider := testProvider{state, store}

	ev := &irc.Event{
		Sender: host, Name: irc.PRIVMSG,
		NetworkID:   netID,
		NetworkInfo: netInfo,
	}

	handler := &commandHandler{}

	id, err := c.Register("", "", cmd.New(ext, command, dsc, handler, cmd.AnyKind, cmd.AnyScope,
		"*user1", "~user2", "[*user3]", "~users..."),
	)
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	ev.Args = []string{nick, command + " nick nick"}
	err = c.Dispatch(writer, ev, provider)
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

	ev.Args = []string{nick, command + " *user nick"}
	err = c.Dispatch(writer, ev, provider)
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

	ev.Args = []string{nick, command + " *user nick *user"}
	err = c.Dispatch(writer, ev, provider)
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

	ev.Args = []string{nick, command + " *user nick *user nick nick"}
	err = c.Dispatch(writer, ev, provider)
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

	success := c.Unregister(id)
	if !success {
		t.Error("Handler could not be unregistered.")
	}
}

func TestCmds_DispatchErrors(t *testing.T) {
	t.Parallel()

	c := NewCommandDispatcher(pfxer, NewCore(nil))

	_, writer := newWriter()
	state, store, _ := setupForAuth()
	provider := testProvider{state, store}

	ev := &irc.Event{
		Sender: host, Name: irc.PRIVMSG,
		NetworkID:   netID,
		NetworkInfo: netInfo,
	}

	handler := &commandHandler{}
	id, err := c.Register("", "", cmd.New(ext, command, dsc, handler, cmd.AnyKind, cmd.AnyScope,
		"*user1", "~user2", "[*user3]", "~users..."),
	)
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	ev.Args = []string{nick, command + " *baduser nick"}
	err = c.Dispatch(writer, ev, provider)
	c.WaitForHandlers()
	err = chkErr(err, fmt.Sprintf(errFmtUserNotRegistered, "baduser"))
	if err != nil {
		t.Error(err)
	}

	ev.Args = []string{nick, command + " * nick"}
	err = c.Dispatch(writer, ev, provider)
	c.WaitForHandlers()
	if err == nil {
		t.Error("should have had an error about a username")
	}

	ev.Args = []string{nick, command + " self nick"}
	err = c.Dispatch(writer, ev, provider)
	c.WaitForHandlers()
	err = chkErr(err, fmt.Sprintf(errFmtUserNotAuthed, "self"))
	if err != nil {
		t.Error(err)
	}

	ev.Args = []string{nick, command + " nick badnick"}
	err = c.Dispatch(writer, ev, provider)
	c.WaitForHandlers()
	err = chkErr(err, fmt.Sprintf(errFmtUserNotFound, "badnick"))
	if err != nil {
		t.Error(err)
	}

	ev.Args = []string{nick, command + " nick nick nick badnick"}
	err = c.Dispatch(writer, ev, provider)
	c.WaitForHandlers()
	err = chkErr(err, fmt.Sprintf(errFmtUserNotFound, "badnick"))
	if err != nil {
		t.Error(err)
	}

	ev.Args = []string{nick, command + " *user nick"}
	err = c.Dispatch(writer, ev, testProvider{state, nil})
	c.WaitForHandlers()
	err = chkErr(err, errMsgStoreDisabled)
	if err != nil {
		t.Error(err)
	}

	ev.Args = []string{nick, command + " nick nick"}
	err = c.Dispatch(writer, ev, testProvider{nil, store})
	c.WaitForHandlers()
	err = chkErr(err, errMsgStateDisabled)
	if err != nil {
		t.Error(err)
	}

	success := c.Unregister(id)
	if !success {
		t.Error("Handler could not be unregistered.")
	}

	id, err = c.Register("", "", cmd.New(ext, command, dsc, handler, cmd.AnyKind, cmd.AnyScope, "~user1"))
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	ev.Args = []string{nick, command + " nick"}
	err = c.Dispatch(writer, ev, testProvider{nil, store})
	c.WaitForHandlers()
	err = chkErr(err, errMsgStateDisabled)
	if err != nil {
		t.Error(err)
	}

	success = c.Unregister(id)
	if !success {
		t.Error("Handler could not be unregistered.")
	}
}

func TestCmds_DispatchVariadicUsers(t *testing.T) {
	t.Parallel()

	c := NewCommandDispatcher(pfxer, NewCore(nil))

	_, writer := newWriter()
	state, store, _ := setupForAuth()
	provider := testProvider{state, store}

	ev := &irc.Event{
		Sender: host, Name: irc.PRIVMSG,
		NetworkID:   netID,
		NetworkInfo: netInfo,
	}

	handler := &commandHandler{}
	id, err := c.Register("", "", cmd.New(ext, command, dsc, handler, cmd.AnyKind, cmd.AnyScope,
		"*users..."),
	)
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	ev.Args = []string{nick, command + " *user nick"}
	err = c.Dispatch(writer, ev, provider)
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
		t.Error("StoredUser var args was not set or empty.")
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

	ev.Args = []string{nick, command + " nick nick badnick"}
	err = c.Dispatch(writer, ev, provider)
	c.WaitForHandlers()
	err = chkErr(err, fmt.Sprintf(errFmtUserNotFound, "badnick"))
	if err != nil {
		t.Error(err)
	}

	ev.Args = []string{nick, command + " nick nick self"}
	err = c.Dispatch(writer, ev, provider)
	c.WaitForHandlers()
	err = chkErr(err, fmt.Sprintf(errFmtUserNotAuthed, "self"))
	if err != nil {
		t.Error(err)
	}

	ev.Args = []string{nick, command + " nick nick *badusername"}
	err = c.Dispatch(writer, ev, provider)
	c.WaitForHandlers()
	err = chkErr(err, fmt.Sprintf(errFmtUserNotRegistered, "badusername"))
	if err != nil {
		t.Error(err)
	}

	success := c.Unregister(id)
	if !success {
		t.Error("Handler could not be unregistered.")
	}
}

func TestCmds_DispatchMixUserAndChan(t *testing.T) {
	t.Parallel()

	c := NewCommandDispatcher(pfxer, NewCore(nil))

	_, writer := newWriter()
	state, store, _ := setupForAuth()
	provider := testProvider{state, store}

	ev := &irc.Event{
		Sender: host, Name: irc.PRIVMSG,
		NetworkID:   netID,
		NetworkInfo: netInfo,
	}

	handler := &commandHandler{}
	id, err := c.Register("", "", cmd.New(ext, command, dsc, handler, cmd.AnyKind, cmd.AnyScope,
		"#chan", "~user"),
	)
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	ev.Args = []string{nick, command + " " + channel + " nick"}
	err = c.Dispatch(writer, ev, provider)
	c.WaitForHandlers()
	if err != nil {
		t.Error(err)
	}

	if handler.targUsers["user"] == nil {
		t.Error("The user argument was nil.")
	}

	success := c.Unregister(id)
	if !success {
		t.Error("Handler could not be unregistered.")
	}
}

func TestCmds_DispatchReflection(t *testing.T) {
	t.Parallel()

	c := NewCommandDispatcher(pfxer, NewCore(nil))

	buffer, writer := newWriter()
	state, _ := setup()
	provider := testProvider{state, nil}

	errMsg := "error"
	handler := &reflectCmdHandler{Error: fmt.Errorf(errMsg)}

	commands := []string{"reflect", "badargnum", "noreturn", "badargs"}
	var ids []uint64
	for _, command := range commands {
		id, err := c.Register("", "", cmd.New(ext, command, dsc, handler, cmd.AnyKind, cmd.AnyScope))
		if err != nil {
			t.Error("Unexpected:", command, err)
		}
		ids = append(ids, id)
	}

	ev := &irc.Event{
		Name: irc.PRIVMSG, Sender: host,
		Args:        []string{"a", "reflect"},
		NetworkID:   netID,
		NetworkInfo: netInfo,
	}
	err := c.Dispatch(writer, ev, provider)
	if err != nil {
		t.Error(err)
	}
	c.WaitForHandlers()

	if handler.CalledBad {
		t.Error("The reflection should call Reflect method instead of Cmd.")
	}
	if !handler.Called {
		t.Error("Expected a call to Reflect.")
	}
	if !strings.Contains(buffer.String(), errMsg) {
		t.Error("Expected:", buffer.String(), "to contain:", errMsg)
	}

	handler.Called, handler.CalledBad = false, false
	ev.Args = []string{"a", "badargnum"}
	err = c.Dispatch(writer, ev, provider)
	if err != nil {
		t.Error(err)
	}
	c.WaitForHandlers()

	if !handler.CalledBad {
		t.Error("Expecting fallback to Cmd when reflection fails.")
	}
	if handler.Called {
		t.Error("Not expecting it to be able to call this handler.")
	}

	handler.Called, handler.CalledBad = false, false
	ev.Args = []string{"a", "noreturn"}
	err = c.Dispatch(writer, ev, provider)
	if err != nil {
		t.Error(err)
	}
	c.WaitForHandlers()

	if !handler.CalledBad {
		t.Error("Expecting fallback to Cmd when reflection fails.")
	}
	if handler.Called {
		t.Error("Not expecting it to be able to call this handler.")
	}

	handler.Called, handler.CalledBad = false, false
	ev.Args = []string{"a", "badargs"}
	err = c.Dispatch(writer, ev, provider)
	if err != nil {
		t.Error(err)
	}
	c.WaitForHandlers()

	if !handler.CalledBad {
		t.Error("Expecting fallback to Cmd when reflection fails.")
	}
	if handler.Called {
		t.Error("Not expecting it to be able to call this handler.")
	}

	for _, id := range ids {
		success := c.Unregister(id)
		if !success {
			t.Error(command, "handler could not be unregistered.")
		}
	}
}

/*
TODO: fix this
func TestCmds_EachCmd(t *testing.T) {
	t.Parallel()

	c := NewCommandDispatcher(pfxer, NewCore(nil))
	var err error

	handler := &errorHandler{}

	id1, err := c.Register("", "", cmd.New(ext, command, dsc, handler, cmd.AnyKind, cmd.AnyScope))
	if err != nil {
		t.Error("Unexpected:", err)
	}
	id2, err := c.Register("", "", cmd.New(ext, "other", dsc, handler,
		cmd.AnyKind, cmd.AnyScope))

	if err != nil {
		t.Error("Unexpected:", err)
	}

	visited := 0
	c.EachCmd("", "", func(command *cmd.Command) bool {
		visited++
		return true
	})

	if visited != 1 {
		t.Error("Expected to stop after one iteration, did:", visited)
	}

	success := c.Unregister(id1)
	if !success {
		t.Error(command, "handler could not be unregistered.")
	}
	success = c.Unregister(id2)
	if !success {
		t.Error("other handler could not be unregistered.")
	}
}
*/

func TestCmds_DispatchAmbiguous(t *testing.T) {
	t.Parallel()

	c := NewCommandDispatcher(pfxer, NewCore(nil))
	if c == nil {
		t.Error("Cmds should not be nil.")
	}

	buffer, writer := newWriter()
	state, store := setup()
	provider := testProvider{state, store}
	handler := &commandHandler{}

	c.Register("", "", cmd.New("one", "command", "d", handler, cmd.AnyKind, cmd.AnyScope))
	c.Register("", "", cmd.New("two", "command", "d", handler, cmd.AnyKind, cmd.AnyScope))

	ev := &irc.Event{
		Name: irc.PRIVMSG, Sender: host,
		Args:        []string{"nick", "command"},
		NetworkID:   netID,
		NetworkInfo: netInfo,
	}

	c.Dispatch(writer, ev, provider)
	c.WaitForHandlers()

	if handler.called {
		t.Error("Handler should not get called.")
	}

	err := chkStr(buffer.String(), "NOTICE nick :"+errFmtAmbiguousCmd)
	if err != nil {
		t.Error(err)
	}
}

func TestCmds_DispatchSpecific(t *testing.T) {
	t.Parallel()

	c := NewCommandDispatcher(pfxer, NewCore(nil))
	if c == nil {
		t.Error("Cmds should not be nil.")
	}

	_, writer := newWriter()
	state, store := setup()
	provider := testProvider{state, store}
	handler1 := &commandHandler{}
	handler2 := &commandHandler{}

	_, err := c.Register("", "", cmd.New("one", "command", "d", handler1, cmd.AnyKind, cmd.AnyScope))
	if err != nil {
		t.Error(err)
	}
	_, err = c.Register("", "", cmd.New("two", "command", "d", handler2, cmd.AnyKind, cmd.AnyScope))
	if err != nil {
		t.Error(err)
	}

	ev := &irc.Event{
		Name: irc.PRIVMSG, Sender: host,
		Args:        []string{"#chan", ".two.command"},
		NetworkID:   netID,
		NetworkInfo: netInfo,
	}

	c.Dispatch(writer, ev, provider)
	c.WaitForHandlers()

	if handler1.called {
		t.Error("Handler should not get called.")
	}
	if !handler2.called {
		t.Error("Handler should get called.")
	}
}

func TestCmds_Panic(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	logger := log15.New()
	logger.SetHandler(log15.StreamHandler(buf, log15.LogfmtFormat()))
	logCore := NewCore(logger)

	c := NewCommandDispatcher(pfxer, logCore)
	panicMsg := "dispatch panic"

	state, store := setup()
	provider := testProvider{state, store}

	handler := panicHandler{
		panicMsg,
	}

	tmpCmd := cmd.New("panic", "panic", "panic desc", handler, cmd.AnyKind, cmd.AnyScope)
	c.Register("", "", tmpCmd)

	ev := irc.NewEvent("", netInfo, irc.PRIVMSG, host, self, "panic")
	err := c.Dispatch(nil, ev, provider)
	if err != nil {
		t.Error(err)
	}

	c.WaitForHandlers()
	logStr := buf.String()

	if logStr == "" {
		t.Error("Expected not empty log.")
	}

	logBytes := buf.Bytes()
	if !bytes.Contains(logBytes, []byte(panicMsg)) {
		t.Errorf("Log does not contain: %s\n%s", panicMsg, logBytes)
	}

	if !bytes.Contains(logBytes, []byte("cmds_test.go")) {
		t.Error("Does not contain a reference to file that panic'd")
	}
}
