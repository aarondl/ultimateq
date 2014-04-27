/*
Package cmd is a more involved dispatcher implementation. In short it
allows users to create commands very easily rather than doing everything by hand
in a privmsg handler.

It uses the data package to achieve command access verification. It also
provides some automatic parsing and handling of the command keyword and
arguments. Cmd keywords become unique for each server and may not be
duplicated.
*/
package cmd

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/dispatch"
	"github.com/aarondl/ultimateq/irc"
)

// Constants used for defining the targets/scope of a command.
const (
	// GLOBAL is the bot-global registration "server name".
	GLOBAL = "GLOBAL"
	// PRIVMSG only listens to irc.PRIVMSG events.
	PRIVMSG = 0x1
	// NOTICE only listens to irc.NOTICE events.
	NOTICE = 0x2
	// PRIVATE only listens to PRIVMSG or NOTICE sent directly to the bot.
	PRIVATE = 0x1
	// PUBLIC only listens to PRIVMSG or NOTICE sent to a channel.
	PUBLIC = 0x2
	// ALL when passed into the msgtype parameter: listens to both
	// PRIVMSG and NOTICE events.
	// When passed into the scope parameter: listens for messages sent both
	// directly to the bot, and to a channel.
	ALL = 0x3
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

var (
	// globalCmdRegistry is a singleton throughout the entire bot, and
	// ensures that a command can only be registered once for each server.
	globalCmdRegistry = make(map[string]*Cmd)
	// protectGlobalReg protects the global registry.
	protectGlobalReg sync.RWMutex
)

// EachCmd allows safe iteration through each command in the registry. The
// return value of the callback can be used to stop iteration by returning true.
func EachCmd(fn func(*Cmd) bool) {
	protectGlobalReg.RLock()
	defer protectGlobalReg.RUnlock()

	for _, cmd := range globalCmdRegistry {
		if fn(cmd) {
			break
		}
	}
}

// CmdHandler is the interface that Cmds expects structs to implement
// in order to be able to handle command events. Although this interface must
// be implemented for fallback, if the type has a method with the same name as
// the command being invoked with the first letter uppercased and the same
// arguments and return types as the Cmd function below (minus the cmd arg),
// it will be called instead of the Cmd function.
//
//	Example:
//	type Handler struct {}
//	func (b *Handler) Cmd(cmd string, d *data.DataEndpoint,
//	    c *cmd.Event) error { return nil }
//	func (b *Handler) Supercommand(d *data.DataEndpoint,
//	    c *cmd.Event) error { return nil }
//
// !supercommand in a channel would invoke the bottom handler.
type CmdHandler interface {
	Cmd(string, irc.Writer, *Event) error
}

// commandTable is used to store all the string->command assocations.
type commandTable map[string]*Cmd

// Cmds allows for registration of commands that can involve user access,
// and provides a rich programming interface for command handling.
type Cmds struct {
	*dispatch.DispatchCore
	prefix      rune
	commands    commandTable
	protectCmds sync.RWMutex
}

// NewCmds initializes a cmds.
func NewCmds(prefix rune, core *dispatch.DispatchCore) *Cmds {
	return &Cmds{
		DispatchCore: core,
		prefix:       prefix,
		commands:     make(commandTable),
	}
}

// Register register's a command with the bot. See documentation for
// Cmd for information about how to use this method, as well as see
// the documentation for CmdHandler for how to respond to commands
// registered with a cmds.
//
// The server parameter should be the name of the server that's registering this
// command. The special constant GLOBAL should be used for commands that are
// global to the bot. This ensures that no command can be registered to a single
// server twice.
func (c *Cmds) Register(server string, cmd *Cmd) error {
	regName := makeIdentifier(server, cmd.Cmd)
	globalRegName := makeIdentifier(GLOBAL, cmd.Cmd)

	protectGlobalReg.RLock()
	_, hasServer := globalCmdRegistry[regName]
	_, hasGlobal := globalCmdRegistry[globalRegName]
	protectGlobalReg.RUnlock()
	if hasServer {
		return fmt.Errorf(errFmtDuplicateCmd, regName)
	}
	if hasGlobal {
		return fmt.Errorf(errFmtDuplicateCmd, globalRegName)
	}

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

	protectGlobalReg.Lock()
	c.protectCmds.Lock()
	defer protectGlobalReg.Unlock()
	defer c.protectCmds.Unlock()
	globalCmdRegistry[regName] = cmd
	c.commands[cmd.Cmd] = cmd
	return nil
}

// Unregister unregisters a command from the bot. server should be the name
// of a server it was registered to, or the GLOBAL constant.
func (c *Cmds) Unregister(server, cmd string) (found bool) {
	protectGlobalReg.Lock()
	c.protectCmds.Lock()
	defer c.protectCmds.Unlock()
	defer protectGlobalReg.Unlock()

	globalCmd := makeIdentifier(server, cmd)

	if _, has := globalCmdRegistry[globalCmd]; has {
		delete(globalCmdRegistry, globalCmd)
		found = true
	}
	if _, has := c.commands[cmd]; has {
		delete(c.commands, cmd)
		found = true
	}
	return
}

// Dispatch dispatches an IrcEvent into the cmds event handlers.
func (c *Cmds) Dispatch(networkID string, overridePrefix rune,
	ev *irc.Event, writer irc.Writer, locker data.Locker) (err error) {

	// Filter non privmsg/notice
	msgtype := 0
	switch ev.Name {
	case irc.PRIVMSG:
		msgtype = PRIVMSG
	case irc.NOTICE:
		msgtype = NOTICE
	}

	if msgtype == 0 {
		return nil
	}

	// Get command name or die trying
	fields := strings.Fields(ev.Args[1])
	if len(fields) == 0 {
		return nil
	}
	cmd := fields[0]

	ch := ""
	nick := irc.Nick(ev.Sender)
	msgscope := PRIVATE
	isChan, hasChan := c.CheckTarget(ev)

	// If it's a channel message, ensure we're active on the channel and
	// that the user has supplied the prefix in his command.
	if isChan {
		firstChar := rune(cmd[0])
		missingOverride := overridePrefix == 0 || firstChar != overridePrefix
		missingPrefix := overridePrefix != 0 || firstChar != c.prefix
		if !hasChan || (missingOverride && missingPrefix) {
			return nil
		}

		cmd = cmd[1:]
		ch = ev.Target()
		msgscope = PUBLIC
	}

	var command *Cmd
	var ok bool
	c.protectCmds.RLock()
	defer c.protectCmds.RUnlock()
	if command, ok = c.commands[cmd]; !ok {
		return nil
	}

	if 0 == (msgtype&command.Msgtype) || 0 == (msgscope&command.Msgscope) {
		return nil
	}

	var cmdEv = &Event{
		locker: locker,
		Event:  ev,
	}

	var args []string
	if len(fields) > 1 {
		args = fields[1:]
	}

	state := locker.OpenState(networkID)
	store := locker.OpenStore()
	cmdEv.State = state
	cmdEv.Store = store

	if command.RequireAuth {
		if cmdEv.UserAccess, err = filterAccess(store, command, networkID,
			ch, ev); err != nil {

			cmdEv.Close()
			writer.Notice(nick, err.Error())
			return
		}
	}

	if err = c.filterArgs(networkID, command, ch, isChan, args, cmdEv, ev,
		state, store); err != nil {

		cmdEv.Close()
		writer.Notice(nick, err.Error())
		return
	}

	if state != nil {
		cmdEv.User = state.GetUser(ev.Sender)
		if isChan {
			if cmdEv.Channel == nil {
				cmdEv.Channel = state.GetChannel(ch)
			}
			cmdEv.UserChannelModes = state.GetUsersChannelModes(ev.Sender, ch)
		}
	}

	c.HandlerStarted()
	go func() {
		defer dispatch.PanicHandler()
		defer c.HandlerFinished()
		defer cmdEv.Close()
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
	ev *irc.Event) (*data.UserAccess, error) {

	hasLevel := command.ReqLevel != 0
	hasFlags := len(command.ReqFlags) != 0

	if store == nil {
		return nil, errors.New(errMsgStoreDisabled)
	}

	var access = store.GetAuthedUser(server, ev.Sender)
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
func (c *Cmds) filterArgs(server string, command *Cmd, channel string,
	isChan bool, msgArgs []string, ev *Event, ircEvent *irc.Event,
	state *data.State, store *data.Store) (err error) {

	ev.args = make(map[string]string)

	i, j := 0, 0
	for i = 0; i < len(command.args); i, j = i+1, j+1 {
		arg := &command.args[i]
		req, opt, varg, ch, nick, user := REQUIRED&arg.Type != 0,
			OPTIONAL&arg.Type != 0, VARIADIC&arg.Type != 0,
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
func (c *Cmds) parseChanArg(command *Cmd, ev *Event,
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
			ev.Channel = state.GetChannel(channel)
			ev.TargetChannel = ev.Channel
			return false, nil
		}
		ev.args[name] = msgArgs[index]
		ev.TargetChannel = state.GetChannel(msgArgs[index])
		return true, nil
	} else if isFirstChan {
		ev.args[name] = msgArgs[index]
		ev.TargetChannel = state.GetChannel(msgArgs[index])
		return true, nil
	}

	return false, fmt.Errorf(errFmtArgumentNotChannel, msgArgs[index])
}

// parseUserArg takes user arguments and assigns them to the correct structures
// in a command data struct.
func (c *Cmds) parseUserArg(ev *Event, state *data.State,
	store *data.Store, srv, name string, t argType, users ...string) error {

	vargs := (t & VARIADIC) != 0
	nUsers := len(users)

	var access *data.UserAccess
	var user *data.User
	var err error

	addData := func(index int) {
		if access != nil {
			if vargs {
				ev.TargetVarUserAccess[index] = access
			} else {
				ev.TargetUserAccess[name] = access
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
			ev.TargetVarUserAccess = make([]*data.UserAccess, nUsers)
		} else {
			if ev.TargetUserAccess == nil {
				ev.TargetUserAccess = make(map[string]*data.UserAccess)
			}
		}
		for i, u := range users {
			access, user, err = ev.FindAccessByUser(srv, u)
			if err != nil {
				return err
			}
			addData(i)
		}
	case NICK:
		for i, u := range users {
			user, err = ev.FindUserByNick(u)
			if err != nil {
				return err
			}
			addData(i)
		}
	}

	return nil
}

// GetPrefix returns the prefix used by this cmds instance.
func (c *Cmds) GetPrefix() rune {
	return c.prefix
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
