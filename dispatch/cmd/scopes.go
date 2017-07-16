package cmd

// Kind is the kind of messages to listen to.
type Kind int

// Scope is the scope of the messages to listen to.
type Scope int

// Constants used for defining the targets/scope of a command.
const (
	// KindPrivmsg only listens to irc.PRIVMSG events.
	Privmsg Kind = 0x1
	// KindNotice only listens to irc.NOTICE events.
	Notice Kind = 0x2
	// AnyKind listens to both irc.PRIVMSG and irc.NOTICE events.
	AnyKind Kind = 0x3

	// Private only listens to PRIVMSG or NOTICE sent directly to the bot.
	Private Scope = 0x1
	// PUBLIC only listens to PRIVMSG or NOTICE sent to a channel.
	Public Scope = 0x2
	// AnyScope listens to events sent to a channel or directly to the bot.
	AnyScope Scope = 0x3
)
