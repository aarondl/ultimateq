package cmd

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// rgxArgs checks a single argument to see if it matches the
	// forms: arg #arg [arg] or arg...
	rgxArgs = regexp.MustCompile(
		`(?i)^(\[[~\*]?[a-z0-9]+\]|[~\*]?[a-z0-9]+(\.\.\.)?|#[a-z0-9]+)$`)
)

type argType int

// These are for internal use of the command structure to deal with parsing
// and storing argument metadata.
const (
	REQUIRED argType = 1 << iota
	OPTIONAL
	VARIADIC
	CHANNEL
	NICK
	USER

	TYPEMASK = REQUIRED | OPTIONAL | VARIADIC | CHANNEL
	USERMASK = NICK | USER

	argStripChars = `#~*[].`
)

// argument is a type to hold argument information.
type argument struct {
	Original string
	Name     string
	Type     argType
}

// Cmd holds all the information about a command.
type Cmd struct {
	// The name of the command.
	Cmd string
	// Extension is the name of the extension registering this command.
	Extension string
	// Description is a description of the command's function.
	Description string
	// Kind is the kind of messages this command reacts to, may be the
	// any of the constants: PRIVMSG, NOTICE or ALLKINDS.
	Kind MsgKind
	// Scope is the scope of the messages this command reacts to. May be
	// any of the constants: PRIVATE, PUBLIC or ALLSCOPES.
	Scope MsgScope
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
	Handler CmdHandler
	// args stores data about each argument after it's parsed.
	args    []argument
	reqArgs int
	optArgs int
}

// MkCmd is a helper method to easily create a Cmd. See the documentation
// for Cmd on what each parameter is.
func MkCmd(ext, desc, cmd string, handler CmdHandler, kind MsgKind,
	scope MsgScope, args ...string) *Cmd {
	return &Cmd{
		Cmd:         cmd,
		Extension:   ext,
		Description: desc,
		Handler:     handler,
		Kind:        kind,
		Scope:       scope,
		Args:        args,
	}
}

// MkAuthCmd is a helper method to easily create an authenticated Cmd. See
// the documentation on Cmd for what each parameter is.
func MkAuthCmd(ext, desc, cmd string, handler CmdHandler,
	kind MsgKind, scope MsgScope, reqLevel uint8, reqFlags string,
	args ...string) *Cmd {

	command := MkCmd(ext, desc, cmd, handler, kind, scope, args...)
	command.RequireAuth = true
	command.ReqLevel = reqLevel
	command.ReqFlags = reqFlags
	return command
}

// parseArgs parses and sets the arguments for a command.
func (c *Cmd) parseArgs() error {
	nArgs := len(c.Args)
	if nArgs == 0 {
		return nil
	}

	c.args = make([]argument, nArgs)

	var chanArg, required, optional, variadic bool

	for i := 0; i < nArgs; i++ {
		arg := strings.ToLower(c.Args[i])
		if !rgxArgs.MatchString(arg) {
			return fmt.Errorf(errFmtArgumentForm, arg)
		}

		argMeta := &c.args[i]
		argMeta.Original = arg
		argMeta.Name = strings.Trim(c.Args[i], argStripChars)
		for j := 0; j < i; j++ {
			if c.args[j].Name == argMeta.Name {
				return fmt.Errorf(errFmtArgumentDupName, argMeta.Name)
			}
		}

		modifier := arg[0]
		if modifier == '[' {
			modifier = arg[1]
		}
		switch modifier {
		case '#':
			if chanArg {
				return fmt.Errorf(errFmtArgumentDupChan, arg)
			} else if required || optional || variadic {
				return fmt.Errorf(errFmtArgumentOrderChan, arg)
			}
			argMeta.Type = CHANNEL
			chanArg = true
		case '~':
			argMeta.Type = NICK
		case '*':
			argMeta.Type = USER
		}

		switch arg[len(arg)-1] {
		case ']':
			if variadic {
				return fmt.Errorf(errFmtArgumentOrderOpt, arg)
			}
			argMeta.Type |= OPTIONAL
			optional = true
			c.optArgs++
		case '.':
			if variadic {
				return fmt.Errorf(errFmtArgumentDupVargs, arg)
			}
			argMeta.Type |= VARIADIC
			variadic = true
		default:
			if optional {
				return fmt.Errorf(errFmtArgumentOrderReq, arg)
			}
			argMeta.Type |= REQUIRED
			required = true
			c.reqArgs++
		}
	}

	return nil
}
