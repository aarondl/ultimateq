package cmd

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/dispatch"
	"github.com/aarondl/ultimateq/irc"
	"github.com/inconshreveable/log15"
)

var (
	netID   = "irc.test.net"
	netInfo = irc.NewNetworkInfo()
)

func init() {
	data.StoredUserPwdCost = 4 // See constant for bcrypt.MinCost
}

func newWriter() (*bytes.Buffer, irc.Writer) {
	b := &bytes.Buffer{}
	return b, irc.Helper{b}
}

type badLocker struct {
	*data.State
	*data.Store
}

func (b badLocker) ReadState(networkID string, fn func(*data.State)) bool {
	fn(b.State)
	return true
}
func (b badLocker) OpenState(networkID string) *data.State { return b.State }
func (b badLocker) CloseState(networkID string)            {}

func (b badLocker) ReadStore(fn func(*data.Store)) bool {
	fn(b.Store)
	return true
}
func (b badLocker) WriteStore(fn func(*data.Store)) bool {
	fn(b.Store)
	return true
}
func (b badLocker) OpenReadStore() *data.Store  { return b.Store }
func (b badLocker) CloseReadStore()             {}
func (b badLocker) OpenWriteStore() *data.Store { return b.Store }
func (b badLocker) CloseWriteStore()            {}

type commandHandler struct {
	called          bool
	cmd             string
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
	state           *data.State
	store           *data.Store
}

