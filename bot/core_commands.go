package bot

import (
	"errors"
	"fmt"
	"github.com/aarondl/ultimateq/data"
	cmds "github.com/aarondl/ultimateq/dispatch/commander"
	"github.com/aarondl/ultimateq/irc"
	"log"
)

const (
	register = `register`
	auth     = `auth`
	logout   = `logout`
	adduser  = `adduser`
	deluser  = `rmuser`
	passwd   = `passwd`
	addhost  = `addhost`
	rmhost   = `rmhost`

	errFmtCommandRegister = `bot: A core command registration failed: %v`
	errMsgInternal        = `There was an internal error, try again later.`
	errFmtInternal        = `commander: Error processing command %v (%v)`

	errMsgAuthed    = `You are already authenticated.`
	errMsgNotAuthed = `You are not authenticated.`
	errFmtUsername  = `The username [%v] is already registered.`

	registerSuccess = `Registered [%v] successfully. You have been ` +
		`automatically authenticated.`
	registerSuccessFirst = `You registered [%v] successfully, ` +
		`as the first user, you have been given all permissions and ` +
		`privileges as well as been automatically authenticated. \o/`
	authSuccess   = `Successfully authenticated [%v].`
	logoutSuccess = `Successfully logged out.`
)

// coreCommands is the bot's command handling struct. The bot itself uses
// the cmds to implement user management.
type coreCommands struct {
	b *Bot
}

// CreateCoreCommands initializes the core commands and registers them with the
// bot.
func CreateCoreCommands(b *Bot) (*coreCommands, error) {
	makeError := func(originalError error) error {
		return errors.New(fmt.Sprintf(errFmtCommandRegister, originalError))
	}

	c := &coreCommands{b}

	var err error
	err = b.RegisterCommand(register, c, cmds.PRIVMSG, cmds.PRIVATE,
		"password", "[username]")
	if err != nil {
		return nil, makeError(err)
	}
	err = b.RegisterCommand(auth, c, cmds.PRIVMSG, cmds.PRIVATE,
		"password", "[username]")
	if err != nil {
		return nil, makeError(err)
	}
	err = b.RegisterCommand(logout, c, cmds.PRIVMSG, cmds.PRIVATE)
	if err != nil {
		return nil, makeError(err)
	}
	err = b.RegisterAuthedCommand(adduser, c, cmds.PRIVMSG, cmds.PRIVATE,
		0, "A")
	if err != nil {
		return nil, makeError(err)
	}
	err = b.RegisterAuthedCommand(deluser, c, cmds.PRIVMSG, cmds.PRIVATE,
		0, "A")
	if err != nil {
		return nil, makeError(err)
	}
	err = b.RegisterAuthedCommand(passwd, c, cmds.PRIVMSG, cmds.PRIVATE,
		0, "A")
	if err != nil {
		return nil, makeError(err)
	}
	err = b.RegisterAuthedCommand(addhost, c, cmds.PRIVMSG, cmds.PRIVATE,
		0, "A")
	if err != nil {
		return nil, makeError(err)
	}
	err = b.RegisterAuthedCommand(rmhost, c, cmds.PRIVMSG, cmds.PRIVATE,
		0, "A")
	if err != nil {
		return nil, makeError(err)
	}

	return &coreCommands{b}, nil
}

// unregisterCoreCommands unregisters all core commands. Made for testing.
func (c *coreCommands) unregisterCoreCommands() {
	c.b.UnregisterCommand(register)
	c.b.UnregisterCommand(auth)
	c.b.UnregisterCommand(logout)
	c.b.UnregisterCommand(adduser)
	c.b.UnregisterCommand(deluser)
	c.b.UnregisterCommand(passwd)
	c.b.UnregisterCommand(addhost)
	c.b.UnregisterCommand(rmhost)
}

// Command is responsible for parsing all of the commands.
func (c *coreCommands) Command(cmd string, m *irc.Message, d *data.DataEndpoint,
	cd *cmds.CommandData) (internal error) {

	var external error

	log.Printf("bot: Core command executed (%v)", cmd)

	/*defer func() {
		if r := recover(); r != nil {
			log.Println("FATAL:", r)
			log.Printf("%+v", d)
			log.Printf("%+v", cd)
		}
	}()*/

	switch cmd {
	case register:
		internal, external = c.register(d, cd)
	case auth:
		internal, external = c.auth(d, cd)
	case logout:
		internal, external = c.logout(d, cd)
	case adduser:
	case deluser:
	case passwd:
	case addhost:
	case rmhost:
	}

	if internal != nil {
		log.Printf("bot: Core command (%v) error: %v", cmd, internal)
	}

	return external
}

// register register's a user to the bot with an optional user name.
func (c *coreCommands) register(d *data.DataEndpoint,
	cd *cmds.CommandData) (internal, external error) {

	var access *data.UserAccess

	pwd := cd.GetArg("password")
	uname := cd.GetArg("username")
	if len(uname) == 0 {
		uname = cd.User.GetUsername()
	}

	access = cd.GetAuthedUser(d.GetKey(), cd.User.GetFullhost())
	if access != nil {
		return nil, fmt.Errorf(errMsgAuthed)
	}

	access, internal = cd.FindUser(uname)
	if internal != nil {
		return
	}
	if access != nil {
		return nil, fmt.Errorf(errFmtUsername, uname)
	}

	isFirst, internal := cd.IsFirst()
	if internal != nil {
		return
	}

	access, internal = data.CreateUserAccess(uname, pwd)
	if internal != nil {
		return
	}
	if isFirst {
		access.Global = &data.Access{^uint8(0), ^uint64(0)}
	}

	internal = cd.AddUser(access)
	if internal != nil {
		return
	}

	_, internal = cd.AuthUser(d.GetKey(), cd.User.GetFullhost(), uname, pwd)
	if internal != nil {
		return
	}

	nick := cd.User.GetNick()
	if isFirst {
		d.Notice(nick, fmt.Sprintf(registerSuccessFirst, uname))
	} else {
		d.Notice(nick, fmt.Sprintf(registerSuccess, uname))
	}

	return
}

// auth authenticates a user.
func (c *coreCommands) auth(d *data.DataEndpoint, cd *cmds.CommandData) (
	internal, external error) {

	pwd := cd.GetArg("password")
	uname := cd.GetArg("username")
	if len(uname) == 0 {
		uname = cd.User.GetUsername()
	}

	access := cd.GetAuthedUser(d.GetKey(), cd.User.GetFullhost())
	if access != nil {
		external = errors.New(errMsgAuthed)
		return
	}

	_, err := cd.AuthUser(d.GetKey(), cd.User.GetFullhost(), uname, pwd)
	if err != nil {
		if authErr, ok := err.(data.AuthError); ok {
			external = authErr
		} else {
			internal = authErr
		}
		return
	}

	d.Notice(cd.User.GetNick(), fmt.Sprintf(authSuccess, uname))
	return
}

// logout logs out a user.
func (c *coreCommands) logout(d *data.DataEndpoint, cd *cmds.CommandData) (
	internal, external error) {

	access := cd.GetAuthedUser(d.GetKey(), cd.User.GetFullhost())
	if access == nil {
		external = errors.New(errMsgNotAuthed)
	} else {
		cd.Logout(d.GetKey(), cd.User.GetFullhost())
		d.Notice(cd.User.GetNick(), logoutSuccess)
	}

	return
}
