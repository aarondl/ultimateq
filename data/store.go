package data

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cznic/kv"
)

// defaultTimeout is the default amount of time after being "unseen" that
// a person will be auto de-authed after.
var defaultTimeout = time.Minute * 5

// AuthFailure is inside AuthErrors to describe why authentication failed.
type AuthFailure int

// These errors are in the AuthError. FailureType field.
const (
	AuthErrBadPassword AuthFailure = iota + 1
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
	FailureType AuthFailure
}

// Error builds the error string for an AuthError.
func (a AuthError) Error() string {
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

// Store is used to store StoredUser objects, and cache their lookup.
type Store struct {
	db *kv.DB

	protect  sync.Mutex
	cache    map[string]*StoredUser
	authed   map[string]string
	timeouts map[string]time.Time
}

// NewStore initializes a store type.
func NewStore(prov DbProvider) (*Store, error) {
	db, err := prov()
	if err != nil {
		return nil, err
	}

	s := &Store{
		db:       db,
		cache:    make(map[string]*StoredUser),
		authed:   make(map[string]string),
		timeouts: make(map[string]time.Time),
	}

	return s, nil
}

// Close closes the underlying database.
func (s *Store) Close() error {
	return s.db.Close()
}

// GlobalUsers gets users with global access
func (s *Store) GlobalUsers() ([]*StoredUser, error) {
	return iterate(s.db, func(ua *StoredUser) bool {
		a, ok := ua.GetAccess("", "")
		return ok && !a.IsZero()
	})
}

// NetworkUsers gets users with Network access
func (s *Store) NetworkUsers(network string) ([]*StoredUser, error) {
	return iterate(s.db, func(ua *StoredUser) bool {
		a, ok := ua.GetAccess(network, "")
		return ok && !a.IsZero()
	})
}

// ChanUsers gets users with access to a channel
func (s *Store) ChanUsers(network, channel string) ([]*StoredUser, error) {
	return iterate(s.db, func(ua *StoredUser) bool {
		a, ok := ua.GetAccess(network, channel)
		return ok && !a.IsZero()
	})
}

func iterate(db *kv.DB, filter func(*StoredUser) bool) ([]*StoredUser, error) {
	list := make([]*StoredUser, 0)

	e, err := db.SeekFirst()
	switch {
	case err == io.EOF:
		return nil, nil
	case err != nil:
		return nil, err
	}

	var stop error
	var val []byte
	for ; stop == nil; _, val, stop = e.Next() {
		if ua, err := deserializeUser(val); err == nil && filter(ua) {
			list = append(list, ua)
		}
	}

	return list, nil
}

// SaveUser saves a user to the database.
func (s *Store) SaveUser(ua *StoredUser) error {
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

	s.protect.Lock()
	s.checkCacheLimits()
	s.cache[ua.Username] = ua.Clone()
	s.protect.Unlock()
	return nil
}

// RemoveUser removes a user from the database, returns true if successful.
func (s *Store) RemoveUser(username string) (removed bool, err error) {
	username = strings.ToLower(username)
	var exists *StoredUser

	s.protect.Lock()
	defer s.protect.Unlock()

	exists, err = s.findUser(username)
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

// AuthUserTmp temporarily authenticates a user. StoredUser will be not nil iff
// the user is found and authenticates successfully.
func (s *Store) AuthUserTmp(
	network, host, username, password string) (*StoredUser, error) {

	return s.authUser(network, host, username, password, true)
}

// AuthUserPerma permanently authenticates a user. StoredUser will be not nil
// iff the user is found and authenticates successfully.
func (s *Store) AuthUserPerma(
	network, host, username, password string) (*StoredUser, error) {

	return s.authUser(network, host, username, password, false)
}

// AuthUser authenticates a user. StoredUser will be not nil iff the user
// is found and authenticates successfully.
func (s *Store) authUser(
	network, host, username, password string, temp bool) (*StoredUser, error) {

	username = strings.ToLower(username)
	var user *StoredUser
	var err error

	s.protect.Lock()
	defer s.protect.Unlock()

	if uname, ok := s.authed[network+host]; ok {
		return s.findUser(uname)
	}

	user, err = s.findUser(username)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, AuthError{
			fmt.Sprintf(errFmtUserNotFound, username),
			AuthErrUserNotFound,
		}
	}

	if !user.HasMask(host) {
		return nil, AuthError{
			fmt.Sprintf(errFmtBadHost, host, username),
			AuthErrHostNotFound,
		}
	}

	if !user.VerifyPassword(password) {
		return nil, AuthError{
			fmt.Sprintf(errFmtBadPassword, username),
			AuthErrBadPassword,
		}
	}

	if temp {
		s.timeouts[network+host] = time.Now().UTC().Add(defaultTimeout)
	}
	s.authed[network+host] = username
	return user.Clone(), nil
}

// AuthedUser looks up a user that was authenticated previously.
func (s *Store) AuthedUser(network, host string) *StoredUser {
	s.protect.Lock()
	defer s.protect.Unlock()
	if username, ok := s.authed[network+host]; ok {
		user, _ := s.findUser(username)
		return user
	}
	return nil
}

// Logout logs an authenticated host out.
func (s *Store) Logout(network, host string) {
	s.protect.Lock()
	defer s.protect.Unlock()
	delete(s.authed, network+host)
}

// LogoutByUsername logs an authenticated username out.
func (s *Store) LogoutByUsername(username string) {
	s.protect.Lock()
	defer s.protect.Unlock()

	username = strings.ToLower(username)
	var hosts []string
	for host, uname := range s.authed {
		if uname == username {
			hosts = append(hosts, host)
		}
	}

	for _, h := range hosts {
		delete(s.authed, h)
	}
}

// Update sets timeouts for seen and unseen users and invokes a reap on users
// who have expired their auth timeouts.
func (s *Store) Update(network string, update StateUpdate) {
	s.protect.Lock()
	defer s.protect.Unlock()

	for _, seen := range update.Seen {
		delete(s.timeouts, network+seen)
	}
	for _, unseen := range update.Unseen {
		if _, ok := s.timeouts[network+unseen]; !ok {
			s.timeouts[network+unseen] = time.Now().UTC().Add(defaultTimeout)
		}
	}
	if len(update.Nick) > 0 {
		s.authed[network+update.Nick[1]] = s.authed[network+update.Nick[0]]
		delete(s.timeouts, network+update.Nick[0])
		delete(s.authed, network+update.Nick[0])
	}
	if len(update.Quit) > 0 {
		delete(s.timeouts, network+update.Quit)
		delete(s.authed, network+update.Quit)
	}

	s.reap()
}

// reap removes users who have exceeded their temporary auths.
func (s *Store) reap() {
	for key, date := range s.timeouts {
		if time.Now().UTC().After(date) {
			delete(s.authed, key)
			delete(s.timeouts, key)
		}
	}
}

// FindUser looks up a user based on username. It caches the result if found.
func (s *Store) FindUser(username string) (user *StoredUser, err error) {
	s.protect.Lock()
	defer s.protect.Unlock()

	return s.findUser(username)
}

// findUser gets a user from the database or the cache, caches if found.
// warning: Assumes the cache is locked
func (s *Store) findUser(username string) (user *StoredUser, err error) {
	username = strings.ToLower(username)

	if cached, ok := s.cache[username]; ok {
		user = cached.Clone()
		return user, nil
	}

	user, err = s.fetchUser(username)
	if user == nil || err != nil {
		return user, err
	}

	s.checkCacheLimits()
	s.cache[username] = user.Clone()
	return
}

// fetchUser gets a user from the database based on username.
func (s *Store) fetchUser(username string) (user *StoredUser, err error) {
	username = strings.ToLower(username)
	var serialized []byte
	serialized, err = s.db.Get(nil, []byte(username))
	if err != nil || serialized == nil {
		return
	}

	user, err = deserializeUser(serialized)
	return
}

// SaveChannel saves a channel to the database.
func (s *Store) SaveChannel(sc *StoredChannel) error {
	var err error
	var serialized []byte

	serialized, err = sc.serialize()
	if err != nil {
		return err
	}

	s.db.Set([]byte(sc.makeID()), serialized)
	if err != nil {
		return err
	}

	return nil
}

// RemoveChannel removes a channel from the database, returns true if successful
func (s *Store) RemoveChannel(netID, name string) (removed bool, err error) {
	var exists *StoredChannel
	exists, err = s.FindChannel(netID, name)
	if err != nil || exists == nil {
		return
	}

	ch := StoredChannel{NetID: netID, Name: name}
	key := ch.makeID()

	err = s.db.Delete([]byte(key))
	if err != nil {
		return
	}
	removed = true
	return
}

// FindChannel looks up a channel based on name.
func (s *Store) FindChannel(netID, name string) (channel *StoredChannel,
	err error) {

	ch := StoredChannel{NetID: netID, Name: name}
	key := ch.makeID()

	var serialized []byte
	serialized, err = s.db.Get(nil, []byte(key))
	if err != nil || serialized == nil {
		return
	}

	channel, err = deserializeChannel(serialized)
	return
}

// Channels returns a slice of the channels found in the database.
func (s *Store) Channels() ([]*StoredChannel, error) {
	list := make([]*StoredChannel, 0)

	e, err := s.db.SeekFirst()
	switch {
	case err == io.EOF:
		return nil, nil
	case err != nil:
		return nil, err
	}

	var stop error
	var val []byte
	for ; stop == nil; _, val, stop = e.Next() {
		if ua, err := deserializeChannel(val); err == nil {
			list = append(list, ua)
		}
	}

	return list, nil
}

// checkCacheLimits verifies if adding one to the size of the cache will
// cross it's boundaries, if so, it dumps the cache.
func (s *Store) checkCacheLimits() {
	if len(s.cache)+1 > nMaxCache {
		s.cache = make(map[string]*StoredUser)
	}
}

// HasAny checks to see if there are any users in the database.
func (s *Store) HasAny() (has bool, err error) {
	var k, v []byte
	k, v, err = s.db.First()
	return k != nil || v != nil, err
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
