package bot

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/dispatch/cmd"
	"github.com/aarondl/ultimateq/irc"
)

var rgxFlags = regexp.MustCompile(`[A-Za-z]+`)

const (
	extension = `core`
	register  = `register`
	auth      = `auth`
	logout    = `logout`
	access    = `access`
	users     = `users`
	gusers    = `gusers`
	susers    = `susers`
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

	help = `help`

	errFmtRegister = `bot: A core command registration failed: %v`
	errMsgInternal = `There was an internal error, try again later.`
	errFmtInternal = `commander: Error processing command %v (%v)`
	errFmtExpired  = `commander: Data expired between locks. ` +
		`Could not find user [%v]`
	cmdExec          = "bot: Core command executed"
	errInternalError = "bot: Core command error"
	errInternalPanic = "bot: Core command error"

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
	sgiveDesc    = `Gives network access to a user.` +
		` Arguments can be numeric levels or flags.`
	sgiveSuccess = `User [%v] now has: (%v) network-wide.`
	giveDesc     = `Gives channel access to a user.` +
		` Arguments can be numeric levels or flags.`
	giveSuccess = `User [%v] now has: (%v) on %v`
	gtakeDesc   = `Takes global access from a user. If no arguments are ` +
		`given, takes the level access, otherwise removes the given flags. ` +
		`Use all to take all access.`
	stakeDesc = `Takes network access from a user. If no arguments are ` +
		`given, takes the level access, otherwise removes the given flags. ` +
		`Use all to take all access.`
	takeDesc = `Takes channel access from a user. If no arguments are ` +
		`given, takes the level access, otherwise removes the given flags. ` +
		`Use all to take all access.`

	giveFailure = `Invalid arguments, must be numeric accesses from 1-255 or ` +
		`flags in the range: A-Za-z.`
	giveFailureHas = `User [%v](%v) already has: %v`
	takeFailure    = `Invalid arguments, leave empty to delete level access, ` +
		`specific flags to delete those flags, or the keyword all to delete ` +
		`everything. (given: %v)`
	takeFailureNo = `No action taken. User [%v](%v) has none of the given ` +
		`access to remove.`

	gusersDesc    = `Lists all the users added to the global access list.`
	gusersNoUsers = `No users for %v`
	gusersHead    = `Showing %v users:`

	usersDesc = `Lists all the users added to the channel's access list. ` +
		`If no channel specified then list for current channel.`
	usersNoUsers = `No users for %v`
	usersHead    = `Showing %v users for %v:`

	susersDesc    = `Lists all the users added to the network's access list. `
	susersNoUsers = `No users for %v`
	susersHead    = `Showing %v users for %v:`

	usersListHeadUser   = `User`
	usersListHeadAccess = `Access`
	usersList           = `%-*v %v`

	helpSuccess      = `Cmds:`
	helpSuccessUsage = `Usage: `
	helpFailure      = `No help available for (%v), try "help" for a list of ` +
		`all commands.`
	helpDesc = `Help with no arguments shows all commands, help with an ` +
		`argument performs a search, if only one match is found gives ` +
		`detailed information about that command.`
)

type (
	argv           []string
	giveHelperFunc func(*data.StoredUser, uint8, string) (string, bool)
	takeHelperFunc func(*data.StoredUser, bool, bool, string) (string, bool)
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
	{gusers, gusersDesc, false, true, 0, ``, nil},
	{susers, susersDesc, false, true, 0, ``, nil},
	{users, usersDesc, false, true, 0, ``, argv{`[chan]`}},
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
	{help, helpDesc, true, true, 0, ``, argv{`[command]`}},
}

// coreCmds is the bot's command handling struct. The bot itself uses
// the cmds to implement user management.
type coreCmds struct {
	b *Bot
}

