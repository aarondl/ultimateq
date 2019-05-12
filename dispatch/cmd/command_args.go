package cmd

import (
	"regexp"
	"strings"

	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/irc"
	"github.com/pkg/errors"
)

const (
	errFmtInternal     = "cmd: Internal Error Occurred: %v"
	errFmtArgumentForm = `cmd: Arguments must look like: ` +
		`#name OR [~|*]name OR [[~|*]name] OR [~|*]name... (given: %v)`
	errFmtArgumentDupName = `cmd: Argument names must be unique ` +
		`(given: %v)`
	errFmtArgumentOrderChan = `cmd: The channel argument must come ` +
		`first. (given: %v)`
	errFmtArgumentDupChan = `cmd: Only one #channel argument is ` +
		`allowed (given: %v)`
	errFmtArgumentOrderReq = `cmd: Required arguments must come before ` +
		`all [optional] and varargs... arguments. (given: %v)`
	errFmtArgumentOrderOpt = `cmd: Optional arguments must come before ` +
		`varargs... arguments. (given: %v)`
	errFmtArgumentDupVargs = `cmd: Only one varargs... argument is ` +
		`allowed (given: %v)`
	errMsgStateDisabled = "Error: Cannot use nick or user parameter commands " +
		"when state is disabled."
	errMsgStoreDisabled = "Access Denied: Cannot use authenticated commands, " +
		"nick or user parameters when store is disabled."
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
)

type argType int

// These are for internal use of the command structure to deal with parsing
// and storing argument metadata.
const (
	argTypeRequired argType = 1 << iota
	argTypeOptional
	argTypeVariadic
	argTypeChannel
	argTypeNick
	argTypeUser

	argTypeTypeMask = argTypeRequired | argTypeOptional | argTypeVariadic | argTypeChannel
	argTypeUSERMASK = argTypeNick | argTypeUser

	argStripChars = `#~*[].`
)

func (a argType) has(b argType) bool {
	return a&b == b
}

var (
	// rgxArgs checks a single argument to see if it matches the
	// forms: arg #arg [arg] or arg...
	rgxArgs = regexp.MustCompile(
		`(?i)^(\[[~\*]?[a-z0-9]+\]|[~\*]?[a-z0-9]+(\.\.\.)?|#[a-z0-9]+)$`)
)

// args stores data about each argument after it's parsed.
type commandArgs struct {
	args []argument
	reqs int
	opts int
}

// argument is a type to hold argument information.
type argument struct {
	Original string
	Name     string
	Type     argType
}

// parseArgs parses and sets the arguments for a command.
func (c *Command) parseArgs() (err error) {
	nArgs := len(c.Args)
	if nArgs == 0 {
		return nil
	}

	var cArgs commandArgs
	cArgs.args = make([]argument, nArgs)

	var chanArg, required, optional, variadic bool

	for i := 0; i < nArgs; i++ {
		arg := strings.ToLower(c.Args[i])
		if !rgxArgs.MatchString(arg) {
			return errors.Errorf(errFmtArgumentForm, arg)
		}

		argMeta := &cArgs.args[i]
		argMeta.Original = arg
		argMeta.Name = strings.Trim(c.Args[i], argStripChars)
		for j := 0; j < i; j++ {
			if cArgs.args[j].Name == argMeta.Name {
				return errors.Errorf(errFmtArgumentDupName, argMeta.Name)
			}
		}

		modifier := arg[0]
		if modifier == '[' {
			modifier = arg[1]
		}
		switch modifier {
		case '#':
			if chanArg {
				return errors.Errorf(errFmtArgumentDupChan, arg)
			} else if required || optional || variadic {
				return errors.Errorf(errFmtArgumentOrderChan, arg)
			}
			argMeta.Type = argTypeChannel
			chanArg = true
		case '~':
			argMeta.Type = argTypeNick
		case '*':
			argMeta.Type = argTypeUser
		}

		switch arg[len(arg)-1] {
		case ']':
			if variadic {
				return errors.Errorf(errFmtArgumentOrderOpt, arg)
			}
			argMeta.Type |= argTypeOptional
			optional = true
			cArgs.opts++
		case '.':
			if variadic {
				return errors.Errorf(errFmtArgumentDupVargs, arg)
			}
			argMeta.Type |= argTypeVariadic
			variadic = true
		default:
			if optional {
				return errors.Errorf(errFmtArgumentOrderReq, arg)
			}
			argMeta.Type |= argTypeRequired
			required = true
			cArgs.reqs++
		}
	}

	c.parsedArgs = cArgs
	return nil
}

