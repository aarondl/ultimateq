/*
Package data is used to store and fetch data about the irc environment.

It comes in two major pieces; State and Store. These are the main types
that provide access to all others.

State

State is the state about the irc world. Who is online, in what channel, what
modes are on the channel or user, what topic is set. All of this information
is readable (not writeable) using State. This is per-network data, so each
network has it's own State database.

When using state the important types are: User and Channel. These types provide
you with the information about the entities themselves. Relationships like
the modes a user has on a channel are retrieved by special helpers.
Examples follow.

	// The client's personal information is stored in the Self instance.
	mynick := state.selfUser.Nick()

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
	hasGoodEnoughLevel := su.HasChannelLevel(networkID, "#channelname", 100)
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
	"sync"

	"github.com/aarondl/ultimateq/irc"
)

var (
	errNetInfoMissing = errors.New("data: NetworkInfo missing")
)

// Self is the client's user, a special case since user modes must be stored.
// Despite using the ChannelModes type, these are actually irc user modes that
// have nothing to do with channels. For example +i or +x on some networks.
type Self struct {
	User
	ChannelModes
}

// State is the main data container. It represents the state on a network
// including all channels, users, and the client's self.
type State struct {
	selfUser  *User
	selfModes ChannelModes

	channels map[string]*Channel
	users    map[string]*User

	channelUsers map[string]map[string]channelUser
	userChannels map[string]map[string]userChannel

	kinds *modeKinds

	protect sync.RWMutex
}

// NewState creates a state from an irc.NetworkInfo instance.
func NewState(netInfo *irc.NetworkInfo) (*State, error) {
	state := &State{}
	if err := state.SetNetworkInfo(netInfo); err != nil {
		return nil, err
	}
	state.selfModes = NewChannelModes(&modeKinds{})

	state.channels = make(map[string]*Channel)
	state.users = make(map[string]*User)
	state.channelUsers = make(map[string]map[string]channelUser)
	state.userChannels = make(map[string]map[string]userChannel)

	return state, nil
}

// SetNetworkInfo updates the network information of the state.
func (s *State) SetNetworkInfo(ni *irc.NetworkInfo) error {
	s.protect.Lock()
	defer s.protect.Unlock()

	if ni == nil {
		return errNetInfoMissing
	}

	if s.kinds != nil {
		return s.kinds.update(ni.Prefix(), ni.Chanmodes())
	}

	kinds, err := newModeKinds(ni.Prefix(), ni.Chanmodes())
	if err != nil {
		return err
	}
	s.kinds = kinds
	return nil
}

// Self retrieves the user that the state identifies itself with. Usually the
// client that is using the data package.
func (s *State) Self() Self {
	return Self{*s.selfUser, s.selfModes.Clone()}
}

// User fetches a user by nickname or host if he exists. The bool returned
// is false if the user does not exist.
func (s *State) User(nickorhost string) (User, bool) {
	s.protect.RLock()
	defer s.protect.RUnlock()

	nick := strings.ToLower(irc.Nick(nickorhost))
	var user User
	u, ok := s.users[nick]
	if ok {
		user = *u
	}
	return user, ok
}

// Channel returns a channel by name if it exists. The bool returned is false
// if the channel does not exist.
func (s *State) Channel(channel string) (Channel, bool) {
	s.protect.RLock()
	defer s.protect.RUnlock()

	var ch Channel
	c, ok := s.channels[strings.ToLower(channel)]
	if ok {
		ch = *c.Clone()
	}
	return ch, ok
}

// UserModes gets the channel modes of a nick or host for the given channel.
// The bool returned is false if the user or the channel does not exist.
func (s *State) UserModes(nickorhost, channel string) (UserModes, bool) {
	s.protect.RLock()
	defer s.protect.RUnlock()

	var modes UserModes
	if channelModes := s.userModes(nickorhost, channel); channelModes != nil {
		modes = *channelModes
		return modes, true
	}

	return modes, false
}

// NUsers returns the number of users in the database.
func (s *State) NUsers() int {
	s.protect.RLock()
	defer s.protect.RUnlock()

	return len(s.users)
}

// NChannels returns the number of channels in the database.
func (s *State) NChannels() int {
	s.protect.RLock()
	defer s.protect.RUnlock()

	return len(s.channels)
}

// NChannelsByUser returns the number of channels for a user in the database.
// The returned bool is false if the user doesn't exist.
func (s *State) NChannelsByUser(nickorhost string) (n int, ok bool) {
	s.protect.RLock()
	defer s.protect.RUnlock()

	var ucs map[string]userChannel
	nick := strings.ToLower(irc.Nick(nickorhost))
	if ucs, ok = s.userChannels[nick]; ok {
		n = len(ucs)
	}
	return n, ok
}

// NUsersByChannel returns the number of users for a channel in the database.
// The returned bool is false if the channel doesn't exist.
func (s *State) NUsersByChannel(channel string) (n int, ok bool) {
	s.protect.RLock()
	defer s.protect.RUnlock()

	var cus map[string]channelUser
	channel = strings.ToLower(channel)
	if cus, ok = s.channelUsers[channel]; ok {
		n = len(cus)
	}
	return n, ok
}

// EachUser iterates through the users.
// To stop iteration early return true from the fn function parameter.
func (s *State) EachUser(fn func(User) bool) {
	s.protect.RLock()
	defer s.protect.RUnlock()

	for _, u := range s.users {
		if fn(*u) {
			break
		}
	}
}

// EachChannel iterates through the channels.
// To stop iteration early return true from the fn function parameter.
func (s *State) EachChannel(fn func(Channel) bool) {
	s.protect.RLock()
	defer s.protect.RUnlock()

	for _, c := range s.channels {
		if fn(*c.Clone()) {
			break
		}
	}
}

// Users returns a string array of all the users.
func (s *State) Users() []string {
	s.protect.RLock()
	defer s.protect.RUnlock()

	ret := make([]string, 0, len(s.users))
	for _, u := range s.users {
		ret = append(ret, u.Host.String())
	}
	return ret
}

// Channels returns a string array of all the channels.
func (s *State) Channels() []string {
	s.protect.RLock()
	defer s.protect.RUnlock()

	ret := make([]string, 0, len(s.channels))
	for _, c := range s.channels {
		ret = append(ret, c.Name)
	}
	return ret
}

// ChannelsByUser returns a string array of the channels a user is on.
func (s *State) ChannelsByUser(nickorhost string) []string {
	s.protect.RLock()
	defer s.protect.RUnlock()

	nick := strings.ToLower(irc.Nick(nickorhost))
	if ucs, ok := s.userChannels[nick]; ok {
		ret := make([]string, 0, len(ucs))
		for _, uc := range ucs {
			ret = append(ret, uc.Channel.Name)
		}
		return ret
	}
	return nil
}

// UsersByChannel returns a string array of the users on a channel.
func (s *State) UsersByChannel(channel string) []string {
	s.protect.RLock()
	defer s.protect.RUnlock()

	channel = strings.ToLower(channel)
	if cus, ok := s.channelUsers[channel]; ok {
		ret := make([]string, 0, len(cus))
		for _, cu := range cus {
			ret = append(ret, cu.User.Host.String())
		}
		return ret
	}
	return nil
}

// IsOn checks if a user is on a specific channel.
func (s *State) IsOn(nickorhost, channel string) bool {
	s.protect.RLock()
	defer s.protect.RUnlock()

	nick := strings.ToLower(irc.Nick(nickorhost))
	channel = strings.ToLower(channel)

	if chans, ok := s.userChannels[nick]; ok {
		_, ok = chans[channel]
		return ok
	}
	return false
}

// user looks up a user without locking.
func (s *State) user(nickorhost string) *User {
	return s.users[strings.ToLower(irc.Nick(nickorhost))]
}

// channel looks up a channel without locking.
func (s *State) channel(name string) *Channel {
	return s.channels[strings.ToLower(irc.Nick(name))]
}

// userModes does the same thing as UserModes without locks.
func (s *State) userModes(nickorhost, channel string) *UserModes {
	nick := strings.ToLower(irc.Nick(nickorhost))
	channel = strings.ToLower(channel)

	if nicks, ok := s.channelUsers[channel]; ok {
		if cu, ok := nicks[nick]; ok {
			return cu.UserModes
		}
	}

	return nil
}

// addUser adds a user to the database.
func (s *State) addUser(nickorhost string) bool {
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
		return false
	}

	nick := strings.ToLower(irc.Nick(nickorhost))
	var user *User
	var ok bool
	if user, ok = s.users[nick]; ok {
		if excl && at && user.Host.String() != nickorhost {
			user.Host = irc.Host(nickorhost)
		}
	} else {
		user = NewUser(nickorhost)
		s.users[nick] = user
		return true
	}
	return false
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
	var ok bool
	if ch, ok = s.channels[chankey]; !ok {
		ch = NewChannel(channel, s.kinds)
		s.channels[chankey] = ch
	}
	return ch
}

// removeChannel deletes a channel from the database.
func (s *State) removeChannel(channel string) (unseen []string) {
	channel = strings.ToLower(channel)
	for _, cus := range s.userChannels {
		delete(cus, channel)
	}

	for _, cu := range s.channelUsers[channel] {
		nick := strings.ToLower(cu.User.Nick())
		if nick == s.selfUser.Nick() {
			continue
		}
		if ucs, ok := s.userChannels[nick]; ok {
			if len(ucs) == 0 {
				unseen = append(unseen, string(cu.User.Host))
				delete(s.users, strings.ToLower(nick))
			}
		}
	}
	delete(s.channelUsers, channel)
	delete(s.channels, channel)

	return unseen
}

// addToChannel adds a user by nick or fullhost to the channel
func (s *State) addToChannel(nickorhost, channel string) {
	var user *User
	var ch *Channel
	var cu map[string]channelUser
	var uc map[string]userChannel
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
		cu = make(map[string]channelUser, 1)
	} else {
		_, cuhas = s.channelUsers[channel][nick]
	}

	if uc, ok = s.userChannels[nick]; !ok {
		uc = make(map[string]userChannel, 1)
	} else {
		_, uchas = s.userChannels[nick][channel]
	}

	if cuhas || uchas {
		return
	}

	modes := NewUserModes(s.kinds)
	cu[nick] = newChannelUser(user, &modes)
	uc[channel] = newUserChannel(ch, &modes)
	s.channelUsers[channel] = cu
	s.userChannels[nick] = uc
}

// removeFromChannel removes a user by nick or fullhost from the channel
func (s *State) removeFromChannel(nickorhost, channel string) {
	var cu map[string]channelUser
	var uc map[string]userChannel
	var ok bool

	nick := strings.ToLower(irc.Nick(nickorhost))
	channel = strings.ToLower(channel)

	if cu, ok = s.channelUsers[channel]; ok {
		delete(cu, nick)
	}

	shouldRemove := false
	if uc, ok = s.userChannels[nick]; ok {
		delete(uc, channel)

		shouldRemove = len(uc) == 0
	}

	if shouldRemove {
		s.removeUser(nick)
	}
}

// StateUpdate is produced by the Update method to summarize the updates made to
// the database. Although anything can use this information it's created so it
// can in turn be passed into the store for processing users who have
// been renamed or disappeared.
type StateUpdate struct {
	Nick   []string
	Unseen []string
	Seen   []string
	Quit   string
}

// Update uses the irc.IrcMessage to modify the database accordingly.
func (s *State) Update(ev *irc.Event) (update StateUpdate) {
	s.protect.Lock()
	defer s.protect.Unlock()

	switch ev.Name {
	case irc.NICK:
		update.Nick = s.nick(ev)
	case irc.JOIN:
		update.Seen = s.join(ev)
	case irc.PART:
		update.Unseen = s.part(ev)
	case irc.QUIT:
		update.Quit = s.quit(ev)
	case irc.KICK:
		update.Seen, update.Unseen = s.kick(ev)
	case irc.MODE:
		update.Seen = s.mode(ev)
	case irc.TOPIC:
		update.Seen = s.topic(ev)
	case irc.RPL_TOPIC:
		s.rplTopic(ev)
	case irc.PRIVMSG, irc.NOTICE:
		update.Seen = s.msg(ev)
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

	return update
}

// nick alters the state of the database when a NICK message is received.
func (s *State) nick(ev *irc.Event) []string {
	s.addUser(ev.Sender)
	nick, username, host := ev.SplitHost()
	newnick := ev.Args[0]
	newuser := irc.Host(newnick + "!" + username + "@" + host)

	nick = strings.ToLower(nick)
	newnick = strings.ToLower(newnick)

	if user, ok := s.users[nick]; ok {
		user.Host = newuser
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

	return []string{ev.Sender, string(newuser)}
}

// join alters the state of the database when a JOIN message is received.
func (s *State) join(ev *irc.Event) []string {
	var seen []string
	if ev.Sender == string(s.selfUser.Host) {
		s.addChannel(ev.Args[0])
	} else {
		seen = []string{ev.Sender}
	}
	s.addUser(ev.Sender)
	s.addToChannel(ev.Sender, ev.Args[0])
	return seen
}

// part alters the state of the database when a PART message is received.
func (s *State) part(ev *irc.Event) []string {
	if ev.Sender == string(s.selfUser.Host) {
		return s.removeChannel(ev.Args[0])
	} else {
		s.removeFromChannel(ev.Sender, ev.Args[0])
		if s.user(ev.Sender) == nil {
			return []string{ev.Sender}
		}
	}
	return nil
}

// quit alters the state of the database when a QUIT message is received.
func (s *State) quit(ev *irc.Event) string {
	if ev.Sender != string(s.selfUser.Host) {
		s.removeUser(ev.Sender)
		return ev.Sender
	}

	return ""
}

// kick alters the state of the database when a KICK message is received.
func (s *State) kick(ev *irc.Event) (seen []string, unseen []string) {
	if ev.Args[1] == s.selfUser.Nick() {
		s.removeChannel(ev.Args[0])
	} else {
		s.addUser(ev.Sender)
		oldUser := s.user(ev.Args[1])
		var oldHost string
		if oldUser != nil {
			oldHost = string(oldUser.Host)
		}

		s.removeFromChannel(ev.Args[1], ev.Args[0])

		if len(oldHost) > 0 && s.user(ev.Args[1]) == nil {
			return []string{ev.Sender}, []string{oldHost}
		}
	}

	return []string{ev.Sender}, nil
}

// mode alters the state of the database when a MODE message is received.
func (s *State) mode(ev *irc.Event) []string {
	target := strings.ToLower(ev.Args[0])
	if ev.IsTargetChan() {
		s.addUser(ev.Sender)
		if ch, ok := s.channels[target]; ok {
			pos, neg := ch.Modes.Apply(strings.Join(ev.Args[1:], " "))
			for i := 0; i < len(pos); i++ {
				nick := strings.ToLower(pos[i].Arg)
				s.channelUsers[target][nick].SetMode(pos[i].Mode)
			}
			for i := 0; i < len(neg); i++ {
				nick := strings.ToLower(neg[i].Arg)
				s.channelUsers[target][nick].UnsetMode(neg[i].Mode)
			}
		}
		return []string{ev.Sender}
	} else if target == s.selfUser.Nick() {
		s.selfModes.Apply(ev.Args[1])
	}
	return nil
}

// topic alters the state of the database when a TOPIC message is received.
func (s *State) topic(ev *irc.Event) []string {
	chname := strings.ToLower(ev.Args[0])
	if ch, ok := s.channels[chname]; ok {
		s.addUser(ev.Sender)
		if len(ev.Args) >= 2 {
			ch.Topic = ev.Args[1]
		} else {
			ch.Topic = ""
		}
	}
	return []string{ev.Sender}
}

// rplTopic alters the state of the database when a RPL_TOPIC message is
// received.
func (s *State) rplTopic(ev *irc.Event) {
	chname := strings.ToLower(ev.Args[1])
	if ch, ok := s.channels[chname]; ok {
		ch.Topic = ev.Args[2]
	}
}

// msg alters the state of the database when a PRIVMSG or NOTICE message is
// received.
func (s *State) msg(ev *irc.Event) []string {
	if ev.IsTargetChan() {
		s.addUser(ev.Sender)
		s.addToChannel(ev.Sender, ev.Args[0])
		return []string{ev.Sender}
	}
	return nil
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
	s.selfUser = user
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
		for ; j < len(s.kinds.userPrefixes); j++ {
			if s.kinds.userPrefixes[j][1] == rune(users[i][0]) {
				mode = s.kinds.userPrefixes[j][0]
				break
			}
		}
		if j < len(s.kinds.userPrefixes) {
			nick := users[i][1:]
			s.addUser(nick)
			s.addToChannel(nick, channel)
			s.userModes(nick, channel).SetMode(mode)
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
	s.user(fullhost).Realname = realname
	for _, modechar := range modes {
		if mode := s.kinds.Mode(modechar); mode != 0 {
			if uc := s.userModes(fullhost, channel); uc != nil {
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
	if ch := s.channel(channel); ch != nil {
		ch.Modes.Apply(modes)
	}
}

// rplBanList alters the state of the database when a RPL_BANLIST message is
// received.
func (s *State) rplBanList(ev *irc.Event) {
	channel := ev.Args[1]
	if ch := s.channel(channel); ch != nil {
		ch.AddBan(ev.Args[2])
	}
}
