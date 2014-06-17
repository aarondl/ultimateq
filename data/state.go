/*
Package data is used to store and fetch data about the irc environment.

It comes in three major pieces; State, Store, Locker. These are the main types
that provide access to all others.

Locker

We have to start with a discussion on how before we get to what. The data
package assumes that the world is pleasant and provides no protections for
concurrent access via multiple goroutines. The solution to this problem for
clients that wish to use the data package in a concurrent world is to implement
the data.Locker interface.

ultimateq/bot.Bot is one such implementing client. The following example uses
the locker interface on the bot.Bot type to safely access the state database
for the testnetwork.

	// Where b is a *bot.Bot from the ultimateq/bot package.
	b.ReadState("testnetwork", func(state *data.State) {
		// Do what we have to with state. For the duration of this function
		// it is safe to read from.
	})

If you find the lambda syntax burdensome, then you may use the alternative
syntax:

	store := b.OpenWriteStore()
	defer b.CloseWriteStore()

Keep in mind that since there is locking, that means that users consuming the
interface must be mindful to keep locks for as short a duration as possible.
The locks are read-writer locks so multiple readers can access in parallel but
it's not possible to update during this time and something bad could happen
if a reader doesn't allow the writer to update for an extended period of time.

State

State is the state about the irc world. Who is online, in what channel, what
modes are on the channel or user, what topic is set. All of this information
is readable (not writeable) using State. This is per-network data, so each
network has it's own State database.

When using state the important types are: User, Channel, ChannelUser and
UserChannel. These types provide you with the information, and the many
state.Get* methods can retrieve instances of these types to query.
Examples follow.

	// The client's personal information is stored in the Self instance.
	mynick := state.Self.Nick()

	state.EachChannel(func (ch *Channel) {
		fmt.Println(ch.Name)
	})

	user := state.GetUser("nick!user@host") //Can look up by host or nick.
	if user != nil {
		fmt.Println(user)
	}

Store

Store is about writing persisted data, and authenticating stored entities.
Store is interested in persisting two types of objects: StoredChannels and
StoredUsers. Both types embed a JSONStorer to store extension-specific data and
can be used in any way that's desireable by extensions to persist their data
across sessions.

StoredChannel is simply a JSONStorer with the channel and network ID to separate
it in the key value database. JSONStorer is a map that can be used directly
or using the marshalling helpers PutJSON and GetJSON.

	sc := NewStoredChannel(networkID, "#channelname")

	// Store an array
	err := sc.JSONPut("myfriends", []string{"happy", "go", "lucky"})
	store.SaveChannel(sc)

	// Retrieve the array
	sc = store.FindChannel(networkID, "#channelname")
	var array []string
	found, err := sc.JSONGet("myfriends", &array)

StoredUser in addition to it's JSONStorer interface, has a multi-tiered user
access scheme. There is the potential for a global level, for each network,
and for each channel to have it's own Access that defines a set of
permissions (level and flags). These permissions cascade down so that when
querying the permissions of the channel, the global and network permissions will
also be present. These permissions are protected by a username and crypted
password as well as optional whitelist host masks.
The Store can authenticate against all of these credentials, see Authentication
section below.

	su := store.FindUser("username")

	// Check some permissions
	hasGoodEnougLevel := su.HasChannelLevel(networkID, "#channelname", 100)
	global := su.GetGlobal()
	fmt.Println(global.HasFlags("abc"))

	// Write some permissions
	su.GrantGlobalFlags("abc")
	su.RevokeChannelFlags(networkID, "#channelname", "a")

	// Must save afterwards.
	store.SaveUser(su)

Authentication is done by the store for users in order to become "authed" and
succeed in subsequent GetAuthedUser() calls, a user must succeed in an AuthUser
call providing the username and password as well a host to bind this
authentication to. Use this system by calling the functions: AuthUser,
GetAuthedUser, Logout.
*/
package data

import (
	"errors"
	"strings"

	"github.com/aarondl/ultimateq/irc"
)

var (
	errProtoCapsMissing = errors.New("data: Protocaps missing")
)

