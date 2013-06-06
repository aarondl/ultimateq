package data

import (
	. "launchpad.net/gocheck"
)

func (s *s) TestModeKinds_Create(c *C) {
	m := CreateModeKinds("a", "b", "c")
	c.Check(m.kinds['a'], Equals, ARGS_ADDRESS)
	c.Check(m.kinds['b'], Equals, ARGS_ALWAYS)
	c.Check(m.kinds['c'], Equals, ARGS_ONSET)
	c.Check(m.kinds['d'], Equals, ARGS_NONE)

	m = CreateModeKinds("a", "b", "c")
	c.Check(m.kinds['a'], Equals, ARGS_ADDRESS)
	c.Check(m.kinds['b'], Equals, ARGS_ALWAYS)
	c.Check(m.kinds['c'], Equals, ARGS_ONSET)
	c.Check(m.kinds['d'], Equals, ARGS_NONE)

	m.Update("d", "c", "b")
	c.Check(m.kinds['d'], Equals, ARGS_ADDRESS)
	c.Check(m.kinds['c'], Equals, ARGS_ALWAYS)
	c.Check(m.kinds['b'], Equals, ARGS_ONSET)
	c.Check(m.kinds['a'], Equals, ARGS_NONE)
}

func (s *s) TestModeKinds_CreateCSV(c *C) {
	m, err := CreateModeKindsCSV("")
	c.Check(err, NotNil)

	m, err = CreateModeKindsCSV(",,,")
	c.Check(err, IsNil)
	m, err = CreateModeKindsCSV(",")
	c.Check(err, NotNil)

	m, err = CreateModeKindsCSV("a,b,c,d")
	c.Check(m.kinds['a'], Equals, ARGS_ADDRESS)
	c.Check(m.kinds['b'], Equals, ARGS_ALWAYS)
	c.Check(m.kinds['c'], Equals, ARGS_ONSET)
	c.Check(m.kinds['d'], Equals, ARGS_NONE)
}

func (s *s) TestModeKindsUpdate(c *C) {
	m := CreateModeKinds("a", "b", "c")
	c.Check(m.kinds['a'], Equals, ARGS_ADDRESS)
	c.Check(m.kinds['b'], Equals, ARGS_ALWAYS)
	c.Check(m.kinds['c'], Equals, ARGS_ONSET)
	c.Check(m.kinds['d'], Equals, ARGS_NONE)

	err := m.UpdateCSV("d,c,b,a")
	c.Check(err, IsNil)
	c.Check(m.kinds['d'], Equals, ARGS_ADDRESS)
	c.Check(m.kinds['c'], Equals, ARGS_ALWAYS)
	c.Check(m.kinds['b'], Equals, ARGS_ONSET)
	c.Check(m.kinds['a'], Equals, ARGS_NONE)

	m.Update("a", "b", "c")
	c.Check(m.kinds['a'], Equals, ARGS_ADDRESS)
	c.Check(m.kinds['b'], Equals, ARGS_ALWAYS)
	c.Check(m.kinds['c'], Equals, ARGS_ONSET)
	c.Check(m.kinds['d'], Equals, ARGS_NONE)

	err = m.UpdateCSV("")
	c.Check(err, NotNil)
}
