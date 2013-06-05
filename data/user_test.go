package data

import (
	"fmt"
	. "launchpad.net/gocheck"
)

func (s *s) TestUser_Create(c *C) {
	u := CreateUser("")
	c.Assert(u, IsNil)

	u = CreateUser("nick")
	c.Assert(u, NotNil)
	c.Assert(u.GetNick(), Equals, "nick")
	c.Assert(u.GetFullhost(), Equals, "nick")

	u = CreateUser("nick!user@host")
	c.Assert(u, NotNil)
	c.Assert(u.GetNick(), Equals, "nick")
	c.Assert(u.GetUsername(), Equals, "user")
	c.Assert(u.GetHost(), Equals, "host")
	c.Assert(u.GetFullhost(), Equals, "nick!user@host")
}

func (s *s) TestUser_Realname(c *C) {
	u := CreateUser("nick!user@host")
	u.Realname("realname realname")
	c.Assert(u.GetRealname(), Equals, "realname realname")
}

func (s *s) TestUser_String(c *C) {
	u := CreateUser("nick")
	str := fmt.Sprint(u)
	c.Assert(str, Equals, "nick")

	u = CreateUser("nick!user@host")
	str = fmt.Sprint(u)
	c.Assert(str, Equals, "nick nick!user@host")

	u = CreateUser("nick")
	u.Realname("realname realname")
	str = fmt.Sprint(u)
	c.Assert(str, Equals, "nick realname realname")

	u = CreateUser("nick!user@host")
	u.Realname("realname realname")
	str = fmt.Sprint(u)
	c.Assert(str, Equals, "nick nick!user@host realname realname")
}
