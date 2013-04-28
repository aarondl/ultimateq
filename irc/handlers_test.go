package irc

import . "launchpad.net/gocheck"

/*=====================================*\
 Types to test privmsg handlers
\*=====================================*/
type testPrivmsgHandler struct {
	callback func(msg *Privmsg)
}

func (p testPrivmsgHandler) Privmsg(msg *Privmsg) {
	p.callback(msg)
}

/*=====================================*\
 Types to test privmsg user handlers
\*=====================================*/
type testPrivmsgUserHandler struct {
	callback func(msg *PrivmsgTarget)
}

func (p testPrivmsgUserHandler) PrivmsgUser(msg *PrivmsgTarget) {
	p.callback(msg)
}

/*=====================================*\
 Types to test privmsg channel handlers
\*=====================================*/
type testPrivmsgChannelHandler struct {
	callback func(msg *PrivmsgTarget)
}

func (p testPrivmsgChannelHandler) PrivmsgChannel(msg *PrivmsgTarget) {
	p.callback(msg)
}

/*=====================================*\
 Types to test Notice handlers
\*=====================================*/
type testNoticeHandler struct {
	callback func(msg *Notice)
}

func (p testNoticeHandler) Notice(msg *Notice) {
	p.callback(msg)
}

/*=====================================*\
 Types to test Notice user handlers
\*=====================================*/
type testNoticeUserHandler struct {
	callback func(msg *NoticeTarget)
}

func (p testNoticeUserHandler) NoticeUser(msg *NoticeTarget) {
	p.callback(msg)
}

/*=====================================*\
 Types to test Notice channel handlers
\*=====================================*/
type testNoticeChannelHandler struct {
	callback func(msg *NoticeTarget)
}

func (p testNoticeChannelHandler) NoticeChannel(msg *NoticeTarget) {
	p.callback(msg)
}

func testMessageSetup(handler interface{}, msg *IrcMessage) {
	d := CreateDispatcher()
	d.finder, _ = CreateChannelFinder("#")
	d.Register(msg.Name, handler)
	d.Dispatch(msg.Name, msg)
}

func (s *s) TestMsgTypes_Privmsg(c *C) {
	var pmsg *Privmsg
	tpmh := testPrivmsgHandler{func(msg *Privmsg) {
		pmsg = msg
	}}
	ircmsg := &IrcMessage{
		Name:   PRIVMSG,
		Args:   []string{"#chan", "msg arg"},
		Sender: "user@host.com",
	}
	testMessageSetup(tpmh, ircmsg)
	c.Assert(pmsg.Channel, Equals, ircmsg.Args[0])
	c.Assert(pmsg.User, Equals, "")
	c.Assert(pmsg.Message, Equals, ircmsg.Args[1])
	c.Assert(pmsg.Sender, Equals, ircmsg.Sender)

	ircmsg.Args[0] = "user"
	testMessageSetup(tpmh, ircmsg)
	c.Assert(pmsg.Channel, Equals, "")
	c.Assert(pmsg.User, Equals, ircmsg.Args[0])
	c.Assert(pmsg.Message, Equals, ircmsg.Args[1])
	c.Assert(pmsg.Sender, Equals, ircmsg.Sender)
}

func (s *s) TestMsgTypes_PrivmsgUser(c *C) {
	var pumsg *PrivmsgTarget
	tpmuh := testPrivmsgUserHandler{func(msg *PrivmsgTarget) {
		pumsg = msg
	}}
	ircmsg := &IrcMessage{
		Name:   PRIVMSG,
		Args:   []string{"user", "msg arg"},
		Sender: "user@host.com",
	}
	testMessageSetup(tpmuh, ircmsg)
	c.Assert(pumsg.Sender, Equals, ircmsg.Sender)
	c.Assert(pumsg.Target, Equals, ircmsg.Args[0])
	c.Assert(pumsg.Message, Equals, ircmsg.Args[1])
}

func (s *s) TestMsgTypes_PrivmsgChannel(c *C) {
	var pcmsg *PrivmsgTarget
	tpmuh := testPrivmsgChannelHandler{func(msg *PrivmsgTarget) {
		pcmsg = msg
	}}
	ircmsg := &IrcMessage{
		Name:   PRIVMSG,
		Args:   []string{"#chan", "msg arg"},
		Sender: "user@host.com",
	}
	testMessageSetup(tpmuh, ircmsg)
	c.Assert(pcmsg.Sender, Equals, ircmsg.Sender)
	c.Assert(pcmsg.Target, Equals, ircmsg.Args[0])
	c.Assert(pcmsg.Message, Equals, ircmsg.Args[1])
}

func (s *s) TestMsgTypes_Notice(c *C) {
	var umsg *Notice
	tpmh := testNoticeHandler{func(msg *Notice) {
		umsg = msg
	}}
	ircmsg := &IrcMessage{
		Name:   NOTICE,
		Args:   []string{"#chan", "msg arg"},
		Sender: "user@host.com",
	}
	testMessageSetup(tpmh, ircmsg)
	c.Assert(umsg.Channel, Equals, ircmsg.Args[0])
	c.Assert(umsg.User, Equals, "")
	c.Assert(umsg.Message, Equals, ircmsg.Args[1])
	c.Assert(umsg.Sender, Equals, ircmsg.Sender)

	ircmsg.Args[0] = "user"
	testMessageSetup(tpmh, ircmsg)
	c.Assert(umsg.Channel, Equals, "")
	c.Assert(umsg.User, Equals, ircmsg.Args[0])
	c.Assert(umsg.Message, Equals, ircmsg.Args[1])
	c.Assert(umsg.Sender, Equals, ircmsg.Sender)
}

func (s *s) TestMsgTypes_NoticeUser(c *C) {
	var numsg *NoticeTarget
	tpmuh := testNoticeUserHandler{func(msg *NoticeTarget) {
		numsg = msg
	}}
	ircmsg := &IrcMessage{
		Name:   NOTICE,
		Args:   []string{"user", "msg arg"},
		Sender: "user@host.com",
	}
	testMessageSetup(tpmuh, ircmsg)
	c.Assert(numsg.Sender, Equals, ircmsg.Sender)
	c.Assert(numsg.Target, Equals, ircmsg.Args[0])
	c.Assert(numsg.Message, Equals, ircmsg.Args[1])
}

func (s *s) TestMsgTypes_NoticeChannel(c *C) {
	var ncmsg *NoticeTarget
	tpmuh := testNoticeChannelHandler{func(msg *NoticeTarget) {
		ncmsg = msg
	}}
	ircmsg := &IrcMessage{
		Name:   NOTICE,
		Args:   []string{"#chan", "msg arg"},
		Sender: "user@host.com",
	}
	testMessageSetup(tpmuh, ircmsg)
	c.Assert(ncmsg.Sender, Equals, ircmsg.Sender)
	c.Assert(ncmsg.Target, Equals, ircmsg.Args[0])
	c.Assert(ncmsg.Message, Equals, ircmsg.Args[1])
}
