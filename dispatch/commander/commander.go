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
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/dispatch"
	"github.com/aarondl/ultimateq/irc"
	"regexp"
	"strings"
	"sync"
)

// Constants used for defining the targets/scope of a command.
const (
	// The bot-global registration "server name".
	GLOBAL = "GLOBAL"
	// PRIVMSG only listens to irc.PRIVMSG events.
	PRIVMSG = 0x1
	// NOTICE only listens to irc.NOTICE events.
	NOTICE = 0x2
	// PRIVATE only listens to PRIVMSG or NOTICE sent directly to the bot.
	PRIVATE = 0x1
	// PUBLIC only listens to PRIVMSG or NOTICE sent to a channel.
	PUBLIC = 0x2
	// When passed into the msgtype parameter: ALL listens to both PRIVMSG and
	// NOTICE events.
	// When passed into the scope parameter: ALL listens for messages sent both
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

	errMsgStoreDisabled = "Access Denied: Cannot use authenticated commands " +
		"or access parameters when store is disabled."
	errMsgStateDisabled = "Error: Cannot use user parameter commands " +
		"when state is disabled."
	errMsgNotAuthed          = "Access Denied: You are not authenticated."
	errFmtInsuffLevel        = "Access Denied: [%v] level required."
	errFmtInsuffFlags        = "Access Denied: [%v] flag(s) required."
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

// Internal constants.
const (
	attribUser = iota + 1
	attribAuthed
	varArgs            = -1
	argNamesStripChars = "#*~[]."
)

var (
	// commandArgRegexp checks a single argument to see if it matches the
	// forms: arg #arg [arg] or arg...
	commandArgRegexp = regexp.MustCompile(
		`(?i)^(\[[~\*]?[a-z0-9]+\]|[~\*]?[a-z0-9]+(\.\.\.)?|#[a-z0-9]+)$`)

	// globalCommandRegistry is a singleton throughout the entire bot, and
	// ensures that a command can only be registered once for each server.
	globalCommandRegistry = make(map[string]*Command)
	// protectGlobalReg protects the global registry.
	protectGlobalReg sync.RWMutex
)

// CommandHandler is the interface that Commander expects structs to implement
// in order to be able to handle command events.
type CommandHandler interface {
	Command(string, *irc.Message, *data.DataEndpoint, *CommandData) error
}

// commandTable is used to store all the string->command assocations.
type commandTable map[string]*Command

// Commander allows for registration of commands that can involve user access,
// and provides rich programming interface for command handling.
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
func (c *Commander) Dispatch(server string, msg *irc.IrcMessage,
	ep *data.DataEndpoint) (err error) {

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

	fields := strings.Fields(msg.Args[1])
	if len(fields) == 0 {
		return nil
	}
	cmd := fields[0]

	ch := ""
	msgscope := PRIVATE
	isChan, hasChan := c.CheckTarget(msg.Args[0])

	if isChan {
		if !hasChan || rune(cmd[0]) != c.prefix {
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

	var cmdata = CommandData{
		ep: ep,
	}

	var args []string
	if len(fields) > 1 {
		args = fields[1:]
	}

	cmdArgs, cmdArgnames := command.Args, command.argnames
	reqArgs, optArgs := command.reqArgs, command.optArgs
	cmdArgAttribs := command.argAttrib
	reqAdj, optAdj := 0, 0

	var chanArg string
	if command.chanArg {
		if chanArg, err = c.filterChanArgs(command, args, isChan); err != nil {
			ep.Notice(irc.Mask(msg.Sender).GetNick(), err.Error())
			return
		} else if len(chanArg) > 0 {
			args = args[1:]
		} else {
			chanArg = ch
		}

		cmdArgs = cmdArgs[1:]
		cmdArgnames = cmdArgnames[1:]
		cmdArgAttribs = cmdArgAttribs[1:]
		if isChan {
			optAdj = 1
		} else {
			reqAdj = 1
		}
	}

	cmdata.args, err = filterArgs(cmdArgs, cmdArgnames, reqArgs, optArgs,
		reqAdj, optAdj, args)

	if err != nil {
		ep.Notice(irc.Mask(msg.Sender).GetNick(), err.Error())
		return
	}
	if command.chanArg {
		if cmdata.args == nil {
			cmdata.args = map[string]string{
				command.argnames[0]: chanArg,
			}
		} else {
			cmdata.args[command.argnames[0]] = chanArg
		}
	}

	state := ep.OpenState()
	store := ep.OpenStore()
	cmdata.State = state
	cmdata.Store = store

	if command.RequireAuth {
		if cmdata.UserAccess, err = filterAccess(store, command,
			server, ch, ep, msg); err != nil {

			cmdata.Close()
			ep.Notice(irc.Mask(msg.Sender).GetNick(), err.Error())
			return
		}
	}

	if state != nil {
		cmdata.User = state.GetUser(msg.Sender)
		if command.chanArg {
			cmdata.TargetChannel = state.GetChannel(chanArg)
		}
		if isChan {
			cmdata.Channel = state.GetChannel(ch)
			cmdata.UserChannelModes = state.GetUsersChannelModes(msg.Sender, ch)
		}
	}

	if err = populateUserArgs(server, cmdArgs, cmdArgnames, args, cmdArgAttribs,
		&cmdata, state, store); err != nil {
		cmdata.Close()
		ep.Notice(irc.Mask(msg.Sender).GetNick(), err.Error())
		return
	}

	c.HandlerStarted()
	go func() {
		defer cmdata.Close()
		err := command.Handler.Command(cmd, &irc.Message{msg}, ep, &cmdata)
		if err != nil {
			ep.Notice(irc.Mask(msg.Sender).GetNick(), err.Error())
		}
		c.HandlerFinished()
	}()

	return nil
}

// filterChanArgs checks for a channel argument.
func (c *Commander) filterChanArgs(cmd *Command, args []string, isChan bool) (
	channel string, err error) {

	if !cmd.chanArg {
		return
	}
	if isChan {
		if len(args) == 0 {
			return
		}
		if isFirstChan, _ := c.CheckTarget(args[0]); isFirstChan {
			channel = args[0]
		}
	} else {
		if len(args) == 0 {
			errStr := errExactly
			if cmd.optArgs != 0 {
				errStr = errAtLeast
			}
			err = fmt.Errorf(errFmtNArguments, errStr, cmd.reqArgs+1,
				strings.Join(cmd.Args, " "))
			return
		}
		if isFirstChan, _ := c.CheckTarget(args[0]); isFirstChan {
			channel = args[0]
		} else {
			err = fmt.Errorf(errFmtArgumentNotChannel, args[0])
			return
		}
	}

	return
}

// filterArgs checks to ensure a command has exactly the right number of
// arguments and makes an argError message if not.
func filterArgs(args, argNames []string, reqArgs, optArgs int,
	reqAdj, optAdj int, msgArgs []string) (map[string]string, error) {

	nArgs := len(msgArgs)
	if nArgs > 0 && reqArgs == 0 && optArgs == 0 {
		return nil, errors.New(errMsgUnexpectedArgument)
	}

	minArgs, maxArgs := reqArgs, reqArgs+optArgs
	isVargs := optArgs == varArgs
	if nArgs >= minArgs && (isVargs || nArgs <= maxArgs) {
		if minArgs == 0 && maxArgs == 0 {
			return nil, nil
		}
		return parseArgs(args, argNames, msgArgs), nil
	}

	var errStr string
	var errArgs = reqArgs
	if optArgs >= 0 {
		optArgs += optAdj
	} else {
		reqArgs += reqAdj
	}
	switch true {
	case optArgs == 0:
		errStr = errExactly
	case isVargs, nArgs < minArgs:
		errStr = errAtLeast
	case nArgs > maxArgs:
		errStr = errAtMost
		errArgs = maxArgs
	}
	return nil, fmt.Errorf(errFmtNArguments, errStr, errArgs,
		strings.Join(args, " "))
}

// parseArgs parses the arguments in the command into a map. This function
// does no checking, it should have been lined up before hand.
func parseArgs(args, argNames, msgArgs []string) (retargs map[string]string) {
	retargs = make(map[string]string, len(args))
	used := 0
	for i, arg := range args {
		if used >= len(msgArgs) {
			return
		}
		name := argNames[i]
		switch arg[len(arg)-1] {
		case '.':
			retargs[name] = strings.Join(msgArgs[used:], " ")
		default:
			retargs[name] = msgArgs[used]
			used++
		}
	}
	return
}

// filterAccess ensures that a user has the correct access to perform the given
// command.
func filterAccess(store *data.Store, command *Command, server, channel string,
	ep *data.DataEndpoint, msg *irc.IrcMessage) (*data.UserAccess, error) {

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

// populateUserArgs uses the store and state to look up any ~user or *user
// type parameters in the arguments.
func populateUserArgs(server string, args, argNames, msgArgs []string,
	argAttrib []int, cmdata *CommandData,
	state *data.State, store *data.Store) error {

	for i, arg := range args {
		attrib := argAttrib[i]
		if attrib == 0 {
			continue
		}

		j, vIndex, vargs := 1, i, false
		argname := ""
		if vargs = '.' == arg[len(arg)-1]; !vargs {
			argname = argNames[i]
			if i >= len(msgArgs) {
				return nil
			}
		} else {
			if j = len(msgArgs) - i; j <= 0 {
				return nil
			}
			switch attrib {
			case attribUser:
				cmdata.TargetVarUsers = make([]*data.User, j)
			case attribAuthed:
				cmdata.TargetVarUsers = make([]*data.User, j)
				cmdata.TargetVarUserAccess = make([]*data.UserAccess, j)
			}
		}

		for ; j > 0; j-- {
			mArg := msgArgs[i]
			index := i - vIndex
			if vargs {
				i++
			}
			switch attrib {
			case attribUser:
				user, err := cmdata.FindUserByNick(mArg)
				if err != nil {
					return err
				}
				cmdata.addUser(argname, index, vargs, user)
			case attribAuthed:
				access, user, err := cmdata.FindAccessByUser(server, mArg)
				if err != nil {
					return err
				}
				if user != nil {
					cmdata.addUser(argname, index, vargs, user)
				}
				cmdata.addUserAccess(argname, index, vargs, access)
			}
		}
	}

	return nil
}

// makeIdentifier creates an identifier from a server and a command for
// registration.
func makeIdentifier(server, cmd string) string {
	return server + ":" + cmd
}

// MakeLevelError creates an error to be shown to the user about required access
func MakeLevelError(levelRequired uint8) error {
	return fmt.Errorf(errFmtInsuffLevel, levelRequired)
}

// MakeFlagsError creates an error to be shown to the user about required access
func MakeFlagsError(flagsRequired string) error {
	return fmt.Errorf(errFmtInsuffFlags, flagsRequired)
}

// MakeNotAuthedError creates an error to be shown to the user about their
// target user not being authenticated.
func MakeUserNotAuthedError(user string) error {
	return fmt.Errorf(errFmtUserNotAuthed, user)
}

// MakeUserNotFoundError creates an error to be shown to the user about their
// target user not being found.
func MakeUserNotFoundError(user string) error {
	return fmt.Errorf(errFmtUserNotFound, user)
}