// NewCoreCmds initializes the core commands and registers them with the
// bot.
func NewCoreCmds(b *Bot) (*coreCmds, error) {
	c := &coreCmds{b}
	for _, command := range commands {
		privacy := cmd.PRIVATE
		if command.Public {
			privacy = cmd.ALL
		}
		err := b.RegisterCmd(&cmd.Cmd{
			Cmd:         command.Name,
			Extension:   extension,
			Description: command.Desc,
			Handler:     c,
			Msgtype:     cmd.PRIVMSG,
			Msgscope:    privacy,
			Args:        command.Args,
			RequireAuth: command.Authed,
			ReqLevel:    command.Level,
			ReqFlags:    command.Flags,
		})
		if err != nil {
			return nil, fmt.Errorf(errFmtRegister, err)
		}
	}

	return &coreCmds{b}, nil
}

// unregisterCoreCmds unregisters all core commands. Made for testing.
func (c *coreCmds) unregisterCoreCmds() {
	for _, cmd := range commands {
		c.b.UnregisterCmd(cmd.Name)
	}
}

// Cmd is responsible for parsing all of the commands.
func (c *coreCmds) Cmd(cmd string, w irc.Writer,
	ev *cmd.Event) (internal error) {

	var external error

	c.b.Info(cmdExec, "cmd", cmd)

	defer func() {
		if r := recover(); r != nil {
			c.b.Error(errInternalPanic, "cmd", cmd, "panic", r)
		}
	}()

	switch cmd {
	case register:
		internal, external = c.register(w, ev)
	case auth:
		internal, external = c.auth(w, ev)
	case logout:
		internal, external = c.logout(w, ev)
	case access:
		internal, external = c.access(w, ev)
	case gusers:
		internal, external = c.gusers(w, ev)
	case susers:
		internal, external = c.susers(w, ev)
	case users:
		internal, external = c.users(w, ev)
	case deluser:
		internal, external = c.deluser(w, ev)
	case delme:
		internal, external = c.delme(w, ev)
	case passwd:
		internal, external = c.passwd(w, ev)
	case masks:
		internal, external = c.masks(w, ev)
	case addmask:
		internal, external = c.addmask(w, ev)
	case delmask:
		internal, external = c.delmask(w, ev)
	case resetpasswd:
		internal, external = c.resetpasswd(w, ev)
	case ggive:
		internal, external = c.ggive(w, ev)
	case sgive:
		internal, external = c.sgive(w, ev)
	case give:
		internal, external = c.give(w, ev)
	case gtake:
		internal, external = c.gtake(w, ev)
	case stake:
		internal, external = c.stake(w, ev)
	case take:
		internal, external = c.take(w, ev)
	case help:
		internal, external = c.help(w, ev)
	}

	if internal != nil {
		c.b.Error(errInternalError, "cmd", cmd, "err", internal)
	}

	return external
}

// register register's a user to the bot with an optional user name.
func (c *coreCmds) register(w irc.Writer,
	ev *cmd.Event) (internal, external error) {

	var access *data.StoredUser

	pwd := ev.GetArg("password")
	uname := ev.GetArg("username")
	if len(uname) == 0 {
		uname = strings.TrimLeft(ev.User.Username(), "~")
	}

	access = ev.StoredUser
	if access == nil {
		access = ev.GetAuthedUser(ev.NetworkID, ev.User.Host())
	}
	if access != nil {
		return nil, fmt.Errorf(errMsgAuthed)
	}

	access, internal = ev.FindUser(uname)
	if internal != nil {
		return
	}
	if access != nil {
		return nil, fmt.Errorf(registerFailure, uname)
	}

	access, internal = data.NewStoredUser(uname, pwd)
	if internal != nil {
		return
	}

	host, nick := ev.User.Host(), ev.User.Nick()

	ev.Close()
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

	internal = store.SaveUser(access)
	if internal != nil {
		return
	}

	_, internal = store.AuthUser(ev.NetworkID, host, uname, pwd)
	if internal != nil {
		return
	}

	if isFirst {
		w.Noticef(nick, registerSuccessFirst, uname)
	} else {
		w.Noticef(nick, registerSuccess, uname)
	}

	return
}

