package irc

import (
	. "launchpad.net/gocheck"
	"testing"
)

func Test(t *testing.T) { TestingT(t) } //Hook into testing package

type testSuite struct{}

var _ = Suite(&testSuite{})

func (s *testSuite) TestParseFragment_Basic(c *C) {
	tokid := "id"
	tok, _ := parseFragment(tokid)
	c.Assert(tok.id, Equals, tokid)
}

func (s *testSuite) TestParseFragment_Finality(c *C) {
	tok, _ := parseFragment(":id")
	c.Assert(tok.final, Equals, true)
}

func (s *testSuite) TestParseFragment_Args(c *C) {
	tok, _ := parseFragment("*id")
	c.Assert(tok.args, Equals, true)
}

func (s *testSuite) TestParseFragment_Channel(c *C) {
	tok, _ := parseFragment("#id")
	c.Assert(tok.channel, Equals, true)
}

func (s *testSuite) TestParseFragment_None(c *C) {
	tok, _ := parseFragment("id")
	c.Assert(tok.final, Equals, false)
	c.Assert(tok.args, Equals, false)
	c.Assert(tok.channel, Equals, false)
}

func (s *testSuite) TestParseFragment_Both(c *C) {
	tok, _ := parseFragment("#*id")
	c.Assert(tok.final, Equals, false)
	c.Assert(tok.args, Equals, true)
	c.Assert(tok.channel, Equals, true)
}

func (s *testSuite) TestParseFragment_Error(c *C) {
	_, err := parseFragment("&lol")
	c.Assert(err, NotNil)
	c.Assert(err, Equals, errIllegalIdentifiers)

	_, err = parseFragment("lol*")
	c.Assert(err, NotNil)
	c.Assert(err, Equals, errIllegalIdentifiers)
}

func (s *testSuite) TestParseFragment_FinalCannotBeChannel(c *C) {
	chain, err := createFragmentChain([]string{":#id"})
	c.Assert(chain, IsNil)
	c.Assert(err, Equals, errFinalCantBeChannel)
}

func (s *testSuite) TestParseFragment_FinalCannotBeArgs(c *C) {
	chain, err := createFragmentChain([]string{":*id"})
	c.Assert(chain, IsNil)
	c.Assert(err, Equals, errFinalCantBeArgs)
}

func (s *testSuite) TestParseFragmentChain_Basic(c *C) {
	chain, _ := createFragmentChain([]string{"id", "fun"})
	c.Assert(chain.id, Equals, "id")
	c.Assert(chain.next.id, Equals, "fun")
}

func (s *testSuite) TestParseFragmentChain_OptionalChain(c *C) {
	chain, err := createFragmentChain([]string{"[id", "more", "fun]"})
	c.Assert(err, IsNil)
	c.Assert(chain.optional, NotNil)
	c.Assert(chain.optional.id, Equals, "id")
	c.Assert(chain.optional.next.id, Equals, "more")
	c.Assert(chain.optional.next.next.id, Equals, "fun")

	chain, err = createFragmentChain([]string{"[hello]", "[more]", "[there]"})
	c.Assert(err, IsNil)
	c.Assert(chain.optional, NotNil)
	c.Assert(chain.optional.id, Equals, "hello")
	c.Assert(chain.next.optional, NotNil)
	c.Assert(chain.next.optional.id, Equals, "more")
	c.Assert(chain.next.next.optional, NotNil)
	c.Assert(chain.next.next.optional.id, Equals, "there")

	chain, err = createFragmentChain([]string{"[hello", "[more]]", "[there]"})
	c.Assert(err, IsNil)
	c.Assert(chain.optional, NotNil)
	c.Assert(chain.optional.id, Equals, "hello")
	c.Assert(chain.optional.next.optional, NotNil)
	c.Assert(chain.optional.next.optional.id, Equals, "more")
	c.Assert(chain.next.optional, NotNil)
	c.Assert(chain.next.optional.id, Equals, "there")
}

func (s *testSuite) TestParseFragmentChain_IllegalIdentifiers(c *C) {
	chain, err := createFragmentChain([]string{"hello", "[[more]]", "there]"})
	c.Assert(chain, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err, Equals, errIllegalIdentifiers)
}

func (s *testSuite) TestParseFragmentChain_BracketMismatch(c *C) {
	chain, err := createFragmentChain([]string{"hello", "[[more]", "there"})
	c.Assert(chain, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err, Equals, errBracketMismatch)
}

func (s *testSuite) TestParseFragmentChain_ArgsAfterFinal(c *C) {
	chain, err := createFragmentChain([]string{":id", "fun"})
	c.Assert(chain, IsNil)
	c.Assert(err, Equals, errArgsAfterFinal)
}

func (s *testSuite) TestParseFragmentChain_RequiredAfterOptionalArg(c *C) {
	chain, err := createFragmentChain([]string{"[id]", "fun"})
	c.Assert(chain, IsNil)
	c.Assert(err, Equals, errRequiredAfterOptionalArg)
}
