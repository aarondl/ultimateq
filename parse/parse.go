/*
parse package deals with parsing the irc protocol
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
		`^(?::(\S+) )?([A-Z0-9]+)((?: (?:[^:\s][^\s]*))*)(?: :(.*))?$`)
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

// Parse produces an IrcMessage from a string. The string is an irc
// protocol message, split by \r\n, and \r\n should not be
// present at the end of the string.
func Parse(str string) (*irc.IrcMessage, error) {
	parts := ircRegex.FindStringSubmatch(str)
	if parts == nil {
		return nil, ParseError{Msg: errMsgParseFailure, Irc: str}
	}

	msg := &irc.IrcMessage{}
	msg.Sender = parts[1]
	msg.Name = parts[2]
	if parts[3] != "" {
		msg.Args = strings.Split(strings.TrimLeft(parts[3], " "), " ")
	}

	if parts[4] != "" {
		if msg.Args != nil {
			msg.Args = append(msg.Args, parts[4])
		} else {
			msg.Args = []string{parts[4]}
		}
	}

	return msg, nil
}
