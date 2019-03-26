package cmd

import (
	"regexp"
	"strings"

	"github.com/aarondl/ultimateq/irc"
	"github.com/pkg/errors"
)

var (
	rgxCmd = regexp.MustCompile(`^[a-z][a-z0-9]*$`)
)

// Handler for command types
type Handler interface {
	Cmd(name string, writer irc.Writer, event *Event) error
}

// HandlerFunc implements Handler
type HandlerFunc func(name string, writer irc.Writer, event *Event) error

// Cmd implements Handler
func (h HandlerFunc) Cmd(name string, writer irc.Writer, event *Event) error {
	return h(name, writer, event)
}

// Command holds all the information about a command.
type Command struct {
	// The name of the command.
	Name string
	// Extension is the name of the extension registering this command.
	Extension string
	// Description is a description of the command's function.
	Description string
	// Kind is the kind of messages this command reacts to, may be
	// any of the constants: Privmsg, Notice or AllKinds.
	Kind Kind
	// Scope is the scope of the messages this command reacts to, may be
	// any of the constants: Private, Public or Allscopes.
	Scope Scope
	// Args is the arguments for the command. Each argument must be in it's own
	// element, be named with flags optionally prefixing the name, and have the
	// form of one of the following:
	// #channel: This form is for requiring a target channel for the command.
	//     If this parameter is present and a message directly to the bot is
	//     received this parameter is required and if it's missing an error
	//     will be returned.
	//     If this parameter is present and a message to a channel is received
	//     the there is two cases: 1) The first parameter given is a channel,
	//     this then becomes the TargetChannel. 2) The first parameter given
	//     is non existent or not a channel, the current channel then becomes
	//     the TargetChannel.
	// required: This form marks a required attribute and it must be present
	//     or an error will be returned. It must come after #channel but before
	//     [optional] and varargs... arguments.
	// [optional]: This form is an optional argument. It must come before after
	//     required but before varargs... arguments.
	// varargs...: This form is a variadic argument, there may be 0 or more
	//     arguments to satisfy this parameter and they will all be parsed
	//     together as one string by the commander. This must come at the end.
	// There are two types of flags available:
	// ~: This flag is a nickname flag. If this flag is present the bot
	//     will look up the nickname given in the state database, if it does
	//     not exist an error will occur.
	// *: This flag is a user flag. It looks up a user based on nick OR
	//     username. If any old nickname is given, it first looks up the user
	//     in the state database, and then checks his authentication record
	//     to get his username (and therefore access).  If the name is prefixed
	//     by a *, then it looks up the user based on username directly. If
	//     the user is not found (via nickname), not authed (via username)
	//     the command will fail.
	Args []string
	// RequireAuth is whether or not this command requires authentication.
	RequireAuth bool
	// ReqLevel is the required level for use.
	ReqLevel uint8
	// ReqFlags is the required flags for use.
	ReqFlags string
	// Handler the handler structure that will handle events for this command.
	Handler Handler

	parsedArgs commandArgs
}

// New is a helper method to easily create a Command. See the documentation
// for Command on what each parameter is. Panics if the args are invalid.
func New(
	ext,
	cmd,
	desc string,
	handler Handler,
	kind Kind,
	scope Scope,
	args ...string) *Command {

	command, err := NewErr(ext, cmd, desc, handler, kind, scope, args...)
	if err != nil {
		panic(err)
	}

	return command
}

// NewErr is like New but does not panic
func NewErr(
	ext,
	cmd,
	desc string,
	handler Handler,
	kind Kind,
	scope Scope,
	args ...string) (*Command, error) {

	cmd = strings.ToLower(cmd)
	if !rgxCmd.MatchString(cmd) {
		return nil, errors.Errorf("command name must start with a letter, and can be followed only be letters and numbers: %s", cmd)
	}

	command := &Command{
		Name:        strings.ToLower(cmd),
		Extension:   strings.ToLower(ext),
		Description: desc,
		Handler:     handler,
		Kind:        kind,
		Scope:       scope,
		Args:        args,
	}

	if err := command.parseArgs(); err != nil {
		return nil, err
	}

	return command, nil
}

// NewAuthed is a helper method to easily create an authenticated Command. See
// the documentation on Command for what each parameter is.
// Panics if the args are invalid.
func NewAuthed(
	ext,
	cmd,
	desc string,
	handler Handler,
	kind Kind,
	scope Scope,
	reqLevel uint8,
	reqFlags string,
	args ...string) *Command {

	command, err := NewAuthedErr(ext, desc, cmd, handler, kind, scope, reqLevel, reqFlags, args...)
	if err != nil {
		panic(err)
	}

	return command
}

// NewAuthedErr is the same as NewAuthed but returns an error instead of panics
func NewAuthedErr(
	ext,
	cmd,
	desc string,
	handler Handler,
	kind Kind,
	scope Scope,
	reqLevel uint8,
	reqFlags string,
	args ...string) (*Command, error) {

	command, err := NewErr(ext, desc, cmd, handler, kind, scope, args...)
	if err != nil {
		return nil, err
	}
	command.RequireAuth = true
	command.ReqLevel = reqLevel
	command.ReqFlags = reqFlags

	return command, nil
}
