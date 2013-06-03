package data

import (
	. "launchpad.net/gocheck"
	"strings"
)

func (s *s) TestChannel_Create(c *C) {
	name := "#CHAN"
	ch := CreateChannel(name)
	c.Assert(ch, NotNil)
	c.Assert(ch.GetName(), Equals, strings.ToLower(name))
	c.Assert(ch.GetTopic(), Equals, "")
	c.Assert(ch.Modes, NotNil)
}

func (s *s) TestChannel_GettersSetters(c *C) {
	name := "#chan"
	topic := "topic"

	ch := CreateChannel(name)
	c.Assert(ch.GetName(), Equals, name)
	ch.Topic(topic)
	c.Assert(ch.GetTopic(), Equals, topic)
}

func (s *s) TestChannel_Bans(c *C) {
	bans := []WildMask{"ban1", "ban2"}
	ch := CreateChannel("")

	ch.Bans(bans)
	for i := 0; i < len(ch.bans); i++ {
		c.Assert(ch.bans[i], Equals, bans[i])
	}
	bans[0] = "ban3"
	c.Assert(ch.bans[0], Not(Equals), bans[0])

	ch.Bans(bans)
	chbans := ch.GetBans()
	chbans[0] = "ban4"
	c.Assert(ch.bans[0], Equals, bans[0])

	c.Assert(ch.HasBan("ban2"), Equals, true)
	c.Assert(ch.DeleteBan("ban2"), Equals, true)
	c.Assert(ch.HasBan("ban2"), Equals, false)
	c.Assert(ch.DeleteBan("ban2"), Equals, false)

	c.Assert(ch.HasBan("ban2"), Equals, false)
	ch.AddBan("ban2")
	c.Assert(ch.HasBan("ban2"), Equals, true)
}

func (s *s) TestChannel_IsBanned(c *C) {
	bans := []WildMask{"*!*@host.com", "nick!*@*"}
	ch := CreateChannel("")
	ch.Bans(bans)
	c.Assert(ch.IsBanned("nick"), Equals, true)
	c.Assert(ch.IsBanned("notnick"), Equals, false)
	c.Assert(ch.IsBanned("nick!user@host"), Equals, true)
	c.Assert(ch.IsBanned("notnick!user@host"), Equals, false)
	c.Assert(ch.IsBanned("notnick!user@host.com"), Equals, true)
}
