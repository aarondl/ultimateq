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
	banmasks := []string{"ban1", "ban2"}

	ch := CreateChannel(name)
	c.Assert(ch.GetName(), Equals, name)
	ch.Topic(topic)
	c.Assert(ch.GetTopic(), Equals, topic)

	ch.Banmasks(banmasks)
	for i := 0; i < len(ch.banmasks); i++ {
		c.Assert(ch.banmasks[i], Equals, banmasks[i])
	}
	banmasks[0] = "ban3"
	c.Assert(ch.banmasks[0], Not(Equals), banmasks[0])

	ch.Banmasks(banmasks)
	masks := ch.GetBanmasks()
	masks[0] = "ban4"
	c.Assert(ch.banmasks[0], Equals, banmasks[0])

	c.Assert(ch.HasBanmask("ban2"), Equals, true)
	c.Assert(ch.DeleteBanmask("ban2"), Equals, true)
	c.Assert(ch.HasBanmask("ban2"), Equals, false)
	c.Assert(ch.DeleteBanmask("ban2"), Equals, false)
}
