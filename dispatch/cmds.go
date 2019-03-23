package dispatch

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/dispatch/cmd"
	"github.com/aarondl/ultimateq/irc"
	"github.com/pkg/errors"
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
	errFmtUserNotRegistered = "Error: User [%v] is not registered."
	errFmtUserNotAuthed     = "Error: User [%v] is not authenticated."
	errFmtUserNotFound      = "Error: User [%v] could not be found."
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

// NewCommandDispatcher initializes a cmds.
func NewCommandDispatcher(fetcher pfxFetcher, core *DispatchCore) *CommandDispatcher {
	return &CommandDispatcher{
		DispatchCore: core,
		fetcher:      fetcher,
		trie:         newTrie(false),
	}
}

// Register a command with the bot. See documentation for
// Cmd for information about how to use this method, as well as see
// the documentation for CmdHandler for how to respond to commands issued by
// users. Network and Channel may be given to restrict which networks/channels
// this event will fire on.
func (c *CommandDispatcher) Register(network, channel string, command *cmd.Command) (uint64, error) {
	switch {
	case len(command.Name) == 0:
		return 0, errors.New(errMsgCmdRequired)
	case len(command.Extension) == 0:
		return 0, errors.New(errMsgExtRequired)
	case len(command.Description) == 0:
		return 0, errors.New(errMsgDescRequired)
	case command.Handler == nil:
		return 0, errors.New(errMsgHandlerRequired)
	}

	c.mutTrie.Lock()
	defer c.mutTrie.Unlock()
	handlers := c.trie.handlers("", "", command.Name)
	for _, h := range handlers {
		handlerCmd := h.(*cmd.Command)
		if handlerCmd.Name == command.Name && handlerCmd.Extension == command.Extension {
			return 0, errors.Errorf(errFmtDuplicateCmd, command.Extension+"."+command.Name)
		}
	}

	hid := c.trie.register(network, channel, command.Name, command)
	if hid == errTrieNotUnique {
		return 0, errors.Errorf(errFmtDuplicateCmd, command.Name)
	}

	return hid, nil
}

// Unregister a command previously registered with the bot. Use the id returned
// from Register to do so.
func (c *CommandDispatcher) Unregister(id uint64) (found bool) {
	c.mutTrie.RLock()
	defer c.mutTrie.RUnlock()

	return c.trie.unregister(id)
}

// Dispatch dispatches an IrcEvent into the cmds event handlers.
func (c *CommandDispatcher) Dispatch(writer irc.Writer, ev *irc.Event,
	provider data.Provider) (err error) {

	// Filter non privmsg/notice
	var kind cmd.Kind
	switch ev.Name {
	case irc.PRIVMSG:
		kind = cmd.Privmsg
	case irc.NOTICE:
		kind = cmd.Notice
	}

	if kind == 0 {
		return nil
	}

	// Get command name or die trying
	fields := strings.Fields(ev.Args[1])
	if len(fields) == 0 {
		return nil
	}
	commandName := strings.ToLower(fields[0])

	ch := ""
	nick := irc.Nick(ev.Sender)
	scope := cmd.Private
	isChan := len(ev.Args) > 0 && ev.IsTargetChan()

	// If it's a channel message, ensure we're active on the channel and
	// that the user has supplied the prefix in his command.
	if isChan {
		ch = ev.Target()
		prefix := c.fetcher(ev.NetworkID, ev.Target())

		firstChar := rune(commandName[0])
		if firstChar != prefix {
			return nil
		}

		commandName = commandName[1:]
		scope = cmd.Public
	}

	// Check if they've supplied the more specific ext.cmd form.
	var ext string
	if ln, dot := len(commandName), strings.IndexRune(commandName, '.'); ln >= 3 && dot > 0 {
		if ln-dot-1 == 0 {
			return nil
		}
		ext = commandName[:dot]
		commandName = commandName[dot+1:]
	}

	c.mutTrie.RLock()
	defer c.mutTrie.RUnlock()
	handlers := c.trie.handlers(ev.NetworkID, ch, commandName)

	var command *cmd.Command
	switch len(handlers) {
	case 0:
		// Do nothing, we found nothing to handle this
		return nil
	case 1:
		command = handlers[0].(*cmd.Command)
	default:
		var remaining []*cmd.Command
		var collisions []string
		for _, h := range handlers {
			actualCmd := h.(*cmd.Command)

			if len(ext) == 0 || actualCmd.Extension == ext {
				remaining = append(remaining, actualCmd)
			}

			fullName := fmt.Sprintf("%s.%s", actualCmd.Extension, actualCmd.Name)
			collisions = append(collisions, fullName)
		}

		if len(remaining) == 1 {
			command = remaining[0]
		} else {
			err := errors.Errorf(errFmtAmbiguousCmd, commandName, strings.Join(collisions, ","))
			writer.Notice(nick, err.Error())
			return err
		}
	}

	if 0 == (kind&command.Kind) || 0 == (scope&command.Scope) {
		return nil
	}

	// Start building up the event.
	var cmdEv = &cmd.Event{
		Event: ev,
	}

	var args []string
	if len(fields) > 1 {
		args = fields[1:]
	}

	state := provider.State(ev.NetworkID)
	store := provider.Store()

	if command.RequireAuth {
		if cmdEv.StoredUser, err = filterAccess(store, command, ev.NetworkID,
			ch, ev); err != nil {

			writer.Notice(nick, err.Error())
			return err
		}
	}

	if err = cmd.ProcessArgs(ev.NetworkID, command, ch, isChan, args, cmdEv, ev,
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
		ok, err := cmdNameDispatch(command.Handler, commandName, writer, cmdEv)
		if !ok {
			err = command.Handler.Cmd(command.Name, writer, cmdEv)
		}
		if err != nil {
			writer.Notice(nick, err.Error())
		}
	}()

	return nil
}

