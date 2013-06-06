package data

import (
	"fmt"
	. "launchpad.net/gocheck"
)

func (s *s) TestUser_Create(c *C) {
	u := CreateUser("")
	c.Check(u, IsNil)

	u = CreateUser("nick")
	c.Check(u, NotNil)
	c.Check(u.GetNick(), Equals, "nick")
	c.Check(u.GetFullhost(), Equals, "nick")

	u = CreateUser("nick!user@host")
	c.Check(u, NotNil)
	c.Check(u.GetNick(), Equals, "nick")
	c.Check(u.GetUsername(), Equals, "user")
	c.Check(u.GetHost(), Equals, "host")
	c.Check(u.GetFullhost(), Equals, "nick!user@host")
}

func (s *s) TestUser_Realname(c *C) {
	u := CreateUser("nick!user@host")
	u.Realname("realname realname")
	c.Check(u.GetRealname(), Equals, "realname realname")
}

func (s *s) TestUser_String(c *C) {
	u := CreateUser("nick")
	str := fmt.Sprint(u)
	c.Check(str, Equals, "nick")

	u = CreateUser("nick!user@host")
	str = fmt.Sprint(u)
	c.Check(str, Equals, "nick nick!user@host")

	u = CreateUser("nick")
	u.Realname("realname realname")
	str = fmt.Sprint(u)
	c.Check(str, Equals, "nick realname realname")

	u = CreateUser("nick!user@host")
	u.Realname("realname realname")
	str = fmt.Sprint(u)
	c.Check(str, Equals, "nick nick!user@host realname realname")
}
