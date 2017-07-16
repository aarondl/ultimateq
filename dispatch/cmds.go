package dispatch

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/irc"
)

// Error messages.
const (
	errFmtInternal     = "cmd: Internal Error Occurred: %v"
	errFmtDuplicateCmd = `cmd: ` +
		`Duplicate command registration attempted (%v)`
	errMsgCmdRequired     = `cmd: Cmd name cannot be empty.`
	errMsgExtRequired     = `cmd: Extension name cannot be empty.`
	errMsgDescRequired    = `cmd: Description cannot be empty.`
	errMsgHandlerRequired = `cmd: Handler required for command registration.`

	errMsgStoreDisabled = "Access Denied: Cannot use authenticated commands, " +
		"nick or user parameters when store is disabled."
	errMsgStateDisabled = "Error: Cannot use nick or user parameter commands " +
		"when state is disabled."
	errMsgNotAuthed = "Access Denied: You are not authenticated. " +
		"To authenticate message me: auth <password>. " +
		"To create an account message me: register <password>."
	errFmtInsuffLevel        = "Access Denied: (%v) level required."
	errFmtInsuffGlobalLevel  = "Access Denied: (%v) global level required."
	errFmtInsuffServerLevel  = "Access Denied: (%v) server level required."
	errFmtInsuffChannelLevel = "Access Denied: (%v) channel level required."
	errFmtInsuffFlags        = "Access Denied: (%v) flag(s) required."
	errFmtInsuffGlobalFlags  = "Access Denied: (%v) global flag(s) required."
	errFmtInsuffServerFlags  = "Access Denied: (%v) server flag(s) required."
	errFmtInsuffChannelFlags = "Access Denied: (%v) channel flag(s) required."
	errFmtCmdNotFound        = `Error: Command not found (%v), try "help".`
	errFmtAmbiguousCmd       = "Error: Ambiguous command (%v) found matching:" +
		` [%v], try "help".`
	errFmtUserNotRegistered  = "Error: User [%v] is not registered."
	errFmtUserNotAuthed      = "Error: User [%v] is not authenticated."
	errFmtUserNotFound       = "Error: User [%v] could not be found."
	errMsgMissingUsername    = "Error: Username must follow *, found nothing."
	errMsgUnexpectedArgument = "Error: No arguments expected."
	errFmtNArguments         = "Error: Expected %v %v arguments. (%v)"
	errFmtArgumentNotChannel = "Error: Expected a valid channel. (given: %v)"
	errAtLeast               = "at least"
	errExactly               = "exactly"
	errAtMost                = "at most"

	errFmtArgumentForm = `cmd: Arguments must look like: ` +
		`#name OR [~|*]name OR [[~|*]name] OR [~|*]name... (given: %v)`
	errFmtArgumentOrderReq = `cmd: Required arguments must come before ` +
		`all [optional] and varargs... arguments. (given: %v)`
	errFmtArgumentOrderOpt = `cmd: Optional arguments must come before ` +
		`varargs... arguments. (given: %v)`
	errFmtArgumentDupName = `cmd: Argument names must be unique ` +
		`(given: %v)`
	errFmtArgumentDupVargs = `cmd: Only one varargs... argument is ` +
		`allowed (given: %v)`
	errFmtArgumentOrderChan = `cmd: The channel argument must come ` +
		`first. (given: %v)`
	errFmtArgumentDupChan = `cmd: Only one #channel argument is ` +
		`allowed (given: %v)`
)

// pfxFetcher is used to look up prefixes for different areas of configuration.
type pfxFetcher func(network, channel string) rune

// CommandDispatcher allows for registration of commands that can involve user access,
// and provides a rich programming interface for command handling.
type CommandDispatcher struct {
	*DispatchCore
	fetcher pfxFetcher

	mutTrie sync.RWMutex
	trie    *trie
}

// NewCmds initializes a cmds.
func NewCommandDispatcher(fetcher pfxFetcher, core *DispatchCore) *Cmds {
	return &Cmds{
		DispatchCore: core,
		fetcher:      fetcher,
		commands:     make(commandTable),
	}
}

