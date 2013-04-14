// Testing package for irc module.
package irc

import (
	. "launchpad.net/gocheck"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

type testSuite struct {}
var _ = Suite(&testSuite{})

func (s *testSuite) TestHello(c *C) {
	c.Check("hello", Equals, IrcHello())
}
