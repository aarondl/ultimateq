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

	c.Check(p.RFC(), Equals, "RFC8812")
	c.Check(p.IRCD(), Equals, "gIRCd")
	c.Check(p.Casemapping(), Equals, "scii")
	c.Check(p.Prefix(), Equals, "(v)+")
	c.Check(p.Chantypes(), Equals, "#&")
	c.Check(p.Chanmodes(), Equals, "eI,k,l,imnOPRstz")
	c.Check(p.Chanlimit(), Equals, 10)
	c.Check(p.Channellen(), Equals, 49)
	c.Check(p.Nicklen(), Equals, 8)
	c.Check(p.Topiclen(), Equals, 489)
	c.Check(p.Awaylen(), Equals, 126)
	c.Check(p.Kicklen(), Equals, 399)
	c.Check(p.Modes(), Equals, 4)
	c.Check(p.Extra("EXCEPTS"), Equals, "e")
	c.Check(p.Extra("PENALTY"), Equals, "true")
	c.Check(p.Extra("INVEX"), Equals, "I")
	c.Check(p.Extra("NICK"), Equals, "")
}
