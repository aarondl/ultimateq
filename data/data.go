/*
data package is used to turn irc.IrcMessages into a stateful database.
*/
package data

import (
	"github.com/aarondl/ultimateq/irc"
	"strings"
)

// Self is the bot's user, he's a special case since he has to hold a Modeset.
type Self struct {
	*User
	*ChannelModes
}

// Store is the main data container. It represents the state on a server
// including all channels, users, and self.
type Store struct {
	Self Self

	channels map[string]*Channel
	users    map[string]*User

	channelUsers map[string]map[string]*ChannelUser
	userChannels map[string]map[string]*UserChannel

	selfkinds *ChannelModeKinds
	kinds     *ChannelModeKinds
	umodes    *UserModeKinds
	cfinder   *irc.ChannelFinder
}

// CreateStore creates a store from an irc protocaps instance.
func CreateStore(caps *irc.ProtoCaps) (*Store, error) {
	store := &Store{}
	err := store.Protocaps(caps)

	if err != nil {
		return nil, err
	}

	store.Self.ChannelModes = CreateChannelModes(store.selfkinds)

	store.channels = make(map[string]*Channel)
	store.users = make(map[string]*User)
	store.channelUsers = make(map[string]map[string]*ChannelUser)
	store.userChannels = make(map[string]map[string]*UserChannel)

	return store, nil
}

// Protocaps updates the protocaps of the store.
func (s *Store) Protocaps(caps *irc.ProtoCaps) error {
	selfkinds := CreateChannelModeKinds("", "", "", caps.Usermodes())
	kinds, err := CreateChannelModeKindsCSV(caps.Chanmodes())
	if err != nil {
		return err
	}
	modes, err := CreateUserModeKinds(caps.Prefix())
	if err != nil {
		return err
	}
	cfinder, err := irc.CreateChannelFinder(caps.Chantypes())
	if err != nil {
		return err
	}

	s.selfkinds = selfkinds
	s.kinds = kinds
	s.umodes = modes
	s.cfinder = cfinder
	return nil
}

// GetUser returns the user if he exists.
func (s *Store) GetUser(nickorhost string) *User {
	nick := strings.ToLower(Mask(nickorhost).GetNick())
	return s.users[nick]
}

// GetChannel returns the channel if it exists.
func (s *Store) GetChannel(channel string) *Channel {
	return s.channels[strings.ToLower(channel)]
}

// GetUserByChannel fetches a user based
func (s *Store) GetUsersChannelModes(nickorhost, channel string) *UserModes {
	nick := strings.ToLower(Mask(nickorhost).GetNick())
	channel = strings.ToLower(channel)

	if nicks, ok := s.channelUsers[channel]; ok {
		if cu, ok := nicks[nick]; ok {
			return cu.UserModes
		}
	}

	return nil
}

// IsOn checks if a user is on a specific channel.
func (s *Store) IsOn(nickorhost, channel string) bool {
	nick := strings.ToLower(Mask(nickorhost).GetNick())
	channel = strings.ToLower(channel)

	if chans, ok := s.userChannels[nick]; ok {
		if _, ok = chans[channel]; ok {
			return true
		}
	}
	return false
}

// addUser adds a user to the database.
func (s *Store) addUser(nickorhost string) *User {
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

	nick := strings.ToLower(Mask(nickorhost).GetNick())
	var user *User
	var ok bool
	if user, ok = s.users[nick]; ok {
		if excl && at && user.GetFullhost() != nickorhost {
			user.mask = Mask(nickorhost)
		}
	} else {
		user = CreateUser(nickorhost)
		s.users[nick] = user
	}
	return user
}

// removeUser deletes a user from the database.
func (s *Store) removeUser(nickorhost string) {
	nick := strings.ToLower(Mask(nickorhost).GetNick())
	for _, cus := range s.channelUsers {
		delete(cus, nick)
	}

	delete(s.userChannels, nick)
	delete(s.users, nick)
}

// addChannel adds a channel to the database.
func (s *Store) addChannel(channel string) *Channel {
	channel = strings.ToLower(channel)
	var ch *Channel
	if ch, ok := s.channels[channel]; !ok {
		ch = CreateChannel(channel, s.kinds)
		s.channels[channel] = ch
	}
	return ch
}

// removeChannel deletes a channel from the database.
func (s *Store) removeChannel(channel string) {
	channel = strings.ToLower(channel)
	for _, cus := range s.userChannels {
		delete(cus, channel)
	}

	delete(s.channelUsers, channel)
	delete(s.channels, channel)
}

// addToChannel adds a user by nick or fullhost to the channel
func (s *Store) addToChannel(nickorhost, channel string) {
	var user *User
	var ch *Channel
	var cu map[string]*ChannelUser
	var uc map[string]*UserChannel
	var ok, cuhas, uchas bool

	nick := strings.ToLower(Mask(nickorhost).GetNick())
	channel = strings.ToLower(channel)

	if user, ok = s.users[nick]; !ok {
		user = s.addUser(nickorhost)
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

	modes := CreateUserModes(s.umodes)
	cu[nick] = CreateChannelUser(user, modes)
	uc[channel] = CreateUserChannel(ch, modes)
	s.channelUsers[channel] = cu
	s.userChannels[nick] = uc
}

// removeFromChannel removes a user by nick or fullhost from the channel
func (s *Store) removeFromChannel(nickorhost, channel string) {
	var cu map[string]*ChannelUser
	var uc map[string]*UserChannel
	var ok bool

	nick := strings.ToLower(Mask(nickorhost).GetNick())
	channel = strings.ToLower(channel)

	if _, ok = s.users[nick]; !ok {
		s.addUser(nickorhost)
		return
	}

	if cu, ok = s.channelUsers[channel]; ok {
		delete(cu, nick)
	}

	if uc, ok = s.userChannels[nick]; ok {
		delete(uc, channel)
	}
}

// Update uses the irc.IrcMessage to modify the database accordingly.
func (s *Store) Update(m *irc.IrcMessage) {
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
	case irc.RPL_TOPIC:
		s.rpl_topic(m)
	case irc.PRIVMSG, irc.NOTICE:
		s.msg(m)
	case irc.RPL_WELCOME:
		s.rpl_welcome(m)
	case irc.RPL_NAMREPLY:
		s.rpl_namereply(m)
	case irc.RPL_WHOREPLY:
		s.rpl_whoreply(m)
	case irc.RPL_CHANNELMODEIS:
		s.rpl_channelmodeis(m)
	case irc.RPL_BANLIST:
		s.rpl_banlist(m)

		// TODO: Handle Whois
	}
}

