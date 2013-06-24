package irc

import (
	. "launchpad.net/gocheck"
	"strings"
)

var (
	serverId = "irc.gamesurge.net"

	_s0 = `NICK irc.test.net testircd-1.2 acCior abcde`

	_s1 = `NICK RFC8812 IRCD=gIRCd CASEMAPPING=scii PREFIX=(v)+ ` +
		`CHANTYPES=#& CHANMODES=a,b,c,d CHANLIMIT=#&+:10`

	_s2 = `NICK CHANNELLEN=49 NICKLEN=8 TOPICLEN=489 AWAYLEN=126 KICKLEN=399 ` +
		`MODES=4 MAXLIST=beI:49 EXCEPTS=e INVEX=I PENALTY`

	capsTest0 = &IrcMessage{
		Name:   RPL_MYINFO,
		Args:   strings.Split(_s0, " "),
		Sender: serverId,
	}
	capsTest1 = &IrcMessage{
		Name:   RPL_ISUPPORT,
		Args:   append(strings.Split(_s1, " "), "are supported by this server"),
		Sender: serverId,
	}
	capsTest2 = &IrcMessage{
		Name:   RPL_ISUPPORT,
		Args:   append(strings.Split(_s2, " "), "are supported by this server"),
		Sender: serverId,
	}
)

func (s *s) TestProtoCaps(c *C) {
	p := CreateProtoCaps()

	p.ParseMyInfo(capsTest0)
	p.ParseISupport(capsTest1)
	p.ParseISupport(capsTest2)

	c.Check(p.ServerName(), Equals, "irc.test.net")
	c.Check(p.IrcdVersion(), Equals, "testircd-1.2")
	c.Check(p.Usermodes(), Equals, "acCior")
	c.Check(p.LegacyChanmodes(), Equals, "abcde")
	c.Check(p.RFC(), Equals, "RFC8812")
	c.Check(p.IRCD(), Equals, "gIRCd")
	c.Check(p.Casemapping(), Equals, "scii")
	c.Check(p.Prefix(), Equals, "(v)+")
	c.Check(p.Chantypes(), Equals, "#&")
	c.Check(p.Chanmodes(), Equals, "a,b,c,d")
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

func (s *s) TestProtoCaps_Merge(c *C) {
	p1 := CreateProtoCaps()
	p2 := CreateProtoCaps()

	mergeTest1 := &IrcMessage{
		Args: []string{"NICK", "CHANTYPES=#&"},
	}
	mergeTest2 := &IrcMessage{
		Args: []string{"NICK", "CHANTYPES=~"},
	}

	p1.ParseISupport(mergeTest1)
	p2.ParseISupport(mergeTest2)
	c.Check(p1.Chantypes(), Equals, "#&")
	c.Check(p2.Chantypes(), Equals, "~")
	p1.Merge(p2)
	c.Check(p1.Chantypes(), Equals, "#&~")
}
