package bot

import (
	"fmt"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/dispatch"
	"github.com/aarondl/ultimateq/dispatch/cmd"
	"github.com/aarondl/ultimateq/irc"
	"github.com/pkg/errors"
)

var rgxFlags = regexp.MustCompile(`^[A-Za-z]+$`)

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

	errFmtRegister   = `bot: A core command registration failed: %v`
	errMsgInternal   = `There was an internal error, try again later.`
	errFmtInternal   = `commander: Error processing command %v (%v)`
	errFmtExpired    = `commander: Could not find user [%v]`
	cmdExec          = "bot: Core command executed"
	errInternalError = "bot: Core command error"
	errInternalPanic = "bot: Core command panic"

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
	gusersNoUsers = `No global users`
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
	helpSuccessUsage = `Usage: %v %v`
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
	{
		Name:   register,
		Desc:   registerDesc,
		Authed: false,
		Public: false,
		Level:  0,
		Flags:  ``,
		Args:   argv{`password`, `[username]`},
	},
	{
		Name:   auth,
		Desc:   authDesc,
		Authed: false,
		Public: false,
		Level:  0,
		Flags:  ``,
		Args:   argv{`password`, `[username]`},
	},
	{
		Name:   logout,
		Desc:   logoutDesc,
		Authed: true,
		Public: true,
		Level:  0,
		Flags:  ``,
		Args:   argv{`[*user]`},
	},
	{
		Name:   access,
		Desc:   accessDesc,
		Authed: true,
		Public: true,
		Level:  0,
		Flags:  ``,
		Args:   argv{`[*user]`},
	},
	{
		Name:   gusers,
		Desc:   gusersDesc,
		Authed: false,
		Public: true,
		Level:  0,
		Flags:  ``,
		Args:   nil,
	},
	{
		Name:   susers,
		Desc:   susersDesc,
		Authed: false,
		Public: true,
		Level:  0,
		Flags:  ``,
		Args:   nil,
	},
	{
		Name:   users,
		Desc:   usersDesc,
		Authed: false,
		Public: true,
		Level:  0,
		Flags:  ``,
		Args:   argv{`[chan]`},
	},
	{
		Name:   deluser,
		Desc:   deluserDesc,
		Authed: true,
		Public: true,
		Level:  0,
		Flags:  ``,
		Args:   argv{`*user`},
	},
	{
		Name:   delme,
		Desc:   delmeDesc,
		Authed: true,
		Public: true,
		Level:  0,
		Flags:  ``,
		Args:   nil,
	},
	{
		Name:   passwd,
		Desc:   passwdDesc,
		Authed: true,
		Public: false,
		Level:  0,
		Flags:  ``,
		Args:   argv{`oldpassword`, `newpassword`},
	},
	{
		Name:   masks,
		Desc:   masksDesc,
		Authed: true,
		Public: false,
		Level:  0,
		Flags:  ``,
		Args:   argv{`[*user]`},
	},
	{
		Name:   addmask,
		Desc:   addmaskDesc,
		Authed: true,
		Public: false,
		Level:  0,
		Flags:  ``,
		Args:   argv{`mask`, `[*user]`},
	},
	{
		Name:   delmask,
		Desc:   delmaskDesc,
		Authed: true,
		Public: false,
		Level:  0,
		Flags:  ``,
		Args:   argv{`mask`, `[*user]`},
	},
	{
		Name:   resetpasswd,
		Desc:   resetpasswdDesc,
		Authed: true,
		Public: false,
		Level:  0,
		Flags:  ``,
		Args:   argv{`nick`, `*user`},
	},
	{
		Name:   ggive,
		Desc:   ggiveDesc,
		Authed: true,
		Public: true,
		Level:  0,
		Flags:  `G`,
		Args:   argv{`*user`, `levelOrFlags...`},
	},
	{
		Name:   sgive,
		Desc:   sgiveDesc,
		Authed: true,
		Public: true,
		Level:  0,
		Flags:  `GS`,
		Args:   argv{`*user`, `levelOrFlags...`},
	},
	{
		Name:   give,
		Desc:   giveDesc,
		Authed: true,
		Public: true,
		Level:  0,
		Flags:  `GSC`,
		Args:   argv{`#chan`, `*user`, `levelOrFlags...`},
	},
	{
		Name:   gtake,
		Desc:   gtakeDesc,
		Authed: true,
		Public: true,
		Level:  0,
		Flags:  `G`,
		Args:   argv{`*user`, `[allOrFlags]`},
	},
	{
		Name:   stake,
		Desc:   stakeDesc,
		Authed: true,
		Public: true,
		Level:  0,
		Flags:  `GS`,
		Args:   argv{`*user`, `[allOrFlags]`},
	},
	{
		Name:   take,
		Desc:   takeDesc,
		Authed: true,
		Public: true,
		Level:  0,
		Flags:  `GSC`,
		Args:   argv{`#chan`, `*user`, `[allOrFlags]`},
	},
	{
		Name:   help,
		Desc:   helpDesc,
		Authed: false,
		Public: true,
		Level:  0,
		Flags:  ``,
		Args:   argv{`[command]`},
	},
}