func (b *commandHandler) Cmd(cmd string,
	w irc.Writer, ev *Event) (err error) {

	b.called = true
	b.cmd = cmd
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
	b.args = ev.args
	b.state = ev.State
	b.store = ev.Store

	// Test Coverage obviously will work.
	for k, v := range ev.args {
		if ev.GetArg(k) != v {
			return fmt.Errorf("The argument was not accessible by GetArg")
		}
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

func (e *errorHandler) Cmd(_ string, _ irc.Writer, _ *Event) error {

	return e.Error
}

type reflectCmdHandler struct {
	Called    bool
	CalledBad bool
	Error     error
}

func (b *reflectCmdHandler) Cmd(cmd string, w irc.Writer,
	ev *Event) (err error) {
	b.CalledBad = true
	return
}

func (b *reflectCmdHandler) Reflect(_ irc.Writer, _ *Event) (err error) {
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

func (p panicHandler) Cmd(cmd string, w irc.Writer, ev *Event) error {
	panic(p.PanicMessage)
	return nil
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
	state, err = data.NewState(netInfo)
	if err != nil {
		panic(err)
	}

	state.Update(&irc.Event{
		Sender: server, Name: irc.RPL_WELCOME,
		Args:        []string{"welcome", self},
		NetworkInfo: netInfo,
	})
	state.Update(&irc.Event{
		Sender: self, Name: irc.JOIN,
		Args:        []string{channel},
		NetworkInfo: netInfo,
	})
	state.Update(&irc.Event{
		Sender: host, Name: irc.JOIN,
		Args:        []string{channel},
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
	_, err = store.AuthUser(server, "nick!user@host", "user", "pass")
	if err != nil {
		panic(err)
	}

	return
}

var core = dispatch.NewDispatchCore(nil)
var prefix = '.'

func TestCmds(t *testing.T) {
	c := NewCmds(prefix, core)
	if c == nil {
		t.Fatal("Cmds should not be nil.")
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

func TestCmds_Register(t *testing.T) {
	c := NewCmds(prefix, core)

	var handler = &commandHandler{}

	var success bool
	var err error
	err = c.Register(GLOBAL, MkCmd(ext, dsc, cmd, nil, ALL, ALL))
	err = chkErr(err, errMsgHandlerRequired)
	if err != nil {
		t.Error(err)
	}

	helper := func(args ...string) *Cmd {
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
	err = chkErr(err, errFmtDuplicateCmd)
	if err != nil {
		t.Error(err)
	}

	err = c.Register("otherserv", helper())
	err = chkErr(err, errFmtDuplicateCmd)
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

func TestCmds_RegisterProtected(t *testing.T) {
	c := NewCmds(prefix, core)

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

func TestCmds_Dispatch(t *testing.T) {
	dcore := dispatch.NewDispatchCore(nil)
	c := NewCmds(prefix, dcore)
	c.AddChannels(channel)
	if c == nil {
		t.Error("Cmds should not be nil.")
	}

	buffer, writer := newWriter()
	state, store := setup()
	locker := badLocker{state, store}

	ccmd := string(c.prefix) + cmd
	cmsg := []string{channel, ccmd}
	notcmd := []string{nick, "not a command"}
	cnotcmd := []string{channel, string(c.prefix) + "not a command"}
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
	atLeastOneArgErr := fmt.Sprintf(errFmtNArguments, errAtLeast, 1, "%v")

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
		{nil, ALL, ALL, irc.PRIVMSG, notcmd, false, errFmtCmdNotFound},
		{nil, ALL, ALL, irc.PRIVMSG, cnotcmd, false, errFmtCmdNotFound},
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
		{arg1chan1req, ALL, ALL, irc.PRIVMSG, cmsg, false, atLeastOneArgErr},
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

		// Uppercase
		{nil, ALL, ALL, irc.PRIVMSG, []string{"nick", "CMD"}, true, ""},
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

		ev := &irc.Event{
			Sender:      host,
			Name:        test.Name,
			Args:        test.MsgArgs,
			NetworkInfo: netInfo,
		}
		err = c.Dispatch(server, 0, writer, ev, locker)
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
			if handler.w == nil {
				t.Error("The writer was not passed to the handler.")
			}
			if handler.ev == nil {
				t.Error("The event was not passed to the handler.")
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

func TestCmds_DispatchAuthed(t *testing.T) {
	c := NewCmds(prefix, core)

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
	locker := badLocker{state, store}
	user.GrantGlobal(100, "a")

	for _, test := range table {
		buffer.Reset()
		handler := &commandHandler{}

		err := c.Register(GLOBAL, MkAuthCmd(ext, dsc, cmd, handler, ALL, ALL,
			test.LevelReq, test.Flags))
		if err != nil {
			t.Errorf("Failed to register test: [%v]\n(%v)", err, test)
			continue
		}

		ev := &irc.Event{
			Sender:      test.Sender,
			Name:        irc.PRIVMSG,
			Args:        []string{channel, string(prefix) + cmd},
			NetworkInfo: netInfo,
		}

		err = c.Dispatch(server, 0, writer, ev, locker)
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
			if handler.ev == nil {
				t.Error("The event was not passed to the handler.")
			}
			if handler.w == nil {
				t.Error("The writer was not passed to the handler.")
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

func TestCmds_DispatchNils(t *testing.T) {
	c := NewCmds(prefix, core)
	_, writer := newWriter()
	locker := badLocker{}

	ev := &irc.Event{
		Sender:      host,
		Name:        irc.PRIVMSG,
		Args:        []string{channel, string(prefix) + cmd},
		NetworkInfo: netInfo,
		NetworkID:   netID,
	}

	handler := &commandHandler{}

	err := c.Register(GLOBAL,
		MkAuthCmd(ext, dsc, cmd, handler, ALL, ALL, 100, "a"))
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	err = c.Dispatch(server, 0, writer, ev, locker)
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
	err = c.Dispatch(server, 0, writer, ev, locker)
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

func TestCmds_DispatchReturns(t *testing.T) {
	c := NewCmds(prefix, core)
	buffer, writer := newWriter()
	locker := badLocker{}

	ev := &irc.Event{
		Sender:      host,
		Name:        irc.PRIVMSG,
		Args:        []string{channel, string(prefix) + cmd},
		NetworkInfo: netInfo,
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
		err = c.Dispatch(server, 0, writer, ev, locker)
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

func TestCmds_DispatchChannel(t *testing.T) {
	c := NewCmds(prefix, core)

	_, writer := newWriter()
	state, store := setup()
	locker := badLocker{state, store}

	ev := &irc.Event{
		Sender: host, Name: irc.PRIVMSG,
		NetworkInfo: netInfo,
	}

	handler := &commandHandler{}

	err := c.Register(GLOBAL,
		MkCmd(ext, dsc, cmd, handler, ALL, ALL, "#channelArg"))
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	ev.Args = []string{channel, string(prefix) + cmd}
	err = c.Dispatch(server, 0, writer, ev, locker)
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
	ev.Args = []string{channel, string(prefix) + cmd + " " + channel}
	err = c.Dispatch(server, 0, writer, ev, locker)
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
	ev.Args = []string{nick, cmd}
	err = c.Dispatch(server, 0, writer, ev, locker)
	c.WaitForHandlers()
	err = chkErr(err, errFmtNArguments)
	if err != nil {
		t.Error(err)
	}

	ev.Args = []string{nick, cmd + " " + channel}
	err = c.Dispatch(server, 0, writer, ev, badLocker{nil, store})
	c.WaitForHandlers()
	err = chkErr(err, errMsgStateDisabled)
	if err != nil {
		t.Error(err)
	}

	handler.targChan = nil
	handler.args = nil
	ev.Args = []string{nick, cmd + " " + channel}
	err = c.Dispatch(server, 0, writer, ev, locker)
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

	ev.Args = []string{channel, string(prefix) + cmd + " " + channel + " arg"}
	err = c.Dispatch(server, 0, writer, ev, locker)
	c.WaitForHandlers()
	err = chkErr(err, fmt.Sprintf(errFmtNArguments, errAtMost, 1, "%v"))
	if err != nil {
		t.Error(err)
	}

	ev.Args = []string{nick, cmd}
	err = c.Dispatch(server, 0, writer, ev, locker)
	c.WaitForHandlers()
	err = chkErr(err, fmt.Sprintf(errFmtNArguments, errAtLeast, 1, "%v"))
	if err != nil {
		t.Error(err)
	}

	success := c.Unregister(GLOBAL, cmd)
	if !success {
		t.Error("Handler could not be unregistered.")
	}
}

func TestCmds_DispatchUsers(t *testing.T) {
	c := NewCmds(prefix, core)

	_, writer := newWriter()
	state, store, _ := setupForAuth()
	locker := badLocker{state, store}

	ev := &irc.Event{
		Sender: host, Name: irc.PRIVMSG,
		NetworkInfo: netInfo,
	}

	handler := &commandHandler{}

	err := c.Register(GLOBAL, MkCmd(ext, dsc, cmd, handler, ALL, ALL,
		"*user1", "~user2", "[*user3]", "~users..."),
	)
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	ev.Args = []string{nick, cmd + " nick nick"}
	err = c.Dispatch(server, 0, writer, ev, locker)
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

	ev.Args = []string{nick, cmd + " *user nick"}
	err = c.Dispatch(server, 0, writer, ev, locker)
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

	ev.Args = []string{nick, cmd + " *user nick *user"}
	err = c.Dispatch(server, 0, writer, ev, locker)
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

	ev.Args = []string{nick, cmd + " *user nick *user nick nick"}
	err = c.Dispatch(server, 0, writer, ev, locker)
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

func TestCmds_DispatchErrors(t *testing.T) {
	c := NewCmds(prefix, core)

	_, writer := newWriter()
	state, store, _ := setupForAuth()
	locker := badLocker{state, store}

	ev := &irc.Event{
		Sender: host, Name: irc.PRIVMSG,
		NetworkInfo: netInfo,
	}

	handler := &commandHandler{}
	err := c.Register(GLOBAL, MkCmd(ext, dsc, cmd, handler, ALL, ALL,
		"*user1", "~user2", "[*user3]", "~users..."),
	)
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	ev.Args = []string{nick, cmd + " *baduser nick"}
	err = c.Dispatch(server, 0, writer, ev, locker)
	c.WaitForHandlers()
	err = chkErr(err, fmt.Sprintf(errFmtUserNotRegistered, "baduser"))
	if err != nil {
		t.Error(err)
	}

	ev.Args = []string{nick, cmd + " * nick"}
	err = c.Dispatch(server, 0, writer, ev, locker)
	c.WaitForHandlers()
	err = chkErr(err, errMsgMissingUsername)
	if err != nil {
		t.Error(err)
	}

	ev.Args = []string{nick, cmd + " self nick"}
	err = c.Dispatch(server, 0, writer, ev, locker)
	c.WaitForHandlers()
	err = chkErr(err, fmt.Sprintf(errFmtUserNotAuthed, "self"))
	if err != nil {
		t.Error(err)
	}

	ev.Args = []string{nick, cmd + " nick badnick"}
	err = c.Dispatch(server, 0, writer, ev, locker)
	c.WaitForHandlers()
	err = chkErr(err, fmt.Sprintf(errFmtUserNotFound, "badnick"))
	if err != nil {
		t.Error(err)
	}

	ev.Args = []string{nick, cmd + " nick nick nick badnick"}
	err = c.Dispatch(server, 0, writer, ev, locker)
	c.WaitForHandlers()
	err = chkErr(err, fmt.Sprintf(errFmtUserNotFound, "badnick"))
	if err != nil {
		t.Error(err)
	}

	ev.Args = []string{nick, cmd + " *user nick"}
	err = c.Dispatch(server, 0, writer, ev, badLocker{state, nil})
	c.WaitForHandlers()
	err = chkErr(err, errMsgStoreDisabled)
	if err != nil {
		t.Error(err)
	}

	ev.Args = []string{nick, cmd + " nick nick"}
	err = c.Dispatch(server, 0, writer, ev, badLocker{nil, store})
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

	ev.Args = []string{nick, cmd + " nick"}
	err = c.Dispatch(server, 0, writer, ev, badLocker{nil, store})
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

func TestCmds_DispatchVariadicUsers(t *testing.T) {
	c := NewCmds(prefix, core)

	_, writer := newWriter()
	state, store, _ := setupForAuth()
	locker := badLocker{state, store}

	ev := &irc.Event{
		Sender: host, Name: irc.PRIVMSG,
		NetworkInfo: netInfo,
	}

	handler := &commandHandler{}
	var err error
	err = c.Register(GLOBAL, MkCmd(ext, dsc, cmd, handler, ALL, ALL,
		"*users..."),
	)
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	ev.Args = []string{nick, cmd + " *user nick"}
	err = c.Dispatch(server, 0, writer, ev, locker)
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

	ev.Args = []string{nick, cmd + " nick nick badnick"}
	err = c.Dispatch(server, 0, writer, ev, locker)
	c.WaitForHandlers()
	err = chkErr(err, fmt.Sprintf(errFmtUserNotFound, "badnick"))
	if err != nil {
		t.Error(err)
	}

	ev.Args = []string{nick, cmd + " nick nick self"}
	err = c.Dispatch(server, 0, writer, ev, locker)
	c.WaitForHandlers()
	err = chkErr(err, fmt.Sprintf(errFmtUserNotAuthed, "self"))
	if err != nil {
		t.Error(err)
	}

	ev.Args = []string{nick, cmd + " nick nick *badusername"}
	err = c.Dispatch(server, 0, writer, ev, locker)
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

func TestCmds_DispatchMixUserAndChan(t *testing.T) {
	c := NewCmds(prefix, core)

	_, writer := newWriter()
	state, store, _ := setupForAuth()
	locker := badLocker{state, store}

	ev := &irc.Event{
		Sender: host, Name: irc.PRIVMSG,
		NetworkInfo: netInfo,
	}

	handler := &commandHandler{}
	var err error
	err = c.Register(GLOBAL, MkCmd(ext, dsc, cmd, handler, ALL, ALL,
		"#chan", "~user"),
	)
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	ev.Args = []string{nick, cmd + " " + channel + " nick"}
	err = c.Dispatch(server, 0, writer, ev, locker)
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

func TestCmds_DispatchReflection(t *testing.T) {
	c := NewCmds(prefix, core)
	var err error

	buffer, writer := newWriter()
	state, _ := setup()
	locker := badLocker{state, nil}

	errMsg := "error"
	handler := &reflectCmdHandler{Error: fmt.Errorf(errMsg)}

	cmds := []string{"reflect", "badargnum", "noreturn", "badargs"}
	for _, command := range cmds {
		err = c.Register(GLOBAL, MkCmd(ext, dsc, command, handler, ALL, ALL))
		if err != nil {
			t.Error("Unexpected:", command, err)
		}
	}

	ev := &irc.Event{
		Name: irc.PRIVMSG, Sender: host,
		Args:        []string{"a", "reflect"},
		NetworkInfo: netInfo,
	}
	err = c.Dispatch(server, 0, writer, ev, locker)
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
	err = c.Dispatch(server, 0, writer, ev, locker)
	c.WaitForHandlers()

	if !handler.CalledBad {
		t.Error("Expecting fallback to Cmd when reflection fails.")
	}
	if handler.Called {
		t.Error("Not expecting it to be able to call this handler.")
	}

	handler.Called, handler.CalledBad = false, false
	ev.Args = []string{"a", "noreturn"}
	err = c.Dispatch(server, 0, writer, ev, locker)
	c.WaitForHandlers()

	if !handler.CalledBad {
		t.Error("Expecting fallback to Cmd when reflection fails.")
	}
	if handler.Called {
		t.Error("Not expecting it to be able to call this handler.")
	}

	handler.Called, handler.CalledBad = false, false
	ev.Args = []string{"a", "badargs"}
	err = c.Dispatch(server, 0, writer, ev, locker)
	c.WaitForHandlers()

	if !handler.CalledBad {
		t.Error("Expecting fallback to Cmd when reflection fails.")
	}
	if handler.Called {
		t.Error("Not expecting it to be able to call this handler.")
	}

	for _, command := range cmds {
		success := c.Unregister(GLOBAL, command)
		if !success {
			t.Error(command, "handler could not be unregistered.")
		}
	}
}

func TestCmds_DispatchOverridePrefix(t *testing.T) {
	c := NewCmds(prefix, core)
	var err error

	_, writer := newWriter()
	state, _ := setup()
	locker := badLocker{state, nil}

	handler := &commandHandler{}
	ev := &irc.Event{
		Name: irc.PRIVMSG, Sender: host,
		NetworkInfo: netInfo,
	}

	err = c.Register(GLOBAL, MkCmd(ext, dsc, cmd, handler, ALL, ALL))
	if err != nil {
		t.Error("Unexpected:", cmd, err)
	}

	handler.called = false
	ev.Args = []string{channel, string(prefix) + cmd}
	err = c.Dispatch(server, 0, writer, ev, locker)
	c.WaitForHandlers()

	if !handler.called {
		t.Error("Expected a call to Cmd.")
	}

	handler.called = false
	ev.Args = []string{channel, string(prefix) + cmd}
	err = c.Dispatch(server, '!', writer, ev, locker)
	c.WaitForHandlers()

	if handler.called {
		t.Error("Expected no call to Cmd.")
	}

	handler.called = false
	ev.Args = []string{channel, "!" + cmd}
	err = c.Dispatch(server, '!', writer, ev, locker)
	c.WaitForHandlers()

	if !handler.called {
		t.Error("Expected a call to Cmd.")
	}

	handler.called = false
	ev.Args = []string{channel, ":" + cmd}
	err = c.Dispatch(server, '!', writer, ev, locker)
	c.WaitForHandlers()

	if handler.called {
		t.Error("Expected no call to Cmd.")
	}

	success := c.Unregister(GLOBAL, cmd)
	if !success {
		t.Error(cmd, "handler could not be unregistered.")
	}
}

func TestCmds_EachCmd(t *testing.T) {
	c := NewCmds(prefix, core)
	var err error

	handler := &errorHandler{}

	err = c.Register(GLOBAL, MkCmd(ext, dsc, cmd, handler, ALL, ALL))
	if err != nil {
		t.Error("Unexpected:", err)
	}
	err = c.Register(GLOBAL, MkCmd(ext, dsc, "other", handler, ALL, ALL))
	if err != nil {
		t.Error("Unexpected:", err)
	}

	visited := 0
	EachCmd(func(command *Cmd) bool {
		visited++
		return true
	})

	if visited != 1 {
		t.Error("Expected to stop after one iteration.")
	}

	success := c.Unregister(GLOBAL, cmd)
	if !success {
		t.Error(cmd, "handler could not be unregistered.")
	}
	success = c.Unregister(GLOBAL, "other")
	if !success {
		t.Error("other handler could not be unregistered.")
	}
}

type lockWriter struct {
	*bytes.Buffer
	write chan struct{}
}

func (l *lockWriter) Write(b []byte) (int, error) {
	n, err := l.Buffer.Write(b)
	l.write <- struct{}{}
	return n, err
}

func TestCmds_Panic(t *testing.T) {
	ch := make(chan struct{}, 1)
	lk := &lockWriter{&bytes.Buffer{}, ch}
	logger := log15.New()
	logger.SetHandler(log15.StreamHandler(lk, log15.LogfmtFormat()))
	logCore := dispatch.NewDispatchCore(logger)

	c := NewCmds(prefix, logCore)
	panicMsg := "dispatch panic"

	state, store := setup()
	locker := badLocker{state, store}

	handler := panicHandler{
		panicMsg,
	}

	tmpCmd := MkCmd("panic", "panic desc", "panic", handler, ALL, ALL)
	c.Register(GLOBAL, tmpCmd)

	ev := irc.NewEvent("", netInfo, irc.PRIVMSG, host, self, "panic")
	err := c.Dispatch(server, 0, nil, ev, locker)
	if err != nil {
		t.Error(err)
	}

	c.WaitForHandlers()
	<-ch
	logStr := lk.String()

	if logStr == "" {
		t.Error("Expected not empty log.")
	}

	logBytes := lk.Bytes()
	if !bytes.Contains(logBytes, []byte(panicMsg)) {
		t.Errorf("Log does not contain: %s\n%s", panicMsg, logBytes)
	}

	if !bytes.Contains(logBytes, []byte("cmds_test.go")) {
		t.Error("Does not contain a reference to file that panic'd")
	}
}
