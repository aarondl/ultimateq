package data

import (
	//"github.com/aarondl/ultimateq/irc"
	. "launchpad.net/gocheck"
	"testing"
)

func Test(t *testing.T) { TestingT(t) } //Hook into testing package
type s struct{}

var _ = Suite(&s{})

//var testUserModes = CreateUserModes("(ov)@+")

func (s *s) TestStore(c *C) {
	//st := CreateStore()
}
