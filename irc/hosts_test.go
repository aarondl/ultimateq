package irc

import (
	"testing"
)

func TestHost(t *testing.T) {
	var host Host = "nick!user@host"

	if s := host.Nick(); s != "nick" {
		t.Errorf("Expected: nick, got: %s", s)
	}
	if s := host.Username(); s != "user" {
		t.Errorf("Expected: user, got: %s", s)
	}
	if s := host.Hostname(); s != "host" {
		t.Errorf("Expected: host, got: %s", s)
	}
	if s := host.String(); s != string(host) {
		t.Errorf("Expected: %v, got: %s", string(host), s)
	}

	host = "nick@user!host"
	if s := host.Nick(); s != "nick" {
		t.Errorf("Expected: nick, got: %s", s)
	}
	if s := host.Username(); len(s) != 0 {
		t.Errorf("Expected: empty string, got: %s", s)
	}
	if s := host.Hostname(); len(s) != 0 {
		t.Errorf("Expected: empty string, got: %s", s)
	}
	if s := host.String(); s != string(host) {
		t.Errorf("Expected: %v, got: %s", string(host), s)
	}

	host = "nick"
	if s := host.Nick(); s != "nick" {
		t.Errorf("Expected: nick, got: %s", s)
	}
	if s := host.Username(); len(s) != 0 {
		t.Errorf("Expected: empty string, got: %s", s)
	}
	if s := host.Hostname(); len(s) != 0 {
		t.Errorf("Expected: empty string, got: %s", s)
	}
	if s := host.String(); s != string(host) {
		t.Errorf("Expected: %v, got: %s", string(host), s)
	}
}

func TestHost_SplitHost(t *testing.T) {
	var nick, user, hostname string

	nick, user, hostname = Host("nick!user@host").Split()
	if s := nick; s != "nick" {
		t.Errorf("Expected: nick, got: %s", s)
	}
	if s := user; s != "user" {
		t.Errorf("Expected: user, got: %s", s)
	}
	if s := hostname; s != "host" {
		t.Errorf("Expected: host, got: %s", s)
	}

	nick, user, hostname = Host("ni ck!user@host").Split()
	if s := nick; len(s) != 0 {
		t.Errorf("Expected: empty string, got: %s", s)
	}
	if s := user; len(s) != 0 {
		t.Errorf("Expected: empty string, got: %s", s)
	}
	if s := hostname; len(s) != 0 {
		t.Errorf("Expected: empty string, got: %s", s)
	}
}

func TestHost_IsValid(t *testing.T) {
	tests := []struct {
		Host    Host
		IsValid bool
	}{
		{"", false},
		{"!@", false},
		{"nick", false},
		{"nick!", false},
		{"nick@", false},
		{"nick@host!user", false},
		{"nick!user@host", true},
	}

	for _, test := range tests {
		if result := test.Host.IsValid(); result != test.IsValid {
			t.Errorf("Expected '%v'.IsValid() to be %v.", test.Host, test.IsValid)
		}
	}
}

func TestMask_Split(t *testing.T) {
	var nick, user, host string
	nick, user, host = Mask("n?i*ck!u*ser@h*o?st").Split()
	if s := nick; s != "n?i*ck" {
		t.Errorf("Expected: n?i*ck, got: %s", s)
	}
	if s := user; s != "u*ser" {
		t.Errorf("Expected: u*ser, got: %s", s)
	}
	if s := host; s != "h*o?st" {
		t.Errorf("Expected: h*o?st, got: %s", s)
	}

	nick, user, host = Mask("n?i* ck!u*ser@h*o?st").Split()
	if s := nick; len(s) != 0 {
		t.Errorf("Expected: empty string, got: %s", s)
	}
	if s := user; len(s) != 0 {
		t.Errorf("Expected: empty string, got: %s", s)
	}
	if s := host; len(s) != 0 {
		t.Errorf("Expected: empty string, got: %s", s)
	}
}

func TestMask_IsValid(t *testing.T) {
	tests := []struct {
		Mask    Mask
		IsValid bool
	}{
		{"", false},
		{"!@", false},
		{"n?i*ck", false},
		{"n?i*ck!", false},
		{"n?i*ck@", false},
		{"n*i?ck@h*o?st!u*ser", false},
		{"n?i*ck!u*ser@h*o?st", true},
	}

	for _, test := range tests {
		if result := test.Mask.IsValid(); result != test.IsValid {
			t.Errorf("Expected '%v'.IsValid() to be %v.",
				test.Mask, test.IsValid)
		}
	}
}

func TestMask_Match(t *testing.T) {
	var mask Mask
	var host Host
	if !mask.Match(host) {
		t.Error("Expected empty case to evaluate true.")
	}

	if !Mask("nick!*@*").Match("nick!@") {
		t.Error("Expected trivial case to evaluate true.")
	}

	host = "nick!user@host"

	positiveMasks := []Mask{
		// Default
		`nick!user@host`,
		// *'s
		`*`, `*!*@*`, `**!**@**`, `*@host`, `**@host`,
		`nick!*`, `nick!**`, `*nick!user@host`, `**nick!user@host`,
		`nick!user@host*`, `nick!user@host**`,
		// ?'s
		`ni?k!us?r@ho?st`, `ni??k!us??r@ho??st`, `????!????@????`,
		`?ick!user@host`, `??ick!user@host`, `?nick!user@host`,
		`??nick!user@host`, `nick!user@hos?`, `nick!user@hos??`,
		`nick!user@host?`, `nick!user@host??`,
		// Combination
		`?*nick!user@host`, `*?nick!user@host`, `??**nick!user@host`,
		`**??nick!user@host`,
		`nick!user@host?*`, `nick!user@host*?`, `nick!user@host??**`,
		`nick!user@host**??`, `nick!u?*?ser@host`, `nick!u?*?ser@host`,
	}

	for i := 0; i < len(positiveMasks); i++ {
		if !positiveMasks[i].Match(host) {
			t.Errorf("Expected: %v to match %v", positiveMasks[i], host)
		}
		if !host.Match(positiveMasks[i]) {
			t.Errorf("Expected: %v to match %v", host, positiveMasks[i])
		}
	}

	negativeMasks := []Mask{
		``, `?nq******c?!*@*`, `nick2!*@*`, `*!*@hostfail`, `*!*@failhost`,
	}

	for i := 0; i < len(negativeMasks); i++ {
		if negativeMasks[i].Match(host) {
			t.Errorf("Expected: %v not to match %v", negativeMasks[i], host)
		}
		if host.Match(negativeMasks[i]) {
			t.Errorf("Expected: %v to match %v", host, negativeMasks[i])
		}
	}

}
