// irc package defines types and classes to be used by most other packages in
// the ultimateq system. It is small and comprised mostly of helper like types
// and constants.
package irc

import "strings"

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

// IrcMessage contains all the information broken out of an irc message.
type IrcMessage struct {
	// Name of the message. Uppercase constant name or numeric.
	Name string
	// Sender that sent the message, a fullhost if one was supplied.
	Sender string
	// The args split by space delimiting.
	Args []string
}

// Split splits string arguments. A convenience method to avoid having to call
// splits and import strings.
func (m *IrcMessage) Split(index int) []string {
	return strings.Split(m.Args[index], ",")
}

