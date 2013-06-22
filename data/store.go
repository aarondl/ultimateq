package data

import (
	"errors"
	"github.com/aarondl/ultimateq/irc"
	"github.com/cznic/kv"
)

var (
	nMaxCache = 1000

	ErrUserNotFound    = errors.New("data: User not found.")
	ErrUserBadPassword = errors.New("data: User password does not match.")
	ErrUserBadHost     = errors.New("data: Host does not match stored hosts.")
)

// Store is used to store UserAccess objects, and cache their lookup.
type Store struct {
	db     *kv.DB
	cache  map[string]*UserAccess
	authed map[string]*UserAccess
}

// CreateStore initializes a store type.
func CreateStore(dbCreate func() (*kv.DB, error)) (*Store, error) {
	db, err := dbCreate()
	if err != nil {
		return nil, err
	}

	s := &Store{
		db:     db,
		cache:  make(map[string]*UserAccess),
		authed: make(map[string]*UserAccess),
	}

	return s, nil
}

// Closes the underlying database.
func (s *Store) Close() error {
	return s.db.Close()
}

// AddUser adds a user to the database.
func (s *Store) AddUser(ua *UserAccess) error {
	var err error
	var serialized []byte

	serialized, err = ua.Serialize()
	if err != nil {
		return err
	}

	s.db.Set([]byte(ua.Username), serialized)
	if err != nil {
		return err
	}

	s.checkCacheLimits()
	s.cache[ua.Username] = ua
	return nil
}

// RemoveUser removes a user from the database, returning the removed user.
func (s *Store) RemoveUser(username string) error {
	delete(s.cache, username)
	return s.db.Delete([]byte(username))
}

// AuthUser authenticates a user. UserAccess will be not nil iff the user
// is found and authenticates successfully.
func (s *Store) AuthUser(
	server, host, username, password string) (*UserAccess, error) {

	var user *UserAccess
	var ok bool
	var err error

	if user, ok = s.authed[server+host]; ok {
		return user, nil
	}

	user, err = s.findUser(username)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, ErrUserNotFound
	}

	if len(user.Masks) > 0 && !user.IsMatch(irc.Mask(host)) {
		return nil, ErrUserBadHost
	}

	if !user.VerifyPassword(password) {
		return nil, ErrUserBadPassword
	}

	s.authed[server+host] = user
	return user, nil
}

// Logout deletes an authenticated host.
func (s *Store) Logout(server, host string) {
	delete(s.authed, server+host)
}

// findUser looks up a user based on username. It caches the result if found.
func (s *Store) findUser(username string) (user *UserAccess, err error) {
	if cached, ok := s.cache[username]; ok {
		user = cached
		return
	}

	user, err = s.fetchUser(username)
	if err != nil {
		return
	}

	s.checkCacheLimits()
	s.cache[username] = user
	return
}

// fetchUser gets a user from the database based on username.
func (s *Store) fetchUser(username string) (user *UserAccess, err error) {
	var serialized []byte
	serialized, err = s.db.Get(nil, []byte(username))
	if err != nil || serialized == nil {
		return
	}

	user, err = deserialize(serialized)
	return
}

// checkCacheLimits verifies if adding one to the size of the cache will
// cross it's boundaries, if so, it dumps the cache.
func (s *Store) checkCacheLimits() {
	if len(s.cache)+1 > nMaxCache {
		s.cache = make(map[string]*UserAccess)
	}
}
