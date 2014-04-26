/*
Package parse has functions to parse the irc protocol into irc.IrcMessages.
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
		`^(?::(\S+) )?([A-Z0-9]+)((?: (?:[^:\s][^\s]*))*)(?: :(.*))?\s*$`)
)

// ParseError is generated when something does not match the regex, irc.Parse
// will return one of these containing the invalid seeming irc protocol string.
type ParseError struct {
	// The invalid irc encountered.
	Irc string
}

// Error satisfies the Error interface for ParseError.
func (p ParseError) Error() string {
	return errMsgParseFailure
}

// Parse produces an IrcMessage from a byte slice. The string is an irc
// protocol message, split by \r\n, and \r\n should not be
// present at the end of the string.
func Parse(str []byte) (*irc.Event, error) {
	parts := ircRegex.FindSubmatch(str)
	if parts == nil {
		return nil, ParseError{Irc: string(str)}
	}

	sender := string(parts[1])
	name := string(parts[2])
	var args []string
	if len(parts[3]) != 0 {
		args = strings.Fields(string(parts[3]))
	}

	if len(parts[4]) != 0 {
		if args != nil {
			args = append(args, string(parts[4]))
		} else {
			args = []string{string(parts[4])}
		}
	}

	return irc.NewEvent("", nil, name, sender, args...), nil
}
