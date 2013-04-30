/*
parse package deals with parsing the irc protocol
*/
package parse

import (
	"regexp"
	"strings"
	"github.com/aarondl/ultimateq/irc"
)

const (
	// errMsgParseFailure is given when the ircRegex fails to parse the protocol
	errMsgParseFailure = "parse: Unable to parse received irc protocol"
)

var (
	// ircRegex is used to parse the parts of irc protocol.
	ircRegex = regexp.MustCompile(
		`^(?::(\S+) )?([A-Z0-9]+)((?: (?:[^:\s]+))*)(?: :(.*))?$`)
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

// Parse produces an IrcMessage from a byte slice. The byte slice is an irc
// protocol message, split by newlines, and newlines should not be
// present.
func Parse(bytes []byte) (*irc.IrcMessage, error) {
	msg := &irc.IrcMessage{}
	str := string(bytes)

	parts := ircRegex.FindStringSubmatch(str)
	if parts == nil {
		return nil, ParseError{Msg: errMsgParseFailure, Irc: str}
	}

	msg.Sender = parts[1]
	msg.Name = parts[2]
	msg.Args = strings.Split(strings.TrimLeft(parts[3], " "), " ")
	if parts[4] != "" {
		msg.Args = append(msg.Args, parts[4])
	}

	return msg, nil
}