// Self is the client's user, a special case since user modes must be stored.
// Despite using the ChannelModes type, these are actually irc user modes that
// have nothing to do with channels. For example +i or +x on some networks.
type Self struct {
	*User
	*ChannelModes
}

// State is the main data container. It represents the state on a network
// including all channels, users, and the client's self.
type State struct {
	Self Self

	channels map[string]*Channel
	users    map[string]*User

	channelUsers map[string]map[string]*ChannelUser
	userChannels map[string]map[string]*UserChannel

	kinds  ChannelModeKinds
	umodes UserModeKinds
}

// NewState creates a state from an irc protocaps instance.
func NewState(netInfo *irc.NetworkInfo) (*State, error) {
	state := &State{}
	if err := state.SetNetworkInfo(netInfo); err != nil {
		return nil, err
	}
	state.Self.ChannelModes = NewChannelModes(&ChannelModeKinds{}, nil)

	state.channels = make(map[string]*Channel)
	state.users = make(map[string]*User)
	state.channelUsers = make(map[string]map[string]*ChannelUser)
	state.userChannels = make(map[string]map[string]*UserChannel)

	return state, nil
}

// SetNetworkInfo updates the network information of the state.
func (s *State) SetNetworkInfo(ni *irc.NetworkInfo) error {
	if ni == nil {
		return errProtoCapsMissing
	}
	kinds, err := NewChannelModeKindsCSV(ni.Chanmodes())
	if err != nil {
		return err
	}
	modes, err := NewUserModeKinds(ni.Prefix())
	if err != nil {
		return err
	}

	s.kinds = *kinds
	s.umodes = *modes
	return nil
}

// GetUser returns the user if he exists.
func (s *State) GetUser(nickorhost string) *User {
	nick := strings.ToLower(irc.Nick(nickorhost))
	return s.users[nick]
}

// GetChannel returns the channel if it exists.
func (s *State) GetChannel(channel string) *Channel {
	return s.channels[strings.ToLower(channel)]
}

// GetUsersChannelModes gets the user modes for the channel or nil if they could
// not be found.
func (s *State) GetUsersChannelModes(nickorhost, channel string) *UserModes {
	nick := strings.ToLower(irc.Nick(nickorhost))
	channel = strings.ToLower(channel)

	if nicks, ok := s.channelUsers[channel]; ok {
		if cu, ok := nicks[nick]; ok {
			return cu.UserModes
		}
	}

	return nil
}

// GetNUsers returns the number of users in the database.
func (s *State) GetNUsers() int {
	return len(s.users)
}

// GetNChannels returns the number of channels in the database.
func (s *State) GetNChannels() int {
	return len(s.channels)
}

// GetNUserChans returns the number of channels for a user in the database.
func (s *State) GetNUserChans(nickorhost string) (n int) {
	nick := strings.ToLower(irc.Nick(nickorhost))
	if ucs, ok := s.userChannels[nick]; ok {
		n = len(ucs)
	}
	return
}

// GetNChanUsers returns the number of users for a channel in the database.
func (s *State) GetNChanUsers(channel string) (n int) {
	channel = strings.ToLower(channel)
	if cus, ok := s.channelUsers[channel]; ok {
		n = len(cus)
	}
	return
}

// EachUser iterates through the users.
func (s *State) EachUser(fn func(*User)) {
	for _, u := range s.users {
		fn(u)
	}
}

// EachChannel iterates through the channels.
func (s *State) EachChannel(fn func(*Channel)) {
	for _, c := range s.channels {
		fn(c)
	}
}

// EachUserChan iterates through the channels a user is on.
func (s *State) EachUserChan(nickorhost string, fn func(*UserChannel)) {
	nick := strings.ToLower(irc.Nick(nickorhost))
	if ucs, ok := s.userChannels[nick]; ok {
		for _, uc := range ucs {
			fn(uc)
		}
	}
	return
}

// EachChanUser iterates through the users on a channel.
func (s *State) EachChanUser(channel string, fn func(*ChannelUser)) {
	channel = strings.ToLower(channel)
	if cus, ok := s.channelUsers[channel]; ok {
		for _, cu := range cus {
			fn(cu)
		}
	}
	return
}

