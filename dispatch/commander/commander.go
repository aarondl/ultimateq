/*
Package commander is a more involved dispatcher package. In short it allows
users to create bot commands very easily rather than doing everything by hand
in a privmsg handler.

It uses the data package to achieve command access verification. It also
provides some automatic parsing and handling of the command keyword and
arguments. Command keywords become unique throughout the bot and may not
be duplicated. There should only be one responding function per command.
*/
package commander

import (
	"errors"
	"fmt"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/dispatch"
	"github.com/aarondl/ultimateq/irc"
	"regexp"
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

	ErrMsgStoreDisabled = "Access Denied: Permissions required but cannot " +
		"check them due to a disabled store."
	ErrMsgNotAuthed = "Access Denied: Permissions required but you are not " +
		"authenticated."
	ErrFmtInsuffLevel        = "Access Denied: Level %v required."
	ErrFmtInsuffFlags        = "Access Denied: %v flag(s) required."
	ErrMsgUnexpectedArgument = "Error: No arguments expected."
	ErrFmtNArguments         = "Error: Expected %v %v arguments. (%v)"
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
	// protectGlobalReg locks the global registration.
	protectGlobalReg sync.RWMutex
)

// CommandData is a convenience structure to gather all the variables that
// come back as well as offer a closing mechanism to close up the opened
// handles to the State and Store.
type CommandData struct {
	ep               *data.DataEndpoint
	User             *data.User
	UserAccess       *data.UserAccess
	UserChannelModes *data.UserModes
	Channel          *data.Channel
	once             sync.Once
}

// Close closes the handles to the internal structures. Close will never return
// an error so it should be ignored.
func (db *CommandData) Close() error {
	db.once.Do(func() {
		db.User = nil
		db.UserAccess = nil
		db.UserChannelModes = nil
		db.Channel = nil
		db.ep.CloseState()
		db.ep.CloseStore()
	})
	return nil
}

// CommandHandler is an interface for command handling.
type CommandHandler interface {
	Command(string, *irc.Message, *data.DataEndpoint, *CommandData)
}

// command holds all the information about a registered command handler.
type command struct {
	Args      []string
	ReqArgs   int
	OptArgs   int
	Callscope int
	ReqLevel  uint8
	ReqFlags  string
	Handler   CommandHandler
}

// setArgs sets the arguments cleanly. Forgoes error handling
func (c *command) setArgs(args ...string) error {
	if len(args) == 0 {
		return nil
	}

	for _, arg := range args {
		if !commandArgRegexp.MatchString(arg) {
			return errors.New(fmt.Sprintf(errFmtArgumentForm, arg))
		}

		switch arg[len(arg)-1] {
		case ']':
			if c.OptArgs == varArgs {
				return errors.New(fmt.Sprintf(errFmtArgumentOrderOpt, arg))
			}
			c.OptArgs++
		case '.':
			if c.OptArgs == varArgs {
				return errors.New(fmt.Sprintf(errFmtArgumentDupVargs, arg))
			}
			c.OptArgs = varArgs
		default:
			if c.OptArgs != 0 {
				return errors.New(fmt.Sprintf(errFmtArgumentOrderReq, arg))
			}
			c.ReqArgs++
		}
	}

	c.Args = make([]string, len(args))
	copy(c.Args, args)
	return nil
}

// commandTable is used to store all the cmd->commandhandler associations.
type commandTable map[string]*command

// Commander allows registration of commands that will later be dispatched to.
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

// Register creates a command with the bot.
func (c *Commander) Register(server, cmd string, handler CommandHandler,
	msgtype, scope int, args ...string) error {

	globalCmd := makeIdentifier(server, cmd)

	command, err :=
		c.createCommand(server, cmd, handler, msgtype, scope, args...)
	if err != nil {
		return err
	}

	protectGlobalReg.Lock()
	defer protectGlobalReg.Unlock()
	globalCommandRegistry[globalCmd] = true
	c.commands[cmd] = command
	return nil
}

// RegisterAuthed creates an authenticated command with the bot.
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
		return nil, errors.New(fmt.Sprintf(errFmtDuplicateCommand, globalCmd))
	}

	if _, has := c.commands[cmd]; has {
		return nil, errors.New(fmt.Sprintf(errFmtDuplicateCommand, cmd))
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

// Unregister unregisters a global command from the bot.
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
		found = found && true
	}
	return
}

// Dispatch dispatches an IrcEvent into the commander's event handlers.
func (c *Commander) Dispatch(msg *irc.IrcMessage, ep *data.DataEndpoint) error {
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

	if err := filterArgs(command, msg); err != nil {
		return err
	}

	var data = CommandData{ep: ep}
	state := ep.OpenState()
	store := ep.OpenStore()

	if access, err := filterAccess(store, command, ch, ep, msg); err != nil {
		return err
	} else {
		data.UserAccess = access
	}

	if state != nil {
		data.User = state.GetUser(msg.Sender)
		if isChan {
			data.Channel = state.GetChannel(ch)
			data.UserChannelModes = state.GetUsersChannelModes(msg.Sender, ch)
		}
	}

	c.HandlerStarted()
	go func() {
		defer data.Close()
		command.Handler.Command(cmd, &irc.Message{msg}, ep, &data)
		c.HandlerFinished()
	}()

	return nil
}

// filterArgs checks to ensure a command has exactly the right number of
// arguments and makes an argError message if not.
func filterArgs(command *command, msg *irc.IrcMessage) error {
	nArgs := len(msg.Args) - 2
	minArgs, maxArgs := command.ReqArgs, command.ReqArgs+command.OptArgs
	isVargs := command.OptArgs == varArgs
	if nArgs >= minArgs && (isVargs || nArgs <= maxArgs) {
		return nil
	}

	if nArgs > 0 && command.ReqArgs == 0 && command.OptArgs == 0 {
		return errors.New(ErrMsgUnexpectedArgument)
	}

	var errStr string
	switch command.OptArgs {
	case 0:
		errStr = errExactly
	case varArgs:
		errStr = errAtLeast
	default:
		errStr = errAtMost
	}
	return errors.New(fmt.Sprintf(
		ErrFmtNArguments, errStr, maxArgs, command.Args,
	))
}

// filterAccess ensures that a user has the correct access to perform the given
// command.
func filterAccess(store *data.Store, command *command, channel string,
	ep *data.DataEndpoint, msg *irc.IrcMessage) (*data.UserAccess, error) {

	hasLevel := command.ReqLevel != 0
	hasFlags := len(command.ReqFlags) != 0
	if !hasLevel && !hasFlags {
		return nil, nil
	}

	if store == nil {
		return nil, errors.New(ErrMsgStoreDisabled)
	}

	var access = store.GetAuthedUser(ep.GetKey(), msg.Sender)
	if access == nil {
		return nil, errors.New(ErrMsgNotAuthed)
	}
	if hasLevel && !access.HasLevel(server, channel, command.ReqLevel) {
		return nil, errors.New(fmt.Sprintf(ErrFmtInsuffLevel, command.ReqLevel))
	}
	if hasFlags && !access.HasFlags(server, channel, command.ReqFlags) {
		return nil, errors.New(fmt.Sprintf(ErrFmtInsuffFlags, command.ReqFlags))
	}

	return access, nil
}

// makeIdentifier creates an identifier from a server and a command for
// registration.
func makeIdentifier(server, cmd string) string {
	return server + ":" + cmd
}