// auth authenticates a user.
func (c *coreCmds) auth(w irc.Writer, ev *cmd.Event) (
	internal, external error) {

	var access *data.StoredUser
	pwd := ev.GetArg("password")
	uname := ev.GetArg("username")
	if len(uname) == 0 {
		uname = strings.TrimLeft(ev.User.Username(), "~")
	}

	host, nick := ev.User.Host(), ev.User.Nick()

	access = ev.StoredUser
	if access == nil {
		access = ev.GetAuthedUser(ev.NetworkID, host)
	}
	if access != nil {
		external = errors.New(errMsgAuthed)
		return
	}

	ev.Close()
	c.b.protectStore.Lock()
	defer c.b.protectStore.Unlock()
	store := c.b.store

	_, err := store.AuthUser(ev.NetworkID, host, uname, pwd)
	if err != nil {
		if authErr, ok := err.(data.AuthError); ok {
			external = authErr
		} else {
			internal = err
		}
		return
	}

	w.Noticef(nick, authSuccess, uname)
	return
}

// logout logs out a user.
func (c *coreCmds) logout(w irc.Writer, ev *cmd.Event) (
	internal, external error) {

	user := ev.TargetStoredUser["user"]
	uname := ""
	host, nick := ev.User.Host(), ev.User.Nick()
	if user != nil {
		if !ev.StoredUser.HasGlobalFlag('G') {
			external = cmd.MakeGlobalFlagsError("G")
			return
		}
		uname = user.Username
	}

	ev.Close()
	c.b.protectStore.Lock()
	defer c.b.protectStore.Unlock()
	store := c.b.store

	if len(uname) != 0 {
		store.LogoutByUsername(uname)
	} else {
		store.Logout(ev.NetworkID, host)
	}
	w.Notice(nick, logoutSuccess)

	return
}

// access outputs the access for the user.
func (c *coreCmds) access(w irc.Writer, ev *cmd.Event) (
	internal, external error) {

	access := ev.TargetStoredUser["user"]
	if access == nil {
		access = ev.StoredUser
	}

	ch := ""
	if ev.Channel != nil {
		ch = ev.Channel.Name()
	}
	w.Noticef(ev.User.Nick(), accessSuccess,
		access.Username, access.String(ev.NetworkID, ch))

	return
}

//gusers provides a list of users with global access
func (c *coreCmds) gusers(w irc.Writer, ev *cmd.Event) (
	internal, external error) {

	var list []*data.StoredUser
	var ua *data.StoredUser

	nick := ev.User.Nick()

	list, internal = c.b.store.GlobalUsers()
	if internal != nil {
		return
	}

	if len(list) == 0 {
		w.Noticef(nick, gusersNoUsers)
		return
	}

	usersWidth := userListWidth(list) + 1
	w.Noticef(nick, gusersHead, len(list))
	w.Noticef(nick, usersList, usersWidth,
		usersListHeadUser, usersListHeadAccess)

	for _, ua = range list {
		ga := ua.GetGlobal()
		w.Noticef(nick, usersList, usersWidth, ua.Username, ga)
	}

	return
}

//susers provides a list of users with network access
func (c *coreCmds) susers(w irc.Writer, ev *cmd.Event) (
	internal, external error) {

	var list []*data.StoredUser
	var ua *data.StoredUser

	nick := ev.User.Nick()

	list, internal = c.b.store.NetworkUsers(ev.NetworkID)
	if internal != nil {
		return
	}

	if len(list) == 0 {
		w.Noticef(nick, susersNoUsers, ev.NetworkID)
		return
	}

	usersWidth := userListWidth(list) + 1
	w.Noticef(nick, susersHead, len(list), ev.NetworkID)
	w.Noticef(nick, usersList, usersWidth,
		usersListHeadUser, usersListHeadAccess)

	for _, ua = range list {
		sa := ua.GetNetwork(ev.NetworkID)
		w.Noticef(nick, usersList, usersWidth, ua.Username, sa)
	}

	return
}