// Register a command with the bot. See documentation for
// Cmd for information about how to use this method, as well as see
// the documentation for CmdHandler for how to respond to commands issued by
// users. Network and Channel may be given to restrict which networks/channels
// this event will fire on.
func (c *CommandDispatcher) Register(network, channel string, cmd *Cmd) error {
	switch {
	case len(cmd.Cmd) == 0:
		return errors.New(errMsgCmdRequired)
	case len(cmd.Extension) == 0:
		return errors.New(errMsgExtRequired)
	case len(cmd.Description) == 0:
		return errors.New(errMsgDescRequired)
	case cmd.Handler == nil:
		return errors.New(errMsgHandlerRequired)
	}

	if err := cmd.parseArgs(); err != nil {
		return err
	}

	key := mkKey(network, channel, cmd.Cmd)
	c.Lock()
	defer c.Unlock()
	if cmdArr, ok := c.commands[key]; ok {
		for _, command := range cmdArr {
			if command.Extension == cmd.Extension && command.Cmd == cmd.Cmd {
				return fmt.Errorf(errFmtDuplicateCmd, cmd.Cmd)
			}
		}

		cmdArr = append(cmdArr, cmd)
		c.commands[key] = cmdArr
	} else {
		c.commands[key] = []*Cmd{cmd}
	}

	return nil
}

// Unregister a command from the bot. If ext is left blank and there are
// multiple event handlers registered under the name 'cmd' it will unregister
// all of them, the safe bet is to provide the ext parameter.
func (c *CommandDispatcher) Unregister(network, channel, ext, cmd string) (found bool) {
	c.Lock()
	defer c.Unlock()

	key := mkKey(network, channel, cmd)
	cmdArr, ok := c.commands[key]
	if !ok {
		return false
	}

	if len(ext) == 0 {
		delete(c.commands, key)
		return true
	}

	for j, i := range cmdArr {
		if i.Extension == ext {
			ln := len(cmdArr) - 1
			cmdArr[j], cmdArr[ln] = cmdArr[ln], cmdArr[j]
			c.commands[key] = cmdArr[:ln]
			return true
		}
	}

	return false
}

// Dispatch dispatches an IrcEvent into the cmds event handlers.
func (c *CommandDispatcher) Dispatch(writer irc.Writer, ev *irc.Event,
	provider data.Provider) (err error) {

	// Filter non privmsg/notice
	var kind MsgKind
	switch ev.Name {
	case irc.PRIVMSG:
		kind = PRIVMSG
	case irc.NOTICE:
		kind = NOTICE
	}

	if kind == MsgKind(0) {
		return nil
	}

	// Get command name or die trying
	fields := strings.Fields(ev.Args[1])
	if len(fields) == 0 {
		return nil
	}
	cmd := strings.ToLower(fields[0])

	ch := ""
	nick := irc.Nick(ev.Sender)
	scope := PRIVATE
	isChan, hasChan := c.CheckTarget(ev)

	// If it's a channel message, ensure we're active on the channel and
	// that the user has supplied the prefix in his command.
	if isChan {
		ch = ev.Target()
		prefix := c.fetcher(ev.NetworkID, ev.Target())

		firstChar := rune(cmd[0])
		if !hasChan || firstChar != prefix {
			return nil
		}

		cmd = cmd[1:]
		scope = PUBLIC
	}

	// Check if they've supplied the more specific ext.cmd form.
	var ext string
	if ln, dot := len(cmd), strings.IndexRune(cmd, '.'); ln >= 3 && dot > 0 {
		if ln-dot-1 == 0 {
			return nil
		}
		ext = cmd[:dot]
		cmd = cmd[dot+1:]
	}

	c.RLock()
	defer c.RUnlock()

	// Find the command in our "db"
	var command *Cmd
	if command, err = c.lookupCmd(ev.NetworkID, ch, ext, cmd); err != nil {
		writer.Notice(nick, err.Error())
		return err
	} else if command == nil {
		return nil
	}

	if 0 == (kind&command.Kind) || 0 == (scope&command.Scope) {
		return nil
	}

	// Start building up the event.
	var cmdEv = &Event{
		Event: ev,
	}

	var args []string
	if len(fields) > 1 {
		args = fields[1:]
	}

	state := provider.State(ev.NetworkID)
	store := provider.Store()
	cmdEv.State = state
	cmdEv.Store = store

	if command.RequireAuth {
		if cmdEv.StoredUser, err = filterAccess(store, command, ev.NetworkID,
			ch, ev); err != nil {

			writer.Notice(nick, err.Error())
			return err
		}
	}

	if err = c.filterArgs(ev.NetworkID, command, ch, isChan, args, cmdEv, ev,
		state, store); err != nil {

		writer.Notice(nick, err.Error())
		return err
	}

	if state != nil {
		if user, ok := state.User(ev.Sender); ok {
			cmdEv.User = &user
		}
		if isChan {
			if channel, ok := state.Channel(ch); ok {
				cmdEv.Channel = &channel
			}
			if modes, ok := state.UserModes(ev.Sender, ch); ok {
				cmdEv.UserChannelModes = &modes
			}
		}
	}

	c.HandlerStarted()
	go func() {
		defer c.HandlerFinished()
		defer c.PanicHandler()
		ok, err := cmdNameDispatch(command.Handler, cmd, writer, cmdEv)
		if !ok {
			err = command.Handler.Cmd(cmd, writer, cmdEv)
		}
		if err != nil {
			writer.Notice(nick, err.Error())
		}
	}()

	return nil
}

