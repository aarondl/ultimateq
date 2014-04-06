package parse

import (
	. "gopkg.in/check.v1"
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
	msg, err := Parse([]byte(strings.Join(testargs, " ")))
	c.Check(err, IsNil)
	c.Check(msg.Sender, Equals, strings.TrimLeft(testargs[0], ":"))
	c.Check(msg.Name, Equals, testargs[1])
	c.Check(len(msg.Args), Equals, 2)
	c.Check(msg.Args[0], Equals, testargs[2])
	c.Check(msg.Args[1], Equals, strings.TrimLeft(testargs[3], ":"))

	msg, err = Parse([]byte(strings.Join(testargs[1:], " ")))
	c.Check(err, IsNil)
	c.Check(msg.Sender, Equals, "")

	msg, err = Parse([]byte(strings.Join(testargs[:len(testargs)-1], " ")))
	c.Check(err, IsNil)
	c.Check(len(msg.Args), Equals, 1)
	c.Check(msg.Args[0], Equals, testargs[2])
}

func (s *s) TestParse_Ping(c *C) {
	test1 := "PING"
	test2 := ":12312323"
	msg, err := Parse([]byte(strings.Join([]string{test1, test2}, " ")))
	c.Check(err, IsNil)
	c.Check(msg.Name, Equals, test1)
	c.Check(msg.Args[0], Equals, test2[1:])
}

func (s *s) TestParse_TrailingSpace(c *C) {
	test1 := "PING"
	test2 := "12312323 "
	msg, err := Parse([]byte(strings.Join([]string{test1, test2}, " ")))
	c.Check(err, IsNil)
	c.Check(msg.Name, Equals, test1)
	c.Check(msg.Args[0], Equals, strings.TrimSpace(test2))
}

func (s *s) TestParse_ExtraSemicolons(c *C) {
	sender := ":irc.test.net"
	name := "005"
	args := []string{
		"nobody1", "RFC2812", "CHANLIMIT=#&:+20", ":are supported",
	}
	raw := []byte(strings.Join([]string{
		sender, name, strings.Join(args, " "),
	}, " "))
	msg, err := Parse(raw)
	c.Check(err, IsNil)
	c.Check(msg.Name, Equals, name)
	c.Check(msg.Sender, Equals, sender[1:])
	c.Check(msg.Args[0], Equals, args[0])
	c.Check(msg.Args[1], Equals, args[1])
	c.Check(msg.Args[2], Equals, args[2])
	c.Check(msg.Args[3], Equals, args[3][1:])
}

func (s *s) TestParse_Error(c *C) {
	_, err := Parse([]byte{})
	c.Check(err.Error(), Equals, errMsgParseFailure)

	irc := "invalid irc message"
	_, err = Parse([]byte(irc))
	e, ok := err.(ParseError)
	c.Check(ok, Equals, true)
	c.Check(e.Irc, Equals, irc)
}
