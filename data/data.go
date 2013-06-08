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
	*Modeset
}

// Store is the main data container. It represents the state on a server
// including all channels, users, and self.
type Store struct {
	Self Self

	channels map[string]*Channel
	users    map[string]*User

	channelUsers map[string][]*ChannelUser
	userChannels map[string][]*UserChannel

	kinds   *ModeKinds
	umodes  *UserModesMeta
	cfinder *irc.ChannelFinder
}

// CreateStore creates a store from an irc protocaps instance.
func CreateStore(caps *irc.ProtoCaps) (*Store, error) {
	kinds, err := CreateModeKindsCSV(caps.Chanmodes())
	if err != nil {
		return nil, err
	}
	modes, err := CreateUserModesMeta(caps.Prefix())
	if err != nil {
		return nil, err
	}
	cfinder, err := irc.CreateChannelFinder(caps.Chantypes())
	if err != nil {
		return nil, err
	}

	return &Store{
		channels:     make(map[string]*Channel),
		users:        make(map[string]*User),
		channelUsers: make(map[string][]*ChannelUser),
		userChannels: make(map[string][]*UserChannel),

		kinds:   kinds,
		umodes:  modes,
		cfinder: cfinder,
	}, nil
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

// IsOn checks if a user is on a specific channel.
func (s *Store) IsOn(nickorhost, channel string) bool {
	nick := strings.ToLower(Mask(nickorhost).GetNick())
	channel = strings.ToLower(channel)
	if chans, ok := s.userChannels[nick]; ok {
		for i := 0; i < len(chans); i++ {
			if chans[i].Channel.GetName() == channel {
				return true
			}
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
		if user.GetFullhost() != nickorhost {
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
	for channel, cus := range s.channelUsers {
		ln := len(cus)
		for i := 0; i < ln; i++ {
			if nick == strings.ToLower(cus[i].User.GetNick()) {
				if ln == 1 {
					delete(s.channelUsers, channel)
				} else {
					if i+1 < ln {
						cus[i], cus[ln-1] = cus[ln-1], cus[i]
					}
					s.channelUsers[channel] = cus[:ln-1]
				}
				break
			}
		}
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
	for user, cus := range s.userChannels {
		ln := len(cus)
		for i := 0; i < ln; i++ {
			if channel == strings.ToLower(cus[i].Channel.GetName()) {
				if ln == 1 {
					delete(s.userChannels, user)
				} else {
					if i+1 < ln {
						cus[i], cus[ln-1] = cus[ln-1], cus[i]
					}
					s.userChannels[user] = cus[:ln-1]
				}
				break
			}
		}
	}

	delete(s.channelUsers, channel)
	delete(s.channels, channel)
}

// addToChannel adds a user by nick or fullhost to the channel
func (s *Store) addToChannel(nickorhost, channel string) {
	var user *User
	var ch *Channel
	var cu []*ChannelUser
	var uc []*UserChannel
	var ok bool

	nick := strings.ToLower(Mask(nickorhost).GetNick())
	channel = strings.ToLower(channel)

	if user, ok = s.users[nick]; !ok {
		user = s.addUser(nickorhost)
	}

	if ch, ok = s.channels[channel]; !ok {
		return
	}

	if cu, ok = s.channelUsers[channel]; !ok {
		cu = make([]*ChannelUser, 0, 1)
	}

	if uc, ok = s.userChannels[nick]; !ok {
		uc = make([]*UserChannel, 0, 1)
	}

	modes := CreateUserModes(s.umodes)
	cu = append(cu, CreateChannelUser(user, modes))
	uc = append(uc, CreateUserChannel(ch, modes))
	s.channelUsers[channel] = cu
	s.userChannels[nick] = uc
}

// removeFromChannel removes a user by nick or fullhost from the channel
func (s *Store) removeFromChannel(nickorhost, channel string) {
	var cu []*ChannelUser
	var uc []*UserChannel
	var ok bool

	nick := strings.ToLower(Mask(nickorhost).GetNick())
	channel = strings.ToLower(channel)

	if _, ok = s.users[nick]; !ok {
		s.addUser(nickorhost)
		return
	}

	if cu, ok = s.channelUsers[channel]; ok {
		ln := len(cu)
		for i := 0; i < ln; i++ {
			if strings.ToLower(cu[i].User.GetNick()) == nick {
				if ln == 1 {
					delete(s.channelUsers, channel)
				} else {
					if i+1 < ln {
						cu[i], cu[ln-1] = cu[ln-1], cu[i]
					}
					s.channelUsers[channel] = cu[:len(cu)-1]
				}
				break
			}
		}
	}

	if uc, ok = s.userChannels[nick]; ok {
		ln := len(uc)
		for i := 0; i < ln; i++ {
			if strings.ToLower(uc[i].Channel.GetName()) == channel {
				if ln == 1 {
					delete(s.userChannels, nick)
				} else {
					if i+1 < ln {
						uc[i], uc[ln-1] = uc[ln-1], uc[i]
					}
					s.userChannels[nick] = uc[:len(uc)-1]
				}
				break
			}
		}
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