// lookupCmd finds the command to execute, returns user-facing errors if not.
func (c *CommandDispatcher) lookupCmd(network, channel, ext, cmd string) (*Cmd, error) {
	network = strings.ToLower(network)
	channel = strings.ToLower(channel)

	cmdArr, ok := c.lookupCmdArr(network, channel, cmd)
	if ok && cmdArr != nil {
		if len(cmdArr) > 1 && len(ext) == 0 {
			b := &bytes.Buffer{}
			for i, command := range cmdArr {
				if i > 0 {
					b.WriteString(", ")
				}
				fmt.Fprintf(b, "%s.%s", command.Extension, command.Cmd)
			}
			return nil, fmt.Errorf(errFmtAmbiguousCmd, cmd, b)
		}

		for _, i := range cmdArr {
			if i.Cmd == cmd && (len(ext) == 0 || i.Extension == ext) {
				return i, nil
			}
		}
	}

	/*if len(ext) == 0 {
		return nil, fmt.Errorf(errFmtCmdNotFound, cmd)
	} else {
		return nil, fmt.Errorf(errFmtCmdNotFound, ext+"."+cmd)
	}*/
	return nil, nil
}

// lookupCmdArr returns the most specific list of commands it can for the given
// key values.
func (c *CommandDispatcher) lookupCmdArr(network, channel, cmd string) ([]*Cmd, bool) {
	var cmdArr []*Cmd
	var ok bool
	if cmdArr, ok = c.commands[mkKey(network, channel, cmd)]; ok {
		return cmdArr, ok
	}
	if cmdArr, ok = c.commands[mkKey(network, "", cmd)]; ok {
		return cmdArr, ok
	}
	if cmdArr, ok = c.commands[mkKey("", channel, cmd)]; ok {
		return cmdArr, ok
	}
	if cmdArr, ok = c.commands[mkKey("", "", cmd)]; ok {
		return cmdArr, ok
	}

	return nil, false
}

