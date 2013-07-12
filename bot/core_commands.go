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

	extension = `core`

	errFmtRegister = `bot: A core command registration failed: %v`
	errMsgInternal = `There was an internal error, try again later.`
	errFmtInternal = `commander: Error processing command %v (%v)`
	errFmtExpired  = `commander: Data expired between locks. ` +
		`Could not find user [%v]`

	errMsgAuthed        = `You are already authenticated.`
	errFmtUserNotFound  = `The user [%v] could not be found.`
	errFmtUserNotAuthed = `The user [%v] is not authenticated.`

	registerDesc    = `Registers an account.`
	registerSuccess = `Registered [%v] successfully. You have been ` +
		`automatically authenticated.`
	registerSuccessFirst = `Registered [%v] successfully. ` +
		`As the first user, you have been given all permissions and ` +
		`privileges as well as being automatically authenticated. \o/`
	registerFailure = `The username [%v] is already registered.`
	authDesc        = `Authenticate a user to an account.`
	authSuccess     = `Successfully authenticated [%v].`
	logoutDesc      = `Logs the user out of the account.`
	logoutSuccess   = `Successfully logged out.`
	accessDesc      = `Access retrieves the access for the user.`
	accessSuccess   = `Access for [%v]: %v`
	deluserDesc     = `Deletes a user account from the bot.`
	deluserSuccess  = `Removed user [%v].`
	deluserFailure  = `User [%v] does not exist.`
	delmeDesc       = `Deletes the current user's account.`
	delmeSuccess    = `Removed your user account [%v].`
	delmeFailure    = `User account could not be removed.`
	passwdDesc      = `Change the current user's account password.`
	passwdSuccess   = `Successfully updated your password.`
	passwdFailure   = `Old password did not match the current password.`
	masksDesc       = `Retrieves the current user's mask list.`
	masksSuccess    = `Masks: %v`
	masksFailure    = `No masks set.`
	addmaskDesc     = `Adds a mask to the current user.`
	addmaskSuccess  = `Host [%v] added successfully.`
	addmaskFailure  = `Host [%v] already exists.`
	delmaskDesc     = `Deletes a mask from the current user.`
	delmaskSuccess  = `Host [%v] removed successfully.`
	delmaskFailure  = `Host [%v] not found.`
)

type argv []string