// users provides a list of users added to a channel
func (c *coreCmds) users(w irc.Writer, ev *cmd.Event) (
	internal, external error) {

	var list []*data.StoredUser
	var ua *data.StoredUser
	var ch string

	if ev.GetArg("chan") != `` {
		ch = ev.GetArg("chan")
	} else {
		if ev.Channel.Name() != `` {
			ch = ev.Channel.Name()
		} else {
			return
		}
	}

	nick := ev.User.Nick()

	list, internal = c.b.store.ChanUsers(ev.NetworkID, ch)
	if internal != nil {
		return
	}

	if len(list) == 0 {
		w.Noticef(nick, usersNoUsers, ch)
		return
	}

	usersWidth := userListWidth(list) + 1
	w.Noticef(nick, usersHead, len(list), ch)
	w.Noticef(nick, usersList, usersWidth,
		usersListHeadUser, usersListHeadAccess)

	for _, ua = range list {
		ca := ua.GetChannel(ev.NetworkID, ch)
		w.Noticef(nick, usersList, usersWidth, ua.Username, ca)
	}

	return
}

// deluser deletes a user
func (c *coreCmds) deluser(w irc.Writer, ev *cmd.Event) (
	internal, external error) {

	param := ev.GetArg("user")
	if !ev.StoredUser.HasGlobalFlag('G') {
		external = cmd.MakeGlobalFlagsError("G")
		return
	}
	uname := ev.TargetStoredUser["user"].Username

	nick := ev.User.Nick()
	ev.Close()
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
		w.Noticef(nick, deluserSuccess, param)
	} else {
		w.Noticef(nick, deluserFailure, param)
	}

	return
}

// delme deletes self
func (c *coreCmds) delme(w irc.Writer, ev *cmd.Event) (
	internal, external error) {

	host, nick := ev.User.Host(), ev.User.Nick()
	uname := ev.StoredUser.Username
	ev.Close()
	c.b.protectStore.Lock()
	defer c.b.protectStore.Unlock()
	store := c.b.store

	removed := false
	store.Logout(ev.NetworkID, host)
	removed, internal = store.RemoveUser(uname)
	if internal != nil {
		return
	}
	if !removed {
		internal = errors.New(delmeFailure)
		return
	}
	w.Noticef(nick, delmeSuccess, uname)
	return
}

// passwd changes a user's password
func (c *coreCmds) passwd(w irc.Writer, ev *cmd.Event) (
	internal, external error) {

	oldpasswd := ev.GetArg("oldpassword")
	newpasswd := ev.GetArg("newpassword")
	nick := ev.User.Nick()
	uname := ev.StoredUser.Username
	if !ev.StoredUser.VerifyPassword(oldpasswd) {
		w.Notice(nick, passwdFailure)
		return
	}

	ev.Close()
	c.b.protectStore.Lock()
	defer c.b.protectStore.Unlock()
	store := c.b.store

	var access *data.StoredUser
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
	internal = store.SaveUser(access)
	if internal != nil {
		return
	}
	w.Notice(nick, passwdSuccess)

	return
}

// masks outputs the masks of the user.
func (c *coreCmds) masks(w irc.Writer, ev *cmd.Event) (
	internal, external error) {

	access := ev.StoredUser
	user := ev.TargetStoredUser["user"]
	if user != nil {
		if !ev.StoredUser.HasGlobalFlag('G') {
			external = cmd.MakeGlobalFlagsError("G")
			return
		}
		access = user
	}

	if len(access.Masks) > 0 {
		w.Noticef(ev.User.Nick(), masksSuccess,
			strings.Join(access.Masks, " "))
	} else {
		w.Notice(ev.User.Nick(), masksFailure)
	}

	return
}

