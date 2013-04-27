package irc

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

func convert(s []string) []byte {
	return []byte(strings.Join(s, " "))
}

func (s *s) TestParse(c *C) {
	msg, err := Parse(convert(testargs))
	c.Assert(err, IsNil)
	c.Assert(msg.User, Equals, strings.TrimLeft(testargs[0], ":"))
	c.Assert(msg.Name, Equals, testargs[1])
	c.Assert(len(msg.Args), Equals, 2)
	c.Assert(msg.Args[0], Equals, testargs[2])
	c.Assert(msg.Args[1], Equals, strings.TrimLeft(testargs[3], ":"))

	msg, err = Parse(convert(testargs[1:]))
	c.Assert(err, IsNil)
	c.Assert(msg.User, Equals, "")

	msg, err = Parse(convert(testargs[:len(testargs)-1]))
	c.Assert(err, IsNil)
	c.Assert(len(msg.Args), Equals, 1)
	c.Assert(msg.Args[0], Equals, testargs[2])
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

func (s *s) TestParse_Split(c *C) {
	msg, _ := Parse(convert(testargs))
	channelnames := msg.Split(0)
	for i, v := range strings.Split(testargs[2], ",") {
		c.Assert(v, Equals, channelnames[i])
	}
}
