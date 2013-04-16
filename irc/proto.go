// Deals with irc protocol parsing
package irc

import (
	"strings"
	"regexp"
	"errors"
)

// Package variables
var protoChains map[string]*protoToken = make(map[string]*protoToken)
var protoIdRegex *regexp.Regexp = regexp.MustCompile(`^[\*#:]*\w+$`)

const (
	// An error message for when invalid tokens appear.
	syntaxErrorMessage string = "irc: Invalid token in proto syntax."
	// An error message for when bracket mismatches occur.
	syntaxBracketMismatch string = "irc: Bracket mismatch in proto syntax."
)

// protoToken: represents a single step in a proto chain
type protoToken struct {
	// The named id of the parameter
	id string
	// Whether or not this should consume all following arguments
	final bool
	// Whether or not space delimited arguments should be consumed
	args bool
	// Whether or not the argument was a channel or not
	channel bool
	// The next chunk to parse in the chain
	next *protoToken
	// An optional piece of the chain
	optional *protoToken
}

// parseProtoChain: parses an entire proto file entry.
func parseProtoChain(protocol []string) (*protoToken, error) {
	var first, next, last *protoToken
	var err error

	for i := 0; i < len(protocol); i++ {
		lbs := 0
		for k := 0; protocol[i][k] == '['; k, lbs = k+1, lbs+1 {}

		if lbs > 0 {
			i, next, err = parseOptionalChain(protocol, i, lbs)
		} else {
			next, err = parseProtoTok(protocol[i])
		}

		if err != nil { return nil, err }

		if first == nil { first = next }
		if last != nil { last.next = next }
		last = next
	}
	return first, nil
}

// parseOptionalChain: Helper function for parseProtoChain
func parseOptionalChain(protocol []string, i int, lbs int) (int, *protoToken, error) {
	for j := i; j < len(protocol); j++ {
		for k := len(protocol[j]) - 1; protocol[j][k] == ']' && lbs > 0; {
			k, lbs = k-1, lbs-1
		}

		if lbs == 0 {
			protocol[i] = protocol[i][1:]
			protocol[j] = protocol[j][:len(protocol[j]) - 1]
			protoChain, err := parseProtoChain(protocol[i:j+1])
			if err != nil { return 0, nil, err }
			return j, &protoToken{optional: protoChain}, nil
		}
	}
	return 0, nil, errors.New(syntaxBracketMismatch)
}

// parseProtoTok: parses a single token in a proto chain
func parseProtoTok(tok string) (*protoToken, error) {
	if !protoIdRegex.MatchString(tok) {
		return nil, errors.New(syntaxErrorMessage)
	}

	proto := new (protoToken)
	for hadPrefix := true;; {
		switch {
		case strings.HasPrefix(tok, ":"):
			proto.final = true
		case strings.HasPrefix(tok, "*"):
			proto.args = true
		case strings.HasPrefix(tok, "#"):
			proto.channel = true
		default:
			hadPrefix = false
		}
		if !hadPrefix { break }
		tok = tok[1:]
	}
	proto.id = tok
	return proto, nil
}
