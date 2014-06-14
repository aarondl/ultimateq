#ultimateq

[![Build Status](https://drone.io/github.com/aarondl/ultimateq/status.png)](https://drone.io/github.com/aarondl/ultimateq/latest) [![Coverage Status](http://coveralls.io/repos/aarondl/ultimateq/badge.png?branch=master)](http://coveralls.io/r/aarondl/ultimateq?branch=master)

An irc bot written in Go.

ultimateq is a distributed irc bot framework. It allows you to create a bot
with a single file (see simple.go for a good example), or to create many
extensions that can run independently and hook them up to one bot, or allow
many bots to connect to them.

Here's a sample of the bot api for some do-nothing connection to an irc server.

```go

import (
    "github.com/aarondl/ultimateq/bot"
    "os/signal"
    "log"
)

func main() {
    cfg := config.NewConfig().FromFile("config.toml")
    b, err := bot.NewBot(cfg)
    if err != nil {
        log.Fatalln("Error creating bot:", err)
    }
    defer b.Close() // Required to close the database.
    
    end := b.Start()
    
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, os.Kill)

    stop := false
    for !stop {
        select {
        case <-quit:
            b.Stop()
            stop = true
        case err, ok := <-end:
            log.Println("Server death:", err)
            stop = !ok
        }
    }    
    log.Println("Shutting down...")
}
```

Here's a quick sample config.yaml for use with the above:

```toml
nick = "Bot"
altnick = "Bot"
username = "notabot"
realname = "A real bot"

[networks.test]
	servers = ["localhost:3337"]
	ssl = true
```

##Package status

The bot is roughly 60% complete. The internal packages are nearly 100%
complete except the outstanding issues that don't involve extensions here:
https://github.com/aarondl/ultimateq/issues/

The following major pieces are currently missing:

* Front ends for people who want an out of the box bot.
* Extensions.
* Nice way to create static modules without the boilerplate of loading a config,
starting the bot etc.

##Packages

####bot
This package ties all the low level plumbing together, using this package's
helpers it should be easy to create a bot and deploy him into the dying world
of irc.

####irc
This package houses the common irc constants and types necessary throughout
the bot. It's supposed to remain a small and dependency-less package that all
packages can utilize.

####config
Config package is used to present a fluent style configuration builder for an
irc bot. It also provides some validation, and file reading/writing.

####parse
This package deals with parsing irc protocols. It is able to consume irc
protocol messages using the Parse method, returning the common irc.Message
type from the irc package.

####dispatch
Dispatch package is meant to register callbacks and dispatch irc.Messages onto
them in an asynchronous method. It also presents many handler types that will
be easy to use for bot-writers.

####dispatch/commander
Commander package is a much more involved version of the dispatcher. Instead of
simply responding to irc raw messages, the commander parses arguments, handles
user authentication and privelege checks. It can also do advanced argument
handling such as returning a user message's target channels or users from the
command.

####inet
Implements the actual connection to an irc server, handles buffering, \r\n
splitting and appending, and logarithmic write-speed throttling.

####extension
This package defines helpers to create an extension for the bot. It should
expose a way to connect/allow connections to the bot via TCP or Unix socket.
And have simple helpers for some network RPC mechanism.

####data
This package holds state information for irc. As well as stores user authentication
and access information in a key-value database. (Many thanks to Jan Merci for this
great package: https://github.com/cznic/kv)
