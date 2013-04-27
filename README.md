#ultimateq

An irc bot written in Go.

ultimateq is designed as a distributed irc bot framework. Where a bot can be just one file, or a collection
of extensions running alongside the bot. A key feature in this bot is that the extensions are actually
processes themselves which will connect to a bot, or allow bots to connect to them via unix or tcp
sockets using RPC mechanisms to communicate.

There are two advantages behind this model. One is that several bots can use a single extension.
This has advantages if the extension is holding a database meant to be shared (although for scalability
you might consider database level scaling with replication, but this is just an example).

The other advantage is that the bot is isolation failure. Because each extension is running in it's own process,
even a fatal and unrecoverable crash in an extension means nothing to the bot.
This adds a resilency to this bot that not many other bots share.

##Packages

###ultimateq
This package ties all the low level plumbing things together, providing a fluent syntax interface
for initializing a bot. This module is designed to be imported and used as a start-point for a
bot. It should also be able to easily register event callbacks and handle them for one-file bots.

```go
import bot "github.com/aarondl/ultimateq"
func main() {
  bot.Nick("mybot").Channels("#C++").Run()
}
```

###extension
This package defines helpers to create an extension for the bot. It should expose a way to connect/allow connections to the bot via TCP or Unix socket. And have simple helpers for some network RPC mechanism (JSON-RPC is a likely candidate).

###irc
This package deals with irc protocols. Parsing, Message construction helpers, and Validation.

The following types should be expected:

- IrcConnection: A connection to an irc server. Wraps calls for irc/network
- IrcParser: An irc message deconstructor.
- IrcMessage: A deconstructed irc message. The result of IrcParser.
- IrcBuilder: A message builder object.
- IrcValidator: An irc protocol validator. Should work with IrcBuilder to ensure validity.

###irc/network
Implements the actual connection to the irc server, handles buffering, logarithmic write-speed throttling.

###data
This package holds data for irc based services. It supplies information about channels, and users
that the bot is aware of.

Persistence may need some attention. Does this bot have built-in user levels and modes for users? Does it
have arbitrary key-value store for extensions?
