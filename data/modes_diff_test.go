package data

import (
	. "launchpad.net/gocheck"
)

func (s *s) TestModeDiff_Create(c *C) {
	diff := CreateModeDiff(testKinds)
	c.Assert(diff, NotNil)
	c.Assert(diff.pos, NotNil)
	c.Assert(diff.neg, NotNil)

	var _ moder = CreateModeDiff(testKinds)
}

func (s *s) TestModeDiff_Apply(c *C) {
	d := CreateModeDiff(testKinds)
	d.Apply("+ab-c 10 ")
	c.Assert(d.IsSet("ab 10"), Equals, true)
	c.Assert(d.IsSet("c"), Equals, false)
	c.Assert(d.IsUnset("c"), Equals, false)

	d = CreateModeDiff(testKinds)
	d.Apply("+b-b 10 10")
	c.Assert(d.IsSet("b 10"), Equals, false)
	c.Assert(d.IsUnset("b 10"), Equals, true)

	d = CreateModeDiff(testKinds)
	d.Apply("-b+b 10 10")
	c.Assert(d.IsSet("b 10"), Equals, true)
	c.Assert(d.IsUnset("b 10"), Equals, false)

	d.Apply("+x-y+z")
	c.Assert(d.IsSet("x"), Equals, true)
	c.Assert(d.IsUnset("y"), Equals, true)
	c.Assert(d.IsSet("z"), Equals, true)
	c.Assert(d.IsUnset("x"), Equals, false)
	c.Assert(d.IsSet("y"), Equals, false)
	c.Assert(d.IsUnset("z"), Equals, false)
}

func (s *s) TestModeDiff_String(c *C) {
	diff := CreateModeDiff(testKinds)
	diff.pos.Set("a", "b host1", "c 1")
	diff.neg.Set("x", "y", "z", "b host2")
	str := diff.String()
	c.Assert(str, Matches, `^\+[abc]{3}-[xyzb]{4}( 1| host1){2}( host2){1}$`)

	diff = CreateModeDiff(testKinds)
	diff.pos.Set("x", "y", "z")
	diff.neg.Set("x", "y", "z")
	str = diff.String()
	c.Assert(str, Matches, `^\+xyz-xyz$`)
}
