/*
data package is used to turn irc.IrcMessages into a stateful database.
*/
package data

// Store is the main data container. It represents the state on a server
// including all channels, users, and self.
type Store struct {
	Channels map[string]Channel
	Users    map[string]User
}
