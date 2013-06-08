package data

import (
	"github.com/aarondl/ultimateq/irc"
	. "launchpad.net/gocheck"
	"testing"
)

func Test(t *testing.T) { TestingT(t) } //Hook into testing package
type s struct{}

var _ = Suite(&s{})

func (s *s) TestStore(c *C) {
	st, err := CreateStore(irc.CreateProtoCaps())
	c.Check(st, NotNil)
	c.Check(err, IsNil)

	// Should die on creating kinds
	fakeCaps := &irc.ProtoCaps{}
	fakeCaps.ParseProtoCaps(&irc.IrcMessage{Args: []string{
		"NICK", "CHANTYPES=#&", "PREFIX=(ov)@+",
	}})
	st, err = CreateStore(fakeCaps)
	c.Check(st, IsNil)
	c.Check(err, NotNil)

	// Should die on creating user modes
	fakeCaps = &irc.ProtoCaps{}
	fakeCaps.ParseProtoCaps(&irc.IrcMessage{Args: []string{
		"NICK", "CHANTYPES=#&", "CHANMODES=a,b,c,d",
	}})
	st, err = CreateStore(fakeCaps)
	c.Check(st, IsNil)
	c.Check(err, NotNil)

	// Should die on creating ChannelFinder
	fakeCaps = &irc.ProtoCaps{}
	fakeCaps.ParseProtoCaps(&irc.IrcMessage{Args: []string{
		"NICK", "CHANTYPES=H", "PREFIX=(ov)@+", "CHANMODES=a,b,c,d",
	}})
	st, err = CreateStore(fakeCaps)
	c.Check(st, IsNil)
	c.Check(err, NotNil)
}

var server = "irc.server.net"
var users = []string{"nick1!user1@host1", "nick2!user2@host2"}
var nicks = []string{"nick1", "nick2"}
var channels = []string{"#CHAN1", "#CHAN2"}

var self = Self{
	User: CreateUser("me!my@host.com"),
}

func (s *s) TestStore_GetUser(c *C) {
	st, err := CreateStore(irc.CreateProtoCaps())
	c.Check(err, IsNil)
	c.Check(st.GetUser(users[0]), IsNil)
	c.Check(st.GetUser(users[1]), IsNil)
	st.addUser(users[0])
	c.Check(st.GetUser(users[0]), NotNil)
	c.Check(st.GetUser(users[1]), IsNil)
	st.addUser(users[1])
	c.Check(st.GetUser(users[0]), NotNil)
	c.Check(st.GetUser(users[1]), NotNil)
}

func (s *s) TestStore_GetChannel(c *C) {
	st, err := CreateStore(irc.CreateProtoCaps())
	c.Check(err, IsNil)
	c.Check(st.GetChannel(channels[0]), IsNil)
	c.Check(st.GetChannel(channels[1]), IsNil)
	st.addChannel(channels[0])
	c.Check(st.GetChannel(channels[0]), NotNil)
	c.Check(st.GetChannel(channels[1]), IsNil)
	st.addChannel(channels[1])
	c.Check(st.GetChannel(channels[0]), NotNil)
	c.Check(st.GetChannel(channels[1]), NotNil)
}

func (s *s) TestStore_IsOn(c *C) {
	st, err := CreateStore(irc.CreateProtoCaps())
	c.Check(err, IsNil)
	c.Check(st.IsOn(users[0], channels[0]), Equals, false)
	st.addChannel(channels[0])
	c.Check(st.IsOn(users[0], channels[0]), Equals, false)
	st.addToChannel(users[0], channels[0])
	c.Check(st.IsOn(users[0], channels[0]), Equals, true)
}

func (s *s) TestStore_UpdateNick(c *C) {
	st, err := CreateStore(irc.CreateProtoCaps())
	c.Check(err, IsNil)
	m := &irc.IrcMessage{
		Name:   irc.NICK,
		Sender: users[0],
		Args:   []string{nicks[1]},
	}

	st.addChannel(channels[0])
	st.addToChannel(users[0], channels[0])

	c.Check(st.GetUser(users[0]), NotNil)
	c.Check(st.GetUser(users[1]), IsNil)
	c.Check(st.IsOn(users[0], channels[0]), Equals, true)
	c.Check(st.IsOn(users[1], channels[0]), Equals, false)

	st.Update(m)

	c.Check(st.GetUser(users[0]), IsNil)
	c.Check(st.GetUser(users[1]), NotNil)
	c.Check(st.IsOn(users[0], channels[0]), Equals, false)
	c.Check(st.IsOn(users[1], channels[0]), Equals, true)

	m.Sender = users[0]
	m.Args = []string{"newnick"}
	st.Update(m)
	c.Check(st.GetUser("newnick"), NotNil)
}

