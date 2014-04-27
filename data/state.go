/*
Package data is used to turn irc.IrcMessages into a stateful database.
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

// Self is the bot's user, he's a special case since he has to hold a Modeset.
type Self struct {
	*User
	*ChannelModes
}

// State is the main data container. It represents the state on a server
// including all channels, users, and self.
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
			s.GetUsersChannelModes(fullhost, channel).SetMode(mode)
		}
	}
}

// rplChannelModeIs alters the state of the database when a RPL_CHANNELMODEIS
// message is received.
func (s *State) rplChannelModeIs(ev *irc.Event) {
	channel := ev.Args[1]
	modes := strings.Join(ev.Args[2:], " ")
	s.GetChannel(channel).Apply(modes)
}

// rplBanList alters the state of the database when a RPL_BANLIST message is
// received.
func (s *State) rplBanList(ev *irc.Event) {
	channel := ev.Args[1]
	s.GetChannel(channel).AddBan(ev.Args[2])
}