// cmdNameDispatch attempts to dispatch an event to a function named the same
// as the command with an uppercase letter (no camel case). The arguments
// must be the exact same as the CmdHandler.Cmd with the cmd string
// argument removed for this to work.
func cmdNameDispatch(handler CmdHandler, cmd string, writer irc.Writer,
	ev *Event) (dispatched bool, err error) {

	methodName := strings.ToUpper(cmd[:1]) + cmd[1:]

	var fn reflect.Method
	handleType := reflect.TypeOf(handler)
	fn, dispatched = handleType.MethodByName(methodName)
	if !dispatched {
		return
	}

	fnType := fn.Type
	dispatched = fnType.NumIn() == 3 && fnType.NumOut() == 1
	if !dispatched {
		return
	}

	dispatched = reflect.TypeOf(writer).AssignableTo(fnType.In(1)) &&
		reflect.TypeOf(ev).AssignableTo(fnType.In(2)) &&
		reflect.TypeOf(errors.New("")).AssignableTo(fnType.Out(0))
	if !dispatched {
		return
	}

	returnVals := fn.Func.Call([]reflect.Value{
		reflect.ValueOf(handler), reflect.ValueOf(writer), reflect.ValueOf(ev),
	})

	// We have already verified it's type. So this should never fail.
	err, _ = returnVals[0].Interface().(error)
	return
}

// filterAccess ensures that a user has the correct access to perform the given
// command.
func filterAccess(store *data.Store, command *Cmd, server, channel string,
	ev *irc.Event) (*data.StoredUser, error) {

	hasLevel := command.ReqLevel != 0
	hasFlags := len(command.ReqFlags) != 0

	if store == nil {
		return nil, errors.New(errMsgStoreDisabled)
	}

	var access = store.AuthedUser(server, ev.Sender)
	if access == nil {
		return nil, errors.New(errMsgNotAuthed)
	}
	if hasLevel && !access.HasLevel(server, channel, command.ReqLevel) {
		return nil, fmt.Errorf(errFmtInsuffLevel, command.ReqLevel)
	}
	if hasFlags && !access.HasFlags(server, channel, command.ReqFlags) {
		return nil, fmt.Errorf(errFmtInsuffFlags, command.ReqFlags)
	}

	return access, nil
}

// filterArgs parses all the arguments. It looks up channel and user arguments
// using the state and store, and generally populates the Event struct
// with argument information.
func (c *CommandDispatcher) filterArgs(server string, command *Cmd, channel string,
	isChan bool, msgArgs []string, ev *Event, ircEvent *irc.Event,
	state *data.State, store *data.Store) (err error) {

	ev.args = make(map[string]string)

	i, j := 0, 0
	for i = 0; i < len(command.args); i, j = i+1, j+1 {
		arg := &command.args[i]
		req, opt, varg, ch, nick, user := argTypeRequired&arg.Type != 0,
			argTypeOptional&arg.Type != 0, VARIADIC&arg.Type != 0,
			CHANNEL&arg.Type != 0, NICK&arg.Type != 0, USER&arg.Type != 0

		switch {
		case ch:
			if state == nil {
				return errors.New(errMsgStateDisabled)
			}
			var consumed bool
			if consumed, err = c.parseChanArg(command, ev, state, j,
				msgArgs, channel, isChan); err != nil {
				return
			} else if !consumed {
				j--
			}
		case req:
			if j >= len(msgArgs) {
				nReq := command.reqArgs
				if command.args[0].Type&CHANNEL != 0 && isChan {
					nReq--
				}
				return fmt.Errorf(errFmtNArguments, errAtLeast, nReq,
					strings.Join(command.Args, " "))
			}
			ev.args[arg.Name] = msgArgs[j]
		case opt:
			if j >= len(msgArgs) {
				return
			}
			ev.args[arg.Name] = msgArgs[j]
		case varg:
			if j >= len(msgArgs) {
				return
			}
			ev.args[arg.Name] = strings.Join(msgArgs[j:], " ")
		}

		if nick || user {
			if varg {
				err = c.parseUserArg(ev, state, store, server, arg.Name,
					arg.Type, msgArgs[j:]...)
			} else {
				err = c.parseUserArg(ev, state, store, server, arg.Name,
					arg.Type, msgArgs[j])
			}
			if err != nil {
				return
			}
		}

		if varg {
			j = len(msgArgs)
			break
		}
	}

	if j < len(msgArgs) {
		if j == 0 {
			return errors.New(errMsgUnexpectedArgument)
		}
		return fmt.Errorf(errFmtNArguments, errAtMost,
			command.reqArgs+command.optArgs,
			strings.Join(command.Args, " "))
	}
	return nil
}

