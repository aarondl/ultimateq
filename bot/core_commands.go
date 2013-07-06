package bot

import (
	"errors"
	"fmt"
	"github.com/aarondl/ultimateq/data"
	cmds "github.com/aarondl/ultimateq/dispatch/commander"
	"github.com/aarondl/ultimateq/irc"
	"log"
	"strings"
)

const (
	register = `register`
	auth     = `auth`
	logout   = `logout`
	access   = `access`
	deluser  = `deluser`
	delme    = `delme`
	passwd   = `passwd`
	masks    = `masks`
	addmask  = `addmask`
	delmask  = `delmask`

	errFmtCommandRegister = `bot: A core command registration failed: %v`
	errMsgInternal        = `There was an internal error, try again later.`
	errFmtInternal        = `commander: Error processing command %v (%v)`

	errMsgAuthed        = `You are already authenticated.`
	errFmtUserNotFound  = `The user [%v] could not be found.`
	errFmtUserNotAuthed = `The user [%v] is not authenticated.`

	registerSuccess = `Registered [%v] successfully. You have been ` +
		`automatically authenticated.`
	registerSuccessFirst = `Registered [%v] successfully. ` +
		`As the first user, you have been given all permissions and ` +
		`privileges as well as being automatically authenticated. \o/`
	registerFailure = `The username [%v] is already registered.`
	authSuccess     = `Successfully authenticated [%v].`
	logoutSuccess   = `Successfully logged out.`
	deluserSuccess  = `Removed user [%v].`
	deluserFailure  = `User [%v] does not exist.`
	delmeSuccess    = `Removed your user account [%v].`
	delmeFailure    = `User account could not be removed.`
	passwdSuccess   = `Successfully updated your password.`
	passwdFailure   = `Old password did not match, try again.`
	masksFailure    = "No masks set."
	addmaskSuccess  = `Host [%v] added successfully.`
	addmaskFailure  = `Host [%v] already exists.`
	delmaskSuccess  = `Host [%v] removed successfully.`
	delmaskFailure  = `Host [%v] not found.`
)

type argv []string

var commands = []struct {
	Name   string
	Authed bool
	Level  uint8
	Flags  string
	Args   []string
}{
	{register, false, 0, ``, argv{`password`, `[username]`}},
	{auth, false, 0, ``, argv{`password`, `[username]`}},
	{logout, true, 0, ``, nil},
	{access, true, 0, ``, argv{`[user]`}},
	{deluser, true, 0, `A`, argv{`user`}},
	{delme, true, 0, ``, nil},
	{passwd, true, 0, ``, argv{`oldpassword`, `newpassword`}},
	{masks, true, 0, ``, nil},
	{addmask, true, 0, ``, argv{`mask`}},
	{delmask, true, 0, ``, argv{`mask`}},
}

// coreCommands is the bot's command handling struct. The bot itself uses
// the cmds to implement user management.
type coreCommands struct {
	b *Bot
}

// CreateCoreCommands initializes the core commands and registers them with the
// bot.
func CreateCoreCommands(b *Bot) (*coreCommands, error) {
	c := &coreCommands{b}

	var err error
	for _, cmd := range commands {
		if cmd.Authed {
			err = b.RegisterAuthedCommand(cmd.Name, c, cmds.PRIVMSG,
				cmds.PRIVATE, cmd.Level, cmd.Flags, cmd.Args...)
		} else {
			err = b.RegisterCommand(cmd.Name, c, cmds.PRIVMSG, cmds.PRIVATE,
				cmd.Args...)
		}
		if err != nil {
			return nil, fmt.Errorf(errFmtCommandRegister, err)
		}
	}

	return &coreCommands{b}, nil
}