// ProcessArgs parses all the arguments. It looks up channel and user arguments
// using the state and store, and generally populates the Event struct
// with argument information.
func ProcessArgs(server string, command *Command, channel string,
	isChan bool, msgArgs []string, ev *Event, ircEvent *irc.Event,
	state *data.State, store *data.Store) (err error) {

	ev.Args = make(map[string]string)

	i, j := 0, 0
	for i = 0; i < len(command.Args); i, j = i+1, j+1 {
		arg := command.parsedArgs.args[i]

		switch {
		case arg.Type.has(argTypeChannel):
			if state == nil {
				return errors.New(errMsgStateDisabled)
			}
			var consumed bool
			if consumed, err = parseChanArg(command, ev, state, j,
				msgArgs, channel, isChan); err != nil {
				return
			} else if !consumed {
				j--
			}
		case arg.Type.has(argTypeRequired):
			if j >= len(msgArgs) {
				nReq := command.parsedArgs.reqs
				if command.parsedArgs.args[0].Type.has(argTypeChannel) && isChan {
					nReq--
				}
				return errors.Errorf(errFmtNArguments, errAtLeast, nReq,
					strings.Join(command.Args, " "))
			}
			ev.Args[arg.Name] = msgArgs[j]
		case arg.Type.has(argTypeOptional):
			if j >= len(msgArgs) {
				return
			}
			ev.Args[arg.Name] = msgArgs[j]
		case arg.Type.has(argTypeVariadic):
			if j >= len(msgArgs) {
				return
			}
			ev.Args[arg.Name] = strings.Join(msgArgs[j:], " ")
		}

		if arg.Type.has(argTypeNick) || arg.Type.has(argTypeUser) {
			if arg.Type.has(argTypeVariadic) {
				err = parseUserArg(ev, state, store, server, arg.Name,
					arg.Type, msgArgs[j:]...)
			} else {
				err = parseUserArg(ev, state, store, server, arg.Name,
					arg.Type, msgArgs[j])
			}
			if err != nil {
				return
			}
		}

		if arg.Type.has(argTypeVariadic) {
			j = len(msgArgs)
			break
		}
	}

	if j < len(msgArgs) {
		if j == 0 {
			return errors.New(errMsgUnexpectedArgument)
		}
		return errors.Errorf(errFmtNArguments, errAtMost,
			command.parsedArgs.reqs+command.parsedArgs.opts,
			strings.Join(command.Args, " "))
	}
	return nil
}

// parseChanArg checks the argument provided and ensures it's a valid situation
// for the channel arg to be in (isChan & validChan) | (isChan & missing) |
// (!isChan & validChan)
func parseChanArg(command *Command, ev *Event,
	state *data.State,
	index int, msgArgs []string, channel string, isChan bool) (bool, error) {

	var isFirstChan bool
	if index < len(msgArgs) {
		isFirstChan = ev.Event.NetworkInfo.IsChannel(msgArgs[index])
	} else if !isChan {
		return false, errors.Errorf(errFmtNArguments, errAtLeast,
			command.parsedArgs.reqs, strings.Join(command.Args, " "))
	}

	name := command.parsedArgs.args[index].Name
	if isChan {
		if !isFirstChan {
			ev.Args[name] = channel
			if ch, ok := state.Channel(channel); ok {
				ev.Channel = &ch
				ev.TargetChannel = &ch
			}
			return false, nil
		}
		ev.Args[name] = msgArgs[index]
		if ch, ok := state.Channel(msgArgs[index]); ok {
			ev.TargetChannel = &ch
		}
		return true, nil
	} else if isFirstChan {
		ev.Args[name] = msgArgs[index]
		if ch, ok := state.Channel(msgArgs[index]); ok {
			ev.TargetChannel = &ch
		}
		return true, nil
	}

	return false, errors.Errorf(errFmtArgumentNotChannel, msgArgs[index])
}

// parseUserArg takes user arguments and assigns them to the correct structures
// in a command data struct.
func parseUserArg(ev *Event, state *data.State,
	store *data.Store, srv, name string, t argType, users ...string) error {

	vargs := t.has(argTypeVariadic)
	nUsers := len(users)

	var access *data.StoredUser
	var user *data.User
	var err error

	addData := func(index int) {
		if access != nil {
			if vargs {
				ev.TargetVarStoredUsers[index] = access
			} else {
				ev.TargetStoredUsers[name] = access
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

	switch t & argTypeUSERMASK {
	case argTypeUser:
		if vargs {
			ev.TargetVarStoredUsers = make([]*data.StoredUser, nUsers)
		} else {
			if ev.TargetStoredUsers == nil {
				ev.TargetStoredUsers = make(map[string]*data.StoredUser)
			}
		}
		for i, u := range users {
			access, user, err = findAccessByUser(state, store, ev, srv, u)
			if err != nil {
				return err
			}
			addData(i)
		}
	case argTypeNick:
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

// findUserByNick finds a user by their nickname. An error is returned if
// they were not found.
func findUserByNick(state *data.State, ev *Event, nick string) (*data.User, error) {
	if state == nil {
		return nil, errors.New(errMsgStateDisabled)
	}

	if user, ok := state.User(nick); ok {
		return &user, nil
	}

	return nil, errors.Errorf(errFmtUserNotFound, nick)
}

// findAccessByUser locates a user's access based on their nick or
// username. To look up by username instead of nick use the * prefix before the
// username in the string. The user parameter is returned when a nickname lookup
// is done. An error occurs if the user is not found, the user is not authed,
// the username is not registered, etc.
func findAccessByUser(state *data.State, store *data.Store, ev *Event,
	server, nickOrUser string) (
	access *data.StoredUser, user *data.User, err error) {
	if store == nil {
		err = errors.New(errMsgStoreDisabled)
		return access, user, err
	}

	switch nickOrUser[0] {
	case '*':
		if len(nickOrUser) == 1 {
			err = errors.New(errMsgMissingUsername)
			return access, user, err
		}
		uname := nickOrUser[1:]
		access, err = store.FindUser(uname)
		if access == nil {
			err = errors.Errorf(errFmtUserNotRegistered, uname)
			return access, user, err
		}
	default:
		if state == nil {
			err = errors.New(errMsgStateDisabled)
			return access, user, err
		}

		u, ok := state.User(nickOrUser)
		if !ok {
			err = errors.Errorf(errFmtUserNotFound, nickOrUser)
			return access, user, err
		}
		user = &u

		access = store.AuthedUser(server, user.Host.String())
		if access == nil {
			err = errors.Errorf(errFmtUserNotAuthed, nickOrUser)
			return
		}
	}

	if err != nil {
		err = errors.Errorf(errFmtInternal, err)
	}
	return access, user, err
}
