package data

import (
	"github.com/aarondl/ultimateq/irc"
	. "launchpad.net/gocheck"
	"testing"
)

func Test(t *testing.T) { TestingT(t) } //Hook into testing package
type s struct{}

var _ = Suite(&s{})

func (s *s) TestStore(c *C) {
	st, err := CreateStore(irc.CreateProtoCaps())
	c.Check(st, NotNil)
	c.Check(err, IsNil)

	// Should die on creating kinds
	fakeCaps := &irc.ProtoCaps{}
	fakeCaps.ParseProtoCaps(&irc.IrcMessage{Args: []string{
		"NICK", "PREFIX=(ov)@+",
	}})
	st, err = CreateStore(fakeCaps)
	c.Check(st, IsNil)
	c.Check(err, NotNil)

	// Should die on creating user modes
	fakeCaps = &irc.ProtoCaps{}
	fakeCaps.ParseProtoCaps(&irc.IrcMessage{Args: []string{
		"NICK", "CHANMODES=a,b,c,d",
	}})
	st, err = CreateStore(fakeCaps)
	c.Check(st, IsNil)
	c.Check(err, NotNil)
}
