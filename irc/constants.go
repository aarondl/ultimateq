package irc

// IRC Messages, these messages are 1-1 constant to string lookups for ease of
// use when registering handlers etc.
const (
	PRIVMSG = "PRIVMSG"
	NOTICE  = "NOTICE"
	QUIT    = "QUIT"
	JOIN    = "JOIN"
	PART    = "PART"
)

// Pseudo Messages, these messages are not real messages defined by the irc
// protocol but the bot provides them to allow for additional messages to be
// handled such as connect or disconnects which the irc protocol has no protocol
// defined for.
const (
	RAW        = "RAW"
	CONNECT    = "CONNECT"
	DISCONNECT = "DISCONNECT"
)
