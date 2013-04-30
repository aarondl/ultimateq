package irc

import (
	. "launchpad.net/gocheck"
	"testing"
	"strings"
)

func Test(t *testing.T) { TestingT(t) } //Hook into testing package

type s struct{}

var _ = Suite(&s{})

func (s *s) TestIrcMessage_Test(c *C) {
	args := []string{"#chan1", "#chan2"}
	msg := IrcMessage{
		Args: []string{strings.Join(args, ",")},
	}
	for i, v := range msg.Split(0) {
		c.Assert(args[i], Equals, v)
	}
}