// GetUsers returns a string array of all the users.
func (s *State) GetUsers() []string {
	ret := make([]string, 0, len(s.users))
	for _, u := range s.users {
		ret = append(ret, u.Host())
	}
	return ret
}

// GetChannels returns a string array of all the channels.
func (s *State) GetChannels() []string {
	ret := make([]string, 0, len(s.channels))
	for _, c := range s.channels {
		ret = append(ret, c.Name())
	}
	return ret
}

// GetUserChans returns a string array of the channels a user is on.
func (s *State) GetUserChans(nickorhost string) []string {
	nick := strings.ToLower(irc.Nick(nickorhost))
	if ucs, ok := s.userChannels[nick]; ok {
		ret := make([]string, 0, len(ucs))
		for _, uc := range ucs {
			ret = append(ret, uc.Channel.Name())
		}
		return ret
	}
	return nil
}

// GetChanUsers returns a string array of the users on a channel.
func (s *State) GetChanUsers(channel string) []string {
	channel = strings.ToLower(channel)
	if cus, ok := s.channelUsers[channel]; ok {
		ret := make([]string, 0, len(cus))
		for _, cu := range cus {
			ret = append(ret, cu.User.Host())
		}
		return ret
	}
	return nil
}

// IsOn checks if a user is on a specific channel.
func (s *State) IsOn(nickorhost, channel string) bool {
	nick := strings.ToLower(irc.Nick(nickorhost))
	channel = strings.ToLower(channel)

	if chans, ok := s.userChannels[nick]; ok {
		_, ok = chans[channel]
		return ok
	}
	return false
}

// addUser adds a user to the database.
func (s *State) addUser(nickorhost string) *User {
	excl, at, per := false, false, false
	for i := 0; i < len(nickorhost); i++ {
		switch nickorhost[i] {
		case '!':
			excl = true
		case '@':
			at = true
		case '.':
			per = true
		}
	}

	if per && !(excl && at) {
		return nil
	}

	nick := strings.ToLower(irc.Nick(nickorhost))
	var user *User
	var ok bool
	if user, ok = s.users[nick]; ok {
		if excl && at && user.Host() != nickorhost {
			user.host = irc.Host(nickorhost)
		}
	} else {
		user = NewUser(nickorhost)
		s.users[nick] = user
	}
	return user
}

// removeUser deletes a user from the database.
func (s *State) removeUser(nickorhost string) {
	nick := strings.ToLower(irc.Nick(nickorhost))
	for _, cus := range s.channelUsers {
		delete(cus, nick)
	}

	delete(s.userChannels, nick)
	delete(s.users, nick)
}

// addChannel adds a channel to the database.
func (s *State) addChannel(channel string) *Channel {
	chankey := strings.ToLower(channel)
	var ch *Channel
	if ch, ok := s.channels[chankey]; !ok {
		ch = NewChannel(channel, &s.kinds, &s.umodes)
		s.channels[chankey] = ch
	}
	return ch
}

// removeChannel deletes a channel from the database.
func (s *State) removeChannel(channel string) {
	channel = strings.ToLower(channel)
	for _, cus := range s.userChannels {
		delete(cus, channel)
	}

	delete(s.channelUsers, channel)
	delete(s.channels, channel)
}

// addToChannel adds a user by nick or fullhost to the channel
func (s *State) addToChannel(nickorhost, channel string) {
	var user *User
	var ch *Channel
	var cu map[string]*ChannelUser
	var uc map[string]*UserChannel
	var ok, cuhas, uchas bool

	nick := strings.ToLower(irc.Nick(nickorhost))
	channel = strings.ToLower(channel)

	if user, ok = s.users[nick]; !ok {
		return
	}

	if ch, ok = s.channels[channel]; !ok {
		return
	}

	if cu, ok = s.channelUsers[channel]; !ok {
		cu = make(map[string]*ChannelUser, 1)
	} else {
		_, cuhas = s.channelUsers[channel][nick]
	}

	if uc, ok = s.userChannels[nick]; !ok {
		uc = make(map[string]*UserChannel, 1)
	} else {
		_, uchas = s.userChannels[nick][channel]
	}

	if cuhas || uchas {
		return
	}

	modes := NewUserModes(&s.umodes)
	cu[nick] = NewChannelUser(user, modes)
	uc[channel] = NewUserChannel(ch, modes)
	s.channelUsers[channel] = cu
	s.userChannels[nick] = uc
}

