/*
Package irc defines types to be used by most other packages in
the ultimateq systee. It is small and comprised mostly of helper like types
and constants.
*/
package irc

import (
	"bytes"
	"strings"
	"time"
)

// Event contains all the information about an irc event.
type Event struct {
	// Name of the event. Uppercase constant name or numeric.
	Name string
	// Sender is the server or user that sent the event, normally a fullhost.
	Sender string
	// Args split by space delimiting.
	Args []string
	// Times is the time this event was received.
	Time time.Time
	// NetworkID is the ID of the network that sent this event.
	NetworkID string
	// NetworkInfo is the networks information.
	NetworkInfo *NetworkInfo
}

// NewEvent constructs a event object that has a timestamp.
func NewEvent(netID string, ni *NetworkInfo, name, sender string,
	args ...string) *Event {

	var setArgs []string
	if len(args) > 0 {
		setArgs = make([]string, len(args))
		copy(setArgs, args)
	}
	return &Event{name, sender, setArgs, time.Now().UTC(), netID, ni}
}

// Nick returns the nick of the sender. Will be empty string if it was
// not able to parse the sender.
func (e *Event) Nick() string {
	return Nick(e.Sender)
}

// Username returns the username of the sender. Will be empty string if it was
// not able to parse the sender.
func (e *Event) Username() string {
	return Username(e.Sender)
}

// Hostname returns the host of the sender. Will be empty string if it was
// not able to parse the sender.
func (e *Event) Hostname() string {
	return Hostname(e.Sender)
}

// SplitHost splits the sender into it's fragments: nick, user, and hostname.
// If the format is not acceptable empty string is returned for everything.
func (e *Event) SplitHost() (nick, user, hostname string) {
	return Split(e.Sender)
}

// SplitArgs splits string arguments. A convenience method to avoid having to
// call splits and import strings.
func (e *Event) SplitArgs(index int) []string {
	return strings.Split(e.Args[index], ",")
}

// Target retrieves the channel or user this event was sent to. Before using
// this method it would be prudent to check that the Event.Name is a message
// that supports a Target argument.
func (e *Event) Target() string {
	return e.Args[0]
}

// IsTargetChan uses the underlying NetworkInfo to decide if this is a channel
// or not. If there is no NetworkInfo it will panic.
func (e *Event) IsTargetChan() bool {
	return e.NetworkInfo.IsChannel(e.Args[0])
}

// Message retrieves the message sent to the user or channel. Before using
// this method it would be prudent to check that the Event.Name is a message
// that supports a Message argument.
func (e *Event) Message() string {
	return e.Args[1]
}

// String turns this back into an IRC style message.
func (e *Event) String() string {
	b := &bytes.Buffer{}
	if len(e.Sender) > 0 {
		b.WriteByte(':')
		b.WriteString(e.Sender)
		b.WriteByte(' ')
	}
	b.WriteString(e.Name)

	lastArg := len(e.Args) - 1
	for i, arg := range e.Args {
		b.WriteByte(' ')
		if lastArg == i && strings.ContainsRune(arg, ' ') {
			b.WriteByte(':')
		}
		b.WriteString(arg)
	}

	return b.String()
}

// IsCTCP checks if this event is a CTCP event. This means it's delimited
// by the CTCPDelim as well as being PRIVMSG or NOTICE only.
func (e *Event) IsCTCP() bool {
	return (e.Name == PRIVMSG || e.Name == NOTICE) && len(e.Args) >= 2 &&
		IsCTCPString(e.Args[1])
}

// UnpackCTCP can be called to retrieve a tag and data from a CTCP event.
func (e *Event) UnpackCTCP() (tag, data string) {
	return CTCPunpackString(e.Args[1])
}
