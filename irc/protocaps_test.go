package irc

import (
	. "launchpad.net/gocheck"
	"strings"
)

func (s *s) TestProtoCaps(c *C) {
	p := CreateProtoCaps()

	s1 := `NICK RFC8812 IRCD=gIRCd CASEMAPPING=scii PREFIX=(v)+ ` +
		`CHANTYPES=#& CHANMODES=eI,k,l,imnOPRstz CHANLIMIT=#&+:10`

	s2 := `NICK CHANNELLEN=49 NICKLEN=8 TOPICLEN=489 AWAYLEN=126 KICKLEN=399 ` +
		`MODES=4 MAXLIST=beI:49 EXCEPTS=e INVEX=I PENALTY`

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

	c.Assert(p.RFC(), Equals, "RFC8812")
	c.Assert(p.IRCD(), Equals, "gIRCd")
	c.Assert(p.Casemapping(), Equals, "scii")
	c.Assert(p.Prefix(), Equals, "(v)+")
	c.Assert(p.Chantypes(), Equals, "#&")
	c.Assert(p.Chanmodes(), Equals, "eI,k,l,imnOPRstz")
	c.Assert(p.Chanlimit(), Equals, 10)
	c.Assert(p.Channellen(), Equals, 49)
	c.Assert(p.Nicklen(), Equals, 8)
	c.Assert(p.Topiclen(), Equals, 489)
	c.Assert(p.Awaylen(), Equals, 126)
	c.Assert(p.Kicklen(), Equals, 399)
	c.Assert(p.Modes(), Equals, 4)
	c.Assert(p.Extra("EXCEPTS"), Equals, "e")
	c.Assert(p.Extra("PENALTY"), Equals, "true")
	c.Assert(p.Extra("INVEX"), Equals, "I")
	c.Assert(p.Extra("NICK"), Equals, "")
}
