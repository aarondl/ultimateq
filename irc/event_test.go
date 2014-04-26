package irc

import (
	"strings"
	"testing"
)

var (
	testArgs = []string{"#chan1", "#chan2"}
	testEv   = NewEvent("", nil, "", "nick!user@host",
		strings.Join(testArgs, ","))
)

func TestIrcEvent_Hostnames(t *testing.T) {
	if "nick" != testEv.Nick() {
		t.Error("Should have nick as a nick, had:", testEv.Nick())
	}
	if "user" != testEv.Username() {
		t.Error("Should have user as a user, had:", testEv.Username())
	}
	if "host" != testEv.Hostname() {
		t.Error("Should have host as a host, had:", testEv.Hostname())
	}

	n, u, h := testEv.SplitHost()
	if "nick" != n {
		t.Error("Should have nick as a nick, had:", testEv.Nick())
	}
	if "user" != u {
		t.Error("Should have user as a user, had:", testEv.Username())
	}
	if "host" != h {
		t.Error("Should have host as a host, had:", testEv.Hostname())
	}
}

func TestIrcEvent_Timestamp(t *testing.T) {
	if 0 == testEv.Time.Unix() {
		t.Error("Expected the timestamp to be set.")
	}
}

func TestIrcEvent_SplitArgs(t *testing.T) {
	for i, v := range testEv.SplitArgs(0) {
		if v != testArgs[i] {
			t.Error("Expected split args to line up with testargs but index",
				i, "was:", testArgs[i])
		}
	}
}

func TestEvent_Target(t *testing.T) {
	args := []string{"#chan", "msg arg"}
	privmsg := &Event{
		Name:   PRIVMSG,
		Args:   args,
		Sender: "user@host.com",
	}

	if targ := privmsg.Target(); targ != args[0] {
		t.Error("Should give the target of the privmsg, got:", targ)
	}
}

func TestEvent_Message(t *testing.T) {
	args := []string{"#chan", "msg arg"}
	notice := &Event{
		Name:   NOTICE,
		Args:   args,
		Sender: "user@host.com",
	}

	if msg := notice.Message(); msg != args[1] {
		t.Error("Should give the message of the notice, got:", msg)
	}
}

func TestEvent_IsTargetChan(t *testing.T) {
	args := []string{"#chan", "msg arg"}
	privmsg := NewEvent("", NewNetworkInfo(), PRIVMSG, "user@host.com", args...)

	if !privmsg.IsTargetChan() {
		t.Error("The target should be a channel!")
	}

	args = []string{"user", "msg arg"}
	notice := NewEvent("", NewNetworkInfo(), NOTICE, "user@host.com", args...)

	if notice.IsTargetChan() {
		t.Error("The target should not be a channel!")
	}
}
