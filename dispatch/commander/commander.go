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

// Internal constants.
const (
	varArgs = -1
	// The bot-global registration "server name".
	GLOBAL = "GLOBAL"
)

// Error messages.
const (
	errFmtDuplicateCommand = `commander: ` +
		`Duplicate command registration attempted (%v)`
	errMsgHandlerRequired = `commander: ` +
		`Handler required for command registration.`

	errMsgStoreDisabled = "Access Denied: Permissions required but cannot " +
		"check them due to a disabled store."
	errMsgNotAuthed = "Access Denied: Permissions required but you are not " +
		"authenticated."
	errFmtInsuffLevel        = "Access Denied: Level %v required."
	errFmtInsuffFlags        = "Access Denied: %v flag(s) required."
	errMsgUnexpectedArgument = "Error: No arguments expected."
	errFmtNArguments         = "Error: Expected %v %v arguments. (%v)"
	errAtLeast               = "at least"
	errExactly               = "exactly"
	errAtMost                = "at most"

	errFmtArgumentForm = `commander: Arguments must look like: ` +
		`name OR [name] OR name... (given: %v)`
	errFmtArgumentOrderReq = `commander: Required arguments must come before ` +
		`all [optional] and varargs... arguments. (given: %v)`
	errFmtArgumentOrderOpt = `commander: Optional arguments must come before ` +
		`varargs... arguments. (given: %v)`
	errFmtArgumentDupVargs = `commander: Only one varargs is allowed ` +
		`(given: %v)`
)

var (
	// commandArgRegexp checks a single argument to see if it matches the
	// forms: arg [arg] or arg...
	commandArgRegexp = regexp.MustCompile(
		`(?i)^(\[[a-z0-9]+\]|[a-z0-9]+(\.\.\.)?)$`)

	// globalCommandRegistry is a singleton throughout the entire bot, and
	// ensures that a command can only be registered once for each server.
	globalCommandRegistry = make(map[string]bool)
	// protectGlobalReg protects the global registry.
	protectGlobalReg sync.RWMutex
)

// CommandData represents the data about the even that's occurred. The commander
// fills the CommandData structure with information about the user and channel
// involved. It also embeds the State and Store for easy access.
//
// CommandData comes with the implication that the State and Store
// have been locked for reading, A typical error handler that quickly does some
// work and returns does not need to worry about calling Close() since it is
// guaranteed to automatically be closed when the
// handler returns. But a call to Close() must be given in a
// command handler that will do some long running processes. Note that all data
// in the CommandData struct becomes volatile and not thread-safe after a call
// to Close() has been made, so the values in the CommandData struct are set to
// nil but extra caution should be made when copying data from this struct and
// calling Close() afterwards since this data is shared between other parts of
// the bot.
//
// Some parts of CommandData will be nil under certain circumstances so elements
// within must be checked for nil for access, see each element's documentation
// for further information.
type CommandData struct {
	ep *data.DataEndpoint
	*data.State
	*data.Store
	// User can be nil if the bot's State is disabled.
	User *data.User
	// UserAccess will be nil when there is no required access.
	UserAccess *data.UserAccess
	// UserChannelModes will be nil when the message was not sent to a channel.
	UserChannelModes *data.UserModes
	// Channel will be nil when the message was not sent to a channel.
	Channel *data.Channel
	args    map[string]string
	once    sync.Once
}

// GetArg gets an argument that was passed in to the command by the user. The
// name of the argument passed into Register() is required to get the argument.
func (cd *CommandData) GetArg(arg string) string {
	return cd.args[arg]
}

// Close closes the handles to the internal structures. Calling Close is not
// required. See CommandData's documentation for when to call this method.
// All CommandData's methods and fields become invalid after a call to Close.
// Close will never return an error so it should be ignored.
func (cd *CommandData) Close() error {
	cd.once.Do(func() {
		cd.User = nil
		cd.UserAccess = nil
		cd.UserChannelModes = nil
		cd.Channel = nil
		cd.State = nil
		cd.Store = nil
		cd.ep.CloseState()
		cd.ep.CloseStore()
	})
	return nil
}

// CommandHandler is the interface that Commander expects structs to implement
// in order to be able to handle command events.
type CommandHandler interface {
	Command(string, *irc.Message, *data.DataEndpoint, *CommandData) error
}

// command holds all the information about a registered command handler.
type command struct {
	Args      []string
	Argnames  []string
	ReqArgs   int
	OptArgs   int
	Callscope int
	ReqLevel  uint8
	ReqFlags  string
	Handler   CommandHandler
}

