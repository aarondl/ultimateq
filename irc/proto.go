/*
irc package deals with the irc protocol from parsing, building to validating.
*/
package irc

import (
	"regexp"
	"strings"
)

const (
	// errMsgParseFailure is given when the ircRegex fails to parse the protocol.
	errMsgParseFailure = "irc: Unable to parse received irc protocol"
)

var (
	// ircRegex is used to parse the parts of irc protocol.
	// note that because Go doesn't appear to support proper multiple capture
	// groups, the args have a space in front of them that must be trimmed,
	// and they must then again be split.
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

func (p ParseError) Error() string {
	return p.Msg
}

// IrcMessage contains all the information broken out of an irc message.
type IrcMessage struct {
	// Name of the message. Uppercase constant name or numeric.
	Name string
	// User that sent the message, a fullhost if one was supplied.
	User string
	// The args split by space delimiting.
	Args []string
}

// Parse parses a byte slice and produces an IrcMessage.
// str: An irc protocol message, split by newlines, and newlines should not be
// present.
func Parse(bytes []byte) (*IrcMessage, error) {
	msg := &IrcMessage{}
	str := string(bytes)

	parts := ircRegex.FindStringSubmatch(str)
	if parts == nil {
		return nil, ParseError{Msg: errMsgParseFailure, Irc: str}
	}

	msg.User = parts[1]
	msg.Name = parts[2]
	msg.Args = strings.Split(strings.TrimLeft(parts[3], " "), " ")
	if parts[4] != "" {
		msg.Args = append(msg.Args, parts[4])
	}

	return msg, nil
}
