/*
Package commander is a more involved dispatcher implementation. In short it
allows users to create commands very easily rather than doing everything by hand
in a privmsg handler.

It uses the data package to achieve command access verification. It also
provides some automatic parsing and handling of the command keyword and
arguments. Command keywords become unique for each server and may not be
duplicated.
*/
package commander

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
	errFmtInternal         = "commander: Internal Error Occurred: %v"
	errFmtDuplicateCommand = `commander: ` +
		`Duplicate command registration attempted (%v)`
	errMsgCmdRequired     = `commander: Command name cannot be empty.`
	errMsgExtRequired     = `commander: Extension name cannot be empty.`
	errMsgDescRequired    = `commander: Description cannot be empty.`
	errMsgHandlerRequired = `commander: ` +
		`Handler required for command registration.`

	errMsgStoreDisabled = "Access Denied: Cannot use authenticated commands, " +
		"nick or user parameters when store is disabled."
	errMsgStateDisabled = "Error: Cannot use nick or user parameter commands " +
		"when state is disabled."
	errMsgNotAuthed          = "Access Denied: You are not authenticated. To authenticate message me AUTH <password> [username]. If you need to create an account message me REGISTER <password> [username]."
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

	errFmtArgumentForm = `commander: Arguments must look like: ` +
		`#name OR [~|*]name OR [[~|*]name] OR [~|*]name... (given: %v)`
	errFmtArgumentOrderReq = `commander: Required arguments must come before ` +
		`all [optional] and varargs... arguments. (given: %v)`
	errFmtArgumentOrderOpt = `commander: Optional arguments must come before ` +
		`varargs... arguments. (given: %v)`
	errFmtArgumentDupName = `commander: Argument names must be unique ` +
		`(given: %v)`
	errFmtArgumentDupVargs = `commander: Only one varargs... argument is ` +
		`allowed (given: %v)`
	errFmtArgumentOrderChan = `commander: The channel argument must come ` +
		`first. (given: %v)`
	errFmtArgumentDupChan = `commander: Only one #channel argument is ` +
		`allowed (given: %v)`
)

var (
	// globalCommandRegistry is a singleton throughout the entire bot, and
	// ensures that a command can only be registered once for each server.
	globalCommandRegistry = make(map[string]*Command)
	// protectGlobalReg protects the global registry.
	protectGlobalReg sync.RWMutex
)

// EachCommand allows safe iteration through each command in the registry. The
// return value of the callback can be used to stop iteration by returning true.
func EachCommand(fn func(*Command) bool) {
	protectGlobalReg.RLock()
	defer protectGlobalReg.RUnlock()

	for _, cmd := range globalCommandRegistry {
		if fn(cmd) {
			break
		}
	}
}

// CommandHandler is the interface that Commander expects structs to implement
// in order to be able to handle command events. Although this interface must
// be implemented for fallback, if the type has a method with the same name as
// the command being invoked with the first letter uppercased and the same
// arguments and return types as the Command function below (minus the cmd arg),
// it will be called instead of the Command function.
//
// Example:
// type Handler struct {}
// func (b *Handler) Command(cmd string, m *irc.Message, d *data.DataEndpoint,
//     c *commander.CommandData) error { return nil }
// func (b *Handler) Supercommand(m *irc.Message, d *data.DataEndpoint,
//     c *commander.CommandData) error { return nil }
//
// !supercommand in a channel would invoke the bottom handler.
type CommandHandler interface {
	Command(string, *irc.Message, *data.DataEndpoint, *CommandData) error
}

// commandTable is used to store all the string->command assocations.
type commandTable map[string]*Command

// Commander allows for registration of commands that can involve user access,
// and provides a rich programming interface for command handling.
type Commander struct {
	*dispatch.DispatchCore
	prefix          rune
	commands        commandTable
	protectCommands sync.RWMutex
}

// CreateCommander initializes a commander.
func CreateCommander(prefix rune, core *dispatch.DispatchCore) *Commander {
	return &Commander{
		DispatchCore: core,
		prefix:       prefix,
		commands:     make(commandTable),
	}
}