// setArgs parses and sets the arguments for a command.
func (c *command) setArgs(args ...string) error {
	if len(args) == 0 {
		return nil
	}

	for i := 0; i < len(args); i++ {
		arg := strings.ToLower(args[i])
		if !commandArgRegexp.MatchString(arg) {
			return formatError(errFmtArgumentForm, arg)
		}

		switch arg[len(arg)-1] {
		case ']':
			if c.OptArgs == varArgs {
				return formatError(errFmtArgumentOrderOpt, arg)
			}
			c.OptArgs++
		case '.':
			if c.OptArgs == varArgs {
				return formatError(errFmtArgumentDupVargs, arg)
			}
			c.OptArgs = varArgs
		default:
			if c.OptArgs != 0 {
				return formatError(errFmtArgumentOrderReq, arg)
			}
			c.ReqArgs++
		}
	}

	c.Args = make([]string, len(args))
	c.Argnames = make([]string, len(args))
	for i := 0; i < len(args); i++ {
		c.Argnames[i] = strings.Trim(args[i], ".[]")
	}
	copy(c.Args, args)
	return nil
}

// commandTable is used to store all the string->command assocations.
type commandTable map[string]*command

// Commander allows for registration of commands that can involve user access,
// and provides rich programming interface for command handling.
type Commander struct {
	*dispatch.DispatchCore
	prefix   rune
	commands commandTable
}

// CreateCommander initializes a commander.
func CreateCommander(prefix rune, core *dispatch.DispatchCore) *Commander {
	return &Commander{
		DispatchCore: core,
		prefix:       prefix,
		commands:     make(commandTable),
	}
}

// Register creates a command with the bot. See documentation on CommandHandler
// to understand the implications of recieving this struct. Server should be
// either the name of a server or the GLOBAL constant. Msgtype can be one
// of the constants: PRIVMSG, NOTICE, ALL and the Scope can be one of the
// constants: PRIVATE, PUBLIC, ALL.
//
// Arguments for a command should be given in the form of an array, one argument
// per element. When a command is received these arguments are checked to ensure
// their existence or optional existence, the user will receive an error if
// the required arguments are not provided, or excessive arguments are provided.
// Each argument should be in the form: required OR [optional] OR varargs...
// [optional] can only follow required (or no) argument, and varargs... can only
// be the last definition in the sequence. Any other ordering is an error and
// such an error will be returned by the Register command. Arguments are parsed
// in and available by name through CommandData's GetArgument function.
func (c *Commander) Register(server, cmd string, handler CommandHandler,
	msgtype, scope int, args ...string) error {

	globalCmd := makeIdentifier(server, cmd)

	command, err :=
		c.createCommand(globalCmd, cmd, handler, msgtype, scope, args...)
	if err != nil {
		return err
	}

	protectGlobalReg.Lock()
	defer protectGlobalReg.Unlock()
	globalCommandRegistry[globalCmd] = true
	c.commands[cmd] = command
	return nil
}

// RegisterAuthed creates a command with the bot. Read the documentation on
// Register to know about the common parameters.
//
// Unique to RegisterAuthed is the level and flags arguments. The level is
// the level required to access this command, if it is 0, there is no level
// required. Required flags can also be given, if it is empty string, there is
// no flags required. If level is 0, and flags are empty, then it behaves
// exactly like a Register call, and Register should be used instead.
func (c *Commander) RegisterAuthed(server, cmd string, handler CommandHandler,
	msgtype, scope int, level uint8, flags string, args ...string) error {

	globalCmd := makeIdentifier(server, cmd)

	command, err :=
		c.createCommand(globalCmd, cmd, handler, msgtype, scope, args...)
	if err != nil {
		return err
	}

	command.ReqLevel = level
	command.ReqFlags = flags

	protectGlobalReg.Lock()
	defer protectGlobalReg.Unlock()
	globalCommandRegistry[globalCmd] = true
	c.commands[cmd] = command
	return nil
}

// register creates a command with the bot.
func (c *Commander) createCommand(globalCmd, cmd string, handler CommandHandler,
	msgtype, scope int, args ...string) (*command, error) {

	if handler == nil {
		return nil, errors.New(errMsgHandlerRequired)
	}

	protectGlobalReg.RLock()
	defer protectGlobalReg.RUnlock()
	if _, has := globalCommandRegistry[globalCmd]; has {
		return nil, formatError(errFmtDuplicateCommand, globalCmd)
	}

	command := &command{
		Callscope: msgtype | (scope << 2),
		Handler:   handler,
	}

	if err := command.setArgs(args...); err != nil {
		return nil, err
	}

	return command, nil
}

