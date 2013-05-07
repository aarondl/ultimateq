package bot

import (
	"log"
	"regexp"
)

var (
	// From the RFC:
	// nickname   =  ( letter / special ) *8( letter / digit / special / "-" )
	// letter     =  %x41-5A / %x61-7A  ; A-Z / a-z
	// digit      =  %x30-39            ; 0-9
	// special    =  %x5B-60 / %x7B-7D  ; [ ] \ ` _ ^ { | }
	// We make an excemption to the 9 char limit since few servers today
	// enforce it, and the RFC also states that clients should handle longer
	// names.
	// Test that the name is a valid IRC nickname
	nicknameRegex = regexp.MustCompile(`^(?i)[a-z\[\]{}|^_\\` + "`]" +
		`[a-z0-9\[\]{}|^_\\` + "`" + `]{0,30}$`)

	/* Channels names are strings (beginning with a '&', '#', '+' or '!'
	character) of length up to fifty (50) characters.  Apart from the
	requirement that the first character is either '&', '#', '+' or '!',
	the only restriction on a channel name is that it SHALL NOT contain
	any spaces (' '), a control G (^G or ASCII 7), a comma (',').  Space
	is used as parameter separator and command is used as a list item
	separator by the protocol).  A colon (':') can also be used as a
	delimiter for the channel mask.  Channel names are case insensitive.

	Grammar:
	channelid  = 5( %x41-5A / digit )   ; 5( A-Z / 0-9 )
	chanstring = any octet except NUL, BELL, CR, LF, " ", "," and ":"
	channel    =  ( "#" / "+" / ( "!" channelid ) / "&" ) chanstring
					[ ":" chanstring ] */
	channelRegex = regexp.MustCompile(
		`^(?i)[#&+!][^\s\000\007,]{1,49}$`)
)

// Config enables the fluent configuration of an irc bot.
type Config struct {
	name string
}

type ServerConfig struct {
	host string
	port uint

	nick string
	altnick string
	realname string
	hostname string
}

// Returns a bot configuration interface
func Configure() *Config {
	return &Config{}
}

func (c *Config) Server(string host) *ServerConfig {
}

// Set the name of the bot
func (bot *Config) Name(name string) *Config {
	// Check if it's a valid nickname
	if !nicknameRegex.MatchString(name) {
		log.Println("Failed to create bot. Bad nickname.")
		return nil
	}

	// Name was valid
	bot.name = name
	return bot
}