func (s *s) TestStore_UpdateJoin(c *C) {
	st, err := CreateStore(irc.CreateProtoCaps())
	st.Self = self
	c.Check(err, IsNil)
	m := &irc.IrcMessage{
		Name:   irc.JOIN,
		Sender: users[0],
		Args:   []string{channels[0]},
	}

	st.addChannel(channels[0])
	c.Check(st.IsOn(users[0], channels[0]), Equals, false)
	st.Update(m)
	c.Check(st.IsOn(users[0], channels[0]), Equals, true)

	st, _ = CreateStore(irc.CreateProtoCaps())
	st.Self = self
	st.addChannel(channels[0])

	c.Check(st.IsOn(users[0], channels[0]), Equals, false)
	st.Update(m)
	c.Check(st.IsOn(users[0], channels[0]), Equals, true)
	c.Check(st.IsOn(users[1], channels[0]), Equals, false)
}

func (s *s) TestStore_UpdateJoinSelf(c *C) {
	st, err := CreateStore(irc.CreateProtoCaps())
	st.Self = self
	c.Check(err, IsNil)
	m := &irc.IrcMessage{
		Name:   irc.JOIN,
		Sender: string(self.mask),
		Args:   []string{channels[0]},
	}

	c.Check(st.GetChannel(channels[0]), IsNil)
	c.Check(st.IsOn(st.Self.GetNick(), channels[0]), Equals, false)
	st.Update(m)
	c.Check(st.GetChannel(channels[0]), NotNil)
	c.Check(st.IsOn(st.Self.GetNick(), channels[0]), Equals, true)
}

func (s *s) TestStore_UpdatePart(c *C) {
	st, err := CreateStore(irc.CreateProtoCaps())
	st.Self = self
	c.Check(err, IsNil)

	m := &irc.IrcMessage{
		Name:   irc.PART,
		Sender: users[0],
		Args:   []string{channels[0]},
	}

	// Make sure seeing this message will create a user, even if the channel
	// doesn't exist.
	c.Check(st.GetUser(users[0]), IsNil)
	st.Update(m)
	c.Check(st.GetUser(users[0]), NotNil)

	// Test coverage, make sure adding to a channel that doesn't exist does
	// nothing.
	st.addToChannel(users[0], channels[0])
	c.Check(st.IsOn(users[0], channels[0]), Equals, false)

	st.addChannel(channels[0])
	st.addChannel(channels[1])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[0], channels[1])

	c.Check(st.IsOn(users[0], channels[0]), Equals, true)
	c.Check(st.IsOn(users[0], channels[1]), Equals, true)

	st.Update(m)
	c.Check(st.IsOn(users[0], channels[0]), Equals, false)
	c.Check(st.IsOn(users[0], channels[1]), Equals, true)

	m.Args[0] = channels[1]
	st.Update(m)

	c.Check(st.IsOn(users[0], channels[0]), Equals, false)
	c.Check(st.IsOn(users[0], channels[1]), Equals, false)
}

func (s *s) TestStore_UpdatePartSelf(c *C) {
	st, err := CreateStore(irc.CreateProtoCaps())
	st.Self = self
	c.Check(err, IsNil)
	m := &irc.IrcMessage{
		Name:   irc.PART,
		Sender: string(self.mask),
		Args:   []string{channels[0]},
	}

	st.addChannel(channels[0])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(self.GetNick(), channels[0])

	c.Check(st.IsOn(users[0], channels[0]), Equals, true)
	c.Check(st.IsOn(self.GetNick(), channels[0]), Equals, true)
	st.Update(m)
	c.Check(st.IsOn(users[0], channels[0]), Equals, false)
	c.Check(st.IsOn(self.GetNick(), channels[0]), Equals, false)
}

func (s *s) TestStore_UpdateQuit(c *C) {
	st, err := CreateStore(irc.CreateProtoCaps())
	st.Self = self
	c.Check(err, IsNil)
	m := &irc.IrcMessage{
		Name:   irc.QUIT,
		Sender: users[0],
		Args:   []string{"quit message"},
	}

	// Test Quitting when we don't know the user
	st.Update(m)

	st.addChannel(channels[0])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[1], channels[0])

	c.Check(st.IsOn(users[0], channels[0]), Equals, true)
	c.Check(st.GetUser(users[0]), NotNil)
	c.Check(st.IsOn(users[1], channels[0]), Equals, true)
	c.Check(st.GetUser(users[1]), NotNil)

	st.Update(m)

	c.Check(st.IsOn(users[0], channels[0]), Equals, false)
	c.Check(st.GetUser(users[0]), IsNil)
	c.Check(st.IsOn(users[1], channels[0]), Equals, true)
	c.Check(st.GetUser(users[1]), NotNil)

	m.Sender = users[1]
	st.Update(m)

	c.Check(st.IsOn(users[1], channels[0]), Equals, false)
	c.Check(st.GetUser(users[1]), IsNil)
}

