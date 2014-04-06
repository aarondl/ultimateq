package data

import (
	. "gopkg.in/check.v1"
)

func (s *s) TestModeDiff_Create(c *C) {
	diff := CreateModeDiff(testChannelKinds, testUserKinds)
	c.Check(diff, NotNil)
	c.Check(diff.pos, NotNil)
	c.Check(diff.neg, NotNil)

	var _ moder = CreateModeDiff(testChannelKinds, testUserKinds)
}

func (s *s) TestModeDiff_Apply(c *C) {
	d := CreateModeDiff(testChannelKinds, testUserKinds)
	pos, neg := d.Apply("+ab-c 10 ")
	c.Check(len(pos), Equals, 0)
	c.Check(len(neg), Equals, 0)
	c.Check(d.IsSet("ab 10"), Equals, true)
	c.Check(d.IsSet("c"), Equals, false)
	c.Check(d.IsUnset("c"), Equals, false)

	d = CreateModeDiff(testChannelKinds, testUserKinds)
	pos, neg = d.Apply("+b-b 10 10")
	c.Check(len(pos), Equals, 0)
	c.Check(len(neg), Equals, 0)
	c.Check(d.IsSet("b 10"), Equals, false)
	c.Check(d.IsUnset("b 10"), Equals, true)

	d = CreateModeDiff(testChannelKinds, testUserKinds)
	pos, neg = d.Apply("-b+b 10 10")
	c.Check(len(pos), Equals, 0)
	c.Check(len(neg), Equals, 0)
	c.Check(d.IsSet("b 10"), Equals, true)
	c.Check(d.IsUnset("b 10"), Equals, false)

	pos, neg = d.Apply("+x-y+z")
	c.Check(len(pos), Equals, 0)
	c.Check(len(neg), Equals, 0)
	c.Check(d.IsSet("x"), Equals, true)
	c.Check(d.IsUnset("y"), Equals, true)
	c.Check(d.IsSet("z"), Equals, true)
	c.Check(d.IsUnset("x"), Equals, false)
	c.Check(d.IsSet("y"), Equals, false)
	c.Check(d.IsUnset("z"), Equals, false)

	pos, neg = d.Apply("+vx-yo+vz user1 user2 user3")
	c.Check(len(pos), Equals, 2)
	c.Check(len(neg), Equals, 1)
	c.Check(pos[0].Mode, Equals, 'v')
	c.Check(pos[0].Arg, Equals, "user1")
	c.Check(pos[1].Mode, Equals, 'v')
	c.Check(pos[1].Arg, Equals, "user3")
	c.Check(neg[0].Mode, Equals, 'o')
	c.Check(neg[0].Arg, Equals, "user2")
	c.Check(d.IsSet("x"), Equals, true)
	c.Check(d.IsUnset("y"), Equals, true)
	c.Check(d.IsSet("z"), Equals, true)
	c.Check(d.IsUnset("x"), Equals, false)
	c.Check(d.IsSet("y"), Equals, false)
	c.Check(d.IsUnset("z"), Equals, false)
}

func (s *s) TestModeDiff_String(c *C) {
	diff := CreateModeDiff(testChannelKinds, testUserKinds)
	diff.pos.Set("a", "b host1", "c 1")
	diff.neg.Set("x", "y", "z", "b host2")
	str := diff.String()
	c.Check(str, Matches, `^\+[abc]{3}-[xyzb]{4}( 1| host1){2}( host2){1}$`)

	diff = CreateModeDiff(testChannelKinds, testUserKinds)
	diff.pos.Set("x", "y", "z")
	diff.neg.Set("x", "y", "z")
	str = diff.String()
	c.Check(str, Matches, `^\+xyz-xyz$`)
}