// coreCmds is the bot's command handling struct. The bot itself uses
// the cmds to implement user management.
type coreCmds struct {
	b *Bot
}

// NewCoreCmds initializes the core commands and registers them with the
// bot.
func NewCoreCmds(b *Bot) (*coreCmds, error) {
	c := &coreCmds{b: b}
	for _, command := range commands {
		privacy := cmd.Private
		if command.Public {
			privacy = cmd.AnyScope
		}

		var commandObj *cmd.Command
		if command.Authed {
			commandObj = cmd.NewAuthed(
				extension,
				command.Name,
				command.Desc,
				c,
				cmd.Privmsg,
				privacy,
				command.Level,
				command.Flags,
				command.Args...,
			)
		} else {
			commandObj = cmd.New(
				extension,
				command.Name,
				command.Desc,
				c,
				cmd.Privmsg,
				privacy,
				command.Args...,
			)
		}

		_, err := b.RegisterGlobalCmd(commandObj)
		if err != nil {
			return nil, errors.Errorf(errFmtRegister, err)
		}
	}

	return c, nil
}

/*
// unregisterCoreCmds unregisters all core commands. Made for testing.
func (c *coreCmds) unregisterCoreCmds() {
	for _, id := range c.ids {
		c.b.UnregisterCmd(id)
	}
}
*/