// unregisterCoreCommands unregisters all core commands. Made for testing.
func (c *coreCommands) unregisterCoreCommands() {
	for _, cmd := range commands {
		c.b.UnregisterCommand(cmd.Name)
	}
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
	case access:
		internal, external = c.access(d, cd)
	case deluser:
		internal, external = c.deluser(d, cd)
	case delme:
		internal, external = c.delme(d, cd)
	case passwd:
		internal, external = c.passwd(d, cd)
	case masks:
		internal, external = c.masks(d, cd)
	case addmask:
		internal, external = c.addmask(d, cd)
	case delmask:
		internal, external = c.delmask(d, cd)
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

	access = cd.UserAccess
	if access == nil {
		access = cd.GetAuthedUser(d.GetKey(), cd.User.GetFullhost())
	}
	if access != nil {
		return nil, fmt.Errorf(errMsgAuthed)
	}

	access, internal = cd.FindUser(uname)
	if internal != nil {
		return
	}
	if access != nil {
		return nil, fmt.Errorf(registerFailure, uname)
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

	var access *data.UserAccess
	pwd := cd.GetArg("password")
	uname := cd.GetArg("username")
	if len(uname) == 0 {
		uname = cd.User.GetUsername()
	}

	access = cd.UserAccess
	if access == nil {
		access = cd.GetAuthedUser(d.GetKey(), cd.User.GetFullhost())
	}
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

	cd.Logout(d.GetKey(), cd.User.GetFullhost())
	d.Notice(cd.User.GetNick(), logoutSuccess)

	return
}

// access outputs the access for the current user.
func (c *coreCommands) access(d *data.DataEndpoint, cd *cmds.CommandData) (
	internal, external error) {

	return
}

// deluser deletes a user
func (c *coreCommands) deluser(d *data.DataEndpoint, cd *cmds.CommandData) (
	internal, external error) {

	uname := cd.GetArg("user")

	uname, external = getUsername(uname, d.GetKey(), cd)
	if external != nil {
		return
	}

	cd.LogoutByUsername(uname)

	var removed bool
	removed, internal = cd.RemoveUser(uname)
	if internal != nil {
		return
	}

	if removed {
		d.Notice(cd.User.GetNick(), fmt.Sprintf(deluserSuccess, uname))
	} else {
		d.Notice(cd.User.GetNick(), fmt.Sprintf(deluserFailure, uname))
	}

	return
}

// delme deletes self
func (c *coreCommands) delme(d *data.DataEndpoint, cd *cmds.CommandData) (
	internal, external error) {

	removed := false
	access := cd.UserAccess
	cd.Logout(d.GetKey(), cd.User.GetFullhost())
	removed, internal = cd.RemoveUser(access.Username)
	if internal != nil {
		return
	}
	if !removed {
		internal = errors.New(delmeFailure)
		return
	}
	d.Notice(cd.User.GetNick(), fmt.Sprintf(delmeSuccess, access.Username))
	return
}

// passwd changes a user's password
func (c *coreCommands) passwd(d *data.DataEndpoint, cd *cmds.CommandData) (
	internal, external error) {

	access := cd.UserAccess
	oldpasswd := cd.GetArg("oldpassword")
	newpasswd := cd.GetArg("newpassword")

	if access.VerifyPassword(oldpasswd) {
		access.SetPassword(newpasswd)
		cd.AddUser(access)
		d.Notice(cd.User.GetNick(), passwdSuccess)
	} else {
		d.Notice(cd.User.GetNick(), passwdFailure)
	}

	return
}

// masks outputs the masks of the user.
func (c *coreCommands) masks(d *data.DataEndpoint, cd *cmds.CommandData) (
	internal, external error) {

	return
}

// addmask adds a mask to a user.
func (c *coreCommands) addmask(d *data.DataEndpoint, cd *cmds.CommandData) (
	internal, external error) {

	mask := cd.GetArg("mask")

	access := cd.UserAccess
	if access.AddMasks(irc.WildMask(mask)) {
		cd.AddUser(access)
		d.Notice(cd.User.GetNick(), fmt.Sprintf(addmaskSuccess, mask))
	} else {
		d.Notice(cd.User.GetNick(), fmt.Sprintf(addmaskFailure, mask))
	}

	return
}

// delmask deletes a mask from a user.
func (c *coreCommands) delmask(d *data.DataEndpoint, cd *cmds.CommandData) (
	internal, external error) {

	mask := cd.GetArg("mask")

	access := cd.UserAccess
	if access.DelMasks(irc.WildMask(mask)) {
		cd.AddUser(access)
		d.Notice(cd.User.GetNick(), fmt.Sprintf(delmaskSuccess, mask))
	} else {
		d.Notice(cd.User.GetNick(), fmt.Sprintf(delmaskFailure, mask))
	}

	return
}

// getUsername looks up a username. If user is in the form *user, the user part
// is assumed to be a username, it's trimmed and returned. Otherwise, the user
// is assumed to be a nickname. The error returned by this function should be
// sent to the user.
func getUsername(user, key string, cd *cmds.CommandData) (string, error) {
	if strings.HasPrefix(user, "*") {
		user = user[1:]
		if len(user) == 0 {
			return "", fmt.Errorf(errFmtUserNotFound, user)
		}
		return user, nil
	} else {
		if stateUser := cd.GetUser(user); stateUser != nil {
			host := stateUser.GetFullhost()
			access := cd.GetAuthedUser(key, host)
			if access == nil {
				return "", fmt.Errorf(errFmtUserNotAuthed, user)
			}
			return access.Username, nil
		} else {
			return "", fmt.Errorf(errFmtUserNotFound, user)
		}
	}
}

// getUser looks up a user based on nickname. The error returned by this
// function should be sent to the user.
func getUser(user, key string, cd *cmds.CommandData) (*data.UserAccess, error) {
	if stateUser := cd.GetUser(user); stateUser != nil {
		host := stateUser.GetFullhost()
		access := cd.GetAuthedUser(key, host)
		if access == nil {
			return nil, fmt.Errorf(errFmtUserNotAuthed, user)
		}
		return access, nil
	} else {
		return nil, fmt.Errorf(errFmtUserNotFound, user)
	}
}
