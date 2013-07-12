#ultimateq

[![Build Status](https://drone.io/github.com/aarondl/ultimateq/status.png)](https://drone.io/github.com/aarondl/ultimateq/latest)

An irc bot written in Go.

ultimateq is designed as a distributed irc bot framework. Where a bot can be
just one file, or a collection of extensions running alongside the bot. A
key feature in this bot is that the extensions are actually processes
themselves which will connect to a bot, or allow bots to connect to them
via unix or tcp sockets using RPC mechanisms to communicate.

There are two advantages behind this model. One is that several bots can use
a single extension. This has advantages if the extension is holding a
database meant to be shared (although for scalability you might consider
database level scaling with replication, but this is just an example).

The other advantage is that the bot has failure isolation. Because each
extension is running in it's own process, even a fatal and unrecoverable
crash in an extension means nothing to the bot. This adds a resilency to
this bot that not many other bots share.

Here's a sample taste of what the bot api might look like for some do-nothing
connection to an irc server.

```go
import bot "github.com/aarondl/ultimateq"
func main() {
  bot.Nick("mybot").Channels("#C++").Run()
}
```

##Packages

###bot
This package ties all the low level plumbing together, using this package's
helpers it should be easy to create a bot and deploy him into the dying world
of irc.

###irc
This package houses the common irc constants and types necessary throughout
the bot. It's supposed to remain a small and dependency-less package that all
packages can utilize.

###config
Config package is used to present a fluent style configuration builder for an
irc bot. It also provides some validation, and file reading/writing.

###parse
This package deals with parsing irc protocols. It is able to consume irc
protocol messages using the Parse method, returning the common IrcMessage
type from the irc package.

###dispatch
Dispatch package is meant to register callbacks and dispatch IrcMessages onto
them in an asynchronous method. It also presents many handler types that will
be easy to use for bot-writers.

###inet
Implements the actual connection to an irc server, handles buffering, \r\n
splitting and appending, and logarithmic write-speed throttling.

###extension
This package defines helpers to create an extension for the bot. It should
expose a way to connect/allow connections to the bot via TCP or Unix socket.
And have simple helpers for some network RPC mechanism.

###data
This package holds data for irc based services. It supplies information
about channels, and users that the bot is aware of.

Persistence may need some attention. Does this bot have built-in user
levels and modes for users? Does it have arbitrary key-value
store for extensions? How do extensions keep this data up to date? Will
they have their own copy of the data or do they need to query the data on the
master bot all the time?
