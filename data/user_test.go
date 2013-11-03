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
	c.Check(u.Nick(), Equals, "nick")
	c.Check(u.Host(), Equals, "nick")

	u = CreateUser("nick!user@host")
	c.Check(u, NotNil)
	c.Check(u.Nick(), Equals, "nick")
	c.Check(u.Username(), Equals, "user")
	c.Check(u.Hostname(), Equals, "host")
	c.Check(u.Host(), Equals, "nick!user@host")
}

func (s *s) TestUser_Realname(c *C) {
	u := CreateUser("nick!user@host")
	u.SetRealname("realname realname")
	c.Check(u.Realname(), Equals, "realname realname")
}

func (s *s) TestUser_String(c *C) {
	u := CreateUser("nick")
	str := fmt.Sprint(u)
	c.Check(str, Equals, "nick")

	u = CreateUser("nick!user@host")
	str = fmt.Sprint(u)
	c.Check(str, Equals, "nick nick!user@host")

	u = CreateUser("nick")
	u.SetRealname("realname realname")
	str = fmt.Sprint(u)
	c.Check(str, Equals, "nick realname realname")

	u = CreateUser("nick!user@host")
	u.SetRealname("realname realname")
	str = fmt.Sprint(u)
	c.Check(str, Equals, "nick nick!user@host realname realname")
}
