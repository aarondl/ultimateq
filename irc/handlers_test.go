package irc

import (
	. "launchpad.net/gocheck"
)

func (s *s) TestMsgTypes_Privmsg(c *C) {
	args := []string{"#chan", "msg arg"}
	pmsg := &Message{Raw: &IrcMessage{
		Name:   PRIVMSG,
		Args:   args,
		Sender: "user@host.com",
	}}

	c.Assert(pmsg.Target(), Equals, args[0])
	c.Assert(pmsg.Message(), Equals, args[1])
}

func (s *s) TestMsgTypes_Notice(c *C) {
	args := []string{"#chan", "msg arg"}
	notice := &Message{Raw: &IrcMessage{
		Name:   NOTICE,
		Args:   args,
		Sender: "user@host.com",
	}}

	c.Assert(notice.Target(), Equals, args[0])
	c.Assert(notice.Message(), Equals, args[1])
}