// parseChanArg checks the argument provided and ensures it's a valid situation
// for the channel arg to be in (isChan & validChan) | (isChan & missing) |
// (!isChan & validChan)
func (c *CommandDispatcher) parseChanArg(command *Cmd, ev *Event,
	state *data.State,
	index int, msgArgs []string, channel string, isChan bool) (bool, error) {

	var isFirstChan bool
	if index < len(msgArgs) {
		isFirstChan = ev.Event.NetworkInfo.IsChannel(msgArgs[index])
	} else if !isChan {
		return false, fmt.Errorf(errFmtNArguments, errAtLeast,
			command.reqArgs, strings.Join(command.Args, " "))
	}

	name := command.args[index].Name
	if isChan {
		if !isFirstChan {
			ev.args[name] = channel
			if ch, ok := state.Channel(channel); ok {
				ev.Channel = &ch
				ev.TargetChannel = &ch
			}
			return false, nil
		}
		ev.args[name] = msgArgs[index]
		if ch, ok := state.Channel(msgArgs[index]); ok {
			ev.TargetChannel = &ch
		}
		return true, nil
	} else if isFirstChan {
		ev.args[name] = msgArgs[index]
		if ch, ok := state.Channel(msgArgs[index]); ok {
			ev.TargetChannel = &ch
		}
		return true, nil
	}

	return false, fmt.Errorf(errFmtArgumentNotChannel, msgArgs[index])
}

// parseUserArg takes user arguments and assigns them to the correct structures
// in a command data struct.
func (c *CommandDispatcher) parseUserArg(ev *Event, state *data.State,
	store *data.Store, srv, name string, t argType, users ...string) error {

	vargs := (t & VARIADIC) != 0
	nUsers := len(users)

	var access *data.StoredUser
	var user *data.User
	var err error

	addData := func(index int) {
		if access != nil {
			if vargs {
				ev.TargetVarStoredUser[index] = access
			} else {
				ev.TargetStoredUser[name] = access
			}
		}
		if user != nil {
			if vargs {
				ev.TargetVarUsers[index] = user
			} else {
				ev.TargetUsers[name] = user
			}
		}
	}

	if vargs {
		ev.TargetVarUsers = make([]*data.User, nUsers)
	} else {
		if ev.TargetUsers == nil {
			ev.TargetUsers = make(map[string]*data.User)
		}
	}

	switch t & USERMASK {
	case USER:
		if vargs {
			ev.TargetVarStoredUser = make([]*data.StoredUser, nUsers)
		} else {
			if ev.TargetStoredUser == nil {
				ev.TargetStoredUser = make(map[string]*data.StoredUser)
			}
		}
		for i, u := range users {
			access, user, err = findAccessByUser(state, store, ev, srv, u)
			if err != nil {
				return err
			}
			addData(i)
		}
	case NICK:
		for i, u := range users {
			user, err = findUserByNick(state, ev, u)
			if err != nil {
				return err
			}
			addData(i)
		}
	}

	return nil
}

// EachCmd iterates through the commands and passes each one to a callback
// function for consumption. These should be considered read-only. Optionally
// the results can be filtered by network and channel.
// To end iteration prematurely the callback function can return true.
func (c *CommandDispatcher) EachCmd(network, channel string, cb func(Cmd) bool) {
	c.RLock()
	defer c.RUnlock()

	for k, cmdArr := range c.commands {
		if (len(network) != 0 || len(channel) != 0) &&
			!strings.HasPrefix(k, mkKey(network, channel, "")) {

			continue
		}

		brk := false
		for _, cmd := range cmdArr {
			if brk = cb(*cmd); brk {
				break
			}
		}

		if brk {
			break
		}
	}
}

// findUserByNick finds a user by their nickname. An error is returned if
// they were not found.
func findUserByNick(state *data.State, ev *Event, nick string) (*data.User, error) {
	if ev.State == nil {
		return nil, errors.New(errMsgStateDisabled)
	}

	if user, ok := ev.State.User(nick); ok {
		return &user, nil
	}

	return nil, fmt.Errorf(errFmtUserNotFound, nick)
}