// addmask adds a mask to a user.
func (c *coreCmds) addmask(w irc.Writer, ev *cmd.Event) (
	internal, external error) {

	mask := ev.GetArg("mask")
	nick := ev.User.Nick()
	uname := ev.StoredUser.Username

	user := ev.TargetStoredUser["user"]
	if user != nil {
		if !ev.StoredUser.HasGlobalFlag('G') {
			external = cmd.MakeGlobalFlagsError("G")
			return
		}
		uname = user.Username
	}

	ev.Close()
	c.b.protectStore.Lock()
	defer c.b.protectStore.Unlock()
	store := c.b.store

	var access *data.StoredUser
	access, internal = store.FindUser(uname)
	if internal != nil {
		return
	}
	if access == nil {
		internal = fmt.Errorf(errFmtExpired, uname)
		return
	}

	if access.AddMask(mask) {
		internal = store.SaveUser(access)
		if internal != nil {
			return
		}
		w.Noticef(nick, addmaskSuccess, mask)
	} else {
		w.Noticef(nick, addmaskFailure, mask)
	}

	return
}

// delmask deletes a mask from a user.
func (c *coreCmds) delmask(w irc.Writer, ev *cmd.Event) (
	internal, external error) {

	mask := ev.GetArg("mask")
	nick := ev.User.Nick()
	uname := ev.StoredUser.Username

	user := ev.TargetStoredUser["user"]
	if user != nil {
		if !ev.StoredUser.HasGlobalFlag('G') {
			external = cmd.MakeGlobalFlagsError("G")
			return
		}
		uname = user.Username
	}

	ev.Close()
	c.b.protectStore.Lock()
	defer c.b.protectStore.Unlock()
	store := c.b.store

	var access *data.StoredUser
	access, internal = store.FindUser(uname)
	if internal != nil {
		return
	}
	if access == nil {
		internal = fmt.Errorf(errFmtExpired, uname)
		return
	}

	if access.DelMask(mask) {
		internal = store.SaveUser(access)
		if internal != nil {
			return
		}
		w.Noticef(nick, delmaskSuccess, mask)
	} else {
		w.Noticef(nick, delmaskFailure, mask)
	}

	return
}

// resetpasswd resets a user's password
func (c *coreCmds) resetpasswd(w irc.Writer, ev *cmd.Event) (
	internal, external error) {

	uname := ev.TargetStoredUser["user"].Username
	resetnick := ev.TargetUsers["nick"].Nick()
	nick := ev.User.Nick()
	newpasswd := ""

	ev.Close()
	c.b.protectStore.Lock()
	defer c.b.protectStore.Unlock()
	store := c.b.store

	var access *data.StoredUser
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
	internal = store.SaveUser(access)
	if internal != nil {
		return
	}
	w.Notice(nick, resetpasswdSuccess)
	w.Noticef(resetnick, resetpasswdSuccessTarget, nick, newpasswd)

	return
}

// ggive gives global access to a user.
func (c *coreCmds) ggive(w irc.Writer, ev *cmd.Event) (
	internal, external error) {
	return c.giveHelper(w, ev,
		func(a *data.StoredUser, level uint8, flags string) (string, bool) {
			if level > 0 {
				a.GrantGlobalLevel(level)
			}
			if len(flags) != 0 {
				filtered := filterFlags(flags, a.HasGlobalFlag)
				if len(filtered) == 0 && level == 0 {
					return fmt.Sprintf(giveFailureHas, a.Username,
						a.Global, flags), false
				}
				a.GrantGlobalFlags(filtered)
			}
			return fmt.Sprintf(ggiveSuccess, a.Username, a.Global), true
		},
	)
}

