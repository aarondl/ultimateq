package data

import (
	. "launchpad.net/gocheck"
	"strings"
)

func (s *s) TestChannel_Create(c *C) {
	ch := CreateChannel("", testKinds)
	c.Check(ch, IsNil)

	name := "#CHAN"
	ch = CreateChannel(name, testKinds)
	c.Check(ch, NotNil)
	c.Check(ch.GetName(), Equals, strings.ToLower(name))
	c.Check(ch.GetTopic(), Equals, "")
	c.Check(ch.Modeset, NotNil)
}

func (s *s) TestChannel_GettersSetters(c *C) {
	name := "#chan"
	topic := "topic"

	ch := CreateChannel(name, testKinds)
	c.Check(ch.GetName(), Equals, name)
	ch.Topic(topic)
	c.Check(ch.GetTopic(), Equals, topic)
}

func (s *s) TestChannel_Bans(c *C) {
	bans := []string{"ban1", "ban2"}
	ch := CreateChannel("name", testKinds)

	ch.Bans(bans)
	got := ch.GetBans()
	for i := 0; i < len(got); i++ {
		c.Check(got[i], Equals, bans[i])
	}
	bans[0] = "ban3"
	c.Check(got[0], Not(Equals), bans[0])

	c.Check(ch.HasBan("ban2"), Equals, true)
	ch.DeleteBan("ban2")
	c.Check(ch.HasBan("ban2"), Equals, false)

	c.Check(ch.HasBan("ban2"), Equals, false)
	ch.AddBan("ban2")
	c.Check(ch.HasBan("ban2"), Equals, true)
}

func (s *s) TestChannel_IsBanned(c *C) {
	bans := []string{"*!*@host.com", "nick!*@*"}
	ch := CreateChannel("name", testKinds)
	ch.Bans(bans)
	c.Check(ch.IsBanned("nick"), Equals, true)
	c.Check(ch.IsBanned("notnick"), Equals, false)
	c.Check(ch.IsBanned("nick!user@host"), Equals, true)
	c.Check(ch.IsBanned("notnick!user@host"), Equals, false)
	c.Check(ch.IsBanned("notnick!user@host.com"), Equals, true)
}

func (s *s) TestChannel_DeleteBanWild(c *C) {
	bans := []string{"*!*@host.com", "nick!*@*", "nick2!*@*"}
	ch := CreateChannel("name", testKinds)
	ch.Bans(bans)
	c.Check(ch.IsBanned("nick"), Equals, true)
	c.Check(ch.IsBanned("notnick"), Equals, false)
	c.Check(ch.IsBanned("nick!user@host"), Equals, true)
	c.Check(ch.IsBanned("notnick!user@host"), Equals, false)
	c.Check(ch.IsBanned("notnick!user@host.com"), Equals, true)
	c.Check(ch.IsBanned("nick2!user@host"), Equals, true)

	ch.DeleteBans("")
	c.Check(len(ch.GetBans()), Equals, 3)

	ch.DeleteBans("nick")
	c.Check(ch.IsBanned("nick"), Equals, false)
	c.Check(ch.IsBanned("notnick"), Equals, false)
	c.Check(ch.IsBanned("nick!user@host"), Equals, false)
	c.Check(ch.IsBanned("nick2!user@host"), Equals, true)
	c.Check(ch.IsBanned("notnick!user@host"), Equals, false)
	c.Check(ch.IsBanned("notnick!user@host.com"), Equals, true)
	c.Check(ch.IsBanned("nick2!user@host"), Equals, true)

	c.Check(len(ch.GetBans()), Equals, 2)

	ch.DeleteBans("nick2!user@host.com")
	c.Check(ch.IsBanned("nick2!user@host"), Equals, false)
	c.Check(ch.IsBanned("notnick!user@host.com"), Equals, false)
	c.Check(ch.IsBanned("nick2!user@host"), Equals, false)

	c.Check(len(ch.GetBans()), Equals, 0)
	ch.DeleteBans("nick2!user@host.com")
	c.Check(len(ch.GetBans()), Equals, 0)
}

func (s *s) TestChannel_String(c *C) {
	ch := CreateChannel("name", testKinds)
	c.Check(ch.String(), Equals, "name")
}
