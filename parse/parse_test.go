package parse

import (
	. "launchpad.net/gocheck"
	"strings"
	"testing"
)

func Test(t *testing.T) { TestingT(t) } //Hook into testing package

type s struct{}

var _ = Suite(&s{})

var testargs = []string{
	":nick!user@host.com",
	"PRIVMSG",
	"&channel1,#channel2",
	":message1 message2",
}

func (s *s) TestParse(c *C) {
	msg, err := Parse(strings.Join(testargs, " "))
	c.Assert(err, IsNil)
	c.Assert(msg.Sender, Equals, strings.TrimLeft(testargs[0], ":"))
	c.Assert(msg.Name, Equals, testargs[1])
	c.Assert(len(msg.Args), Equals, 2)
	c.Assert(msg.Args[0], Equals, testargs[2])
	c.Assert(msg.Args[1], Equals, strings.TrimLeft(testargs[3], ":"))

	msg, err = Parse(strings.Join(testargs[1:], " "))
	c.Assert(err, IsNil)
	c.Assert(msg.Sender, Equals, "")

	msg, err = Parse(strings.Join(testargs[:len(testargs)-1], " "))
	c.Assert(err, IsNil)
	c.Assert(len(msg.Args), Equals, 1)
	c.Assert(msg.Args[0], Equals, testargs[2])
}

func (s *s) TestParse_Ping(c *C) {
	test1 := "PING"
	test2 := ":12312323"
	msg, err := Parse(strings.Join([]string{test1, test2}, " "))
	c.Assert(err, IsNil)
	c.Assert(msg.Name, Equals, test1)
	c.Assert(msg.Args[0], Equals, test2[1:])
}

func (s *s) TestParse_Error(c *C) {
	_, err := Parse("")
	c.Assert(err.Error(), Equals, errMsgParseFailure)

	irc := "invalid irc message"
	_, err = Parse(irc)
	e, ok := err.(ParseError)
	c.Assert(ok, Equals, true)
	c.Assert(e.Irc, Equals, irc)
}
