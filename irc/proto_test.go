package irc

import (
	. "launchpad.net/gocheck"
	"testing"
)

func Test(t *testing.T) { TestingT(t) } //Hook into testing package

type testSuite struct {}
var _ = Suite(&testSuite{})

func (s *testSuite) TestParseProtoTok_Basic(c *C) {
	tokid := "id"
	tok, _ := parseProtoTok(tokid)
	c.Assert(tok.id, Equals, tokid)
}

func (s *testSuite) TestParseProtoTok_Finality(c *C) {
	tok, _ := parseProtoTok(":id")
	c.Assert(tok.final, Equals, true)
}

func (s *testSuite) TestParseProtoTok_Args(c *C) {
	tok, _ := parseProtoTok("*id")
	c.Assert(tok.args, Equals, true)
}

func (s *testSuite) TestParseProtoTok_Channel(c *C) {
	tok, _ := parseProtoTok("#id")
	c.Assert(tok.channel, Equals, true)
}

func (s *testSuite) TestParseProtoTok_None(c *C) {
	tok, _ := parseProtoTok("id")
	c.Assert(tok.final, Equals, false)
	c.Assert(tok.args, Equals, false)
	c.Assert(tok.channel, Equals, false)
}

func (s *testSuite) TestParseProtoTok_All(c *C) {
	tok, _ := parseProtoTok("#*:id")
	c.Assert(tok.final, Equals, true)
	c.Assert(tok.args, Equals, true)
	c.Assert(tok.channel, Equals, true)

	tok, _ = parseProtoTok(":*#id")
	c.Assert(tok.final, Equals, true)
	c.Assert(tok.args, Equals, true)
	c.Assert(tok.channel, Equals, true)
}

func (s *testSuite) TestParseProtoTok_Error(c *C) {
	_, err := parseProtoTok("&lol")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, syntaxErrorMessage)

	_, err = parseProtoTok("lol*")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, syntaxErrorMessage)
}

func (s *testSuite) TestParseProtoChain_Basic(c *C) {
	chain, _ := parseProtoChain([]string {"id", "fun"})
	c.Assert(chain.id, Equals, "id")
	c.Assert(chain.next.id, Equals, "fun")
}

func (s *testSuite) TestParseProtoChain_OptionalChain(c *C) {
	chain, _ := parseProtoChain([]string {"[id", "more", "fun]"})
	c.Assert(chain.optional, NotNil)
	c.Assert(chain.optional.id, Equals, "id")
	c.Assert(chain.optional.next.id, Equals, "more")
	c.Assert(chain.optional.next.next.id, Equals, "fun")

	chain, _ = parseProtoChain([]string {"[hello]", "[more]", "[there]"})
	c.Assert(chain.optional, NotNil)
	c.Assert(chain.optional.id, Equals, "hello")
	c.Assert(chain.next.optional, NotNil)
	c.Assert(chain.next.optional.id, Equals, "more")
	c.Assert(chain.next.next.optional, NotNil)
	c.Assert(chain.next.next.optional.id, Equals, "there")

	chain, _ = parseProtoChain([]string {"[hello", "[more]]", "[there]"})
	c.Assert(chain.optional, NotNil)
	c.Assert(chain.optional.id, Equals, "hello")
	c.Assert(chain.optional.next.optional, NotNil)
	c.Assert(chain.optional.next.optional.id, Equals, "more")
	c.Assert(chain.next.optional, NotNil)
	c.Assert(chain.next.optional.id, Equals, "there")

	chain, _ = parseProtoChain([]string {"hello", "[[more]", "there]"})
	c.Assert(chain.id, Equals, "hello")
	c.Assert(chain.next.optional.optional, NotNil)
	c.Assert(chain.next.optional.optional.id, Equals, "more")
	c.Assert(chain.next.optional.next, NotNil)
	c.Assert(chain.next.optional.next.id, Equals, "there")
}

func (s *testSuite) TestParseProtoChain_Error(c *C) {
	_, err := parseProtoChain([]string {"hello", "[[more]]", "there]"})
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, syntaxErrorMessage)

	_, err = parseProtoChain([]string {"hello", "[[more]", "there"})
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, syntaxBracketMismatch)
}
