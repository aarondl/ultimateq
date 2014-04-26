package parse

import (
	"strings"
	"github.com/aarondl/ultimateq/irc"
	"testing"
)

func b(s string) []byte {
	return []byte(s)
}

type a []string

func TestParse(t *testing.T) {
	sender := ":nick!user@host.com"
	testargs := []string{
		"&channel1,#channel2",
		":message1 message2",
	}

	wholeMsg := sender + " " + irc.PRIVMSG + " " + strings.Join(testargs, " ")
	noSender := irc.PRIVMSG + " " + strings.Join(testargs, " ")

	tests := []struct{
		Msg []byte
		Name string
		Sender string
		Args []string
		Error bool
	}{
		{b(wholeMsg), irc.PRIVMSG, sender, testargs, false},
		{b(noSender), irc.PRIVMSG, "", testargs, false},
		{b(":irc PING :4005945"), irc.PING, "irc", a{"4005945"}, false},
		{b(":irc PING 4005945 "), irc.PING, "irc", a{"4005945"}, false},
		{b(":irc 005 nobody1 RFC2812 CHANLIMIT=#&:+20 :are supported"),
			irc.RPL_ISUPPORT, "irc",
			a{"nobody1", "RFC2812", "CHANLIMIT=#&:+20", "are supported"},
			false,
		},
		{b("irc fail message"), "", "", nil, true},
	}

	for _, test := range tests {
		ev, err := Parse(test.Msg)

		if !test.Error && err != nil {
			t.Errorf("%s => Unexpected Error: %v", test.Msg, err)
		} else if test.Error && err == nil {
			t.Errorf("%s => Expected error but got nothing", test.Msg)
		} else {
			continue
		}

		if ev.Name != test.Name {
			t.Errorf("%s => Expected name: %v got %v",
				test.Msg, test.Name, ev.Name)
		}

		if ev.Sender != strings.TrimLeft(test.Sender, ":") {
			t.Errorf("%s => Expected sender: %v got %v",
				test.Msg, test.Sender[1:], ev.Sender)
		}

		if len(test.Args) != len(ev.Args) {
			t.Errorf("%s => Expected: %d arguments, got: %d",
				test.Msg, len(test.Args), len(ev.Args))
		}

		for i, expectArg := range test.Args {
			expectArg = strings.TrimLeft(expectArg, ":")
			if ev.Args[i] != expectArg {
				t.Errorf("%s => Expected Arg[%d]: %s, got: %s",
					test.Msg, i, expectArg, ev.Args[i])
			}
		}
	}
}
