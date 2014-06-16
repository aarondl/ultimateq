# ultimateq

[![Build Status](https://drone.io/github.com/aarondl/ultimateq/status.png)](https://drone.io/github.com/aarondl/ultimateq/latest) [![Coverage Status](http://coveralls.io/repos/aarondl/ultimateq/badge.png?branch=master)](http://coveralls.io/r/aarondl/ultimateq?branch=master)

An irc bot framework written in Go.

ultimateq is a distributed irc bot framework. It allows you to create a bot
with a single file (see simple.go for a good example), or to create many
extensions that can run independently and hook them up to one bot, or allow
many bots to connect to them.

What follows is a sample of the bot api for some basic greeter bot.
Keep in mind that he can use much more fine-grained APIs allowing you more
control of how he's run. ALso see simple.go for a much bigger example.

The bot.Run() function reads in a config.toml, sets up keyboard and signal
handlers, and runs the bot until all networks are permanently disconnected.

```go
package main

import (
	"log"

	"github.com/aarondl/ultimateq/bot"
	"github.com/aarondl/ultimateq/dispatch/cmd"
	"github.com/aarondl/ultimateq/irc"
)

// Greeter will be our handler type for both commands and simple events.
type Greeter struct{}

// Cmd is the interface method that's required by all command handlers, this
// method is fallback for any commands that are not reachable through reflection
func (_ Greeter) Cmd(command string, _ irc.Writer, _ *cmd.Event) error {
	switch command {
	case "hello":
		// Do something
	}
	return nil
}

// This method will be invoked via reflection based on the command name. This
// way we don't need the switch case above. If this method did not exist it
// would fallback to the above method when the command was issued by a user.
func (_ Greeter) Hello(w irc.Writer, e *cmd.Event) error {
	// Remember to call e.Close() if you do any long processing here.
	// There are internal database locks engaged at this point to make using
	// them safe.

	// Write out our message.
	// Normally we'd have to check if e.Channel is nil, but it's safe since
	// we registered with the cmd.PUBLIC (no private message allowed) flag for
	// msg scope.
	w.Privmsgf(e.Channel.Name(), "Hello to you too %s!", e.Nick())

	return nil
}

/*
Currently one of these functions is required to handle events, they all do
some basic event filtering, for example PrivmsgChannel will only get PRIVMSG
events that are sent to a channel.
	HandleRaw(irc.Writer, *irc.Event)
	Privmsg(irc.Writer, *irc.Event)
	PrivmsgUser(irc.Writer, *irc.Event)
	PrivmsgChannel(irc.Writer, *irc.Event)
	Notice(irc.Writer, *irc.Event)
	NoticeUser(irc.Writer, *irc.Event)
	NoticeChannel(irc.Writer, *irc.Event)
	CTCP(irc.Writer, *irc.Event, string, string)
	CTCPChannel(irc.Writer, *irc.Event, string, string)
	CTCPReply(irc.Writer, *irc.Event, string, string)
*/
func (_ Greeter) HandleRaw(w irc.Writer, e *irc.Event) {
	// Because of the way we've registered, only JOIN events will be dispatched,
	// but for later growth we could do this.
	switch e.Name {
	case irc.JOIN:
		// Write the message out.
		w.Privmsgf(e.Target(), "Welcome to %s %s!", e.Target(), e.Nick())
	}
}

func main() {
	err := bot.Run(func(b *bot.Bot) {
		// Basic Command - See cmd package documentation.
		b.RegisterCmd(cmd.MkCmd(
			"myExtension",
			"Says hello to someone",
			"hello",
			&Greeter{},
			cmd.PRIVMSG, cmd.PUBLIC,
		))

		// To make an Authenticated command using the user database simply
		// use the cmd.MkAuthCmd method to create your commands.
		// b.RegisterCmd(cmd.MkAuthCmd(...

		// Basic Handler
		b.Register(irc.JOIN, &Greeter{})
	})

	log.Println(err)
}
```

Here's a quick sample config.toml for use with the above, see the config
package documentation for a full configuration sample.

```toml
nick = "Bot"
altnick = "Bot"
username = "notabot"
realname = "A real bot"

[networks.test]
	servers = ["localhost:3337"]
	ssl = true
```

## Package status

The bot is roughly 60% complete. The internal packages are nearly 100%
complete except the outstanding issues that don't involve extensions here:
https://github.com/aarondl/ultimateq/issues/

The following major pieces are currently missing:

* Front ends for people who want an out of the box bot.
* Extensions.

## Packages

#### bot
This package ties all the low level plumbing together, using this package's
helpers it should be easy to create a bot and deploy him into the dying world
of irc.

#### irc
This package houses the common irc constants and types necessary throughout
the bot. It's supposed to remain a small and dependency-less package that all
packages can utilize.

#### config
Config package is used to present a fluent style configuration builder for an
irc bot. It also provides some validation, and file reading/writing.

#### parse
This package deals with parsing irc protocols. It is able to consume irc
protocol messages using the Parse method, returning the common irc.Event
type from the irc package.

#### dispatch
Dispatch package is meant to register callbacks and dispatch irc.Events onto
them in a new goroutine. It also presents many handler interfaces that help
filter messages.

#### dispatch/cmd
Cmd package is a much more involved version of the dispatcher. Instead of
simply responding to irc raw messages, the commander parses arguments, handles
user authentication and privelege checks. It can also do advanced argument
handling such as returning a user message's target channels or user arguments
from the command.

#### inet
Implements the actual connection to an irc server, handles buffering, \r\n
splitting and appending, filtering, and write-speed throttling.

#### extension
This package defines helpers to create an extension for the bot. It should
expose a way to connect/allow connections to the bot via TCP or Unix socket.
And have simple helpers for some network RPC mechanism.

#### data
This package holds state information for all the irc servers. It can also store
user and channel data, and authenticate users to be used in protected commands
from the cmd package. The database is a key-value store written in Go.
(Many thanks to Jan Merci for this great package: https://github.com/cznic/kv)