var commands = []struct {
	Name   string
	Desc   string
	Authed bool
	Public bool
	Level  uint8
	Flags  string
	Args   []string
}{
	{register, registerDesc, false, false, 0, ``,
		argv{`password`, `[username]`}},
	{auth, authDesc, false, false, 0, ``, argv{`password`, `[username]`}},
	{logout, logoutDesc, true, false, 0, ``, nil},
	{access, accessDesc, true, true, 0, ``, argv{`[*user]`}},
	{deluser, deluserDesc, true, false, 0, `A`, argv{`*user`}},
	{delme, delmeDesc, true, false, 0, ``, nil},
	{passwd, passwdDesc, true, false, 0, ``,
		argv{`oldpassword`, `newpassword`}},
	{masks, masksDesc, true, false, 0, ``, nil},
	{addmask, addmaskDesc, true, false, 0, ``, argv{`mask`}},
	{delmask, delmaskDesc, true, false, 0, ``, argv{`mask`}},
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
	for _, cmd := range commands {
		privacy := cmds.PRIVATE
		if cmd.Public {
			privacy = cmds.ALL
		}
		err := b.RegisterCommand(&cmds.Command{
			Cmd:         cmd.Name,
			Extension:   extension,
			Description: cmd.Desc,
			Handler:     c,
			Msgtype:     cmds.PRIVMSG,
			Msgscope:    privacy,
			Args:        cmd.Args,
			RequireAuth: cmd.Authed,
			ReqLevel:    cmd.Level,
			ReqFlags:    cmd.Flags,
		})
		if err != nil {
			return nil, fmt.Errorf(errFmtRegister, err)
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

	access, internal = data.CreateUserAccess(uname, pwd)
	if internal != nil {
		return
	}

	host, nick := cd.User.GetFullhost(), cd.User.GetNick()

	cd.Close()
	c.b.protectStore.Lock()
	defer c.b.protectStore.Unlock()
	store := c.b.store

	isFirst, internal := store.IsFirst()
	if internal != nil {
		return
	}
	if isFirst {
		access.Global = &data.Access{^uint8(0), ^uint64(0)}
	}

	internal = store.AddUser(access)
	if internal != nil {
		return
	}

	_, internal = store.AuthUser(d.GetKey(), host, uname, pwd)
	if internal != nil {
		return
	}

	if isFirst {
		d.Noticef(nick, registerSuccessFirst, uname)
	} else {
		d.Noticef(nick, registerSuccess, uname)
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

	host, nick := cd.User.GetFullhost(), cd.User.GetNick()

	access = cd.UserAccess
	if access == nil {
		access = cd.GetAuthedUser(d.GetKey(), host)
	}
	if access != nil {
		external = errors.New(errMsgAuthed)
		return
	}

	cd.Close()
	c.b.protectStore.Lock()
	defer c.b.protectStore.Unlock()
	store := c.b.store

	_, err := store.AuthUser(d.GetKey(), host, uname, pwd)
	if err != nil {
		if authErr, ok := err.(data.AuthError); ok {
			external = authErr
		} else {
			internal = err
		}
		return
	}

	d.Noticef(nick, authSuccess, uname)
	return
}

// logout logs out a user.
func (c *coreCommands) logout(d *data.DataEndpoint, cd *cmds.CommandData) (
	internal, external error) {

	host, nick := cd.User.GetFullhost(), cd.User.GetNick()
	cd.Close()
	c.b.protectStore.Lock()
	defer c.b.protectStore.Unlock()
	store := c.b.store

	store.Logout(d.GetKey(), host)
	d.Notice(nick, logoutSuccess)

	return
}

// access outputs the access for the user.
func (c *coreCommands) access(d *data.DataEndpoint, cd *cmds.CommandData) (
	internal, external error) {

	access := cd.TargetUserAccess["user"]
	if access == nil {
		access = cd.UserAccess
	}

	ch := ""
	if cd.Channel != nil {
		ch = cd.Channel.GetName()
	}
	d.Noticef(cd.User.GetNick(), accessSuccess,
		access.Username, access.String(d.GetKey(), ch))

	return
}

// deluser deletes a user
func (c *coreCommands) deluser(d *data.DataEndpoint, cd *cmds.CommandData) (
	internal, external error) {

	param := cd.GetArg("user")
	uname := cd.TargetUserAccess["user"].Username

	nick := cd.User.GetNick()
	cd.Close()
	c.b.protectStore.Lock()
	defer c.b.protectStore.Unlock()
	store := c.b.store

	store.LogoutByUsername(uname)

	var removed bool
	removed, internal = store.RemoveUser(uname)
	if internal != nil {
		return
	}

	if removed {
		d.Noticef(nick, deluserSuccess, param)
	} else {
		d.Noticef(nick, deluserFailure, param)
	}

	return
}

// delme deletes self
func (c *coreCommands) delme(d *data.DataEndpoint, cd *cmds.CommandData) (
	internal, external error) {

	host, nick := cd.User.GetFullhost(), cd.User.GetNick()
	uname := cd.UserAccess.Username
	cd.Close()
	c.b.protectStore.Lock()
	defer c.b.protectStore.Unlock()
	store := c.b.store

	removed := false
	store.Logout(d.GetKey(), host)
	removed, internal = store.RemoveUser(uname)
	if internal != nil {
		return
	}
	if !removed {
		internal = errors.New(delmeFailure)
		return
	}
	d.Noticef(nick, delmeSuccess, uname)
	return
}

// passwd changes a user's password
func (c *coreCommands) passwd(d *data.DataEndpoint, cd *cmds.CommandData) (
	internal, external error) {

	oldpasswd := cd.GetArg("oldpassword")
	newpasswd := cd.GetArg("newpassword")
	nick := cd.User.GetNick()
	uname := cd.UserAccess.Username
	if !cd.UserAccess.VerifyPassword(oldpasswd) {
		d.Notice(nick, passwdFailure)
		return
	}

	cd.Close()
	c.b.protectStore.Lock()
	defer c.b.protectStore.Unlock()
	store := c.b.store

	var access *data.UserAccess
	access, internal = store.FindUser(uname)
	if internal != nil {
		return
	}
	if access == nil {
		internal = fmt.Errorf(errFmtExpired, uname)
		return
	}
	access.SetPassword(newpasswd)
	store.AddUser(access)
	d.Notice(nick, passwdSuccess)

	return
}

// masks outputs the masks of the user.
func (c *coreCommands) masks(d *data.DataEndpoint, cd *cmds.CommandData) (
	internal, external error) {

	access := cd.UserAccess
	if len(access.Masks) > 0 {
		d.Noticef(cd.User.GetNick(), masksSuccess,
			strings.Join(access.Masks, " "))
	} else {
		d.Notice(cd.User.GetNick(), masksFailure)
	}

	return
}

// addmask adds a mask to a user.
func (c *coreCommands) addmask(d *data.DataEndpoint, cd *cmds.CommandData) (
	internal, external error) {

	mask := cd.GetArg("mask")
	nick := cd.User.GetNick()
	uname := cd.UserAccess.Username

	cd.Close()
	c.b.protectStore.Lock()
	defer c.b.protectStore.Unlock()
	store := c.b.store

	var access *data.UserAccess
	access, internal = store.FindUser(uname)
	if internal != nil {
		return
	}
	if access == nil {
		internal = fmt.Errorf(errFmtExpired, uname)
		return
	}

	if access.AddMasks(mask) {
		store.AddUser(access)
		d.Noticef(nick, addmaskSuccess, mask)
	} else {
		d.Noticef(nick, addmaskFailure, mask)
	}

	return
}

// delmask deletes a mask from a user.
func (c *coreCommands) delmask(d *data.DataEndpoint, cd *cmds.CommandData) (
	internal, external error) {

	mask := cd.GetArg("mask")
	nick := cd.User.GetNick()
	uname := cd.UserAccess.Username

	cd.Close()
	c.b.protectStore.Lock()
	defer c.b.protectStore.Unlock()
	store := c.b.store

	var access *data.UserAccess
	access, internal = store.FindUser(uname)
	if internal != nil {
		return
	}
	if access == nil {
		internal = fmt.Errorf(errFmtExpired, uname)
		return
	}

	if access.DelMasks(mask) {
		store.AddUser(access)
		d.Noticef(nick, delmaskSuccess, mask)
	} else {
		d.Noticef(nick, delmaskFailure, mask)
	}

	return
}
