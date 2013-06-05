package data

import (
	. "launchpad.net/gocheck"
	"strings"
)

func (s *s) TestChannel_Create(c *C) {
	name := "#CHAN"
	ch := CreateChannel(name, testKinds)
	c.Assert(ch, NotNil)
	c.Assert(ch.GetName(), Equals, strings.ToLower(name))
	c.Assert(ch.GetTopic(), Equals, "")
	c.Assert(ch.Modeset, NotNil)
}

func (s *s) TestChannel_GettersSetters(c *C) {
	name := "#chan"
	topic := "topic"

	ch := CreateChannel(name, testKinds)
	c.Assert(ch.GetName(), Equals, name)
	ch.Topic(topic)
	c.Assert(ch.GetTopic(), Equals, topic)
}

func (s *s) TestChannel_Bans(c *C) {
	bans := []string{"ban1", "ban2"}
	ch := CreateChannel("", testKinds)

	ch.Bans(bans)
	got := ch.GetBans()
	for i := 0; i < len(got); i++ {
		c.Assert(got[i], Equals, bans[i])
	}
	bans[0] = "ban3"
	c.Assert(got[0], Not(Equals), bans[0])

	c.Assert(ch.HasBan("ban2"), Equals, true)
	ch.DeleteBan("ban2")
	c.Assert(ch.HasBan("ban2"), Equals, false)

	c.Assert(ch.HasBan("ban2"), Equals, false)
	ch.AddBan("ban2")
	c.Assert(ch.HasBan("ban2"), Equals, true)
}

func (s *s) TestChannel_IsBanned(c *C) {
	bans := []string{"*!*@host.com", "nick!*@*"}
	ch := CreateChannel("", testKinds)
	ch.Bans(bans)
	c.Assert(ch.IsBanned("nick"), Equals, true)
	c.Assert(ch.IsBanned("notnick"), Equals, false)
	c.Assert(ch.IsBanned("nick!user@host"), Equals, true)
	c.Assert(ch.IsBanned("notnick!user@host"), Equals, false)
	c.Assert(ch.IsBanned("notnick!user@host.com"), Equals, true)
}

func (s *s) TestChannel_DeleteBanWild(c *C) {
	bans := []string{"*!*@host.com", "nick!*@*", "nick2!*@*"}
	ch := CreateChannel("", testKinds)
	ch.Bans(bans)
	c.Assert(ch.IsBanned("nick"), Equals, true)
	c.Assert(ch.IsBanned("notnick"), Equals, false)
	c.Assert(ch.IsBanned("nick!user@host"), Equals, true)
	c.Assert(ch.IsBanned("notnick!user@host"), Equals, false)
	c.Assert(ch.IsBanned("notnick!user@host.com"), Equals, true)
	c.Assert(ch.IsBanned("nick2!user@host"), Equals, true)

	//ch.DeleteBans("nick!user@host")

	ch.DeleteBans("")
	c.Assert(len(ch.GetBans()), Equals, 3)

	ch.DeleteBans("nick")
	c.Assert(ch.IsBanned("nick"), Equals, false)
	c.Assert(ch.IsBanned("notnick"), Equals, false)
	c.Assert(ch.IsBanned("nick!user@host"), Equals, false)
	c.Assert(ch.IsBanned("nick2!user@host"), Equals, true)
	c.Assert(ch.IsBanned("notnick!user@host"), Equals, false)
	c.Assert(ch.IsBanned("notnick!user@host.com"), Equals, true)
	c.Assert(ch.IsBanned("nick2!user@host"), Equals, true)

	c.Assert(len(ch.GetBans()), Equals, 2)

	ch.DeleteBans("nick2!user@host.com")
	c.Assert(ch.IsBanned("nick2!user@host"), Equals, false)
	c.Assert(ch.IsBanned("notnick!user@host.com"), Equals, false)
	c.Assert(ch.IsBanned("nick2!user@host"), Equals, false)

	c.Assert(len(ch.GetBans()), Equals, 0)
	ch.DeleteBans("nick2!user@host.com")
	c.Assert(len(ch.GetBans()), Equals, 0)
}
