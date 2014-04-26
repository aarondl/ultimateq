package data

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/cznic/kv"
)

// These errors are in the AuthError.FailureType field.
const (
	AuthErrBadPassword = iota + 1
	AuthErrHostNotFound
	AuthErrUserNotFound
)

// These error messages are put into the AuthError's string field and will
// appear when receiving an auth error.
const (
	// errFmtUserNotFound occurs when the database lookup fails.
	errFmtUserNotFound = "User [%v] not found."
	// errFmtBadPassword occurs when the user provides a password that does
	// not match.
	errFmtBadPassword = "Password does not match for user [%v]."
	// errFmtBadHost occurs when a user has hosts defined, and the user's
	// current host is not a match.
	errFmtBadHost = "Host [%v] does not match stored hosts for user [%v]."
)

// AuthError is returned when a user failure occurs (bad password etc.) during
// authentication.
type AuthError struct {
	str         string
	fmtArgs     []interface{}
	FailureType int
}

// Error builds the error string for an AuthError.
func (a AuthError) Error() string {
	if len(a.fmtArgs) > 0 {
		return fmt.Sprintf(a.str, a.fmtArgs...)
	}
	return a.str
}

var (
	// nMaxCache is the number of users to store in the cache.
	nMaxCache = 1000
	// isIninitialized is a key into the database that checks if the first
	// user has been set.
	isInitialized = []byte{0}
)

// DbProvider is a function that provides an internal database.
type DbProvider func() (*kv.DB, error)

// Store is used to store UserAccess objects, and cache their lookup.
type Store struct {
	db           *kv.DB
	cache        map[string]*UserAccess
	protectCache sync.Mutex
	authed       map[string]*UserAccess
	checkedFirst bool
}

// CreateStore initializes a store type.
func CreateStore(prov DbProvider) (*Store, error) {
	db, err := prov()
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

// Close closes the underlying database.
func (s *Store) Close() error {
	return s.db.Close()
}

// GlobalUsers gets users with global access
func (s *Store) GlobalUsers() (list []UserAccess, err error) {
	var val []byte
	var e *kv.Enumerator
	var ua *UserAccess
	var a *Access
	var stop error

	e, err = s.db.SeekFirst()
	if err == io.EOF {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	for ; stop == nil; _, val, stop = e.Next() {
		ua, err = deserialize(val)
		if err != nil {
			err = nil
			continue
		}

		if a = ua.GetGlobal(); a != nil && !a.IsZero() {
			list = append(list, *ua)
		}
	}

	return
}

// ServerUsers gets users with Server access
func (s *Store) ServerUsers(server string) (list []UserAccess, err error) {
	var val []byte
	var e *kv.Enumerator
	var ua *UserAccess
	var a *Access
	var stop error

	e, err = s.db.SeekFirst()
	if err == io.EOF {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	for ; stop == nil; _, val, stop = e.Next() {
		ua, err = deserialize(val)
		if err != nil {
			err = nil
			continue
		}

		if a = ua.GetServer(server); a != nil && !a.IsZero() {
			list = append(list, *ua)
		}
	}

	return
}

// ChanUsers gets users with access to a channel
func (s *Store) ChanUsers(server, channel string) (list []UserAccess, err error) {
	var val []byte
	var e *kv.Enumerator
	var ua *UserAccess
	var a *Access
	var stop error

	e, err = s.db.SeekFirst()
	if err == io.EOF {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	for ; stop == nil; _, val, stop = e.Next() {
		ua, err = deserialize(val)
		if err != nil {
			err = nil
			continue
		}

		if a = ua.GetChannel(server, channel); a != nil && !a.IsZero() {
			list = append(list, *ua)
		}
	}

	return
}

// AddUser adds a user to the database.
func (s *Store) AddUser(ua *UserAccess) error {
	var err error
	var serialized []byte

	serialized, err = ua.serialize()
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

// RemoveUser removes a user from the database, returns true if successful.
func (s *Store) RemoveUser(username string) (removed bool, err error) {
	username = strings.ToLower(username)
	var exists *UserAccess
	exists, err = s.FindUser(username)
	if err != nil || exists == nil {
		return
	}

	delete(s.cache, username)
	err = s.db.Delete([]byte(username))
	if err != nil {
		return
	}
	removed = true
	return
}

// AuthUser authenticates a user. UserAccess will be not nil iff the user
// is found and authenticates successfully.
func (s *Store) AuthUser(
	server, host, username, password string) (*UserAccess, error) {

	username = strings.ToLower(username)
	var user *UserAccess
	var ok bool
	var err error

	if user, ok = s.authed[server+host]; ok {
		return user, nil
	}

	user, err = s.FindUser(username)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, AuthError{
			errFmtUserNotFound,
			[]interface{}{username},
			AuthErrUserNotFound,
		}
	}

	if !user.ValidateMask(host) {
		return nil, AuthError{
			errFmtBadHost,
			[]interface{}{host, username},
			AuthErrHostNotFound,
		}
	}

	if !user.VerifyPassword(password) {
		return nil, AuthError{
			errFmtBadPassword,
			[]interface{}{username},
			AuthErrBadPassword,
		}
	}

	s.authed[server+host] = user
	return user, nil
}

// GetAuthedUser looks up a user that was authenticated previously.
func (s *Store) GetAuthedUser(server, host string) *UserAccess {
	return s.authed[server+host]
}

// Logout logs an authenticated host out.
func (s *Store) Logout(server, host string) {
	delete(s.authed, server+host)
}

// LogoutByUsername logs an authenticated username out.
func (s *Store) LogoutByUsername(username string) {
	username = strings.ToLower(username)
	hosts := make([]string, 0, 1)
	for host, user := range s.authed {
		if user.Username == username {
			hosts = append(hosts, host)
		}
	}

	for i := range hosts {
		delete(s.authed, hosts[i])
	}
}

// FindUser looks up a user based on username. It caches the result if found.
func (s *Store) FindUser(username string) (user *UserAccess, err error) {
	username = strings.ToLower(username)
	// We're writing to cache in a method that should be considered safe by
	// read-only locked friends, so we have to protect the cache.
	s.protectCache.Lock()
	defer s.protectCache.Unlock()

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
	username = strings.ToLower(username)
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

// IsFirst checks to see if the user is the first one in. Returns true if
// so, false if not. Note that this also sets the value immediately, so all
// subsequent calls to IsFirst will return false.
func (s *Store) IsFirst() (isFirst bool, err error) {
	if s.checkedFirst {
		return
	}

	_, isFirst, err = s.db.Put(nil, isInitialized,
		func(key, old []byte) (upd []byte, write bool, err error) {
			if old == nil {
				upd = isInitialized
				write = true
			}
			return
		},
	)

	s.checkedFirst = true
	return
}

// MakeFileStoreProvider is the default way to create a store by using the
// filename and trying to open it.
func MakeFileStoreProvider(filename string) DbProvider {
	return func() (db *kv.DB, err error) {
		opts := &kv.Options{}

		_, err = os.Stat(filename)

		if os.IsNotExist(err) {
			db, err = kv.Create(filename, opts)
		} else {
			db, err = kv.Open(filename, opts)
		}
		return
	}
}

// MemStoreProvider provides memory-only database stores.
func MemStoreProvider() (*kv.DB, error) {
	return kv.CreateMem(&kv.Options{})
}
