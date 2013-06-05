package data

import (
	. "launchpad.net/gocheck"
)

func (s *s) TestModeKinds_Create(c *C) {
	m := CreateModeKinds("a", "b", "c")
	c.Assert(m.kinds['a'], Equals, ARGS_ADDRESS)
	c.Assert(m.kinds['b'], Equals, ARGS_ALWAYS)
	c.Assert(m.kinds['c'], Equals, ARGS_ONSET)
	c.Assert(m.kinds['d'], Equals, ARGS_NONE)

	m = CreateModeKinds("a", "b", "c")
	c.Assert(m.kinds['a'], Equals, ARGS_ADDRESS)
	c.Assert(m.kinds['b'], Equals, ARGS_ALWAYS)
	c.Assert(m.kinds['c'], Equals, ARGS_ONSET)
	c.Assert(m.kinds['d'], Equals, ARGS_NONE)

	m.Update("d", "c", "b")
	c.Assert(m.kinds['d'], Equals, ARGS_ADDRESS)
	c.Assert(m.kinds['c'], Equals, ARGS_ALWAYS)
	c.Assert(m.kinds['b'], Equals, ARGS_ONSET)
	c.Assert(m.kinds['a'], Equals, ARGS_NONE)
}

func (s *s) TestModeKinds_CreateCSV(c *C) {
	m, err := CreateModeKindsCSV("")
	c.Assert(err, Equals, csvParseError)

	m, err = CreateModeKindsCSV(",,,")
	c.Assert(err, IsNil)
	m, err = CreateModeKindsCSV(",")
	c.Assert(err, Equals, csvParseError)

	m, err = CreateModeKindsCSV("a,b,c,d")
	c.Assert(m.kinds['a'], Equals, ARGS_ADDRESS)
	c.Assert(m.kinds['b'], Equals, ARGS_ALWAYS)
	c.Assert(m.kinds['c'], Equals, ARGS_ONSET)
	c.Assert(m.kinds['d'], Equals, ARGS_NONE)
}

func (s *s) TestModeKindsUpdate(c *C) {
	m := CreateModeKinds("a", "b", "c")
	c.Assert(m.kinds['a'], Equals, ARGS_ADDRESS)
	c.Assert(m.kinds['b'], Equals, ARGS_ALWAYS)
	c.Assert(m.kinds['c'], Equals, ARGS_ONSET)
	c.Assert(m.kinds['d'], Equals, ARGS_NONE)

	err := m.UpdateCSV("d,c,b,a")
	c.Assert(err, IsNil)
	c.Assert(m.kinds['d'], Equals, ARGS_ADDRESS)
	c.Assert(m.kinds['c'], Equals, ARGS_ALWAYS)
	c.Assert(m.kinds['b'], Equals, ARGS_ONSET)
	c.Assert(m.kinds['a'], Equals, ARGS_NONE)

	m.Update("a", "b", "c")
	c.Assert(m.kinds['a'], Equals, ARGS_ADDRESS)
	c.Assert(m.kinds['b'], Equals, ARGS_ALWAYS)
	c.Assert(m.kinds['c'], Equals, ARGS_ONSET)
	c.Assert(m.kinds['d'], Equals, ARGS_NONE)
}
