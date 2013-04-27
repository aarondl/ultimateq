package irc

// Used to record the server settings, aids in parsing irc protocol.
type ProtoCaps struct {
	// The channel types supported by the server, usually &#~
	Chantypes string
	// The user prefix and symbol mapping (ov)@+
	Prefix string
	// The status message, whatever this means @+
	Statusmsg string
	// The channel modes allowed to be set by the server.
	Chanmodes string
}
