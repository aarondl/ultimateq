package irc

import "bytes"

const (
	CTCPDelim     = '\x01'
	CTCPLowQuote  = '\x10'
	CTCPHighQuote = '\x5C'
	CTCPSep       = '\x20'
)

func IsCTCP(msg []byte) bool {
	return CTCPDelim == msg[0] && CTCPDelim == msg[len(msg)-1]
}

func IsCTCPString(msg string) bool {
	return CTCPDelim == msg[0] && CTCPDelim == msg[len(msg)-1]
}

// CTCPunpack unpacks a CTCP message.
func CTCPunpack(msg []byte) (tag []byte, data []byte) {
	msg = msg[1 : len(msg)-1]

	msg = ctcpLowLevelUnescape(msg)
	tag, data = ctcpUnpack(msg)
	tag = ctcpHighLevelUnescape(tag)
	if data != nil {
		data = ctcpHighLevelUnescape(data)
	}
	return tag, data
}

// CTCPpack packs a message into CTCP format.
func CTCPpack(tag, data []byte) []byte {
	if data != nil {
		data = ctcpHighLevelEscape(data)
	}
	tag = ctcpHighLevelEscape(tag)

	ret := ctcpPack(tag, data)
	ret = ctcpLowLevelEscape(ret)

	retDelimited := make([]byte, len(ret)+2)
	retDelimited[0] = CTCPDelim
	retDelimited[len(retDelimited)-1] = CTCPDelim
	copy(retDelimited[1:], ret)
	return retDelimited
}

// CTCPunpack unpacks a CTCP message to strings.
func CTCPunpackString(msg string) (tag, data string) {
	t, d := CTCPunpack([]byte(msg))
	return string(t), string(d)
}

// CTCPpackString packs a message into CTCP format from strings.
func CTCPpackString(tag, data string) string {
	ret := CTCPpack([]byte(tag), []byte(data))
	return string(ret)
}

// ctcpUnpack extracts tagging data from the message data.
// X-CHR  ::= '\000' | '\002' .. '\377'
// X-N-AS ::= '\000'  | '\002' .. '\037' | '\041' .. '\377'
// SPC    ::= '\040'
// X-MSG  ::= | X-N-AS+ | X-N-AS+ SPC X-CHR*
func ctcpUnpack(in []byte) ([]byte, []byte) {
	splits := bytes.SplitN(in, []byte{CTCPSep}, 2)

	if len(splits) == 2 {
		return splits[0], splits[1]
	}
	return splits[0], nil
}

// ctcpPack packs tagging data in with the message data.
func ctcpPack(tag []byte, data []byte) []byte {
	if len(data) == 0 {
		return tag
	}

	ret := make([]byte, len(tag)+len(data)+1)
	copy(ret, tag)
	ret[len(tag)] = CTCPSep
	copy(ret[len(tag)+1:], data)
	return ret
}

// ctcpHighLevelEscape escapes the highest level of CTCP message.
// X-DELIM ::= '\x01'
// X-QUOTE ::= '\134' (0x5C)
// X-DELIM --> X-QUOTE 'a' (0x61)
// X-QUOTE --> X-QUOTE X-QUOTE
func ctcpHighLevelEscape(in []byte) []byte {
	out := bytes.Replace(in, []byte{CTCPHighQuote},
		[]byte{CTCPHighQuote, CTCPHighQuote}, -1)
	out = bytes.Replace(out, []byte{0x01}, []byte{CTCPHighQuote, 0x61}, -1)
	return out
}

// ctcpHighLevelUnescape unescapes the ctcp message to get ready for the wire
func ctcpHighLevelUnescape(in []byte) []byte {
	out := bytes.Replace(in, []byte{CTCPHighQuote, 0x61}, []byte{0x01}, -1)
	out = bytes.Replace(out, []byte{CTCPHighQuote, CTCPHighQuote},
		[]byte{CTCPHighQuote}, -1)
	return out
}

// ctcpLowLevelEscape escapes the low level of CTCP message.
// M-QUOTE = M-QUOTE ::= '\xl0'
// NUL     --> M-QUOTE '0'
// NL      --> M-QUOTE 'n'
// CR      --> M-QUOTE 'r'
// M-QUOTE --> M-QUOTE M-QUOTE
func ctcpLowLevelEscape(in []byte) []byte {
	out := bytes.Replace(in, []byte{CTCPLowQuote},
		[]byte{CTCPLowQuote, CTCPLowQuote}, -1)
	out = bytes.Replace(out, []byte{'\r'}, []byte{CTCPLowQuote, '\r'}, -1)
	out = bytes.Replace(out, []byte{'\n'}, []byte{CTCPLowQuote, '\n'}, -1)
	out = bytes.Replace(out, []byte{0x00}, []byte{CTCPLowQuote, 0x00}, -1)
	return out
}

// ctcpLowLevelUnescape unescapes the ctcp message to get ready for the wire
func ctcpLowLevelUnescape(in []byte) []byte {
	out := bytes.Replace(in, []byte{CTCPLowQuote, 0x00}, []byte{0x00}, -1)
	out = bytes.Replace(out, []byte{CTCPLowQuote, '\n'}, []byte{'\n'}, -1)
	out = bytes.Replace(out, []byte{CTCPLowQuote, '\r'}, []byte{'\r'}, -1)
	out = bytes.Replace(out, []byte{CTCPLowQuote, CTCPLowQuote},
		[]byte{CTCPLowQuote}, -1)
	return out
}