// Cmd is responsible for parsing all of the commands.
func (c *coreCmds) Cmd(cmd string, w irc.Writer,
	ev *cmd.Event) (internal error) {

	var external error

	c.b.Info(cmdExec, "cmd", cmd)

	defer func() {
		if r := recover(); r != nil {
			c.b.Error(errInternalPanic, "cmd", cmd, "panic", r)
			var buf [4096]byte
			n := runtime.Stack(buf[:], false)
			c.b.Error(fmt.Sprintf("%s", buf[:n]))
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

	pwd := ev.Args["password"]
	uname := ev.Args["username"]
	if len(uname) == 0 {
		uname = strings.TrimLeft(ev.Username(), "~")
	}

	host := ev.Sender
	nick := ev.Nick()

	store := c.b.Store()
	state := c.b.State(ev.NetworkID)

	access, internal = store.FindUser(uname)
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

	nChans, _ := state.NChannelsByUser(nick)

	var hasAny bool
	hasAny, internal = store.HasAny()
	if internal != nil {
		return
	}
	if !hasAny {
		// Secret from the Access map specifics
		access.Access[":"] = data.Access{Level: ^uint8(0), Flags: ^uint64(0)}
	}

	internal = store.SaveUser(access)
	if internal != nil {
		return
	}

	if nChans > 0 {
		_, internal = store.AuthUserPerma(ev.NetworkID, host, uname, pwd)
	} else {
		_, internal = store.AuthUserTmp(ev.NetworkID, host, uname, pwd)
	}
	if internal != nil {
		return
	}

	uname = strings.ToLower(uname)
	if !hasAny {
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
	pwd := ev.Args["password"]
	uname := ev.Args["username"]
	if len(uname) == 0 {
		uname = strings.TrimLeft(ev.Username(), "~")
	}

	state := c.b.State(ev.NetworkID)
	store := c.b.Store()

	host, nick := ev.Sender, ev.Nick()
	nChans, _ := state.NChannelsByUser(nick)

	access = ev.StoredUser
	if access == nil {
		access = store.AuthedUser(ev.NetworkID, host)
	}
	if access != nil {
		external = errors.New(errMsgAuthed)
		return
	}

	var err error
	if nChans > 0 {
		_, err = store.AuthUserPerma(ev.NetworkID, host, uname, pwd)
	} else {
		_, err = store.AuthUserTmp(ev.NetworkID, host, uname, pwd)
	}
	if err != nil {
		if authErr, ok := err.(data.AuthError); ok {
			external = authErr
		} else {
			internal = err
		}
		return
	}

	w.Noticef(nick, authSuccess, strings.ToLower(uname))
	return
}

// logout logs out a user.
func (c *coreCmds) logout(w irc.Writer, ev *cmd.Event) (
	internal, external error) {

	user := ev.TargetStoredUsers["user"]
	uname := ""
	host, nick := ev.Sender, ev.Nick()
	if user != nil {
		if !ev.StoredUser.HasFlags("", "", "G") {
			external = dispatch.MakeGlobalFlagsError("G")
			return
		}
		uname = user.Username
	}

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

	access := ev.TargetStoredUsers["user"]
	if access == nil {
		access = ev.StoredUser
	}

	ch := ""
	if ev.Channel != nil {
		ch = ev.Channel.Name
	}
	w.Noticef(ev.Nick(), accessSuccess,
		access.Username, access.String(ev.NetworkID, ch))

	return
}

//gusers provides a list of users with global access
func (c *coreCmds) gusers(w irc.Writer, ev *cmd.Event) (
	internal, external error) {

	var list []*data.StoredUser
	var ua *data.StoredUser

	nick := ev.Nick()

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
		w.Noticef(nick, usersList, usersWidth, ua.Username, ua.String("", ""))
	}

	return
}

//susers provides a list of users with network access
func (c *coreCmds) susers(w irc.Writer, ev *cmd.Event) (
	internal, external error) {

	var list []*data.StoredUser
	var ua *data.StoredUser

	nick := ev.Nick()

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
		w.Noticef(nick, usersList, usersWidth, ua.Username, ua.String(ev.NetworkID, ""))
	}

	return
}

// users provides a list of users added to a channel
func (c *coreCmds) users(w irc.Writer, ev *cmd.Event) (
	internal, external error) {

	var list []*data.StoredUser
	var ua *data.StoredUser
	var ch string

	if ev.Args["chan"] != `` {
		ch = ev.Args["chan"]
	} else if ev.Channel != nil && ev.Channel.Name != `` {
		ch = ev.Channel.Name
	} else {
		return
	}

	nick := ev.Nick()

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
		w.Noticef(nick, usersList, usersWidth, ua.Username, ua.String(ev.NetworkID, ch))
	}

	return
}

// deluser deletes a user
func (c *coreCmds) deluser(w irc.Writer, ev *cmd.Event) (
	internal, external error) {

	param := ev.Args["user"]
	if !ev.StoredUser.HasFlags("", "", "G") {
		external = dispatch.MakeGlobalFlagsError("G")
		return
	}
	uname := ev.TargetStoredUsers["user"].Username

	nick := ev.Nick()
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

	host, nick := ev.Sender, ev.Nick()
	uname := ev.StoredUser.Username
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

	oldpasswd := ev.Args["oldpassword"]
	newpasswd := ev.Args["newpassword"]
	nick := ev.Nick()
	uname := ev.StoredUser.Username
	if !ev.StoredUser.VerifyPassword(oldpasswd) {
		w.Notice(nick, passwdFailure)
		return
	}

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
	user := ev.TargetStoredUsers["user"]
	if user != nil {
		if !ev.StoredUser.HasFlags("", "", "G") {
			external = dispatch.MakeGlobalFlagsError("G")
			return
		}
		access = user
	}

	if len(access.Masks) > 0 {
		w.Noticef(ev.Nick(), masksSuccess,
			strings.Join(access.Masks, " "))
	} else {
		w.Notice(ev.Nick(), masksFailure)
	}

	return
}

