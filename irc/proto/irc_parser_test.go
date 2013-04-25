package proto

import (
	. "launchpad.net/gocheck"
	"strings"
)

func (s *testSuite) TestCreateIrcParser(c *C) {
	parser := CreateIrcParser()
	c.Assert(parser, NotNil)
	c.Assert(parser.handlers, NotNil)
	c.Assert(parser, FitsTypeOf, new(IrcParser))
}

func (s *testSuite) TestCreateParseResult(c *C) {
	result := createParseResult()
	c.Assert(result, NotNil)
	c.Assert(result.Args, NotNil)
	c.Assert(result.Argv, NotNil)
	c.Assert(result.Channels, NotNil)
	c.Assert(result, FitsTypeOf, new(ParseResult))
}

func (s *testSuite) TestAddAndRemoveParseTree(c *C) {
	parser := CreateIrcParser()
	upper, lower := "PING", "ping"
	err := parser.AddIrcHandler(upper, nil)
	c.Assert(err, IsNil)
	err = parser.AddIrcHandler(lower, nil)
	c.Assert(err, Equals, errHandlerAlreadyRegistered)
	err = parser.RemoveIrcHandler(upper)
	c.Assert(err, IsNil)
	err = parser.RemoveIrcHandler(lower)
	c.Assert(err, NotNil)
	c.Assert(err, Equals, errHandlerNotRegistered)
}

func (s *testSuite) TestParseNoIrc(c *C) {
	parser := CreateIrcParser()
	_, err := parser.Parse("", nil)
	c.Assert(err, NotNil)
	c.Assert(err, Equals, errNoProtocolGiven)
}

func (s *testSuite) TestParseIrc(c *C) {
	parser := CreateIrcParser()
	chain, err := createFragmentChain([]string{":id"})
	c.Assert(err, IsNil)
	err = parser.AddIrcHandler("PING", chain)
	c.Assert(err, IsNil)
	result, err := parser.Parse("PING :arg1 arg2", nil)
	c.Assert(err, IsNil)
	c.Assert(result.Name, Equals, "ping")
	c.Assert(result.Args["id"], Equals, "arg1 arg2")
}

func (s *testSuite) TestWalkProto_Basic(c *C) {
	id := "id"
	chain, err := createFragmentChain([]string{id})
	proto := []string{"arg"}
	result := createParseResult()
	err = walkProto(chain, proto, result, nil)
	c.Assert(err, IsNil)
	c.Assert(result.Args[id], Equals, proto[0])
}

func (s *testSuite) TestWalkProto_Final(c *C) {
	id := "id"
	chain, err := createFragmentChain([]string{":" + id})
	proto := []string{":arg1", "arg2"}
	result := createParseResult()
	err = walkProto(chain, proto, result, nil)
	c.Assert(err, IsNil)
	c.Assert(result.Args[id], Equals, proto[0][1:]+" "+proto[1])

	result = createParseResult()
	err = walkProto(chain, proto[1:], result, nil)
	c.Assert(err, IsNil)
	c.Assert(result.Args[id], Equals, proto[1])
}

func (s *testSuite) TestWalkProto_Channels(c *C) {
	id, id2 := "id", "id2"
	strs := []string{"#chan1", "#chan2", "#chan3"}
	chain, err := createFragmentChain([]string{"*#" + id, "#" + id2})
	proto := []string{strs[0] + "," + strs[1], strs[2]}
	result := createParseResult()
	err = walkProto(chain, proto, result, &ProtoCaps{Chantypes:"#"})
	c.Assert(err, IsNil)
	c.Assert(result.Channels[id][0], Equals, strs[0])
	c.Assert(result.Channels[id][1], Equals, strs[1])
	c.Assert(result.Channels[id2][0], Equals, strs[2])
	c.Assert(result.Args[id], Equals, proto[0])
	c.Assert(result.Args[id2], Equals, proto[1])
}

func (s *testSuite) TestWalkProto_NArgs(c *C) {
	id := "id"
	strs := []string{"arg1", "arg2", "arg3"}
	chain, err := createFragmentChain([]string{"*" + id})
	result := createParseResult()
	err = walkProto(chain, []string{strings.Join(strs, ",")}, result, nil)
	c.Assert(err, IsNil)
	for i, v := range strs {
		c.Assert(result.Argv[id][i], Equals, v)
	}
}

func (s *testSuite) TestWalkProto_Optionals(c *C) {
	ids := []string{"id1", "id2", "id3"}
	strs := []string{"arg1", "arg2", "arg3"}
	chain, err := createFragmentChain(
		[]string{ids[0], "[" + ids[1] + "]", "[" + ids[2] + "]"},
	)

	result := createParseResult()
	err = walkProto(chain, strs, result, nil)
	c.Assert(err, IsNil)
	for i, v := range ids {
		c.Assert(result.Args[v], Equals, strs[i])
	}

	result = createParseResult()
	err = walkProto(chain, strs[:len(strs)], result, nil)
	c.Assert(err, IsNil)
	for i, v := range ids[:len(ids)] {
		c.Assert(result.Args[v], Equals, strs[i])
	}
}

func (s *testSuite) TestWalkProto_ArgsFollowedNoColonFinal(c *C) {
	chain, err := createFragmentChain([]string{":id"})
	result := createParseResult()
	err = walkProto(chain, []string{"arg1", "arg2"}, result, nil)
	c.Assert(err, Equals, errArgsAfterFinalNoColon)
}

func (s *testSuite) TestWalkProto_ExpectedMoreArguments(c *C) {
	chain, err := createFragmentChain([]string{"id1", "id2"})
	result := createParseResult()
	err = walkProto(chain, []string{"arg1"}, result, nil)
	c.Assert(err, Equals, errExpectedMoreArguments)
}

func (s *testSuite) TestHandleFinalChain(c *C) {
	args := []string{"arg1", "arg2"}
	str, err := handleFinalChain(0, []string{":" + args[0], args[1]})
	c.Assert(err, IsNil)
	c.Assert(str, Equals, strings.Join(args, " "))
	str, err = handleFinalChain(0, []string{args[0]})
	c.Assert(err, IsNil)
	c.Assert(str, Equals, args[0])
	_, err = handleFinalChain(0, []string{args[0], args[1]})
	c.Assert(err, Equals, errArgsAfterFinalNoColon)
	str, err = handleFinalChain(3, []string{args[0], args[1]})
	c.Assert(str, Equals, "")
	c.Assert(err, IsNil)
}

func (s *testSuite) TestValidateChannels(c *C) {
	caps := &ProtoCaps{Chantypes: "&#"}
	chans := []string{"#c1", "&c2", "_c3"}
	chanline := strings.Join(chans, ",")
	valid := validateChannels(chanline, caps)
	c.Assert(valid[0], Equals, chans[0])
	c.Assert(valid[1], Equals, chans[1])
	c.Assert(2, Equals, len(valid))
	valid = validateChannels(chanline, nil)
	c.Assert(0, Equals, len(valid))
	valid = validateChannels("", caps)
	c.Assert(0, Equals, len(valid))
}
