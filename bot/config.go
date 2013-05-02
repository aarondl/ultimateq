package bot

import (
    "regexp"
    "fmt"
)

// Reference: tools.ietf.org/html/rfc2812

// TODO: 
// Some other way of handling attempts at bad configurations apart from 
// returning nil?
// 
// Servers specify [MAX]NICKLEN/[MAX]CHANNELLEN and various other limits, 
// how should this be handled?
//
// Option 1:
//     Specify server in Create("irc.gamesurge.net"), and have the server info
//     loaded into config when the server replies.
// Option 2: (current implementation)
//     Set our own predefined limits and (in the future, perhaps)
//     truncate any data that exceeds server limits upon receiving
//     info from the server.
// Option 3:
//     Do nothing.

// Settings mostly inspired by GameSurge
const (
    MAX_NAME_LENGTH = 30
)

// Configuration interface for IRC bots
type BotConfig struct {
    name string       // Name of the IRC bot
    channels []string // List of channels this bot will join
}

// TODO: Ditch this, and stack errors on the config struct instead.
// Disables some warnings during testing
var testing_mode bool = false;

// Returns a bot configuration interface
func Create() *BotConfig {
    return &BotConfig{}
}

// Check that the bot name is a valid IRC nickname
// TODO: camelCase
func checkValidNickName(name string) bool {
    // Cannot have a bot with no name
    if len(name) == 0 {
        return false
    }

    // Cannot have a bot with a too long name
    if len(name) > MAX_NAME_LENGTH {
        return false
    }

    name_regex := "^[a-zA-Z\\[\\]\\\\`_\\^\\{\\}\\|][a-zA-Z\\d\\[\\]\\\\`_\\^\\{\\}\\|]*$"
    match, err := regexp.MatchString(name_regex, name)

    if err != nil {
        fmt.Printf("Unable to verify if %v is a valid nickname\n", name)
        fmt.Println(err)
        return false
    }

    if !match {
        // TODO: Print a list of rules?
        if !testing_mode {
            fmt.Printf("%v is not a valid nickname for a bot!\n", name)
        }

        return false
    }

    return true
}

// Set the name of the bot
func (bot *BotConfig) Name(name string) *BotConfig {
    // Check if it's a valid nickname
    if !checkValidNickName(name) {
        return nil
    }

    // Name was valid
    bot.name = name
    return bot
}

/*
Channels names are strings (beginning with a '&', '#', '+' or '!'
character) of length up to fifty (50) characters.  Apart from the
requirement that the first character is either '&', '#', '+' or '!',
the only restriction on a channel name is that it SHALL NOT contain
any spaces (' '), a control G (^G or ASCII 7), a comma (',').  Space
is used as parameter separator and command is used as a list item
separator by the protocol).  A colon (':') can also be used as a
delimiter for the channel mask.  Channel names are case insensitive.
*/
/*
Grammar:
channelid  = 5( %x41-5A / digit )   ; 5( A-Z / 0-9 )
chanstring = any octet except NUL, BELL, CR, LF, " ", "," and ":"
channel    =  ( "#" / "+" / ( "!" channelid ) / "&" ) chanstring
                [ ":" chanstring ]
*/
func checkValidChannelName(channel string) bool {
    if len(channel) <= 1 {
        return false
    }

    if len(channel) > 50 {
        return false
    }

    //name_regex := "^[a-zA-Z\\[\\]\\\\`_\\^\\{\\}\\|][a-zA-Z\\d\\[\\]\\\\`_\\^\\{\\}\\|]*$"
    channel_regex := `^([#&+]|![a-zA-Z\d]{5})[^ ,\a\n\r\001]*$`
    match, err := regexp.MatchString(channel_regex, channel)

    if err != nil {
        fmt.Println("Regex failure:", err)
        return false
    }

    if !match {
        return false
    }

    return true
}

// TODO: Add the bot to a channel
func (bot *BotConfig) Channels(channel_name string) *BotConfig {
    return nil
}
