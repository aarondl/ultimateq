/*
data package is used to turn irc.IrcMessages into a stateful database.
*/
package data

import (
	"github.com/aarondl/ultimateq/irc"
)

type Self struct {
	*User
}

// Store is the main data container. It represents the state on a server
// including all channels, users, and self.
type Store struct {
	Self Self

	channels map[string]Channel
	users    map[string]User

	channelUsers map[string]ChannelUser
	userChannels map[string]ChannelUser

	kinds  *ModeKinds
	umodes *UserModes
}

// CreateStore creates a store from an irc protocaps instance.
func CreateStore(caps *irc.ProtoCaps) (*Store, error) {
	kinds, err := CreateModeKindsCSV(caps.Chanmodes())
	if err != nil {
		return nil, err
	}
	modes, err := CreateUserModes(caps.Prefix())
	if err != nil {
		return nil, err
	}

	return &Store{
		channels:     make(map[string]Channel),
		users:        make(map[string]User),
		channelUsers: make(map[string]ChannelUser),
		userChannels: make(map[string]ChannelUser),

		kinds:  kinds,
		umodes: modes,
	}, nil
}
