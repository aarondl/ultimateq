package bot

import (
	. "launchpad.net/gocheck"
	"testing"
)

func Test(t *testing.T) { TestingT(t) } //Hook into testing package

type s struct{}

var _ = Suite(&s{})

// Test the conditions on an initially created bot.
func (s *s) TestInitialState(c *C) {
	bot := Create()

	c.Assert(bot.name, Equals, "")
	c.Assert(len(bot.channels), Equals, 0)
}

// Test that the name satisfies various conditions
func (s *s) TestName(c *C) {
	// Not permissible to have an empty name
	bot := Create().Name("")
	c.Assert(bot, Equals, (*BotConfig)(nil))

	// A bot should have the name passed as param
	bot = Create().Name("Foo")
	c.Assert(bot.name, Equals, "Foo")

	long_name := ""

	for i := 0; i < MAX_NAME_LENGTH+1; i++ {
		long_name += "a"
	}

	// A bot should not have an excessively long name
	bot = Create().Name(long_name)
	c.Assert(bot, Equals, (*BotConfig)(nil))
}

// From the RFC:
// nickname   =  ( letter / special ) *8( letter / digit / special / "-" )
// letter     =  %x41-5A / %x61-7A       ; A-Z / a-z
// digit      =  %x30-39                 ; 0-9
// special    =  %x5B-60 / %x7B-7D; "[", "]", "\", "`", "_", "^", "{", "|", "}"
// We make an excemption to the 9 char limit since few servers today enforce it,
// and the RFC also states that clients should handle longer names.
//
// Test that the name is a valid IRC nickname
func (s *s) TestValidNames(c *C) {
	testing_mode = true

	// Cannot contain spaces
	c.Assert(checkValidNickName("My Name"), Equals, false)

	// Can contain numbers after at least one character
	c.Assert(checkValidNickName("1abc"), Equals, false)
	c.Assert(checkValidNickName("5abc"), Equals, false)
	c.Assert(checkValidNickName("9abc"), Equals, false)
	c.Assert(checkValidNickName("a1bc"), Equals, true)
	c.Assert(checkValidNickName("a5bc"), Equals, true)
	c.Assert(checkValidNickName("a9bc"), Equals, true)

	// Check that our test is otherwise ok
	c.Assert(checkValidNickName("MyNick"), Equals, true)

	// Cannot contain the following characters:
	c.Assert(checkValidNickName("My!Nick"), Equals, false)
	c.Assert(checkValidNickName("My\"Nick"), Equals, false)
	c.Assert(checkValidNickName("My#Nick"), Equals, false)
	c.Assert(checkValidNickName("My$Nick"), Equals, false)
	c.Assert(checkValidNickName("My%Nick"), Equals, false)
	c.Assert(checkValidNickName("My&Nick"), Equals, false)
	c.Assert(checkValidNickName("My'Nick"), Equals, false)
	c.Assert(checkValidNickName("My/Nick"), Equals, false)
	c.Assert(checkValidNickName("My(Nick"), Equals, false)
	c.Assert(checkValidNickName("My)Nick"), Equals, false)
	c.Assert(checkValidNickName("My*Nick"), Equals, false)
	c.Assert(checkValidNickName("My+Nick"), Equals, false)
	c.Assert(checkValidNickName("My,Nick"), Equals, false)
	c.Assert(checkValidNickName("My-Nick"), Equals, false)
	c.Assert(checkValidNickName("My.Nick"), Equals, false)
	c.Assert(checkValidNickName("My/Nick"), Equals, false)
	c.Assert(checkValidNickName("My;Nick"), Equals, false)
	c.Assert(checkValidNickName("My:Nick"), Equals, false)
	c.Assert(checkValidNickName("My<Nick"), Equals, false)
	c.Assert(checkValidNickName("My=Nick"), Equals, false)
	c.Assert(checkValidNickName("My>Nick"), Equals, false)
	c.Assert(checkValidNickName("My?Nick"), Equals, false)
	c.Assert(checkValidNickName("My@Nick"), Equals, false)

	// Can contain the following in any position
	c.Assert(checkValidNickName("[MyNick"), Equals, true)
	c.Assert(checkValidNickName("My[Nick"), Equals, true)
	c.Assert(checkValidNickName("]MyNick"), Equals, true)
	c.Assert(checkValidNickName("My]Nick"), Equals, true)
	c.Assert(checkValidNickName("\\MyNick"), Equals, true)
	c.Assert(checkValidNickName("My\\Nick"), Equals, true)
	c.Assert(checkValidNickName("`MyNick"), Equals, true)
	c.Assert(checkValidNickName("My`Nick"), Equals, true)
	c.Assert(checkValidNickName("_MyNick"), Equals, true)
	c.Assert(checkValidNickName("My_Nick"), Equals, true)
	c.Assert(checkValidNickName("^MyNick"), Equals, true)
	c.Assert(checkValidNickName("My^Nick"), Equals, true)
	c.Assert(checkValidNickName("{MyNick"), Equals, true)
	c.Assert(checkValidNickName("My{Nick"), Equals, true)
	c.Assert(checkValidNickName("|MyNick"), Equals, true)
	c.Assert(checkValidNickName("My|Nick"), Equals, true)
	c.Assert(checkValidNickName("}MyNick"), Equals, true)
	c.Assert(checkValidNickName("My}Nick"), Equals, true)

	// Various sanity tests
	c.Assert(checkValidNickName("@ChanServ"), Equals, false)

	testing_mode = false
}

func (s *s) TestValidChannels(c *C) {
	// Check that the first letter must be {#+!&}
	c.Assert(checkValidChannelName("InvalidChannel"), Equals, false)

	c.Assert(checkValidChannelName("#"), Equals, false)
	c.Assert(checkValidChannelName("+"), Equals, false)
	c.Assert(checkValidChannelName("&"), Equals, false)

	c.Assert(checkValidChannelName("#ValidChannel"), Equals, true)
	c.Assert(checkValidChannelName("+ValidChannel"), Equals, true)
	c.Assert(checkValidChannelName("&ValidChannel"), Equals, true)

	c.Assert(checkValidChannelName("!12345"), Equals, true)

	c.Assert(checkValidChannelName("#Invalid Channel"), Equals, false)
	c.Assert(checkValidChannelName("#Invalid,Channel"), Equals, false)
	c.Assert(checkValidChannelName("#Invalid\aChannel"), Equals, false)

	c.Assert(checkValidChannelName("#c++"), Equals, true)

}
