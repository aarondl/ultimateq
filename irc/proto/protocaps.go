package proto

// Used to record the server settings, aids in parsing.
type ProtoCaps struct {
	Chantypes string
	Prefix    string
	Statusmsg string
	Chanmodes string
}

// Stores the capabilities of the server.
var protoCaps *ProtoCaps = nil

// Sets the server capabilites.
func SetCaps(caps *ProtoCaps) {
	protoCaps = caps
}

// Gets the server capabilities.
func GetCaps() *ProtoCaps {
	return protoCaps
}