// Register register's a command with the bot. See documentation for
// Command for information about how to use this method, as well as see
// the documentation for CommandHandler for how to respond to commands
// registered with a commander.
//
// The server parameter should be the name of the server that's registering this
// command. The special constant GLOBAL should be used for commands that are
// global to the bot. This ensures that no command can be registered to a single
// server twice.
func (c *Commander) Register(server string, cmd *Command) error {
	regName := makeIdentifier(server, cmd.Cmd)
	globalRegName := makeIdentifier(GLOBAL, cmd.Cmd)

	protectGlobalReg.RLock()
	_, hasServer := globalCommandRegistry[regName]
	_, hasGlobal := globalCommandRegistry[globalRegName]
	protectGlobalReg.RUnlock()
	if hasServer {
		return fmt.Errorf(errFmtDuplicateCommand, regName)
	}
	if hasGlobal {
		return fmt.Errorf(errFmtDuplicateCommand, globalRegName)
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
	c.protectCommands.Lock()
	defer protectGlobalReg.Unlock()
	defer c.protectCommands.Unlock()
	globalCommandRegistry[regName] = cmd
	c.commands[cmd.Cmd] = cmd
	return nil
}

// Unregister unregisters a command from the bot. server should be the name
// of a server it was registered to, or the GLOBAL constant.
func (c *Commander) Unregister(server, cmd string) (found bool) {
	protectGlobalReg.Lock()
	c.protectCommands.Lock()
	defer c.protectCommands.Unlock()
	defer protectGlobalReg.Unlock()

	globalCmd := makeIdentifier(server, cmd)

	if _, has := globalCommandRegistry[globalCmd]; has {
		delete(globalCommandRegistry, globalCmd)
		found = true
	}
	if _, has := c.commands[cmd]; has {
		delete(c.commands, cmd)
		found = true
	}
	return
}

// Dispatch dispatches an IrcEvent into the commander's event handlers.
func (c *Commander) Dispatch(server string, overridePrefix rune,
	msg *irc.Message, ep *data.DataEndpoint) (err error) {

	// Filter non privmsg/notice
	msgtype := 0
	switch msg.Name {
	case irc.PRIVMSG:
		msgtype = PRIVMSG
	case irc.NOTICE:
		msgtype = NOTICE
	}

	if msgtype == 0 {
		return nil
	}

	// Get command name or die trying
	fields := strings.Fields(msg.Args[1])
	if len(fields) == 0 {
		return nil
	}
	cmd := fields[0]

	ch := ""
	nick := irc.Nick(msg.Sender)
	msgscope := PRIVATE
	isChan, hasChan := c.CheckTarget(msg.Args[0])

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
		ch = msg.Args[0]
		msgscope = PUBLIC
	}

	var command *Command
	var ok bool
	c.protectCommands.RLock()
	defer c.protectCommands.RUnlock()
	if command, ok = c.commands[cmd]; !ok {
		return nil
	}

	if 0 == (msgtype&command.Msgtype) || 0 == (msgscope&command.Msgscope) {
		return nil
	}

	var cmdata = &CommandData{
		ep: ep,
	}

	var args []string
	if len(fields) > 1 {
		args = fields[1:]
	}

	state := ep.OpenState()
	store := ep.OpenStore()
	cmdata.State = state
	cmdata.Store = store

	if command.RequireAuth {
		if cmdata.UserAccess, err = filterAccess(store, command,
			server, ch, ep, msg); err != nil {

			cmdata.Close()
			ep.Notice(nick, err.Error())
			return
		}
	}

	if err = c.filterArgs(server, command, ch, isChan, args, cmdata, state,
		store); err != nil {

		cmdata.Close()
		ep.Notice(nick, err.Error())
		return
	}

	if state != nil {
		cmdata.User = state.GetUser(msg.Sender)
		if isChan {
			if cmdata.Channel == nil {
				cmdata.Channel = state.GetChannel(ch)
			}
			cmdata.UserChannelModes = state.GetUsersChannelModes(msg.Sender, ch)
		}
	}

	c.HandlerStarted()
	go func() {
		defer dispatch.PanicHandler()
		defer c.HandlerFinished()
		defer cmdata.Close()
		ok, err := cmdNameDispatch(command.Handler, cmd, msg, ep, cmdata)
		if !ok {
			err = command.Handler.Command(cmd, msg, ep, cmdata)
		}
		if err != nil {
			ep.Notice(nick, err.Error())
		}
	}()

	return nil
}

// cmdNameDispatch attempts to dispatch an event to a function named the same
// as the command with an uppercase letter (no camel case). The arguments
// must be the exact same as the CommandHandler.Command with the cmd string
// argument removed for this to work.
func cmdNameDispatch(handler CommandHandler, cmd string, msg *irc.Message,
	ep *data.DataEndpoint, cmdata *CommandData) (dispatched bool, err error) {

	methodName := strings.ToUpper(cmd[:1]) + cmd[1:]

	var fn reflect.Method
	handleType := reflect.TypeOf(handler)
	fn, dispatched = handleType.MethodByName(methodName)
	if !dispatched {
		return
	}

	fnType := fn.Type
	dispatched = fnType.NumIn() == 4 && fnType.NumOut() == 1
	if !dispatched {
		return
	}

	dispatched = reflect.TypeOf(msg).AssignableTo(fnType.In(1)) &&
		reflect.TypeOf(ep).AssignableTo(fnType.In(2)) &&
		reflect.TypeOf(cmdata).AssignableTo(fnType.In(3)) &&
		reflect.TypeOf(errors.New("")).AssignableTo(fnType.Out(0))
	if !dispatched {
		return
	}

	returnVals := fn.Func.Call([]reflect.Value{
		reflect.ValueOf(handler), reflect.ValueOf(msg),
		reflect.ValueOf(ep), reflect.ValueOf(cmdata),
	})

	// We have already verified it's type. So this should never fail.
	err, _ = returnVals[0].Interface().(error)
	return
}