// sgive gives network access to a user.
func (c *coreCmds) sgive(w irc.Writer, ev *cmd.Event) (
	internal, external error) {
	network := ev.NetworkID
	return c.giveHelper(w, ev,
		func(a *data.StoredUser, level uint8, flags string) (string, bool) {
			if level > 0 {
				a.GrantNetworkLevel(network, level)
			}
			if len(flags) != 0 {
				filtered := filterFlags(flags, func(r rune) bool {
					return a.HasNetworkFlag(network, r)
				})

				if len(filtered) == 0 && level == 0 {
					return fmt.Sprintf(giveFailureHas, a.Username,
						a.GetNetwork(network), flags), false
				}
				a.GrantNetworkFlags(network, filtered)
			}
			return fmt.Sprintf(sgiveSuccess, a.Username, a.GetNetwork(network)),
				true
		},
	)
}

// give gives channel access to a user.
func (c *coreCmds) give(w irc.Writer, ev *cmd.Event) (
	internal, external error) {
	network := ev.NetworkID
	channel := ev.GetArg("chan")
	return c.giveHelper(w, ev,
		func(a *data.StoredUser, level uint8, flags string) (string, bool) {
			if level > 0 {
				a.GrantChannelLevel(network, channel, level)
			}
			if len(flags) != 0 {
				filtered := filterFlags(flags, func(r rune) bool {
					return a.HasChannelFlag(network, channel, r)
				})

				if len(filtered) == 0 && level == 0 {
					return fmt.Sprintf(giveFailureHas, a.Username,
						a.GetChannel(network, channel), flags), false
				}
				a.GrantChannelFlags(network, channel, filtered)
			}
			return fmt.Sprintf(giveSuccess, a.Username,
				a.GetChannel(network, channel), channel), true
		},
	)
}