// cmdNameDispatch attempts to dispatch an event to a function named the same
// as the command with an uppercase letter (no camel case). The arguments
// must be the exact same as the CmdHandler.Name with the cmd string
// argument removed for this to work.
func cmdNameDispatch(handler cmd.Handler, cmd string, writer irc.Writer,
	ev *cmd.Event) (dispatched bool, err error) {

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
func filterAccess(store *data.Store, command *cmd.Command, server, channel string,
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
		return nil, errors.Errorf(errFmtInsuffLevel, command.ReqLevel)
	}
	if hasFlags && !access.HasFlags(server, channel, command.ReqFlags) {
		return nil, errors.Errorf(errFmtInsuffFlags, command.ReqFlags)
	}

	return access, nil
}

/*
TODO: Fix this
// EachCmd iterates through the commands and passes each one to a callback
// function for consumption. These should be considered read-only. Optionally
// the results can be filtered by network and channel.
// To end iteration prematurely the callback function can return true.
func (c *CommandDispatcher) EachCmd(network, channel string, cb func(*cmd.Command) bool) {
	c.mutTrie.RLock()
	defer c.mutTrie.RUnlock()

	spew.Dump(c.trie)

	commands := c.trie.handlers(network, channel, "")
	for _, command := range commands {
		realCmd := command.(*cmd.Command)
		if cb(realCmd) {
			return
		}
	}
}
*/

// makeIdentifier creates an identifier from a server and a command for
// registration.
func makeIdentifier(server, cmd string) string {
	return server + ":" + cmd
}

// MakeLevelError creates an error to be shown to the user about required
// access.
func MakeLevelError(levelRequired uint8) error {
	return errors.Errorf(errFmtInsuffLevel, levelRequired)
}

// MakeGlobalLevelError creates an error to be shown to the user about required
// access.
func MakeGlobalLevelError(levelRequired uint8) error {
	return errors.Errorf(errFmtInsuffGlobalLevel, levelRequired)
}

// MakeServerLevelError creates an error to be shown to the user about required
// access.
func MakeServerLevelError(levelRequired uint8) error {
	return errors.Errorf(errFmtInsuffServerLevel, levelRequired)
}

// MakeChannelLevelError creates an error to be shown to the user about required
// access.
func MakeChannelLevelError(levelRequired uint8) error {
	return errors.Errorf(errFmtInsuffChannelLevel, levelRequired)
}

// MakeFlagsError creates an error to be shown to the user about required
// access.
func MakeFlagsError(flagsRequired string) error {
	return errors.Errorf(errFmtInsuffFlags, flagsRequired)
}

// MakeGlobalFlagsError creates an error to be shown to the user about required
// access.
func MakeGlobalFlagsError(flagsRequired string) error {
	return errors.Errorf(errFmtInsuffGlobalFlags, flagsRequired)
}

// MakeServerFlagsError creates an error to be shown to the user about required
// access.
func MakeServerFlagsError(flagsRequired string) error {
	return errors.Errorf(errFmtInsuffServerFlags, flagsRequired)
}

// MakeChannelFlagsError creates an error to be shown to the user about required
// access.
func MakeChannelFlagsError(flagsRequired string) error {
	return errors.Errorf(errFmtInsuffChannelFlags, flagsRequired)
}

// MakeUserNotAuthedError creates an error to be shown to the user about their
// target user not being authenticated.
func MakeUserNotAuthedError(user string) error {
	return errors.Errorf(errFmtUserNotAuthed, user)
}

// MakeUserNotFoundError creates an error to be shown to the user about their
// target user not being found.
func MakeUserNotFoundError(user string) error {
	return errors.Errorf(errFmtUserNotFound, user)
}

// MakeUserNotRegisteredError creates an error to be shown to the user about
// the target user not being registered.
func MakeUserNotRegisteredError(user string) error {
	return errors.Errorf(errFmtUserNotRegistered, user)
}

// mkKey creates a key for storing and retrieving event handlers.
func mkKey(network, channel, event string) string {
	return fmt.Sprintf("%s:%s:%s", network, channel, event)
}
