package bot

import (
	"errors"
	"fmt"
	"github.com/aarondl/ultimateq/data"
	cmds "github.com/aarondl/ultimateq/dispatch/commander"
	"github.com/aarondl/ultimateq/irc"
	"log"
	"regexp"
	"strconv"
	"strings"
)

var rgxFlags = regexp.MustCompile(`[A-Za-z]+`)

const (
	extension = `core`
	register  = `register`
	auth      = `auth`
	logout    = `logout`
	access    = `access`
	deluser   = `deluser`
	delme     = `delme`
	passwd    = `passwd`
	masks     = `masks`
	addmask   = `addmask`
	delmask   = `delmask`

	resetpasswd = `setpasswd`

	ggive      = `ggive`
	sgive      = `sgive`
	give       = `give`
	gtake      = `gtake`
	stake      = `stake`
	take       = `take`
	takeAllArg = `all`

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
	logoutDesc      = `Logs the current user out of the account. Admins can ` +
		`add a user param to log that user out.`
	logoutSuccess  = `Successfully logged out.`
	accessDesc     = `Access retrieves the access for the user.`
	accessSuccess  = `Access for [%v]: %v`
	deluserDesc    = `Deletes a user account from the bot.`
	deluserSuccess = `Removed user [%v].`
	deluserFailure = `User [%v] does not exist.`
	delmeDesc      = `Deletes the current user's account.`
	delmeSuccess   = `Removed your user account [%v].`
	delmeFailure   = `User account could not be removed.`
	passwdDesc     = `Change the current user's account password.`
	passwdSuccess  = `Successfully updated password.`
	passwdFailure  = `Old password did not match the current password.`
	masksDesc      = `Retrieves the current user's mask list. Admins can add` +
		` a user param to see that user's masks.`
	masksSuccess = `Masks: %v`
	masksFailure = `No masks set.`
	addmaskDesc  = `Adds a mask to the current user. Admins can add a user` +
		` param to add a mask to that user.`
	addmaskSuccess = `Host [%v] added successfully.`
	addmaskFailure = `Host [%v] already exists.`
	delmaskDesc    = `Deletes a mask from the current user. Admins can add a` +
		` user param to remove a mask to that user.`
	delmaskSuccess = `Host [%v] removed successfully.`
	delmaskFailure = `Host [%v] not found.`

	resetpasswdDesc          = `Resets a user's password.`
	resetpasswdSuccess       = `Password reset successful.`
	resetpasswdSuccessTarget = `Your password was reset by %v, it is now: %v`

	ggiveDesc = `Gives global access to a user.` +
		` Arguments can be numeric levels or flags.`
	ggiveSuccess = `User [%v] now has: (%v) globally.`
	sgiveDesc    = `Gives server access to a user.` +
		` Arguments can be numeric levels or flags.`
	sgiveSuccess = `User [%v] now has: (%v) server-wide.`
	giveDesc     = `Gives channel access to a user.` +
		` Arguments can be numeric levels or flags.`
	giveSuccess = `User [%v] now has: (%v) on %v`
	gtakeDesc   = `Takes global access from a user. If no arguments are ` +
		`given, takes the level access, otherwise removes the given flags. ` +
		`Use all to take all access.`
	stakeDesc = `Takes server access from a user. If no arguments are ` +
		`given, takes the level access, otherwise removes the given flags. ` +
		`Use all to take all access.`
	takeDesc = `Takes channel access from a user. If no arguments are ` +
		`given, takes the level access, otherwise removes the given flags. ` +
		`Use all to take all access.`

	giveFailure = `Invalid arguments, must be numeric accesses from 1-255 or ` +
		`flags in the range: A-Za-z.`
	takeFailure = `Invalid arguments, leave empty to delete level access, ` +
		`specific flags to delete those flags, or the keyword all to delete ` +
		`everything. (given: %v)`
	takeFailureNo = `No action taken. User [%v](%v) has none of the given ` +
		`accesses to remove.`
)

