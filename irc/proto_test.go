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
	tok := parseProtoTok(tokid)
	c.Assert(tok.id, Equals, tokid)
}

func (s *testSuite) TestParseProtoTok_Finality(c *C) {
	c.Assert(parseProtoTok(":id").final, Equals, true)
}

func (s *testSuite) TestParseProtoTok_Args(c *C) {
	c.Assert(parseProtoTok("*id").args, Equals, true)
}

func (s *testSuite) TestParseProtoTok_Channel(c *C) {
	c.Assert(parseProtoTok("#id").channel, Equals, true)
}

func (s *testSuite) TestParseProtoTok_None(c *C) {
	tok := parseProtoTok("id")
	c.Assert(tok.final, Equals, false)
	c.Assert(tok.args, Equals, false)
	c.Assert(tok.channel, Equals, false)
}

func (s *testSuite) TestParseProtoTok_All(c *C) {
	tok := parseProtoTok("#*:id")
	c.Assert(tok.final, Equals, true)
	c.Assert(tok.args, Equals, true)
	c.Assert(tok.channel, Equals, true)

	tok = parseProtoTok(":*#id")
	c.Assert(tok.final, Equals, true)
	c.Assert(tok.args, Equals, true)
	c.Assert(tok.channel, Equals, true)
}

func (s *testSuite) TestParseProtoChain_Basic(c *C) {
	chain := parseProtoChain([]string {"id", "fun"})
	c.Assert(chain.id, Equals, "id")
	c.Assert(chain.next.id, Equals, "fun")
}

func (s *testSuite) TestParseProtoChain_OptionalChain(c *C) {
	chain := parseProtoChain([]string {"[id", "more", "fun]"})
	c.Assert(chain.optional, NotNil)
	c.Assert(chain.optional.id, Equals, "id")
	c.Assert(chain.optional.next.id, Equals, "more")
	c.Assert(chain.optional.next.next.id, Equals, "fun")

	chain = parseProtoChain([]string {"[hello]", "[more]", "[there]"})
	c.Assert(chain.optional, NotNil)
	c.Assert(chain.optional.id, Equals, "hello")
	c.Assert(chain.next.optional, NotNil)
	c.Assert(chain.next.optional.id, Equals, "more")
	c.Assert(chain.next.next.optional, NotNil)
	c.Assert(chain.next.next.optional.id, Equals, "there")

	chain = parseProtoChain([]string {"[hello", "[more]]", "[there]"})
	c.Assert(chain.optional, NotNil)
	c.Assert(chain.optional.id, Equals, "hello")
	c.Assert(chain.optional.next.optional, NotNil)
	c.Assert(chain.optional.next.optional.id, Equals, "more")
	c.Assert(chain.next.optional, NotNil)
	c.Assert(chain.next.optional.id, Equals, "there")

	chain = parseProtoChain([]string {"hello", "[[more]", "there]"})
	c.Assert(chain.id, Equals, "hello")
	c.Assert(chain.next.optional.optional, NotNil)
	c.Assert(chain.next.optional.optional.id, Equals, "more")
	c.Assert(chain.next.optional.next, NotNil)
	c.Assert(chain.next.optional.next.id, Equals, "there")
}