// removeFromChannel removes a user by nick or fullhost from the channel
func (s *State) removeFromChannel(nickorhost, channel string) {
	var cu map[string]*ChannelUser
	var uc map[string]*UserChannel
	var ok bool

	nick := strings.ToLower(irc.Nick(nickorhost))
	channel = strings.ToLower(channel)

	if cu, ok = s.channelUsers[channel]; ok {
		delete(cu, nick)
	}

	if uc, ok = s.userChannels[nick]; ok {
		delete(uc, channel)
	}
}

// Update uses the irc.IrcMessage to modify the database accordingly.
func (s *State) Update(ev *irc.Event) {
	if len(ev.Sender) > 0 {
		s.addUser(ev.Sender)
	}
	switch ev.Name {
	case irc.NICK:
		s.nick(ev)
	case irc.JOIN:
		s.join(ev)
	case irc.PART:
		s.part(ev)
	case irc.QUIT:
		s.quit(ev)
	case irc.KICK:
		s.kick(ev)
	case irc.MODE:
		s.mode(ev)
	case irc.TOPIC:
		s.topic(ev)
	case irc.RPL_TOPIC:
		s.rplTopic(ev)
	case irc.PRIVMSG, irc.NOTICE:
		s.msg(ev)
	case irc.RPL_WELCOME:
		s.rplWelcome(ev)
	case irc.RPL_NAMREPLY:
		s.rplNameReply(ev)
	case irc.RPL_WHOREPLY:
		s.rplWhoReply(ev)
	case irc.RPL_CHANNELMODEIS:
		s.rplChannelModeIs(ev)
	case irc.RPL_BANLIST:
		s.rplBanList(ev)

		// TODO: Handle Whois
	}
}

// nick alters the state of the database when a NICK message is received.
func (s *State) nick(ev *irc.Event) {
	nick, username, host := ev.SplitHost()
	newnick := ev.Args[0]
	newuser := irc.Host(newnick + "!" + username + "@" + host)

	nick = strings.ToLower(nick)
	newnick = strings.ToLower(newnick)

	if user, ok := s.users[nick]; ok {
		user.host = newuser
		for _, cus := range s.channelUsers {
			if _, ok := cus[nick]; ok {
				cus[newnick] = cus[nick]
				delete(cus, nick)
			}
		}
		if _, ok := s.userChannels[nick]; ok {
			s.userChannels[newnick] = s.userChannels[nick]
			delete(s.userChannels, nick)
		}
		s.users[newnick] = s.users[nick]
		delete(s.users, nick)
	}
}

// join alters the state of the database when a JOIN message is received.
func (s *State) join(ev *irc.Event) {
	if ev.Sender == s.Self.Host() {
		s.addChannel(ev.Args[0])
	}
	s.addToChannel(ev.Sender, ev.Args[0])
}

// part alters the state of the database when a PART message is received.
func (s *State) part(ev *irc.Event) {
	if ev.Sender == s.Self.Host() {
		s.removeChannel(ev.Args[0])
	} else {
		s.removeFromChannel(ev.Sender, ev.Args[0])
	}
}

// quit alters the state of the database when a QUIT message is received.
func (s *State) quit(ev *irc.Event) {
	if ev.Sender != s.Self.Host() {
		s.removeUser(ev.Sender)
	}
}

// kick alters the state of the database when a KICK message is received.
func (s *State) kick(ev *irc.Event) {
	if ev.Args[1] == s.Self.Nick() {
		s.removeChannel(ev.Args[0])
	} else {
		s.removeFromChannel(ev.Args[1], ev.Args[0])
	}
}