// Unregister unregisters a command from the bot. server should be the name
// of a server, or the GLOBAL constant.
func (c *Commander) Unregister(server, cmd string) (found bool) {
	protectGlobalReg.Lock()
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

	ch := ""
	msgscope := PRIVATE
	cmd := msg.Args[1]
	isChan, hasChan := c.CheckTarget(msg.Args[0])

	if isChan {
		if !hasChan || rune(cmd[0]) != c.prefix {
			return nil
		}

		cmd = cmd[1:]
		ch = msg.Args[0]
		msgscope = PUBLIC
	}

	var command *command
	var ok bool
	if command, ok = c.commands[cmd]; !ok {
		return nil
	}

	if cs := command.Callscope; 0 == (msgtype&cs) || 0 == (msgscope&(cs>>2)) {
		return nil
	}

	var cmdata = CommandData{
		ep: ep,
	}

	if cmdata.args, err = filterArgs(command, msg); err != nil {
		ep.Notice(irc.Mask(msg.Sender).GetNick(), err.Error())
		return
	}

	state := ep.OpenState()
	store := ep.OpenStore()
	cmdata.State = state
	cmdata.Store = store

	if a, err := filterAccess(store, command, server, ch, ep, msg); err != nil {
		ep.Notice(irc.Mask(msg.Sender).GetNick(), err.Error())
		return err
	} else {
		cmdata.UserAccess = a
	}

	if state != nil {
		cmdata.User = state.GetUser(msg.Sender)
		if isChan {
			cmdata.Channel = state.GetChannel(ch)
			cmdata.UserChannelModes = state.GetUsersChannelModes(msg.Sender, ch)
		}
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

// filterArgs checks to ensure a command has exactly the right number of
// arguments and makes an argError message if not.
func filterArgs(cmd *command, msg *irc.IrcMessage) (map[string]string, error) {
	nArgs := len(msg.Args) - 2
	minArgs, maxArgs := cmd.ReqArgs, cmd.ReqArgs+cmd.OptArgs
	isVargs := cmd.OptArgs == varArgs
	if nArgs >= minArgs && (isVargs || nArgs <= maxArgs) {
		if minArgs == 0 && maxArgs == 0 {
			return nil, nil
		}
		return parseArgs(cmd.Args, cmd.Argnames, msg.Args[2:]), nil
	}

	if nArgs > 0 && cmd.ReqArgs == 0 && cmd.OptArgs == 0 {
		return nil, errors.New(errMsgUnexpectedArgument)
	}

	var errStr string
	switch cmd.OptArgs {
	case 0:
		errStr = errExactly
	case varArgs:
		errStr = errAtLeast
	default:
		errStr = errAtMost
	}
	return nil, formatError(errFmtNArguments, errStr, maxArgs, cmd.Args)
}

// parseArgs parses the arguments in the command into a map. This function
// does no checking, it should have been lined up before hand.
func parseArgs(cmdArgs, argNames, msgArgs []string) (args map[string]string) {
	args = make(map[string]string, len(cmdArgs))
	used := 0
	for i, arg := range cmdArgs {
		if used >= len(msgArgs) {
			return
		}
		name := argNames[i]
		switch arg[len(arg)-1] {
		case '.':
			args[name] = strings.Join(msgArgs[used:], " ")
		default:
			args[name] = msgArgs[used]
			used++
		}
	}
	return
}

// filterAccess ensures that a user has the correct access to perform the given
// command.
func filterAccess(store *data.Store, command *command, server, channel string,
	ep *data.DataEndpoint, msg *irc.IrcMessage) (*data.UserAccess, error) {

	hasLevel := command.ReqLevel != 0
	hasFlags := len(command.ReqFlags) != 0
	if !hasLevel && !hasFlags {
		return nil, nil
	}

	if store == nil {
		return nil, errors.New(errMsgStoreDisabled)
	}

	var access = store.GetAuthedUser(ep.GetKey(), msg.Sender)
	if access == nil {
		return nil, errors.New(errMsgNotAuthed)
	}
	if hasLevel && !access.HasLevel(server, channel, command.ReqLevel) {
		return nil, formatError(errFmtInsuffLevel, command.ReqLevel)
	}
	if hasFlags && !access.HasFlags(server, channel, command.ReqFlags) {
		return nil, formatError(errFmtInsuffFlags, command.ReqFlags)
	}

	return access, nil
}

// makeIdentifier creates an identifier from a server and a command for
// registration.
func makeIdentifier(server, cmd string) string {
	return server + ":" + cmd
}

// formatError uses a format string to create an error.
func formatError(format string, args ...interface{}) error {
	return errors.New(fmt.Sprintf(format, args...))
}

// MakeLevelError creates an error to be shown to the user about required access
func MakeLevelError(levelRequired uint8) error {
	return formatError(errFmtInsuffLevel, levelRequired)
}

// MakeFlagsError creates an error to be shown to the user about required access
func MakeFlagsError(flagsRequired string) error {
	return formatError(errFmtInsuffFlags, flagsRequired)
}
