package data

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/aarondl/ultimateq/irc"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var (
	// errMissingUnameOrPwd is given when the username or the password of
	// a user is empty string.
	errMissingUnameOrPwd = errors.New("data: Missing username or password")
	// errDuplicateMask is given when a duplicate mask is passed into the
	// NewStoredUser method.
	errDuplicateMask = errors.New("data: Duplicate mask in user creation")
)

const (
	nNewPasswordLen          = 10
	newPasswordStart         = 48
	newPasswordEnd           = 122
	digitSpecialCharsStart   = 58
	digitSpecialCharsEnd     = 64
	lettersSpecialCharsStart = 91
	lettersSpecialCharsEnd   = 96
)

// StoredUser provides access for a user to the bot, networks, and channels.
// This information is protected by a username and crypted password combo.
// Most of StoredUser's access-related methods require a network and a channel,
// but passing in blank strings to these methods allow us to set global,
// channel, and/or network specific access levels.
type StoredUser struct {
	Username   string            `json:"username"`
	Password   []byte            `json:"password"`
	Masks      []string          `json:"masks"`
	Access     map[string]Access `json:"access"`
	JSONStorer `json:"data"`
}

// StoredUserPwdCost is the cost factor for bcrypt. It should not be set
// unless the reasoning is good and the consequences are known.
var StoredUserPwdCost = bcrypt.DefaultCost

// NewStoredUser requires username and password but masks are optional.
func NewStoredUser(un, pw string, masks ...string) (*StoredUser, error) {
	if len(un) == 0 || len(pw) == 0 {
		return nil, errMissingUnameOrPwd
	}

	s := &StoredUser{
		Username:   strings.ToLower(un),
		JSONStorer: make(JSONStorer),
		Access:     make(map[string]Access),
	}

	err := s.SetPassword(pw)
	if err != nil {
		return nil, err
	}

	for _, mask := range masks {
		if !s.AddMask(mask) {
			return nil, errDuplicateMask
		}
	}

	return s, nil
}

func (s *StoredUser) Clone() *StoredUser {
	newStoredUser := &StoredUser{
		Username:   s.Username,
		Password:   make([]byte, len(s.Password)),
		Masks:      make([]string, len(s.Masks)),
		JSONStorer: s.JSONStorer.Clone(),
		Access:     make(map[string]Access, len(s.Access)),
	}

	copy(newStoredUser.Password, s.Password)
	copy(newStoredUser.Masks, s.Masks)

	for k, v := range s.Access {
		newStoredUser.Access[k] = v
	}

	return newStoredUser
}

// createStoredUser creates a useraccess that doesn't care about security.
// Mostly for testing. Will return nil if duplicate masks are given.
func createStoredUser(masks ...string) *StoredUser {
	s := &StoredUser{
		JSONStorer: make(JSONStorer),
		Access:     make(map[string]Access),
	}
	for _, mask := range masks {
		if !s.AddMask(mask) {
			return nil
		}
	}

	return s
}

// SetPassword encrypts the password string, and sets the Password property.
func (s *StoredUser) SetPassword(password string) (err error) {
	var pwd []byte
	pwd, err = bcrypt.GenerateFromPassword([]byte(password), StoredUserPwdCost)
	if err != nil {
		return
	}
	s.Password = pwd
	return
}

// ResetPassword generates a new random password, and sets the user's password
// to that.
func (s *StoredUser) ResetPassword() (newpasswd string, err error) {
	b := make([]byte, nNewPasswordLen)
	for i := 0; i < nNewPasswordLen; i++ {
		r := newPasswordStart + rand.Intn(newPasswordEnd-newPasswordStart)
		if r >= digitSpecialCharsStart && r <= digitSpecialCharsEnd {
			r -= digitSpecialCharsEnd - digitSpecialCharsStart + 1
		} else if r >= lettersSpecialCharsStart && r <= lettersSpecialCharsEnd {
			r -= lettersSpecialCharsEnd - lettersSpecialCharsStart + 1
		}
		b[i] = byte(r)
	}
	newpasswd = string(b)
	err = s.SetPassword(newpasswd)
	return
}

// VerifyPassword checks to see if the given password matches the stored
// password.
func (s *StoredUser) VerifyPassword(password string) bool {
	return nil == bcrypt.CompareHashAndPassword(s.Password, []byte(password))
}

// serialize turns the useraccess into bytes for storage.
func (s *StoredUser) serialize() ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := gob.NewEncoder(buffer)
	err := encoder.Encode(s)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// deserializeUser reverses the Serialize process.
