package proto

import (
	"errors"
	"regexp"
	"strings"
)

var (
	// syntaxErrorMessage: When invalid tokens occur in identifiers.
	errIllegalIdentifiers = errors.New("irc: Invalid token in identifier")
	// errBracketMismatch: When bracket mismatches occur.
	errBracketMismatch = errors.New("irc: Bracket mismatch")
	// errArgsAfterFinal: When the final syntax is given, and then more arguments.
	errArgsAfterFinal = errors.New("irc: Arguments given after final marker")
	// errRequiredAfterOptionalArg: A required argument comes after an optional.
	errRequiredAfterOptionalArg = errors.New(
		"irc: Required argument after optional argument")
	// errFinalCannotBeChannel: Final arguments cannot be channels
	errFinalCantBeChannel = errors.New("irc: Final arguments cannot be channels")
	// errFinalCannotBeArgs: Final arguments cannot be args
	errFinalCantBeArgs = errors.New("irc: Final arguments cannot be args")
	// fragmentIdRegex: Regex to validate identifier tokens.
	fragmentIdRegex *regexp.Regexp = regexp.MustCompile(`^[\*#:]*\w+$`)
)

// fragment: represents a single step in a fragment chain
type fragment struct {
	// The named id of the parameter
	id string
	// Whether or not this should consume all following arguments
	final bool
	// Whether or not space delimited arguments should be consumed
	args bool
	// Whether or not the argument was a channel or not
	channel bool
	// The next chunk to parse in the chain
	next *fragment
	// An optional piece of the chain
	optional *fragment
}

// createProtoChain: creates a parse tree from the protocol syntax
func createFragmentChain(toks []string) (*fragment, error) {
	var first, next, last *fragment
	var err error

	for i := 0; i < len(toks); i++ {
		lbs := 0
		for k := 0; toks[i][k] == '['; k, lbs = k+1, lbs+1 {
			//noop
		}

		if lbs > 0 {
			i, next, err = parseOptionalChain(toks, i, lbs)
		} else {
			next, err = parseFragment(toks[i])
		}

		if err != nil {
			return nil, err
		}

		if first == nil {
			first = next
		}
		if last != nil {
			last.next = next
		}
		if next.final && i+1 != len(toks) {
			return nil, errArgsAfterFinal
		}
		if last != nil && last.optional != nil && next.optional == nil {
			return nil, errRequiredAfterOptionalArg
		}
		last = next
	}
	return first, nil
}

// parseOptionalChain: Helper function for createFragmentChain
func parseOptionalChain(toks []string, i int, lbs int) (int, *fragment, error) {
	for j := i; j < len(toks); j++ {
		for k := len(toks[j]) - 1; toks[j][k] == ']' && lbs > 0; {
			k, lbs = k-1, lbs-1
		}

		if lbs == 0 {
			toks[i] = toks[i][1:]
			toks[j] = toks[j][:len(toks[j])-1]
			protoChain, err := createFragmentChain(toks[i : j+1])
			if err != nil {
				return 0, nil, err
			}
			return j, &fragment{optional: protoChain}, nil
		}
	}
	return 0, nil, errBracketMismatch
}

// parseFragment: parses a single token in the protocol descriptor chain
func parseFragment(tok string) (*fragment, error) {
	if !fragmentIdRegex.MatchString(tok) {
		return nil, errIllegalIdentifiers
	}

	frag := new(fragment)
	for hadPrefix := true; ; {
		switch {
		case strings.HasPrefix(tok, ":"):
			frag.final = true
		case strings.HasPrefix(tok, "*"):
			frag.args = true
		case strings.HasPrefix(tok, "#"):
			frag.channel = true
		default:
			hadPrefix = false
		}
		if !hadPrefix {
			break
		}
		tok = tok[1:]
	}
	if frag.args && frag.final {
		return nil, errFinalCantBeArgs
	}
	if frag.channel && frag.final {
		return nil, errFinalCantBeChannel
	}
	frag.id = tok
	return frag, nil
}
