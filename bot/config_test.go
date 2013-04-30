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

    for i := 0; i < MAX_NAME_LENGTH + 1; i++ {
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
    c.Assert(check_valid_nick("My Name"), Equals, false)

    // Can contain numbers after at least one character
    c.Assert(check_valid_nick("1abc"), Equals, false)
    c.Assert(check_valid_nick("5abc"), Equals, false)
    c.Assert(check_valid_nick("9abc"), Equals, false)
    c.Assert(check_valid_nick("a1bc"), Equals, true)
    c.Assert(check_valid_nick("a5bc"), Equals, true)
    c.Assert(check_valid_nick("a9bc"), Equals, true)

    // Check that our test is otherwise ok
    c.Assert(check_valid_nick("MyNick"), Equals, true);

    // Cannot contain the following characters:
    c.Assert(check_valid_nick("My!Nick"), Equals, false);
    c.Assert(check_valid_nick("My\"Nick"), Equals, false);
    c.Assert(check_valid_nick("My#Nick"), Equals, false);
    c.Assert(check_valid_nick("My$Nick"), Equals, false);
    c.Assert(check_valid_nick("My%Nick"), Equals, false);
    c.Assert(check_valid_nick("My&Nick"), Equals, false);
    c.Assert(check_valid_nick("My'Nick"), Equals, false);
    c.Assert(check_valid_nick("My/Nick"), Equals, false);
    c.Assert(check_valid_nick("My(Nick"), Equals, false);
    c.Assert(check_valid_nick("My)Nick"), Equals, false);
    c.Assert(check_valid_nick("My*Nick"), Equals, false);
    c.Assert(check_valid_nick("My+Nick"), Equals, false);
    c.Assert(check_valid_nick("My,Nick"), Equals, false);
    c.Assert(check_valid_nick("My-Nick"), Equals, false);
    c.Assert(check_valid_nick("My.Nick"), Equals, false);
    c.Assert(check_valid_nick("My/Nick"), Equals, false);
    c.Assert(check_valid_nick("My;Nick"), Equals, false);
    c.Assert(check_valid_nick("My:Nick"), Equals, false);
    c.Assert(check_valid_nick("My<Nick"), Equals, false);
    c.Assert(check_valid_nick("My=Nick"), Equals, false);
    c.Assert(check_valid_nick("My>Nick"), Equals, false);
    c.Assert(check_valid_nick("My?Nick"), Equals, false);
    c.Assert(check_valid_nick("My@Nick"), Equals, false);

    // Can contain the following in any position
    c.Assert(check_valid_nick("[MyNick"), Equals, true);
    c.Assert(check_valid_nick("My[Nick"), Equals, true);
    c.Assert(check_valid_nick("]MyNick"), Equals, true);
    c.Assert(check_valid_nick("My]Nick"), Equals, true);
    c.Assert(check_valid_nick("\\MyNick"), Equals, true);
    c.Assert(check_valid_nick("My\\Nick"), Equals, true);
    c.Assert(check_valid_nick("`MyNick"), Equals, true);
    c.Assert(check_valid_nick("My`Nick"), Equals, true);
    c.Assert(check_valid_nick("_MyNick"), Equals, true);
    c.Assert(check_valid_nick("My_Nick"), Equals, true);
    c.Assert(check_valid_nick("^MyNick"), Equals, true);
    c.Assert(check_valid_nick("My^Nick"), Equals, true);
    c.Assert(check_valid_nick("{MyNick"), Equals, true);
    c.Assert(check_valid_nick("My{Nick"), Equals, true);
    c.Assert(check_valid_nick("|MyNick"), Equals, true);
    c.Assert(check_valid_nick("My|Nick"), Equals, true);
    c.Assert(check_valid_nick("}MyNick"), Equals, true);
    c.Assert(check_valid_nick("My}Nick"), Equals, true);

    // Various sanity tests
    c.Assert(check_valid_nick("@ChanServ"), Equals, false);

    testing_mode = false
}
