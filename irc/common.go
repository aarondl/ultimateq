/*
irc package defines types and classes to be used by most other packages in
the ultimateq system. It is small and comprised mostly of helper like types
and constants.
*/
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

// Sender is the sender of an event, and should allow replies on a writing
// interface as well as a way to identify itself.
type Sender interface {
	// Writes a string to an endpoint that makes sense for the given event.
	Writeln(string) error
	// Retrieves a key to retrieve where this event was generated from.
	GetKey() string
}

// IrcMessage contains all the information broken out of an irc message.
type IrcMessage struct {
	// Name of the message. Uppercase constant name or numeric.
	Name string
	// The server or user that sent the message, a fullhost if one was supplied.
	Sender string
	// The args split by space delimiting.
	Args []string
}

// Split splits string arguments. A convenience method to avoid having to call
// splits and import strings.
func (m *IrcMessage) Split(index int) []string {
	return strings.Split(m.Args[index], ",")
}

// Message type provides a view around an IrcMessage to access it's parts in a
// more convenient way.
type Message struct {
	// Raw is the underlying irc message.
	Raw *IrcMessage
}

// Target retrieves the channel or user this message was sent to.
func (p *Message) Target() string {
	return p.Raw.Args[0]
}

// Message retrieves the message sent to the user or channel.
func (p *Message) Message() string {
	return p.Raw.Args[1]
}
