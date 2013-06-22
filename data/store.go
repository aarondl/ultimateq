package data

import (
	"github.com/aarondl/ultimateq/irc"
	"github.com/cznic/kv"
)

const (
	// nAssumedUsers is the starting number of users allocated for the database.
	nAssumedUsers = 10
)

// Store is used to store UserAccess objects, and cache their lookup.
type Store struct {
	db    *kv.DB
	cache map[string]*UserAccess
}

// CreateStore initializes a store type.
func CreateStore(dbCreate func() (*kv.DB, error)) (*Store, error) {
	db, err := dbCreate()
	if err != nil {
		return nil, err
	}

	s := &Store{
		db:    db,
		cache: make(map[string]*UserAccess),
	}

	return s, nil
}

// AddUser adds a user to the database.
func (s *Store) AddUser(ua *UserAccess) error {
	return nil
}

// RemoveUser removes a user from the database, returning the removed user.
func (s *Store) RemoveUser(username string) *UserAccess {
	return nil
}

// AuthUser authenticates a user and caches the lookup.
func (s *Store) AuthUser(username, password string) (user *UserAccess) {
	_ = irc.CAPS_AWAYLEN
	/*if cached, ok := s.cache[mask]; ok {
		user = cached
	} else if found := s.FindUser(mask); found != nil {
		user = found
		s.cache[mask] = found
	}*/
	return
}

// FindUser looks up a user based username. It caches the result if found.
func (s *Store) FindUser(username string) (user *UserAccess) {
	if cached, ok := s.cache[username]; ok {
		user = cached
	} else if found := s.FindUser(username); found != nil {
		user = found
		s.cache[username] = found
	}

	return nil
}