// gtake takes global access from a user.
func (c *coreCmds) gtake(w irc.Writer, ev *cmd.Event) (
	internal, external error) {
	return c.takeHelper(w, ev,
		func(a *data.StoredUser, all, level bool, flags string) (string, bool) {
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

// stake takes network access from a user.
func (c *coreCmds) stake(w irc.Writer, ev *cmd.Event) (
	internal, external error) {
	network := ev.NetworkID
	return c.takeHelper(w, ev,
		func(a *data.StoredUser, all, level bool, flags string) (string, bool) {
			var save bool
			if all {
				if a.HasNetworkLevel(network, 1) ||
					a.HasNetworkFlags(network, flags) {

					a.RevokeNetwork(network)
					save = true
				}
			} else if level {
				if a.HasNetworkLevel(network, 1) {
					a.RevokeNetworkLevel(network)
					save = true
				}
			} else if a.HasNetworkFlags(network, flags) {
				a.RevokeNetworkFlags(network, flags)
				save = true
			}

			var rstr = a.GetNetwork(network).String()
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
func (c *coreCmds) take(w irc.Writer, ev *cmd.Event) (
	internal, external error) {
	network := ev.NetworkID
	channel := ev.GetArg("chan")
	return c.takeHelper(w, ev,
		func(a *data.StoredUser, all, level bool, flags string) (string, bool) {
			var save bool
			if all {
				if a.HasChannelLevel(network, channel, 1) ||
					a.HasChannelFlags(network, channel, flags) {

					a.RevokeChannel(network, channel)
					save = true
				}
			} else if level {
				if a.HasChannelLevel(network, channel, 1) {
					a.RevokeChannelLevel(network, channel)
					save = true
				}
			} else if a.HasChannelFlags(network, channel, flags) {
				a.RevokeChannelFlags(network, channel, flags)
				save = true
			}

			var rstr = a.GetChannel(network, channel).String()
			if save {
				rstr = fmt.Sprintf(giveSuccess, a.Username, rstr, channel)
			} else {
				rstr = fmt.Sprintf(takeFailureNo, a.Username, rstr)
			}
			return rstr, save
		},
	)
}

// giveHelper parses the args to a give function and executes them in context
func (c *coreCmds) giveHelper(w irc.Writer, ev *cmd.Event,
	g giveHelperFunc) (internal, external error) {

	uname := ev.TargetStoredUser["user"].Username
	args := ev.SplitArg("levelOrFlags")
	nick := ev.User.Nick()

	ev.Close()
	c.b.protectStore.Lock()
	defer c.b.protectStore.Unlock()
	store := c.b.store

	var access *data.StoredUser
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
		str, dosave := g(access, uint8(level), flags)
		if dosave {
			if internal = store.SaveUser(access); internal != nil {
				return
			}
		}
		w.Noticef(nick, str)
	} else {
		w.Noticef(nick, giveFailure)
	}

	return
}

// takeHelper parses the args to a take function and executes them in context
func (c *coreCmds) takeHelper(w irc.Writer, ev *cmd.Event,
	t takeHelperFunc) (internal, external error) {

	uname := ev.TargetStoredUser["user"].Username
	arg := ev.GetArg("allOrFlags")
	nick := ev.User.Nick()

	ev.Close()
	c.b.protectStore.Lock()
	defer c.b.protectStore.Unlock()
	store := c.b.store

	var access *data.StoredUser
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
		if internal = store.SaveUser(access); internal != nil {
			return
		}
	}
	w.Noticef(nick, str)

	return
}

// help searches for commands, and also provides details for specific commands
func (c *coreCmds) help(w irc.Writer, ev *cmd.Event) (
	internal, external error) {

	search := strings.ToLower(ev.GetArg("command"))
	nick := ev.User.Nick()

	var output = make(map[string][]string)
	var exactMatches []*cmd.Cmd

	cmd.EachCmd(func(command *cmd.Cmd) bool {
		write := true

		if len(search) > 0 {
			combined := command.Extension + "." + command.Cmd
			if perfect := combined == search; command.Cmd == search || perfect {
				if exactMatches == nil || perfect {
					exactMatches = []*cmd.Cmd{command}
				} else {
					exactMatches = append(exactMatches, command)
					write = false
				}
				if perfect {
					return true
				}
			} else if !strings.Contains(combined, search) {
				write = false
			}
		}

		if write {
			if arr, ok := output[command.Extension]; ok {
				output[command.Extension] = append(arr, command.Cmd)
			} else {
				output[command.Extension] = []string{command.Cmd}
			}
		}
		return false
	})

	if len(exactMatches) > 1 {
		for _, command := range exactMatches {
			if arr, ok := output[command.Extension]; ok {
				output[command.Extension] = append(arr, command.Cmd)
			} else {
				output[command.Extension] = []string{command.Cmd}
			}
		}
		exactMatches = nil
	}

	if exactMatches != nil {
		exactMatch := exactMatches[0]
		w.Notice(nick, helpSuccess,
			" ", exactMatch.Extension, ".", exactMatch.Cmd)
		w.Notice(nick, exactMatch.Description)
		if len(exactMatch.Args) > 0 {
			w.Notice(nick, helpSuccessUsage, strings.Join(exactMatch.Args, " "))
		}
	} else if len(output) > 0 {
		for extension, commands := range output {
			sort.Strings(commands)
			w.Notice(nick, extension, ":")
			w.Notice(nick, " ", strings.Join(commands, " "))
		}
	} else {
		w.Noticef(nick, helpFailure, search)
	}

	return
}

// filterFlags removes flags that the user already has
func filterFlags(flags string, check func(rune) bool) string {
	buf := bytes.Buffer{}
	for _, flag := range flags {
		if !check(flag) {
			buf.WriteRune(flag)
		}
	}
	return buf.String()
}

// userListWidth calculates the width of the userList's "User" column.
func userListWidth(users []*data.StoredUser) int {
	minl := len(usersListHeadUser)
	for _, u := range users {
		l := len(u.Username)
		if l > minl {
			minl = l
		}
	}

	return minl
}