func deserializeUser(serialized []byte) (*StoredUser, error) {
	buffer := &bytes.Buffer{}
	decoder := gob.NewDecoder(buffer)
	if _, err := buffer.Write(serialized); err != nil {
		return nil, err
	}

	dec := &StoredUser{}
	err := decoder.Decode(dec)
	return dec, err
}

// AddMask adds a mask to this users list of wildcard masks. If a duplicate is
// given it returns false.
func (s *StoredUser) AddMask(mask string) bool {
	mask = strings.ToLower(mask)
	for i := 0; i < len(s.Masks); i++ {
		if mask == s.Masks[i] {
			return false
		}
	}
	s.Masks = append(s.Masks, mask)
	return true
}

// RemoveMask deletes a mask from this users list of wildcard masks. Returns true
// if the mask was found and deleted.
func (s *StoredUser) RemoveMask(mask string) (deleted bool) {
	mask = strings.ToLower(mask)
	length := len(s.Masks)
	for i := 0; i < length; i++ {
		if mask == s.Masks[i] {
			s.Masks[i], s.Masks[length-1] = s.Masks[length-1], s.Masks[i]
			s.Masks = s.Masks[:length-1]
			deleted = true
			break
		}
	}
	return
}

// HasMask checks to see if this user has the given masks.
func (s *StoredUser) HasMask(mask string) (has bool) {
	if len(s.Masks) == 0 {
		return true
	}
	mask = strings.ToLower(mask)
	for _, ourMask := range s.Masks {
		if irc.Host(mask).Match(irc.Mask(ourMask)) {
			has = true
			break
		}
	}
	return
}

// Has checks if a user has the given level and flags. Where his access is
// prioritized thusly: Global > Network > Channel
func (s *StoredUser) Has(network, channel string,
	level uint8, flags ...string) bool {

	var searchBits = getFlagBits(flags...)
	var hasFlags, hasLevel bool

	var check = func(access Access) bool {
		hasLevel = hasLevel || access.HasLevel(level)
		hasFlags = hasFlags || ((searchBits & access.Flags) != 0)
		return hasLevel && hasFlags
	}

	if a, ok := s.Access[mkKey("", "")]; ok && check(a) {
		return true
	}
	if len(network) > 0 {
		if a, ok := s.Access[mkKey(network, "")]; ok && check(a) {
			return true
		}
	}
	if len(channel) > 0 {
		if a, ok := s.Access[mkKey("", channel)]; ok && check(a) {
			return true
		}
	}
	if len(network) > 0 && len(channel) > 0 {
		if a, ok := s.Access[mkKey(network, channel)]; ok && check(a) {
			return true
		}
	}

	return false
}

// HasLevel checks if a user has a given level of access. Where his access is
// prioritized thusly: Global > Network > Channel
func (s *StoredUser) HasLevel(network, channel string, level uint8) bool {
	if a, ok := s.Access[mkKey("", "")]; ok && a.Level >= level {
		return true
	}
	if len(network) > 0 {
		if a, ok := s.Access[mkKey(network, "")]; ok && a.Level >= level {
			return true
		}
	}
	if len(channel) > 0 {
		if a, ok := s.Access[mkKey("", channel)]; ok && a.Level >= level {
			return true
		}
	}
	if len(network) > 0 && len(channel) > 0 {
		if a, ok := s.Access[mkKey(network, channel)]; ok && a.Level >= level {
			return true
		}
	}
	return false
}

// HasFlags checks if a user has a given level of access. Where his access is
// prioritized thusly: Global > Network > Channel
func (s *StoredUser) HasFlags(network, channel string, flags ...string) bool {
	var searchBits = getFlagBits(flags...)
	var haveBits uint64

	var check = func(access Access) (had bool) {
		haveBits |= access.Flags
		return (searchBits & haveBits) == searchBits
	}

	if a, ok := s.Access[mkKey("", "")]; ok && check(a) {
		return true
	}
	if len(network) > 0 {
		if a, ok := s.Access[mkKey(network, "")]; ok && check(a) {
			return true
		}
	}
	if len(channel) > 0 {
		if a, ok := s.Access[mkKey("", channel)]; ok && check(a) {
			return true
		}
	}
	if len(network) > 0 && len(channel) > 0 {
		if a, ok := s.Access[mkKey(network, channel)]; ok && check(a) {
			return true
		}
	}

	return false
}

