package irc

import (
	"strings"
	"testing"
)

var (
	netID = "irc.gamesurge.net"

	_s0 = `NICK irc.test.net testircd-1.2 acCior abcde`

	_s1 = `NICK RFC8812 IRCD=gIRCd CASEMAPPING=scii PREFIX=(v)+ ` +
		`CHANTYPES=#& CHANMODES=a,b,c,d CHANLIMIT=#&+:10`

	_s2 = `NICK CHANNELLEN=49 NICKLEN=8 TOPICLEN=489 AWAYLEN=126 KICKLEN=399 ` +
		`MODES=4 MAXLIST=beI:49 EXCEPTS=e INVEX=I PENALTY`

	capsTest0 = &Event{
		Name:   RPL_MYINFO,
		Args:   strings.Split(_s0, " "),
		Sender: netID,
	}
	capsTest1 = &Event{
		Name:   RPL_ISUPPORT,
		Args:   append(strings.Split(_s1, " "), "are supported by this server"),
		Sender: netID,
	}
	capsTest2 = &Event{
		Name:   RPL_ISUPPORT,
		Args:   append(strings.Split(_s2, " "), "are supported by this server"),
		Sender: netID,
	}
)

func TestNetworkInfo_Parse(t *testing.T) {
	t.Parallel()
	p := NewNetworkInfo()

	p.ParseMyInfo(capsTest0)
	p.ParseISupport(capsTest1)
	p.ParseISupport(capsTest2)

	if exp, val := "irc.test.net", p.ServerName(); val != exp {
		t.Error("Unexpected:", val, "should be:", exp)
	}
	if exp, val := "testircd-1.2", p.IrcdVersion(); val != exp {
		t.Error("Unexpected:", val, "should be:", exp)
	}
	if exp, val := "acCior", p.Usermodes(); val != exp {
		t.Error("Unexpected:", val, "should be:", exp)
	}
	if exp, val := "abcde", p.LegacyChanmodes(); val != exp {
		t.Error("Unexpected:", val, "should be:", exp)
	}
	if exp, val := "RFC8812", p.RFC(); val != exp {
		t.Error("Unexpected:", val, "should be:", exp)
	}
	if exp, val := "gIRCd", p.IRCD(); val != exp {
		t.Error("Unexpected:", val, "should be:", exp)
	}
	if exp, val := "scii", p.Casemapping(); val != exp {
		t.Error("Unexpected:", val, "should be:", exp)
	}
	if exp, val := "(v)+", p.Prefix(); val != exp {
		t.Error("Unexpected:", val, "should be:", exp)
	}
	if exp, val := "#&", p.Chantypes(); val != exp {
		t.Error("Unexpected:", val, "should be:", exp)
	}
	if exp, val := "a,b,c,d", p.Chanmodes(); val != exp {
		t.Error("Unexpected:", val, "should be:", exp)
	}
	if exp, val := 10, p.Chanlimit(); val != exp {
		t.Error("Unexpected:", val, "should be:", exp)
	}
	if exp, val := 49, p.Channellen(); val != exp {
		t.Error("Unexpected:", val, "should be:", exp)
	}
	if exp, val := 8, p.Nicklen(); val != exp {
		t.Error("Unexpected:", val, "should be:", exp)
	}
	if exp, val := 489, p.Topiclen(); val != exp {
		t.Error("Unexpected:", val, "should be:", exp)
	}
	if exp, val := 126, p.Awaylen(); val != exp {
		t.Error("Unexpected:", val, "should be:", exp)
	}
	if exp, val := 399, p.Kicklen(); val != exp {
		t.Error("Unexpected:", val, "should be:", exp)
	}
	if exp, val := 4, p.Modes(); val != exp {
		t.Error("Unexpected:", val, "should be:", exp)
	}
	if exp, val := "e", p.Extra("EXCEPTS"); val != exp {
		t.Error("Unexpected:", val, "should be:", exp)
	}
	if exp, val := "true", p.Extra("PENALTY"); val != exp {
		t.Error("Unexpected:", val, "should be:", exp)
	}
	if exp, val := "I", p.Extra("INVEX"); val != exp {
		t.Error("Unexpected:", val, "should be:", exp)
	}
	if exp, val := "", p.Extra("NICK"); val != exp {
		t.Error("Unexpected:", val, "should be:", exp)
	}
}

func TestNetworkInfo_Clone(t *testing.T) {
	t.Parallel()
	other := "other"
	diff := "different"

	p1 := NewNetworkInfo()
	p1.extras[other] = other
	p2 := p1.Clone()
	p1.chantypes = other
	p1.extras[other] = diff

	if p2.chantypes == other {
		t.Error("Clones should not share memory.")
	}
	if p2.extras[other] != other {
		t.Error("The extras map should be deep copied.")
	}
}

func TestNetworkInfo_IsChannel(t *testing.T) {
	t.Parallel()
	p := NewNetworkInfo()
	p.chantypes = "#&~"
	if test := "#channel"; !p.IsChannel(test) {
		t.Error("Expected:", test, "to be a channel.")
	}
	if test := "&channel"; !p.IsChannel(test) {
		t.Error("Expected:", test, "to be a channel.")
	}
	if test := "n#otchannel"; p.IsChannel(test) {
		t.Error("Expected:", test, "to not be a channel.")
	}
	if p.IsChannel("") {
		t.Error("It should return false when empty.")
	}
}
