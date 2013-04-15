// Deals with irc protocol parsing
package irc

import "strings"

var protoChains map[string]*protoToken = make(map[string]*protoToken)

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
func parseProtoChain(protocol []string) *protoToken {
	var first, next, last *protoToken

	for i := 0; i < len(protocol); i++ {

		//Determine the amount of brackets on the left of the expression
		lbs := 0
		for k := 0; protocol[i][k] == '['; k, lbs = k+1, lbs+1 {}

		if lbs > 0 {
			i, next = parseOptionalChain(protocol, i, lbs)
		} else {
			next = parseProtoTok(protocol[i])
		}
		if first == nil { first = next }
		if last != nil { last.next = next }
		last = next
	}
	return first
}

// parseOptionalChain: Helper function for parseProtoChain
func parseOptionalChain(protocol []string, i int, lbs int) (int, *protoToken) {
	for j := i; j < len(protocol); j++ {
		for k := len(protocol[j]) - 1; protocol[j][k] == ']' && lbs > 0; {
			k, lbs = k-1, lbs-1
		}

		if lbs == 0 {
			protocol[i] = protocol[i][1:]
			protocol[j] = protocol[j][:len(protocol[j]) - 1]
			return j, &protoToken{optional: parseProtoChain(protocol[i:j + 1])}
		}
	}
	return 0, nil // This shouldn't happen.
}

// parseProtoTok: parses a single token in a proto chain
func parseProtoTok(tok string) *protoToken {
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
	return proto
}