// filterAccess ensures that a user has the correct access to perform the given
// command.
func filterAccess(store *data.Store, command *Command, server, channel string,
	ep *data.DataEndpoint, msg *irc.Message) (*data.UserAccess, error) {

	hasLevel := command.ReqLevel != 0
	hasFlags := len(command.ReqFlags) != 0

	if store == nil {
		return nil, errors.New(errMsgStoreDisabled)
	}

	var access = store.GetAuthedUser(ep.GetKey(), msg.Sender)
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
// using the state and store, and generally populates the CommandData struct
// with argument information.
func (c *Commander) filterArgs(server string, command *Command, channel string,
	isChan bool, msgArgs []string, cmdata *CommandData,
	state *data.State, store *data.Store) (err error) {

	cmdata.args = make(map[string]string)

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
			if consumed, err = c.parseChanArg(command, cmdata, state, j,
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
			cmdata.args[arg.Name] = msgArgs[j]
		case opt:
			if j >= len(msgArgs) {
				return
			}
			cmdata.args[arg.Name] = msgArgs[j]
		case varg:
			if j >= len(msgArgs) {
				return
			}
			cmdata.args[arg.Name] = strings.Join(msgArgs[j:], " ")
		}

		if nick || user {
			if varg {
				err = c.parseUserArg(cmdata, state, store, server, arg.Name,
					arg.Type, msgArgs[j:]...)
			} else {
				err = c.parseUserArg(cmdata, state, store, server, arg.Name,
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
func (c *Commander) parseChanArg(command *Command, cmdata *CommandData,
	state *data.State,
	index int, msgArgs []string, channel string, isChan bool) (bool, error) {

	var isFirstChan bool
	if index < len(msgArgs) {
		isFirstChan, _ = c.CheckTarget(msgArgs[index])
	} else if !isChan {
		return false, fmt.Errorf(errFmtNArguments, errAtLeast,
			command.reqArgs, strings.Join(command.Args, " "))
	}

	name := command.args[index].Name
	if isChan {
		if !isFirstChan {
			cmdata.args[name] = channel
			cmdata.Channel = state.GetChannel(channel)
			cmdata.TargetChannel = cmdata.Channel
			return false, nil
		}
		cmdata.args[name] = msgArgs[index]
		cmdata.TargetChannel = state.GetChannel(msgArgs[index])
		return true, nil
	} else if isFirstChan {
		cmdata.args[name] = msgArgs[index]
		cmdata.TargetChannel = state.GetChannel(msgArgs[index])
		return true, nil
	}

	return false, fmt.Errorf(errFmtArgumentNotChannel, msgArgs[index])
}

// parseUserArg takes user arguments and assigns them to the correct structures
// in a command data struct.
func (c *Commander) parseUserArg(cmdata *CommandData, state *data.State,
	store *data.Store, srv, name string, t argType, users ...string) error {

	vargs := (t & VARIADIC) != 0
	nUsers := len(users)

	var access *data.UserAccess
	var user *data.User
	var err error

	addData := func(index int) {
		if access != nil {
			if vargs {
				cmdata.TargetVarUserAccess[index] = access
			} else {
				cmdata.TargetUserAccess[name] = access
			}
		}
		if user != nil {
			if vargs {
				cmdata.TargetVarUsers[index] = user
			} else {
				cmdata.TargetUsers[name] = user
			}
		}
	}

	if vargs {
		cmdata.TargetVarUsers = make([]*data.User, nUsers)
	} else {
		if cmdata.TargetUsers == nil {
			cmdata.TargetUsers = make(map[string]*data.User)
		}
	}

	switch t & USERMASK {
	case USER:
		if vargs {
			cmdata.TargetVarUserAccess = make([]*data.UserAccess, nUsers)
		} else {
			if cmdata.TargetUserAccess == nil {
				cmdata.TargetUserAccess = make(map[string]*data.UserAccess)
			}
		}
		for i, u := range users {
			access, user, err = cmdata.FindAccessByUser(srv, u)
			if err != nil {
				return err
			}
			addData(i)
		}
	case NICK:
		for i, u := range users {
			user, err = cmdata.FindUserByNick(u)
			if err != nil {
				return err
			}
			addData(i)
		}
	}

	return nil
}

// GetPrefix returns the prefix used by this commander instance.
func (c *Commander) GetPrefix() rune {
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
