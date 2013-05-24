package irc

import (
	. "launchpad.net/gocheck"
	"strings"
)

func (s *s) TestProtoCaps(c *C) {
	p := CreateProtoCaps()

	s1 := `RFC2812 IRCD=ngIRCd CASEMAPPING=ascii PREFIX=(ov)@+ CHANTYPES=#&+ ` +
		`CHANMODES=beI,k,l,imnOPRstz CHANLIMIT=#&+:20`

	s2 := `CHANNELLEN=50 NICKLEN=9 TOPICLEN=490 AWAYLEN=127 KICKLEN=400 ` +
		`MODES=5 MAXLIST=beI:50 EXCEPTS=e INVEX=I PENALTY`

	msg1 := &IrcMessage{
		Name:   RPL_BOUNCE,
		Args:   append(strings.Split(s1, " "), "are supported by this server"),
		Sender: "irc.gamesurge.net",
	}
	msg2 := &IrcMessage{
		Name:   RPL_BOUNCE,
		Args:   append(strings.Split(s2, " "), "are supported by this server"),
		Sender: "irc.gamesurge.net",
	}

	p.ParseProtoCaps(msg1)
	p.ParseProtoCaps(msg2)

	c.Assert(p.RFC(), Equals, "RFC2812")
	c.Assert(p.IRCD(), Equals, "ngIRCd")
	c.Assert(p.Casemapping(), Equals, "ascii")
	c.Assert(p.Prefix(), Equals, "(ov)@+")
	c.Assert(p.Chantypes(), Equals, "#&+")
	c.Assert(p.Chanmodes(), Equals, "beI,k,l,imnOPRstz")
	c.Assert(p.Chanlimit(), Equals, 20)
	c.Assert(p.Channellen(), Equals, 50)
	c.Assert(p.Nicklen(), Equals, 9)
	c.Assert(p.Topiclen(), Equals, 490)
	c.Assert(p.Awaylen(), Equals, 127)
	c.Assert(p.Kicklen(), Equals, 400)
	c.Assert(p.Modes(), Equals, 5)
	c.Assert(p.Extra("EXCEPTS"), Equals, "e")
	c.Assert(p.Extra("PENALTY"), Equals, "true")
	c.Assert(p.Extra("INVEX"), Equals, "I")
}
