package irc

import "regexp"

// Used to record the server settings, aids in parsing irc protocol.
type ProtoCaps struct {
	// The user prefix and symbol mapping (ov)@+
	Prefix string
	// The status message, whatever this means @+
	Statusmsg string
	// The channel modes allowed to be set by the server.
	Chanmodes string
	// The channel types supported by the server, usually &#~
	chantypes string
	// A regular expression to search for channels
	chantypesRegex *regexp.Regexp
}

// SetChanTypes sets the channel types for a ProtoCaps object.
// Additionally it creates a chantypesRegex that can be used to pull channels
// with proper prefixes for the connected server out of text.
func (caps *ProtoCaps) SetChantypes(types string) error {
	caps.chantypes = types
	safetypes := ""
	for _, c := range types {
		safetypes += string(`\`) + string(c)
	}
	var err error
	caps.chantypesRegex, err = regexp.Compile(`[` + safetypes + `][^\s,]+`)
	return err
}