// nick alters the state of the database when a NICK message is received.
func (s *Store) nick(m *irc.IrcMessage) {
	nick, username, host := Mask(m.Sender).SplitFullhost()
	newnick := m.Args[0]
	newuser := Mask(newnick + "!" + username + "@" + host)

	nick = strings.ToLower(nick)
	newnick = strings.ToLower(newnick)

	var ok bool
	if _, ok = s.users[nick]; !ok {
		s.addUser(string(newuser))
	} else {
		newnicklow := strings.ToLower(newnick)
		s.userChannels[newnicklow] = s.userChannels[nick]
		delete(s.userChannels, nick)
		s.users[newnicklow] = s.users[nick]
		delete(s.users, nick)
	}
}

// join alters the state of the database when a JOIN message is received.
func (s *Store) join(m *irc.IrcMessage) {
	if m.Sender == s.Self.GetFullhost() {
		s.addChannel(m.Args[0])
	}
	s.addToChannel(m.Sender, m.Args[0])
}

// part alters the state of the database when a PART message is received.
func (s *Store) part(m *irc.IrcMessage) {
	if m.Sender == s.Self.GetFullhost() {
		s.removeChannel(m.Args[0])
	} else {
		s.removeFromChannel(m.Sender, m.Args[0])
	}
}

// quit alters the state of the database when a QUIT message is received.
func (s *Store) quit(m *irc.IrcMessage) {
	if m.Sender != s.Self.GetFullhost() {
		s.removeUser(m.Sender)
	}
}

// kick alters the state of the database when a KICK message is received.
func (s *Store) kick(m *irc.IrcMessage) {
	if m.Args[1] == s.Self.GetNick() {
		s.removeChannel(m.Args[0])
	} else {
		s.removeFromChannel(m.Args[1], m.Args[0])
	}
}

// mode alters the state of the database when a MODE message is received.
func (s *Store) mode(m *irc.IrcMessage) {
	target := strings.ToLower(m.Args[0])
	if s.cfinder.IsChannel(target) {
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
	} else if target == s.Self.GetNick() {
		s.Self.Apply(m.Args[1])
	}
}

// topic alters the state of the database when a TOPIC message is received.
func (s *Store) rpl_topic(m *irc.IrcMessage) {
	chname := strings.ToLower(m.Args[0])
	if ch, ok := s.channels[chname]; ok {
		ch.Topic(m.Args[1])
	}
}

// msg alters the state of the database when a PRIVMSG or NOTICE message is
// received.
func (s *Store) msg(m *irc.IrcMessage) {
	s.addUser(m.Sender)
	if s.cfinder.IsChannel(m.Args[0]) {
		s.addToChannel(m.Sender, m.Args[0])
	}
}

// rpl_welcome alters the state of the database when a RPL_WELCOME message is
// received.
func (s *Store) rpl_welcome(m *irc.IrcMessage) {
	splits := strings.Split(m.Args[1], " ")
	host := splits[len(splits)-1]

	if !strings.ContainsRune(host, '!') || !strings.ContainsRune(host, '@') {
		host = m.Args[0]
	}
	user := CreateUser(host)
	s.Self.User = user
	s.users[user.GetNick()] = user
}

// rpl_namereply alters the state of the database when a RPL_NAMEREPLY
// message is received.
func (s *Store) rpl_namereply(m *irc.IrcMessage) {
	channel := m.Args[2]
	users := strings.Split(m.Args[3], " ")
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
			s.addToChannel(nick, channel)
			s.GetUsersChannelModes(nick, channel).SetMode(mode)
		} else {
			s.addToChannel(users[i], channel)
		}
	}
}

// rpl_whoreply alters the state of the database when a RPL_WHOREPLY message
// is received.
func (s *Store) rpl_whoreply(m *irc.IrcMessage) {
	channel := m.Args[1]
	fullhost := m.Args[5] + "!" + m.Args[2] + "@" + m.Args[3]
	modes := m.Args[6]
	realname := strings.SplitN(m.Args[7], " ", 2)[1]

	s.addToChannel(fullhost, channel)
	s.GetUser(fullhost).Realname(realname)
	chanmode := rune(modes[len(modes)-1])
	mode := s.umodes.GetMode(chanmode)
	if mode != 0 {
		s.GetUsersChannelModes(fullhost, channel).SetMode(mode)
	}
}

// rpl_channelmodeis alters the state of the database when a RPL_CHANNELMODEIS
// message is received.
func (s *Store) rpl_channelmodeis(m *irc.IrcMessage) {
	channel := m.Args[1]
	modes := strings.Join(m.Args[2:], " ")
	s.GetChannel(channel).Apply(modes)
}

// rpl_banlist alters the state of the database when a RPL_BANLIST message is
// received.
func (s *Store) rpl_banlist(m *irc.IrcMessage) {
	channel := m.Args[1]
	s.GetChannel(channel).AddBan(m.Args[2])
}