type (
	argv           []string
	giveHelperFunc func(*data.UserAccess, uint8, string) string
	takeHelperFunc func(*data.UserAccess, bool, bool, string) (string, bool)
)

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
	{logout, logoutDesc, true, true, 0, ``, argv{`[*user]`}},
	{access, accessDesc, true, true, 0, ``, argv{`[*user]`}},
	{deluser, deluserDesc, true, true, 0, ``, argv{`*user`}},
	{delme, delmeDesc, true, true, 0, ``, nil},
	{passwd, passwdDesc, true, false, 0, ``,
		argv{`oldpassword`, `newpassword`}},
	{masks, masksDesc, true, false, 0, ``, argv{`[*user]`}},
	{addmask, addmaskDesc, true, false, 0, ``, argv{`mask`, `[*user]`}},
	{delmask, delmaskDesc, true, false, 0, ``, argv{`mask`, `[*user]`}},
	{resetpasswd, resetpasswdDesc, true, false, 0, ``, argv{`~nick`, `*user`}},
	{ggive, ggiveDesc, true, true, 0, `G`, argv{`*user`, `levelOrFlags...`}},
	{sgive, sgiveDesc, true, true, 0, `GS`, argv{`*user`, `levelOrFlags...`}},
	{give, giveDesc, true, true, 0, `GSC`, argv{`#chan`, `*user`,
		`levelOrFlags...`}},
	{gtake, gtakeDesc, true, true, 0, `G`, argv{`*user`, `[allOrFlags]`}},
	{stake, stakeDesc, true, true, 0, `GS`, argv{`*user`, `[allOrFlags]`}},
	{take, takeDesc, true, true, 0, `GSC`, argv{`#chan`, `*user`,
		`[allOrFlags]`}},
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
	case resetpasswd:
		internal, external = c.resetpasswd(d, cd)
	case ggive:
		internal, external = c.ggive(d, cd)
	case sgive:
		internal, external = c.sgive(d, cd)
	case give:
		internal, external = c.give(d, cd)
	case gtake:
		internal, external = c.gtake(d, cd)
	case stake:
		internal, external = c.stake(d, cd)
	case take:
		internal, external = c.take(d, cd)
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
		uname = strings.TrimLeft(cd.User.GetUsername(), "~")
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
		uname = strings.TrimLeft(cd.User.GetUsername(), "~")
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

	user := cd.TargetUserAccess["user"]
	uname := ""
	host, nick := cd.User.GetFullhost(), cd.User.GetNick()
	if user != nil {
		if !cd.UserAccess.HasGlobalFlag('G') {
			external = cmds.MakeGlobalFlagsError("G")
			return
		}
		uname = user.Username
	}

	cd.Close()
	c.b.protectStore.Lock()
	defer c.b.protectStore.Unlock()
	store := c.b.store

	if len(uname) != 0 {
		store.LogoutByUsername(uname)
	} else {
		store.Logout(d.GetKey(), host)
	}
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
	if !cd.UserAccess.HasGlobalFlag('G') {
		external = cmds.MakeGlobalFlagsError("G")
		return
	}
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
	internal = access.SetPassword(newpasswd)
	if internal != nil {
		return
	}
	internal = store.AddUser(access)
	if internal != nil {
		return
	}
	d.Notice(nick, passwdSuccess)

	return
}

