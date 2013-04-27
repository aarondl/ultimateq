// Deals with irc protocol parsing
package proto

import (
	"errors"
	"regexp"
	"strings"
)

var (
	// errHandlerNotRegistered: Handler not registered.
	errHandlerNotRegistered = errors.New("irc: Handler not registered")
	// errHandlerAlreadyRegistered: Handler previously registered.
	errHandlerAlreadyRegistered = errors.New("irc: Handler already registered")
	// errNoProtocolGiven: Happens when an empty string is given to parse
	errNoProtocolGiven = errors.New("irc: No protocol given")
	// errArgsAfterFinalNoColon: Colon not given, but arguments still followed.
	errArgsAfterFinalNoColon = errors.New(
		"irc: Colon not given, but arguments still followed")
	// errExpectedMoreArguments: Protocol line ends abruptly
	errExpectedMoreArguments = errors.New("irc: Expected more arguments")

	// evNameRegex: Used to check identifiers of messages for validity.
	evNameRegex = regexp.MustCompile(`^[A-Za-z0-9]+$`)
)

// ParseResult: The result of a pass through the parsing tree.
type ParseResult struct {
	// The name of the event
	Name string
	// The fullhost of the sender.
	Sender string
	// args: arguments in plain form
	Args map[string]string
	// nargs: split arguments
	Argv map[string][]string
	// channels: any channels found
	Channels map[string][]string
}

// IrcParser: Handles parsing irc and returning a result on how to handle.
type IrcParser struct {
	handlers map[string]*fragment
}

// CreateIrcParser: Creates an irc parser struct.
func CreateIrcParser() *IrcParser {
	return &IrcParser{handlers: make(map[string]*fragment)}
}

// createParseResult: Creates a parse result struct.
func createParseResult() *ParseResult {
	return &ParseResult{
		Args:     make(map[string]string),
		Argv:     make(map[string][]string),
		Channels: make(map[string][]string),
	}
}

// addHandler: Adds a handler to the IrcParser.
func (p *IrcParser) AddIrcHandler(handler string, tree *fragment) error {
	if !evNameRegex.MatchString(handler) {
		return errIllegalIdentifiers
	}
	handler = strings.ToUpper(handler)
	_, has := p.handlers[handler]
	if has {
		return errHandlerAlreadyRegistered
	}

	p.handlers[handler] = tree
	return nil
}

// removeHandler: Deletes a handler from the IrcParser
func (p *IrcParser) RemoveIrcHandler(handler string) error {
	handler = strings.ToUpper(handler)
	_, has := p.handlers[handler]
	if !has {
		return errHandlerNotRegistered
	}

	delete(p.handlers, handler)
	return nil
}

// parse: Parses an irc protocol string
func (p *IrcParser) Parse(proto string, caps *ProtoCaps) (*ParseResult, error) {
	if len(proto) == 0 {
		return nil, errNoProtocolGiven
	}

	splits := strings.Split(proto, " ")
	result := createParseResult()
	var nameIndex = 0
	if strings.HasPrefix(splits[0], ":") {
		result.Sender = splits[0][1:]
		nameIndex++
	}
	result.Name = strings.ToUpper(splits[nameIndex])
	chain, ok := p.handlers[result.Name]
	if !ok {
		return nil, errHandlerNotRegistered
	}
	err := walkProto(chain, splits[nameIndex+1:], result, nil)
	return result, err
}

// walkProto: Walks the protocol tokens and parse tree to fill a ParseResult
func walkProto(f *fragment, p []string, pr *ParseResult, c *ProtoCaps) error {
	frag, i, err := walkHelper(f, 0, p, pr, c)
	if err != nil {
		return err
	}
	if frag != nil && frag.optional == nil && i >= len(p) {
		return errExpectedMoreArguments
	}
	return nil
}

// walkHelper: Recursive function for walkProto
func walkHelper(
	chain *fragment, i int, proto []string,
	result *ParseResult, caps *ProtoCaps) (*fragment, int, error) {

	var err error = nil
	for chain != nil && i < len(proto) {
		if chain.optional != nil {
			_, i, err = walkHelper(chain.optional, i, proto, result, caps)
			if err != nil {
				break
			}
			chain = chain.next
			continue
		}

		if chain.final {
			var value string
			value, err = handleFinalChain(i, proto)
			if err != nil {
				break
			}
			result.Args[chain.id] = value
			i = len(proto)
		} else {
			if chain.channel && caps != nil && len(caps.Chantypes) > 0 {
				if valid := validateChannels(proto[i], caps); len(valid) > 0 {
					result.Channels[chain.id] = valid
				}
			} else if chain.args {
				result.Argv[chain.id] = strings.Split(proto[i], ",")
			}
			result.Args[chain.id] = proto[i]
		}

		chain = chain.next
		i++
	}
	return chain, i, err
}

func handleFinalChain(index int, proto []string) (string, error) {
	var value string = ""
	if index > len(proto) {
		return "", nil
	}

	if strings.HasPrefix(proto[index], ":") {
		value = proto[index][1:] + " " + strings.Join(proto[index+1:], " ")
	} else if index+1 != len(proto) {
		return "", errArgsAfterFinalNoColon
	} else {
		value = proto[index]
	}
	return value, nil
}

func validateChannels(chans string, caps *ProtoCaps) []string {
	if caps == nil || len(caps.Chantypes) == 0 || len(chans) == 0 {
		return nil
	}

	cs := strings.Split(chans, ",")
	valid := make([]string, 0, len(cs))
	for _, v := range cs {
		for _, c := range caps.Chantypes {
			if c == rune(v[0]) {
				valid = append(valid, v)
			}
		}
	}
	return valid
}
