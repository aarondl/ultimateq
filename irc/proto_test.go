package irc

import (
	. "launchpad.net/gocheck"
	"strings"
	"testing"
)

func Test(t *testing.T) { TestingT(t) } //Hook into testing package

type s struct{}

var _ = Suite(&s{})

func (s *s) TestParse(c *C) {
	args := []string{
		":nick!user@host.com",
		"PRIVMSG",
		"#channel1.#channel2",
		":message1 message2",
	}

	convert := func(s []string) []byte {
		return []byte(strings.Join(s, " "))
	}

	msg, err := Parse(convert(args))
	c.Assert(err, IsNil)
	c.Assert(msg.User, Equals, strings.TrimLeft(args[0], ":"))
	c.Assert(msg.Name, Equals, args[1])
	c.Assert(len(msg.Args), Equals, 2)
	c.Assert(msg.Args[0], Equals, args[2])
	c.Assert(msg.Args[1], Equals, strings.TrimLeft(args[3], ":"))

	msg, err = Parse(convert(args[1:]))
	c.Assert(err, IsNil)
	c.Assert(msg.User, Equals, "")

	msg, err = Parse(convert(args[:len(args)-1]))
	c.Assert(err, IsNil)
	c.Assert(len(msg.Args), Equals, 1)
	c.Assert(msg.Args[0], Equals, args[2])
}

func (s *s) TestParse_Error(c *C) {
	_, err := Parse([]byte{})
	c.Assert(err.Error(), Equals, errMsgParseFailure)

	irc := "invalid irc message"
	_, err = Parse([]byte(irc))
	e, ok := err.(ParseError)
	c.Assert(ok, Equals, true)
	c.Assert(e.Irc, Equals, irc)
}
