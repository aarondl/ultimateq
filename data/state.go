/*
Package data is used to turn irc.IrcMessages into a stateful database.
*/
package data

import (
	"errors"
	"github.com/aarondl/ultimateq/irc"
	"strings"
)

var (
	errProtoCapsMissing = errors.New("data: Protocaps missing.")
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
	caps   *irc.ProtoCaps
}

// CreateState creates a state from an irc protocaps instance.
func CreateState(caps *irc.ProtoCaps) (*State, error) {
	state := &State{}
	if err := state.Protocaps(caps); err != nil {
		return nil, err
	}
	state.Self.ChannelModes = CreateChannelModes(&ChannelModeKinds{}, nil)

	state.channels = make(map[string]*Channel)
	state.users = make(map[string]*User)
	state.channelUsers = make(map[string]map[string]*ChannelUser)
	state.userChannels = make(map[string]map[string]*UserChannel)

	return state, nil
}

// Protocaps updates the protocaps of the state.
func (s *State) Protocaps(caps *irc.ProtoCaps) error {
	if caps == nil {
		return errProtoCapsMissing
	}
	s.caps = caps
	kinds, err := CreateChannelModeKindsCSV(caps.Chanmodes())
	if err != nil {
		return err
	}
	modes, err := CreateUserModeKinds(caps.Prefix())
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
		user = CreateUser(nickorhost)
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
		ch = CreateChannel(channel, &s.kinds, &s.umodes)
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

	modes := CreateUserModes(&s.umodes)
	cu[nick] = CreateChannelUser(user, modes)
	uc[channel] = CreateUserChannel(ch, modes)
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
func (s *State) Update(m *irc.Message) {
	if len(m.Sender) > 0 {
		s.addUser(m.Sender)
	}
	switch m.Name {
	case irc.NICK:
		s.nick(m)
	case irc.JOIN:
		s.join(m)
	case irc.PART:
		s.part(m)
	case irc.QUIT:
		s.quit(m)
	case irc.KICK:
		s.kick(m)
	case irc.MODE:
		s.mode(m)
	case irc.TOPIC:
		s.topic(m)
	case irc.RPL_TOPIC:
		s.rplTopic(m)
	case irc.PRIVMSG, irc.NOTICE:
		s.msg(m)
	case irc.RPL_WELCOME:
		s.rplWelcome(m)
	case irc.RPL_NAMREPLY:
		s.rplNameReply(m)
	case irc.RPL_WHOREPLY:
		s.rplWhoReply(m)
	case irc.RPL_CHANNELMODEIS:
		s.rplChannelModeIs(m)
	case irc.RPL_BANLIST:
		s.rplBanList(m)

		// TODO: Handle Whois
	}
}

// nick alters the state of the database when a NICK message is received.
func (s *State) nick(m *irc.Message) {
	nick, username, host := irc.Host(m.Sender).Split()
	newnick := m.Args[0]
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
func (s *State) join(m *irc.Message) {
	if m.Sender == s.Self.Host() {
		s.addChannel(m.Args[0])
	}
	s.addToChannel(m.Sender, m.Args[0])
}

// part alters the state of the database when a PART message is received.
func (s *State) part(m *irc.Message) {
	if m.Sender == s.Self.Host() {
		s.removeChannel(m.Args[0])
	} else {
		s.removeFromChannel(m.Sender, m.Args[0])
	}
}

// quit alters the state of the database when a QUIT message is received.
func (s *State) quit(m *irc.Message) {
	if m.Sender != s.Self.Host() {
		s.removeUser(m.Sender)
	}
}

// kick alters the state of the database when a KICK message is received.
func (s *State) kick(m *irc.Message) {
	if m.Args[1] == s.Self.Nick() {
		s.removeChannel(m.Args[0])
	} else {
		s.removeFromChannel(m.Args[1], m.Args[0])
	}
}

// mode alters the state of the database when a MODE message is received.
func (s *State) mode(m *irc.Message) {
	target := strings.ToLower(m.Args[0])
	if s.caps.IsChannel(target) {
		if ch, ok := s.channels[target]; ok {
			pos, neg := ch.Apply(strings.Join(m.Args[1:], " "))
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
		s.Self.Apply(m.Args[1])
	}
}

// topic alters the state of the database when a TOPIC message is received.
func (s *State) topic(m *irc.Message) {
	chname := strings.ToLower(m.Args[0])
	if ch, ok := s.channels[chname]; ok {
		ch.SetTopic(m.Args[1])
	}
}

// rplTopic alters the state of the database when a RPL_TOPIC message is
// received.
func (s *State) rplTopic(m *irc.Message) {
	chname := strings.ToLower(m.Args[1])
	if ch, ok := s.channels[chname]; ok {
		ch.SetTopic(m.Args[2])
	}
}

// msg alters the state of the database when a PRIVMSG or NOTICE message is
// received.
func (s *State) msg(m *irc.Message) {
	if s.caps.IsChannel(m.Args[0]) {
		s.addToChannel(m.Sender, m.Args[0])
	}
}

// rplWelcome alters the state of the database when a RPL_WELCOME message is
// received.
func (s *State) rplWelcome(m *irc.Message) {
	splits := strings.Fields(m.Args[1])
	host := splits[len(splits)-1]

	if !strings.ContainsRune(host, '!') || !strings.ContainsRune(host, '@') {
		host = m.Args[0]
	}
	user := CreateUser(host)
	s.Self.User = user
	s.users[user.Nick()] = user
}

// rplNameReply alters the state of the database when a RPL_NAMEREPLY
// message is received.
func (s *State) rplNameReply(m *irc.Message) {
	channel := m.Args[2]
	users := strings.Fields(m.Args[3])
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
func (s *State) rplWhoReply(m *irc.Message) {
	channel := m.Args[1]
	fullhost := m.Args[5] + "!" + m.Args[2] + "@" + m.Args[3]
	modes := m.Args[6]
	realname := strings.SplitN(m.Args[7], " ", 2)[1]

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
func (s *State) rplChannelModeIs(m *irc.Message) {
	channel := m.Args[1]
	modes := strings.Join(m.Args[2:], " ")
	s.GetChannel(channel).Apply(modes)
}

// rplBanList alters the state of the database when a RPL_BANLIST message is
// received.
func (s *State) rplBanList(m *irc.Message) {
	channel := m.Args[1]
	s.GetChannel(channel).AddBan(m.Args[2])
}