func (s *s) TestStore_UpdateKick(c *C) {
	st, err := CreateStore(irc.CreateProtoCaps())
	st.Self = self
	c.Check(err, IsNil)
	m := &irc.IrcMessage{
		Name:   irc.KICK,
		Sender: users[1],
		Args:   []string{channels[0], users[0]},
	}

	st.addChannel(channels[0])
	st.addToChannel(users[0], channels[0])

	c.Check(st.IsOn(users[0], channels[0]), Equals, true)
	st.Update(m)
	c.Check(st.IsOn(users[0], channels[0]), Equals, false)
}

func (s *s) TestStore_UpdateKickSelf(c *C) {
	st, err := CreateStore(irc.CreateProtoCaps())
	st.Self = self
	c.Check(err, IsNil)
	m := &irc.IrcMessage{
		Name:   irc.KICK,
		Sender: users[1],
		Args:   []string{channels[0], st.Self.GetNick()},
	}

	st.addChannel(channels[0])
	st.addToChannel(users[0], channels[0])

	c.Check(st.GetChannel(channels[0]), NotNil)
	st.Update(m)
	c.Check(st.GetChannel(channels[0]), IsNil)
}

func (s *s) TestStore_UpdateMode(c *C) {
	c.Skip("Not Done")
	st, err := CreateStore(irc.CreateProtoCaps())
	c.Check(err, IsNil)
	m := &irc.IrcMessage{
		Name:   "JOIN",
		Sender: users[1],
		Args:   []string{channels[0]},
	}
	st.Update(m)
}

func (s *s) TestStore_UpdateRplMode(c *C) {
}

func (s *s) TestStore_UpdateTopic(c *C) {
	st, err := CreateStore(irc.CreateProtoCaps())
	st.Self = self
	c.Check(err, IsNil)

	m := &irc.IrcMessage{
		Name:   irc.RPL_TOPIC,
		Sender: users[0],
		Args:   []string{channels[0], "topic topic"},
	}

	st.addChannel(channels[0])

	c.Check(st.GetChannel(channels[0]).GetTopic(), Equals, "")
	st.Update(m)
	c.Check(st.GetChannel(channels[0]).GetTopic(), Equals, "topic topic")
}

func (s *s) TestStore_UpdatePrivmsg(c *C) {
	st, err := CreateStore(irc.CreateProtoCaps())
	st.Self = self
	c.Check(err, IsNil)
	m := &irc.IrcMessage{
		Name:   irc.PRIVMSG,
		Sender: users[0],
		Args:   []string{channels[0]},
	}

	c.Check(st.GetUser(users[0]), IsNil)
	st.Update(m)
	c.Check(st.GetUser(users[0]), NotNil)

	m.Sender = server
	size := len(st.users)
	st.Update(m)
	c.Check(len(st.users), Equals, size)
}

func (s *s) TestStore_UpdateNotice(c *C) {
	st, err := CreateStore(irc.CreateProtoCaps())
	st.Self = self
	c.Check(err, IsNil)
	m := &irc.IrcMessage{
		Name:   irc.NOTICE,
		Sender: users[0],
		Args:   []string{channels[0]},
	}

	c.Check(st.GetUser(users[0]), IsNil)
	st.Update(m)
	c.Check(st.GetUser(users[0]), NotNil)

	m.Sender = server
	size := len(st.users)
	st.Update(m)
	c.Check(len(st.users), Equals, size)
}

func (s *s) TestStore_UpdateWelcome(c *C) {
	st, err := CreateStore(irc.CreateProtoCaps())
	c.Check(err, IsNil)
	m := &irc.IrcMessage{
		Name:   irc.RPL_WELCOME,
		Sender: server,
		Args:   []string{nicks[1], "Welcome to"},
	}

	st.Update(m)
	c.Check(st.Self.GetFullhost(), Equals, nicks[1])
	c.Check(st.users[nicks[1]].GetFullhost(), Equals, st.Self.GetFullhost())

	m = &irc.IrcMessage{
		Name:   irc.RPL_WELCOME,
		Sender: server,
		Args:   []string{nicks[1], "Welcome to " + users[1]},
	}

	st.Update(m)
	c.Check(st.Self.GetFullhost(), Equals, users[1])
	c.Check(st.users[nicks[1]].GetFullhost(), Equals, st.Self.GetFullhost())
}
