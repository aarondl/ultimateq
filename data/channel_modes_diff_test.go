package data

import (
	. "launchpad.net/gocheck"
)

func (s *s) TestModeDiff_Create(c *C) {
	diff := CreateModeDiff(testKinds)
	c.Check(diff, NotNil)
	c.Check(diff.pos, NotNil)
	c.Check(diff.neg, NotNil)

	var _ moder = CreateModeDiff(testKinds)
}

func (s *s) TestModeDiff_Apply(c *C) {
	d := CreateModeDiff(testKinds)
	d.Apply("+ab-c 10 ")
	c.Check(d.IsSet("ab 10"), Equals, true)
	c.Check(d.IsSet("c"), Equals, false)
	c.Check(d.IsUnset("c"), Equals, false)

	d = CreateModeDiff(testKinds)
	d.Apply("+b-b 10 10")
	c.Check(d.IsSet("b 10"), Equals, false)
	c.Check(d.IsUnset("b 10"), Equals, true)

	d = CreateModeDiff(testKinds)
	d.Apply("-b+b 10 10")
	c.Check(d.IsSet("b 10"), Equals, true)
	c.Check(d.IsUnset("b 10"), Equals, false)

	d.Apply("+x-y+z")
	c.Check(d.IsSet("x"), Equals, true)
	c.Check(d.IsUnset("y"), Equals, true)
	c.Check(d.IsSet("z"), Equals, true)
	c.Check(d.IsUnset("x"), Equals, false)
	c.Check(d.IsSet("y"), Equals, false)
	c.Check(d.IsUnset("z"), Equals, false)
}

func (s *s) TestModeDiff_String(c *C) {
	diff := CreateModeDiff(testKinds)
	diff.pos.Set("a", "b host1", "c 1")
	diff.neg.Set("x", "y", "z", "b host2")
	str := diff.String()
	c.Check(str, Matches, `^\+[abc]{3}-[xyzb]{4}( 1| host1){2}( host2){1}$`)

	diff = CreateModeDiff(testKinds)
	diff.pos.Set("x", "y", "z")
	diff.neg.Set("x", "y", "z")
	str = diff.String()
	c.Check(str, Matches, `^\+xyz-xyz$`)
}