// addmask adds a mask to a user.
func (c *coreCmds) addmask(w irc.Writer, ev *cmd.Event) (
	internal, external error) {

	mask := ev.Args["mask"]
	nick := ev.Nick()
	uname := ev.StoredUser.Username

	user := ev.TargetStoredUsers["user"]
	if user != nil {
		if !ev.StoredUser.HasFlags("", "", "G") {
			external = dispatch.MakeGlobalFlagsError("G")
			return
		}
		uname = user.Username
	}

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

	mask := ev.Args["mask"]
	nick := ev.Nick()
	uname := ev.StoredUser.Username

	user := ev.TargetStoredUsers["user"]
	if user != nil {
		if !ev.StoredUser.HasFlags("", "", "G") {
			external = dispatch.MakeGlobalFlagsError("G")
			return
		}
		uname = user.Username
	}

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

	if access.RemoveMask(mask) {
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

	uname := ev.TargetStoredUsers["user"].Username
	resetnick := ev.Args["nick"]
	nick := ev.Nick()
	newpasswd := ""

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

	return c.giveHelper(w, ev, "", "")
}

// sgive gives network access to a user.
func (c *coreCmds) sgive(w irc.Writer, ev *cmd.Event) (
	internal, external error) {

	network := ev.NetworkID
	return c.giveHelper(w, ev, network, "")
}

// give gives channel access to a user.
func (c *coreCmds) give(w irc.Writer,
	ev *cmd.Event) (internal, external error) {

	network := ev.NetworkID
	channel := ev.Args["chan"]
	return c.giveHelper(w, ev, network, channel)
}

// gtake takes global access from a user.
func (c *coreCmds) gtake(w irc.Writer, ev *cmd.Event) (
	internal, external error) {

	return c.takeHelper(w, ev, "", "")
}

// stake takes network access from a user.
func (c *coreCmds) stake(w irc.Writer, ev *cmd.Event) (
	internal, external error) {
	network := ev.NetworkID
	return c.takeHelper(w, ev, network, "")
}

// take takes global access from a user.
func (c *coreCmds) take(w irc.Writer, ev *cmd.Event) (
	internal, external error) {
	network := ev.NetworkID
	channel := ev.Args["chan"]
	return c.takeHelper(w, ev, network, channel)
}

// giveHelper parses the args to a give function and executes them in context
func (c *coreCmds) giveHelper(w irc.Writer, ev *cmd.Event,
	network, channel string) (internal, external error) {

	uname := ev.TargetStoredUsers["user"].Username
	args := ev.SplitArg("levelOrFlags")
	nick := ev.Nick()

	store := c.b.store

	var access *data.StoredUser
	if access, internal = store.FindUser(uname); internal != nil {
		return
	} else if access == nil {
		internal = fmt.Errorf(errFmtExpired, uname)
		return
	}

	username := access.Username
	a := ignoreOK(access.GetAccess(network, channel))

	var level uint8
	var flags, filtered string
	var hasFlags, hasLevel bool

	for _, arg := range args {
		if rgxFlags.MatchString(arg) {
			flags += arg
		} else if l, err := strconv.ParseUint(arg, 10, 8); err == nil {
			level = uint8(l)
		} else {
			w.Noticef(nick, giveFailure)
			return
		}
	}

	if (level <= 0 && level > 255) || len(flags) == 0 {
		w.Noticef(nick, giveFailure)
		return
	}

	if level > 0 {
		if a.HasLevel(level) {
			hasLevel = true
		} else {
			access.Grant(network, channel, level)
		}
	}

	if len(flags) != 0 {
		filtered = filterFlags(network, channel, flags, a)
		if len(filtered) == 0 {
			hasFlags = true
		} else {
			access.Grant(network, channel, 0, filtered)
		}
	}

	if (!hasFlags && len(flags) > 0) || (!hasLevel && level > 0) {
		if internal = store.SaveUser(access); internal == nil {
			var msg string
			var newAccess = ignoreOK(access.GetAccess(network, channel))
			switch {
			case len(network) != 0 && len(channel) != 0:
				msg = fmt.Sprintf(giveSuccess, username, newAccess, channel)
			case len(network) != 0:
				msg = fmt.Sprintf(sgiveSuccess, username, newAccess)
			default:
				msg = fmt.Sprintf(ggiveSuccess, username, newAccess)
			}
			w.Noticef(nick, msg)
		}
		return
	}

	var alreadyHas string
	if hasLevel {
		alreadyHas += fmt.Sprintf(">=%d", level)
	}
	if hasFlags {
		if len(alreadyHas) > 0 {
			alreadyHas += " "
		}
		alreadyHas = flags
	}
	w.Noticef(nick, giveFailureHas, access.Username, access.String(network, channel), alreadyHas)

	return
}

// takeHelper parses the args to a take function and executes them in context
func (c *coreCmds) takeHelper(w irc.Writer, ev *cmd.Event,
	network, channel string) (internal, external error) {

	uname := ev.TargetStoredUsers["user"].Username
	arg := ev.Args["allOrFlags"]
	nick := ev.Nick()

	store := c.b.store

	var access *data.StoredUser
	if access, internal = store.FindUser(uname); internal != nil {
		return
	} else if access == nil {
		internal = fmt.Errorf(errFmtExpired, uname)
		return
	}

	username := access.Username
	a := ignoreOK(access.GetAccess(network, channel))

	var all, level bool
	var flags string

	if len(arg) == 0 {
		level = true
	} else if arg == takeAllArg {
		all = true
	} else if rgxFlags.MatchString(arg) {
		flags = filterMissingFlags(network, channel, arg, a)
	} else {
		external = fmt.Errorf(takeFailure, arg)
	}

	var save = true
	if all && (a.HasLevel(1) || len(flags) != 0) {
		access.RevokeLevel(network, channel)
		access.RevokeFlags(network, channel)
	} else if level && a.HasLevel(1) {
		access.RevokeLevel(network, channel)
	} else if len(flags) != 0 {
		access.RevokeFlags(network, channel, flags)
	} else {
		save = false
	}

	if save {
		if internal = store.SaveUser(access); internal == nil {
			var msg string
			var newAccess = ignoreOK(access.GetAccess(network, channel))
			switch {
			case len(network) != 0 && len(channel) != 0:
				msg = fmt.Sprintf(giveSuccess, username, newAccess, channel)
			case len(network) != 0:
				msg = fmt.Sprintf(sgiveSuccess, username, newAccess)
			default:
				msg = fmt.Sprintf(ggiveSuccess, username, newAccess)
			}
			w.Notice(nick, msg)
		}
		return
	}

	w.Noticef(nick, takeFailureNo, username, access.String(network, channel))
	return
}

// help searches for commands, and also provides details for specific commands
func (c *coreCmds) help(w irc.Writer, ev *cmd.Event) (
	internal, external error) {

	search := strings.ToLower(ev.Args["command"])
	nick := ev.Nick()

	var extSorted = make(map[string][]string)
	var fqMatches []*cmd.Command
	var exactMatches []*cmd.Command
	var fuzzyMatches []*cmd.Command
	var extMatches []string

	c.b.cmds.EachCmd("", "", func(command *cmd.Command) bool {
		full := command.Extension + "." + command.Name

		if search == full {
			fqMatches = append(fqMatches, command)
			return false
		}

		shouldOutput := false

		if search == command.Name {
			exactMatches = append(exactMatches, command)
			shouldOutput = true
		} else if strings.Contains(command.Name, search) {
			fuzzyMatches = append(fuzzyMatches, command)
			shouldOutput = true
		}

		if strings.Contains(command.Extension, search) {
			extMatches = append(extMatches, command.Extension)
			shouldOutput = true
		}

		if shouldOutput {
			arr := extSorted[command.Extension]
			extSorted[command.Extension] = append(arr, command.Name)
		}

		return false
	})

	if len(exactMatches) > 0 && len(fqMatches) == 0 &&
		len(fuzzyMatches) == 0 && len(extMatches) == 0 {

		fqMatches = exactMatches
	}

	switch {
	case len(fqMatches) >= 1:
		for _, i := range fqMatches {
			w.Notice(nick, helpSuccess, " ", i.Extension, ".", i.Name)
			w.Notice(nick, i.Description)
			if len(i.Args) == 0 {
				continue
			}
			w.Noticef(nick, helpSuccessUsage, i.Name, strings.Join(i.Args, " "))
		}
	case len(exactMatches) > 0 || len(fuzzyMatches) > 0 || len(extMatches) > 0:
		for extension, commands := range extSorted {
			sort.Strings(commands)
			w.Notice(nick, extension, ":")
			w.Notice(nick, " ", strings.Join(commands, " "))
		}
	default:
		w.Noticef(nick, helpFailure, search)
	}

	return
}

// filterFlags removes flags that the user already has
func filterFlags(network, channel, flags string, access data.Access) string {
	var buf []rune
	for _, flag := range flags {
		if !access.HasFlags(network, channel, string(flag)) {
			buf = append(buf, flag)
		}
	}
	return string(buf)
}

// filterMissingFlags removes flags that the user does not have
func filterMissingFlags(network, channel, flags string, access data.Access) string {
	var buf []rune
	for _, flag := range flags {
		if access.HasFlags(network, channel, string(flag)) {
			buf = append(buf, flag)
		}
	}
	return string(buf)
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

// ignoreOK allows us to easily return an access object even if it's zero'd
func ignoreOK(access data.Access, ok bool) data.Access {
	return access
}