// masks outputs the masks of the user.
func (c *coreCommands) masks(d *data.DataEndpoint, cd *cmds.CommandData) (
	internal, external error) {

	access := cd.UserAccess
	user := cd.TargetUserAccess["user"]
	if user != nil {
		if !cd.UserAccess.HasGlobalFlag('G') {
			external = cmds.MakeGlobalFlagsError("G")
			return
		}
		access = user
	}

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

	user := cd.TargetUserAccess["user"]
	if user != nil {
		if !cd.UserAccess.HasGlobalFlag('G') {
			external = cmds.MakeGlobalFlagsError("G")
			return
		}
		uname = user.Username
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

	if access.AddMasks(mask) {
		internal = store.AddUser(access)
		if internal != nil {
			return
		}
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

	user := cd.TargetUserAccess["user"]
	if user != nil {
		if !cd.UserAccess.HasGlobalFlag('G') {
			external = cmds.MakeGlobalFlagsError("G")
			return
		}
		uname = user.Username
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

	if access.DelMasks(mask) {
		internal = store.AddUser(access)
		if internal != nil {
			return
		}
		d.Noticef(nick, delmaskSuccess, mask)
	} else {
		d.Noticef(nick, delmaskFailure, mask)
	}

	return
}

// resetpasswd resets a user's password
func (c *coreCommands) resetpasswd(d *data.DataEndpoint, cd *cmds.CommandData) (
	internal, external error) {

	uname := cd.TargetUserAccess["user"].Username
	resetnick := cd.TargetUsers["nick"].GetNick()
	nick := cd.User.GetNick()
	newpasswd := ""

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
	newpasswd, internal = access.ResetPassword()
	if internal != nil {
		return
	}
	internal = store.AddUser(access)
	if internal != nil {
		return
	}
	d.Notice(nick, resetpasswdSuccess)
	d.Noticef(resetnick, resetpasswdSuccessTarget, nick, newpasswd)

	return
}

// ggive gives global access to a user.
func (c *coreCommands) ggive(d *data.DataEndpoint, cd *cmds.CommandData) (
	internal, external error) {
	return c.giveHelper(d, cd,
		func(a *data.UserAccess, level uint8, flags string) string {
			if level > 0 {
				a.GrantGlobalLevel(level)
			}
			if len(flags) != 0 {
				a.GrantGlobalFlags(flags)
			}
			return fmt.Sprintf(ggiveSuccess, a.Username, a.Global.String())
		},
	)
}

// sgive gives server access to a user.
func (c *coreCommands) sgive(d *data.DataEndpoint, cd *cmds.CommandData) (
	internal, external error) {
	server := d.GetKey()
	return c.giveHelper(d, cd,
		func(a *data.UserAccess, level uint8, flags string) string {
			if level > 0 {
				a.GrantServerLevel(server, level)
			}
			if len(flags) != 0 {
				a.GrantServerFlags(server, flags)
			}
			return fmt.Sprintf(sgiveSuccess, a.Username, a.GetServer(server))
		},
	)
}

// give gives global access to a user.
func (c *coreCommands) give(d *data.DataEndpoint, cd *cmds.CommandData) (
	internal, external error) {
	server := d.GetKey()
	channel := cd.GetArg("chan")
	return c.giveHelper(d, cd,
		func(a *data.UserAccess, level uint8, flags string) string {
			if level > 0 {
				a.GrantChannelLevel(server, channel, level)
			}
			if len(flags) != 0 {
				a.GrantChannelFlags(server, channel, flags)
			}
			return fmt.Sprintf(giveSuccess, a.Username, channel,
				a.GetChannel(server, channel))
		},
	)
}

// gtake takes global access from a user.
func (c *coreCommands) gtake(d *data.DataEndpoint, cd *cmds.CommandData) (
	internal, external error) {
	return c.takeHelper(d, cd,
		func(a *data.UserAccess, all, level bool, flags string) (string, bool) {
			var save bool
			if all {
				if a.HasGlobalLevel(1) || a.HasGlobalFlags(flags) {
					a.RevokeGlobal()
					save = true
				}
			} else if level {
				if a.HasGlobalLevel(1) {
					a.RevokeGlobalLevel()
					save = true
				}
			} else if a.HasGlobalFlags(flags) {
				a.RevokeGlobalFlags(flags)
				save = true
			}

			var rstr = a.Global.String()
			if save {
				rstr = fmt.Sprintf(ggiveSuccess, a.Username, rstr)
			} else {
				rstr = fmt.Sprintf(takeFailureNo, a.Username, rstr)
			}
			return rstr, save
		},
	)
}

// stake takes server access from a user.
func (c *coreCommands) stake(d *data.DataEndpoint, cd *cmds.CommandData) (
	internal, external error) {
	server := d.GetKey()
	return c.takeHelper(d, cd,
		func(a *data.UserAccess, all, level bool, flags string) (string, bool) {
			var save bool
			if all {
				if a.HasServerLevel(server, 1) ||
					a.HasServerFlags(server, flags) {

					a.RevokeServer(server)
					save = true
				}
			} else if level {
				if a.HasServerLevel(server, 1) {
					a.RevokeServerLevel(server)
					save = true
				}
			} else if a.HasServerFlags(server, flags) {
				a.RevokeServerFlags(server, flags)
				save = true
			}

			var rstr = a.GetServer(server).String()
			if save {
				rstr = fmt.Sprintf(sgiveSuccess, a.Username, rstr)
			} else {
				rstr = fmt.Sprintf(takeFailureNo, a.Username, rstr)
			}
			return rstr, save
		},
	)
}

// take takes global access from a user.
func (c *coreCommands) take(d *data.DataEndpoint, cd *cmds.CommandData) (
	internal, external error) {
	server := d.GetKey()
	channel := cd.GetArg("chan")
	return c.takeHelper(d, cd,
		func(a *data.UserAccess, all, level bool, flags string) (string, bool) {
			var save bool
			if all {
				if a.HasChannelLevel(server, channel, 1) ||
					a.HasChannelFlags(server, channel, flags) {

					a.RevokeChannel(server, channel)
					save = true
				}
			} else if level {
				if a.HasChannelLevel(server, channel, 1) {
					a.RevokeChannelLevel(server, channel)
					save = true
				}
			} else if a.HasChannelFlags(server, channel, flags) {
				a.RevokeChannelFlags(server, channel, flags)
				save = true
			}

			var rstr = a.GetChannel(server, channel).String()
			if save {
				rstr = fmt.Sprintf(giveSuccess, a.Username, rstr)
			} else {
				rstr = fmt.Sprintf(takeFailureNo, a.Username, rstr)
			}
			return rstr, save
		},
	)
}

// giveHelper parses the args to a give function and executes them in context
func (c *coreCommands) giveHelper(d *data.DataEndpoint, cd *cmds.CommandData,
	g giveHelperFunc) (internal, external error) {

	uname := cd.TargetUserAccess["user"].Username
	args := cd.SplitArg("levelOrFlags")
	nick := cd.User.GetNick()

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

	var level uint64
	var flags string

	for _, arg := range args {
		if rgxFlags.MatchString(arg) {
			flags += arg
		} else if level, internal =
			strconv.ParseUint(arg, 10, 8); internal != nil {
			return
		}
	}

	if (level > 0 && level < 256) || len(flags) > 0 {
		str := g(access, uint8(level), flags)
		if internal = store.AddUser(access); internal != nil {
			return
		}
		d.Noticef(nick, str)
	} else {
		d.Noticef(nick, giveFailure)
	}

	return
}

// takeHelper parses the args to a take function and executes them in context
func (c *coreCommands) takeHelper(d *data.DataEndpoint, cd *cmds.CommandData,
	t takeHelperFunc) (internal, external error) {

	uname := cd.TargetUserAccess["user"].Username
	arg := cd.GetArg("allOrFlags")
	nick := cd.User.GetNick()

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

	var all, level bool
	var flags string

	if len(arg) == 0 {
		level = true
	} else if arg == takeAllArg {
		all = true
	} else {
		if rgxFlags.MatchString(arg) {
			flags = arg
		} else {
			external = fmt.Errorf(takeFailure, arg)
		}
	}

	str, dosave := t(access, all, level, flags)
	if dosave {
		if internal = store.AddUser(access); internal != nil {
			return
		}
	}
	d.Noticef(nick, str)

	return
}
