/*
Package parse has functions to parse the irc protocol into irc.IrcMessages.
*/
package parse

import (
	"github.com/aarondl/ultimateq/irc"
	"regexp"
	"strings"
)

const (
	// errMsgParseFailure is given when the ircRegex fails to parse the protocol
	errMsgParseFailure = "parse: Unable to parse received irc protocol"
)

var (
	// ircRegex is used to parse the parts of irc protocol.
	ircRegex = regexp.MustCompile(
		`^(?::(\S+) )?([A-Z0-9]+)((?: (?:[^:\s][^\s]*))*)(?: :(.*))?\s*$`)
)

// ParseError is generated when something does not match the regex, irc.Parse
// will return one of these containing the invalid seeming irc protocol string.
type ParseError struct {
	// The message
	Msg string
	// The invalid irc encountered.
	Irc string
}

// Error satisfies the Error interface for ParseError.
func (p ParseError) Error() string {
	return p.Msg
}

// Parse produces an IrcMessage from a byte slice. The string is an irc
// protocol message, split by \r\n, and \r\n should not be
// present at the end of the string.
func Parse(str []byte) (*irc.Message, error) {
	parts := ircRegex.FindSubmatch(str)
	if parts == nil {
		return nil, ParseError{Msg: errMsgParseFailure, Irc: string(str)}
	}

	msg := &irc.Message{}
	msg.Sender = string(parts[1])
	msg.Name = string(parts[2])
	if len(parts[3]) != 0 {
		msg.Args = strings.Fields(string(parts[3]))
	}

	if len(parts[4]) != 0 {
		if msg.Args != nil {
			msg.Args = append(msg.Args, string(parts[4]))
		} else {
			msg.Args = []string{string(parts[4])}
		}
	}

	return msg, nil
}