// HasFlag checks if a user has a given flag. Where his access is
// prioritized thusly: Global > Network > Channel
// func (s *StoredUser) HasFlag(network, channel string, flag rune) bool {
// 	if a, ok := s.Access[mkKey("", "")]; ok && a.HasFlag(flag) {
// 		return true
// 	}
// 	if len(network) > 0 {
// 		if a, ok := s.Access[mkKey(network, "")]; ok && a.HasFlag(flag) {
// 			return true
// 		}
// 	}
// 	if len(channel) > 0 {
// 		if a, ok := s.Access[mkKey("", channel)]; ok && a.HasFlag(flag) {
// 			return true
// 		}
// 	}
// 	if len(network) > 0 && len(channel) > 0 {
// 		if a, ok := s.Access[mkKey(network, channel)]; ok && a.HasFlag(flag) {
// 			return true
// 		}
// 	}
// 	return false
// }

// GetAccess returns access using the network and channel provided. The bool
// returns false if the user has no explicit permissions for the level
// requested.
func (s *StoredUser) GetAccess(network, channel string) (Access, bool) {
	a, ok := s.Access[mkKey(network, channel)]
	return a, ok
}

// Grant sets level and flags for a user. To set more specific access
// provide network and channel names.
// A level of 0 will not set anything. Use revoke to remove the level.
// Empty flags will not set anything. Use revoke to remove the flags.
func (s *StoredUser) Grant(
	network, channel string, level uint8, flags ...string) {

	key := mkKey(network, channel)
	access := s.Access[key]
	changed := false
	if len(flags) > 0 {
		access.SetFlags(flags...)
		changed = true
	}
	if level > 0 {
		access.Level = level
		changed = true
	}
	if changed {
		s.Access[key] = access
	}
}

// Revoke removes a user's access. To revoke more specific access
// provide network and channel names.
func (s *StoredUser) Revoke(network, channel string) {
	key := mkKey(network, channel)
	delete(s.Access, key)
}

// RevokeLevel removes level access.
func (s *StoredUser) RevokeLevel(network, channel string) {
	key := mkKey(network, channel)
	access, ok := s.Access[key]
	changed := false
	if ok && access.Level > 0 {
		access.Level = 0
		changed = true
	}
	if changed {
		s.Access[key] = access
	}
}

// RevokeFlags removes flags from the user.
// Leaving flags empty removes ALL flags.
func (s *StoredUser) RevokeFlags(network, channel string, flags ...string) {
	key := mkKey(network, channel)
	access, ok := s.Access[key]
	changed := false
	if ok && access.Flags > 0 {
		if len(flags) == 0 {
			access.Flags = 0
		} else {
			access.ClearFlags(flags...)
		}
		changed = true
	}
	if changed {
		s.Access[key] = access
	}
}

// RevokeAll removes all access from the user.
func (s *StoredUser) RevokeAll() {
	s.Access = make(map[string]Access)
}

// String turns StoredUser's access into a user consumable format.
func (s *StoredUser) String(network, channel string) string {
	var b = &bytes.Buffer{}

	if len(s.Access) == 0 {
		return none
	}

	s.writeIt(b, ":") // Always write global

	var keys []string
	for k, _ := range s.Access {
		if k == ":" {
			continue
		}

		if len(network) != 0 && !strings.HasPrefix(k, network) {
			continue
		}
		if len(channel) != 0 && !strings.HasSuffix(k, channel) {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// write keys out in 3 waves:
	// things that don't start with : and end with : (network:)
	// things that start with : and don't end with : (:channel)
	// things that don't start with : and don't end with : (network:channel)
	for out := 0; out <= 2; out++ {
		for _, k := range keys {
			switch f, l := k[0], k[len(k)-1]; {
			case out == 0 && f != ':' && l == ':':
				s.writeIt(b, k)
			case out == 1 && f == ':' && l != ':':
				s.writeIt(b, k)
			case out == 2 && f != ':' && l != ':':
				s.writeIt(b, k)
			}
		}
	}

	if b.Len() == 0 {
		return none
	}
	return b.String()
}

func (s *StoredUser) writeIt(b *bytes.Buffer, key string) {
	spl := strings.Split(key, ":")
	network, channel := spl[0], spl[1]

	access, ok := s.Access[key]
	if !ok || (access.Level == 0 && access.Flags == 0) {
		return
	}

	if b.Len() > 0 {
		b.WriteByte(' ')
	}
	n, c := len(network) > 0, len(channel) > 0
	switch {
	case n && c:
		fmt.Fprintf(b, "%s:%s(%v)", network, channel, access)
	case n:
		fmt.Fprintf(b, "%s(%v)", network, access)
	case c:
		fmt.Fprintf(b, "%s(%v)", channel, access)
	default:
		fmt.Fprintf(b, "G(%v)", access)
	}
}

// mkKey gets a key for accessing the Access map.
func mkKey(network, channel string) string {
	return fmt.Sprintf("%s:%s",
		strings.ToLower(network), strings.ToLower(channel))
}