// findAccessByUser locates a user's access based on their nick or
// username. To look up by username instead of nick use the * prefix before the
// username in the string. The user parameter is returned when a nickname lookup
// is done. An error occurs if the user is not found, the user is not authed,
// the username is not registered, etc.
func findAccessByUser(state *data.State, store *data.Store, ev *Event, server, nickOrUser string) (
	access *data.StoredUser, user *data.User, err error) {
	if store == nil {
		err = errors.New(errMsgStoreDisabled)
		return
	}

	switch nickOrUser[0] {
	case '*':
		if len(nickOrUser) == 1 {
			err = errors.New(errMsgMissingUsername)
			return
		}
		uname := nickOrUser[1:]
		access, err = store.FindUser(uname)
		if access == nil {
			err = fmt.Errorf(errFmtUserNotRegistered, uname)
			return
		}
	default:
		if ev.State == nil {
			err = errors.New(errMsgStateDisabled)
			return
		}

		if u, ok := state.User(nickOrUser); !ok {
			err = fmt.Errorf(errFmtUserNotFound, nickOrUser)
			return
		} else {
			user = &u
		}

		access = store.AuthedUser(server, user.Host.String())
		if access == nil {
			err = fmt.Errorf(errFmtUserNotAuthed, nickOrUser)
			return
		}
	}

	if err != nil {
		err = fmt.Errorf(errFmtInternal, err)
	}
	return
}

// makeIdentifier creates an identifier from a server and a command for
// registration.
func makeIdentifier(server, cmd string) string {
	return server + ":" + cmd
}

// MakeLevelError creates an error to be shown to the user about required
// access.
func MakeLevelError(levelRequired uint8) error {
	return fmt.Errorf(errFmtInsuffLevel, levelRequired)
}

// MakeGlobalLevelError creates an error to be shown to the user about required
// access.
func MakeGlobalLevelError(levelRequired uint8) error {
	return fmt.Errorf(errFmtInsuffGlobalLevel, levelRequired)
}

// MakeServerLevelError creates an error to be shown to the user about required
// access.
func MakeServerLevelError(levelRequired uint8) error {
	return fmt.Errorf(errFmtInsuffServerLevel, levelRequired)
}

// MakeChannelLevelError creates an error to be shown to the user about required
// access.
func MakeChannelLevelError(levelRequired uint8) error {
	return fmt.Errorf(errFmtInsuffChannelLevel, levelRequired)
}

// MakeFlagsError creates an error to be shown to the user about required
// access.
func MakeFlagsError(flagsRequired string) error {
	return fmt.Errorf(errFmtInsuffFlags, flagsRequired)
}

// MakeGlobalFlagsError creates an error to be shown to the user about required
// access.
func MakeGlobalFlagsError(flagsRequired string) error {
	return fmt.Errorf(errFmtInsuffGlobalFlags, flagsRequired)
}

// MakeServerFlagsError creates an error to be shown to the user about required
// access.
func MakeServerFlagsError(flagsRequired string) error {
	return fmt.Errorf(errFmtInsuffServerFlags, flagsRequired)
}

// MakeChannelFlagsError creates an error to be shown to the user about required
// access.
func MakeChannelFlagsError(flagsRequired string) error {
	return fmt.Errorf(errFmtInsuffChannelFlags, flagsRequired)
}

// MakeUserNotAuthedError creates an error to be shown to the user about their
// target user not being authenticated.
func MakeUserNotAuthedError(user string) error {
	return fmt.Errorf(errFmtUserNotAuthed, user)
}

// MakeUserNotFoundError creates an error to be shown to the user about their
// target user not being found.
func MakeUserNotFoundError(user string) error {
	return fmt.Errorf(errFmtUserNotFound, user)
}

// MakeUserNotRegisteredError creates an error to be shown to the user about
// the target user not being registered.
func MakeUserNotRegisteredError(user string) error {
	return fmt.Errorf(errFmtUserNotRegistered, user)
}

// mkKey creates a key for storing and retrieving event handlers.
func mkKey(network, channel, event string) string {
	return fmt.Sprintf("%s:%s:%s", network, channel, event)
}
