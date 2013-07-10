package commander

import (
	"fmt"
	"strings"
)

// Command holds all the information about a command.
type Command struct {
	// The name of the command.
	Cmd string
	// Extension is the name of the extension registering this command.
	Extension string
	// Description is a description of the command's function.
	Description string
	// Msgtype is the type of messages this command reacts to, may be the
	// any of the constants: PRIVMSG, NOTICE or ALL.
	Msgtype int
	// Msgscope is the scope of the messages this command reacts to. May be
	// any of the constants: PRIVATE, PUBLIC or ALL.
	Msgscope int
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
	Handler   CommandHandler
	argnames  []string
	argAttrib []int
	chanArg   bool
	reqArgs   int
	optArgs   int
}

// MkCmd is a helper method to easily create a Command. See the documentation
// for Command on what each parameter is.
func MkCmd(ext, desc, cmd string, handler CommandHandler, msgtype, msgscope int,
	args ...string) *Command {
	return &Command{
		Cmd:         cmd,
		Extension:   ext,
		Description: desc,
		Handler:     handler,
		Msgtype:     msgtype,
		Msgscope:    msgscope,
		Args:        args,
	}
}

// MkAuthCmd is a helper method to easily create an authenticated Command. See
// the documentation on Command for what each parameter is.
func MkAuthCmd(ext, desc, cmd string, handler CommandHandler,
	msgtype, msgscope int, reqLevel uint8, reqFlags string,
	args ...string) *Command {

	command := MkCmd(ext, desc, cmd, handler, msgtype, msgscope, args...)
	command.RequireAuth = true
	command.ReqLevel = reqLevel
	command.ReqFlags = reqFlags
	return command
}

// setArgs parses and sets the arguments for a command.
func (c *Command) parseArgs() error {
	nArgs := len(c.Args)
	if nArgs == 0 {
		return nil
	}

	c.argAttrib = make([]int, nArgs)

	for i := 0; i < nArgs; i++ {
		arg := strings.ToLower(c.Args[i])
		if !commandArgRegexp.MatchString(arg) {
			return fmt.Errorf(errFmtArgumentForm, arg)
		}

		modifier := arg[0]
		if modifier == '[' {
			modifier = arg[1]
		}
		switch modifier {
		case '#':
			if c.chanArg {
				return fmt.Errorf(errFmtArgumentDupChan, arg)
			} else if c.reqArgs != 0 || c.optArgs != 0 {
				return fmt.Errorf(errFmtArgumentOrderChan, arg)
			}
			c.chanArg = true
			continue
		case '~':
			c.argAttrib[i] = attribUser
		case '*':
			c.argAttrib[i] = attribAuthed
		}

		switch arg[len(arg)-1] {
		case ']':
			if c.optArgs == varArgs {
				return fmt.Errorf(errFmtArgumentOrderOpt, arg)
			}
			c.optArgs++
		case '.':
			if c.optArgs == varArgs {
				return fmt.Errorf(errFmtArgumentDupVargs, arg)
			}
			c.optArgs = varArgs
		default:
			if c.optArgs != 0 {
				return fmt.Errorf(errFmtArgumentOrderReq, arg)
			}
			c.reqArgs++
		}
	}

	c.argnames = make([]string, nArgs)
	for i := 0; i < len(c.Args); i++ {
		c.argnames[i] = strings.Trim(c.Args[i], argNamesStripChars)
		for j := 0; j < i; j++ {
			if c.argnames[j] == c.argnames[i] {
				return fmt.Errorf(errFmtArgumentDupName, c.argnames[i])
			}
		}
	}
	return nil
}
