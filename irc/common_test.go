package irc

import (
	. "launchpad.net/gocheck"
	"strings"
	"testing"
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
		c.Check(args[i], Equals, v)
	}
}

func (s *s) TestMsgTypes_Privmsg(c *C) {
	args := []string{"#chan", "msg arg"}
	pmsg := &Message{Raw: &IrcMessage{
		Name:   PRIVMSG,
		Args:   args,
		Sender: "user@host.com",
	}}

	c.Check(pmsg.Target(), Equals, args[0])
	c.Check(pmsg.Message(), Equals, args[1])
}

func (s *s) TestMsgTypes_Notice(c *C) {
	args := []string{"#chan", "msg arg"}
	notice := &Message{Raw: &IrcMessage{
		Name:   NOTICE,
		Args:   args,
		Sender: "user@host.com",
	}}

	c.Check(notice.Target(), Equals, args[0])
	c.Check(notice.Message(), Equals, args[1])
}