// mode alters the state of the database when a MODE message is received.
func (s *State) mode(ev *irc.Event) {
	target := strings.ToLower(ev.Args[0])
	if ev.IsTargetChan() {
		if ch, ok := s.channels[target]; ok {
			pos, neg := ch.Apply(strings.Join(ev.Args[1:], " "))
			for i := 0; i < len(pos); i++ {
				nick := strings.ToLower(pos[i].Arg)
				s.channelUsers[target][nick].SetMode(pos[i].Mode)
			}
			for i := 0; i < len(neg); i++ {
				nick := strings.ToLower(neg[i].Arg)
				s.channelUsers[target][nick].UnsetMode(neg[i].Mode)
			}
		}
	} else if target == s.Self.Nick() {
		s.Self.Apply(ev.Args[1])
	}
}

// topic alters the state of the database when a TOPIC message is received.
func (s *State) topic(ev *irc.Event) {
	chname := strings.ToLower(ev.Args[0])
	if ch, ok := s.channels[chname]; ok {
		ch.SetTopic(ev.Args[1])
	}
}

// rplTopic alters the state of the database when a RPL_TOPIC message is
// received.
func (s *State) rplTopic(ev *irc.Event) {
	chname := strings.ToLower(ev.Args[1])
	if ch, ok := s.channels[chname]; ok {
		ch.SetTopic(ev.Args[2])
	}
}

// msg alters the state of the database when a PRIVMSG or NOTICE message is
// received.
func (s *State) msg(ev *irc.Event) {
	if ev.IsTargetChan() {
		s.addToChannel(ev.Sender, ev.Args[0])
	}
}

// rplWelcome alters the state of the database when a RPL_WELCOME message is
// received.
func (s *State) rplWelcome(ev *irc.Event) {
	splits := strings.Fields(ev.Args[1])
	host := splits[len(splits)-1]

	if !strings.ContainsRune(host, '!') || !strings.ContainsRune(host, '@') {
		host = ev.Args[0]
	}
	user := NewUser(host)
	s.Self.User = user
	s.users[strings.ToLower(user.Nick())] = user
}

// rplNameReply alters the state of the database when a RPL_NAMEREPLY
// message is received.
func (s *State) rplNameReply(ev *irc.Event) {
	channel := ev.Args[2]
	users := strings.Fields(ev.Args[3])
	for i := 0; i < len(users); i++ {
		j := 0
		mode := rune(0)
		for ; j < len(s.umodes.modeInfo); j++ {
			if s.umodes.modeInfo[j][1] == rune(users[i][0]) {
				mode = s.umodes.modeInfo[j][0]
				break
			}
		}
		if j < len(s.umodes.modeInfo) {
			nick := users[i][1:]
			s.addUser(nick)
			s.addToChannel(nick, channel)
			s.GetUsersChannelModes(nick, channel).SetMode(mode)
		} else {
			s.addUser(users[i])
			s.addToChannel(users[i], channel)
		}
	}
}

// rplWhoReply alters the state of the database when a RPL_WHOREPLY message
// is received.
func (s *State) rplWhoReply(ev *irc.Event) {
	channel := ev.Args[1]
	fullhost := ev.Args[5] + "!" + ev.Args[2] + "@" + ev.Args[3]
	modes := ev.Args[6]
	realname := strings.SplitN(ev.Args[7], " ", 2)[1]

	s.addUser(fullhost)
	s.addToChannel(fullhost, channel)
	s.GetUser(fullhost).SetRealname(realname)
	for _, modechar := range modes {
		if mode := s.umodes.GetMode(modechar); mode != 0 {
			if uc := s.GetUsersChannelModes(fullhost, channel); uc != nil {
				uc.SetMode(mode)
			}
		}
	}
}

// rplChannelModeIs alters the state of the database when a RPL_CHANNELMODEIS
// message is received.
func (s *State) rplChannelModeIs(ev *irc.Event) {
	channel := ev.Args[1]
	modes := strings.Join(ev.Args[2:], " ")
	if ch := s.GetChannel(channel); ch != nil {
		ch.Apply(modes)
	}
}

// rplBanList alters the state of the database when a RPL_BANLIST message is
// received.
func (s *State) rplBanList(ev *irc.Event) {
	channel := ev.Args[1]
	if ch := s.GetChannel(channel); ch != nil {
		ch.AddBan(ev.Args[2])
	}
}
