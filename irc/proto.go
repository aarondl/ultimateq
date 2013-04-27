/*
irc package deals with the irc protocol from parsing, building to validating.
*/
package irc

import (
	"regexp"
	"strings"
)

const (
	// nStringsAssumed is the number of channels assumed to be in each irc message
	// if this number is too small, there could be memory thrashing due to append
	nChannelsAssumed = 1
	// errMsgParseFailure is given when the ircRegex fails to parse the protocol.
	errMsgParseFailure = "irc: Unable to parse received irc protocol"
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

// IrcMessage contains all the information broken out of an irc message.
type IrcMessage struct {
	// Name of the message. Uppercase constant name or numeric.
	Name string
	// User that sent the message, a fullhost if one was supplied.
	User string
	// The args split by space delimiting.
	Args []string
	// The cached list of channels
	channels []string
}

// Split splits string arguments. A convenience method to avoid having to call
// splits and import strings.
func (m *IrcMessage) Split(index int) []string {
	return strings.Split(m.Args[index], ",")
}

// Channels retrieves all the channels in the object using the channel regex
// created by ProtoCaps when SetChanTypes is called. It ensures only the first
// batch of channels is returned, and it also caches per IrcMessage object.
func (m *IrcMessage) Channels(caps *ProtoCaps) []string {
	if m.channels != nil {
		return m.channels
	}
	m.channels = make([]string, 0, nChannelsAssumed)

	for _, arg := range m.Args {
		if strings.Contains(arg, " ") {
			continue
		}

		for _, channel := range caps.chantypesRegex.FindAllString(arg, -1) {
			m.channels = append(m.channels, channel)
		}
		if len(m.channels) > 0 {
			break
		}
	}
	return m.channels
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
