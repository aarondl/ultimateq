package data

import (
	"github.com/aarondl/ultimateq/irc"
)

const (
	// nAssumedUsers is the starting number of users allocated for the database.
	nAssumedUsers = 10
)

// Store is used to authenticate user fullhosts against masks. Once
// authenticated they can have three levels of access: global, server and
// channel.
type Store struct {
	Users []*UserAccess
}

// CreateStore initializes a store type.
func CreateStore() *Store {
	return &Store{
		Users: make([]*UserAccess, nAssumedUsers),
	}
}

// FindUser looks up a user based on mask.
func (s *Store) FindUser(mask irc.Mask) *UserAccess {
	for _, user := range s.Users {
		if user.IsMatch(mask) {
			return user
		}
	}

	return nil
}

// AddUser adds a user to the global level.
func (s *Store) AddUser(mask string, level uint8, flags ...string) *Access {
	if user := s.FindUser(irc.Mask(mask)); user != nil {
	} else {
	}
	return &Access{}
}

// AddServerUser adds a user to the server level.
func (s *Store) AddServerUser(mask string, server string,
	level uint8, flags ...string) *Access {

	return CreateAccess(level, flags...)
}

// AddChannelUser adds a user to the channel level.
func (s *Store) AddChannelUser(mask string, server string, channel string,
	level uint8, flags ...string) *Access {

	return CreateAccess(level, flags...)
}

// AuthUser authenticates a user. Channel may be empty if there is no channel
// available.
func (s *Store) AuthUser(fullhost, server, channel string) *Access {
	return nil
}
