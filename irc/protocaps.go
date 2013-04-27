package irc

// Used to record the server settings, aids in parsing.
type ProtoCaps struct {
	Chantypes string
	Prefix    string
	Statusmsg string
	Chanmodes string
}
