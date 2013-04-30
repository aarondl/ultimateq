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

type BotConfig struct {
    name string
    channels []string // Assuming the bot can be in multiple channels.
}

// Disables some warnings during testing
var testing_mode bool = false;

// Returns a bot configuration interface
func Create() *BotConfig {
    return &BotConfig{}
}

// Check that the bot name is a valid IRC nickname
func check_valid_nick(name string) bool {
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
    // Cannot have a bot with no name
    if len(name) == 0 {
        return nil
    }

    // Cannot have a bot with a too long name
    if len(name) > MAX_NAME_LENGTH {
        return nil
    }

    // Check if it's a valid nickname
    if !check_valid_nick(name) {
        return nil
    }

    // Name was valid
    bot.name = name
    return bot
}

// TODO: Add the bot to a channel
func (bot *BotConfig) Channels(channel_name string) *BotConfig {
    return nil
}
