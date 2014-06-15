package data

import (
	"bytes"
	"encoding/gob"
	"errors"
	"math/rand"
	"strings"
	"time"

	"code.google.com/p/go.crypto/bcrypt"
	"github.com/aarondl/ultimateq/irc"
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

// StoredUser provides access for a user to the bot, servers, and channels.
// This information is protected by a username and crypted password combo.
type StoredUser struct {
	Username string
	Password []byte
	Masks    []string
	Global   *Access
	Server   map[string]*Access
	Channel  map[string]map[string]*Access
}

// StoredUserPwdCost is the cost factor for bcrypt. It should not be set
// unless the reasoning is good and the consequences are known.
var StoredUserPwdCost = bcrypt.DefaultCost

// NewStoredUser initializes an access user. Requires username and password,
// but masks are optional.
func NewStoredUser(un, pw string,
	masks ...string) (*StoredUser, error) {

	if len(un) == 0 || len(pw) == 0 {
		return nil, errMissingUnameOrPwd
	}

	a := &StoredUser{
		Username: strings.ToLower(un),
	}

	err := a.SetPassword(pw)
	if err != nil {
		return nil, err
	}

	for _, mask := range masks {
		if !a.AddMask(mask) {
			return nil, errDuplicateMask
		}
	}

	return a, nil
}

// createStoredUser creates a useraccess that doesn't care about security.
// Mostly for testing. Will return nil if duplicate masks are given.
func createStoredUser(masks ...string) *StoredUser {
	a := &StoredUser{}
	for _, mask := range masks {
		if !a.AddMask(mask) {
			return nil
		}
	}

	return a
}

// ensureServer that the server access object is created.
func (a *StoredUser) ensureServer(server string) (access *Access) {
	server = strings.ToLower(server)
	if a.Server == nil {
		a.Server = make(map[string]*Access)
	}
	if access = a.Server[server]; access == nil {
		access = NewAccess(0)
		a.Server[server] = access
	}
	return
}

// ensureChannel ensures that the server access object is created.
func (a *StoredUser) ensureChannel(server, channel string) (access *Access) {
	server = strings.ToLower(server)
	channel = strings.ToLower(channel)
	var chans map[string]*Access
	if a.Channel == nil {
		a.Channel = make(map[string]map[string]*Access)
	}
	if chans = a.Channel[server]; chans == nil {
		a.Channel[server] = make(map[string]*Access)
		chans = a.Channel[server]
	}
	if access = chans[channel]; access == nil {
		access = NewAccess(0)
		chans[channel] = access
	}
	return
}

// doServer get's the server access, calls a callback if the server exists.
func (a *StoredUser) doServer(server string, do func(string, *Access)) {
	server = strings.ToLower(server)
	if access, ok := a.Server[server]; ok {
		do(server, access)
	}
}

// doChannel get's the server access, calls a callback if the channel exists.
func (a *StoredUser) doChannel(server, channel string,
	do func(string, string, *Access)) {

	server = strings.ToLower(server)
	channel = strings.ToLower(channel)
	if chanMap, ok := a.Channel[server]; ok {
		if access, ok := chanMap[channel]; ok {
			do(server, channel, access)
		}
	}
}

// SetPassword encrypts the password string, and sets the Password property.
func (a *StoredUser) SetPassword(password string) (err error) {
	var pwd []byte
	pwd, err = bcrypt.GenerateFromPassword([]byte(password), StoredUserPwdCost)
	if err != nil {
		return
	}
	a.Password = pwd
	return
}

// ResetPassword generates a new random password, and sets the user's password
// to that.
func (a *StoredUser) ResetPassword() (newpasswd string, err error) {
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
	err = a.SetPassword(newpasswd)
	return
}

// VerifyPassword checks to see if the given password matches the stored
// password.
func (a *StoredUser) VerifyPassword(password string) bool {
	return nil == bcrypt.CompareHashAndPassword(a.Password, []byte(password))
}

// serialize turns the useraccess into bytes for storage.
func (a *StoredUser) serialize() ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := gob.NewEncoder(buffer)
	err := encoder.Encode(a)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// deserialize reverses the Serialize process.
func deserialize(serialized []byte) (*StoredUser, error) {
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
func (a *StoredUser) AddMask(mask string) bool {
	mask = strings.ToLower(mask)
	for i := 0; i < len(a.Masks); i++ {
		if mask == a.Masks[i] {
			return false
		}
	}
	a.Masks = append(a.Masks, mask)
	return true
}

// DelMask deletes a mask from this users list of wildcard masks. Returns true
// if the mask was found and deleted.
func (a *StoredUser) DelMask(mask string) (deleted bool) {
	mask = strings.ToLower(mask)
	length := len(a.Masks)
	for i := 0; i < length; i++ {
		if mask == a.Masks[i] {
			a.Masks[i], a.Masks[length-1] = a.Masks[length-1], a.Masks[i]
			a.Masks = a.Masks[:length-1]
			deleted = true
			break
		}
	}
	return
}

// ValidateMask checks to see if this user has the given masks.
func (a *StoredUser) ValidateMask(mask string) (has bool) {
	if len(a.Masks) == 0 {
		return true
	}
	mask = strings.ToLower(mask)
	for _, ourMask := range a.Masks {
		if irc.Host(mask).Match(irc.Mask(ourMask)) {
			has = true
			break
		}
	}
	return
}

// Has checks if a user has the given level and flags. Where his access is
// overridden thusly: Global > Server > Channel
func (a *StoredUser) Has(server, channel string,
	level uint8, flags ...string) bool {

	server = strings.ToLower(server)
	channel = strings.ToLower(channel)

	var searchBits = getFlagBits(flags...)
	var hasFlags, hasLevel bool

	var check = func(access *Access) bool {
		if access != nil {
			hasLevel = hasLevel || access.HasLevel(level)
			hasFlags = hasFlags || ((searchBits & access.Flags) != 0)
		}
		return hasLevel && hasFlags
	}

	if check(a.Global) {
		return true
	}
	if check(a.Server[server]) {
		return true
	}
	if chans, ok := a.Channel[server]; ok {
		if check(chans[channel]) {
			return true
		}
	}

	return false
}

// HasLevel checks if a user has a given level of access. Where his access is
// overridden thusly: Global > Server > Channel
func (a *StoredUser) HasLevel(server, channel string, level uint8) bool {
	if a.HasGlobalLevel(level) {
		return true
	}
	if a.HasServerLevel(server, level) {
		return true
	}
	if a.HasChannelLevel(server, channel, level) {
		return true
	}
	return false
}

// HasFlags checks if a user has a given level of access. Where his access is
// overridden thusly: Global > Server > Channel
func (a *StoredUser) HasFlags(server, channel string, flags ...string) bool {
	var searchBits = getFlagBits(flags...)

	server = strings.ToLower(server)
	channel = strings.ToLower(channel)

	var check = func(access *Access) (had bool) {
		if access != nil {
			had = (searchBits & access.Flags) != 0
		}
		return
	}

	if check(a.Global) {
		return true
	}
	if check(a.Server[server]) {
		return true
	}
	if chans, ok := a.Channel[server]; ok {
		if check(chans[channel]) {
			return true
		}
	}

	return false
}

// HasFlag checks if a user has a given flag. Where his access is
// overridden thusly: Global > Server > Channel
func (a *StoredUser) HasFlag(server, channel string, flag rune) bool {
	if a.HasGlobalFlag(flag) {
		return true
	}
	if a.HasServerFlag(server, flag) {
		return true
	}
	if a.HasChannelFlag(server, channel, flag) {
		return true
	}
	return false
}

// GrantGlobal sets both Level and Flags at the same time.
func (a *StoredUser) GrantGlobal(level uint8, flags ...string) {
	a.GrantGlobalLevel(level)
	a.GrantGlobalFlags(flags...)
}

// GrantGlobalFlags sets global flags.
func (a *StoredUser) GrantGlobalFlags(flags ...string) {
	if a.Global == nil {
		a.Global = &Access{}
	}
	a.Global.SetFlags(flags...)
}

// GrantGlobalLevel sets global level.
func (a *StoredUser) GrantGlobalLevel(level uint8) {
	if a.Global == nil {
		a.Global = &Access{}
	}
	a.Global.Level = level
}

// RevokeGlobal removes a user's global access.
func (a *StoredUser) RevokeGlobal() {
	a.Global = nil
}

// RevokeGlobalLevel removes global access.
func (a *StoredUser) RevokeGlobalLevel() {
	if a.Global != nil {
		a.Global.Level = 0
	}
}

// RevokeGlobalFlags removes flags from the global level.
func (a *StoredUser) RevokeGlobalFlags(flags ...string) {
	if a.Global != nil {
		a.Global.ClearFlags(flags...)
	}
}

// GetGlobal returns the global access.
func (a *StoredUser) GetGlobal() *Access {
	return a.Global
}

// HasGlobalLevel checks a user to see if their global level access is equal
// or above the specified access.
func (a *StoredUser) HasGlobalLevel(level uint8) (has bool) {
	if a.Global != nil {
		has = a.Global.HasLevel(level)
	}
	return
}

// HasGlobalFlags checks a user to see if their global level flags contain the
// given flags.
func (a *StoredUser) HasGlobalFlags(flags ...string) (has bool) {
	if a.Global != nil {
		has = a.Global.HasFlags(flags...)
	}
	return
}

// HasGlobalFlag checks a user to see if their global level flags contain the
// given flag.
func (a *StoredUser) HasGlobalFlag(flag rune) (has bool) {
	if a.Global != nil {
		has = a.Global.HasFlag(flag)
	}
	return
}

// GrantServer sets both Level and Flags at the same time.
func (a *StoredUser) GrantServer(server string, level uint8, flags ...string) {
	a.ensureServer(server).SetAccess(level, flags...)
}

// GrantServerFlags sets server flags.
func (a *StoredUser) GrantServerFlags(server string, flags ...string) {
	a.ensureServer(server).SetFlags(flags...)
}

// GrantServerLevel sets server level.
func (a *StoredUser) GrantServerLevel(server string, level uint8) {
	a.ensureServer(server).Level = level
}

// RevokeServer removes a user's server access.
func (a *StoredUser) RevokeServer(server string) {
	a.doServer(server, func(srv string, _ *Access) {
		delete(a.Server, srv)
	})
}

// RevokeServerLevel removes server access.
func (a *StoredUser) RevokeServerLevel(server string) {
	a.doServer(server, func(_ string, access *Access) {
		access.Level = 0
	})
}

// RevokeServerFlags removes flags from the server level.
func (a *StoredUser) RevokeServerFlags(server string, flags ...string) {
	a.doServer(server, func(_ string, access *Access) {
		access.ClearFlags(flags...)
	})
}

// GetServer gets the server access for the given server.
func (a *StoredUser) GetServer(server string) (access *Access) {
	a.doServer(server, func(_ string, acc *Access) {
		access = acc
	})
	return
}

// HasServerLevel checks a user to see if their server level access is equal
// or above the specified access.
func (a *StoredUser) HasServerLevel(server string, level uint8) (has bool) {
	a.doServer(server, func(_ string, access *Access) {
		has = access.HasLevel(level)
	})
	return
}

// HasServerFlags checks a user to see if their server level flags contain the
// given flags.
func (a *StoredUser) HasServerFlags(server string, flags ...string) (has bool) {
	a.doServer(server, func(_ string, access *Access) {
		has = access.HasFlags(flags...)
	})
	return
}

// HasServerFlag checks a user to see if their server level flags contain the
// given flag.
func (a *StoredUser) HasServerFlag(server string, flag rune) (has bool) {
	a.doServer(server, func(_ string, access *Access) {
		has = access.HasFlag(flag)
	})
	return
}

// GrantChannel sets both Level and Flags at the same time.
func (a *StoredUser) GrantChannel(server, channel string, level uint8,
	flags ...string) {

	a.ensureChannel(server, channel).SetAccess(level, flags...)
}

// GrantChannelFlags sets channel flags.
func (a *StoredUser) GrantChannelFlags(server, channel string,
	flags ...string) {

	a.ensureChannel(server, channel).SetFlags(flags...)
}

// GrantChannelLevel sets channel level.
func (a *StoredUser) GrantChannelLevel(server, channel string, level uint8) {
	a.ensureChannel(server, channel).Level = level
}

// RevokeChannel removes a user's channel access.
func (a *StoredUser) RevokeChannel(server, channel string) {
	a.doChannel(server, channel, func(srv, ch string, _ *Access) {
		delete(a.Channel[srv], ch)
	})
}

// RevokeChannelLevel removes channel access.
func (a *StoredUser) RevokeChannelLevel(server, channel string) {
	a.doChannel(server, channel, func(_, _ string, access *Access) {
		access.Level = 0
	})
}

// RevokeChannelFlags removes flags from the channel level.
func (a *StoredUser) RevokeChannelFlags(server, channel string,
	flags ...string) {
	a.doChannel(server, channel, func(_, _ string, access *Access) {
		access.ClearFlags(flags...)
	})
}

// GetChannel gets the server access for the given channel.
func (a *StoredUser) GetChannel(server, channel string) (access *Access) {
	a.doChannel(server, channel, func(_, _ string, acc *Access) {
		access = acc
	})
	return
}

// HasChannelLevel checks a user to see if their channel level access is equal
// or above the specified access.
func (a *StoredUser) HasChannelLevel(server, channel string,
	level uint8) (has bool) {

	a.doChannel(server, channel, func(_, _ string, access *Access) {
		has = access.HasLevel(level)
	})
	return
}

// HasChannelFlags checks a user to see if their channel level flags contain the
// given flags.
func (a *StoredUser) HasChannelFlags(server, channel string,
	flags ...string) (has bool) {

	a.doChannel(server, channel, func(_, _ string, access *Access) {
		has = access.HasFlags(flags...)
	})
	return
}

// HasChannelFlag checks a user to see if their channel level flags contain the
// given flag.
func (a *StoredUser) HasChannelFlag(server, channel string,
	flag rune) (has bool) {

	a.doChannel(server, channel, func(_, _ string, access *Access) {
		has = access.HasFlag(flag)
	})
	return
}

// String turns StoredUser into a user consumable format.
func (a *StoredUser) String(server, channel string) (str string) {
	var wrote bool

	if a.Global != nil && (a.Global.Level > 0 || a.Global.Flags > 0) {
		str += "G(" + a.Global.String() + ")"
		wrote = true
	}

	a.doServer(server, func(_ string, srv *Access) {
		if wrote {
			str += " "
		}
		str += "S(" + srv.String() + ")"
		wrote = true
	})

	server = strings.ToLower(server)
	channel = strings.ToLower(channel)
	if chsrv, ok := a.Channel[server]; ok {
		if len(channel) != 0 {
			if ch, ok := chsrv[channel]; ok && (ch.Level > 0 || ch.Flags > 0) {
				if wrote {
					str += " "
				}
				str += channel + "(" + ch.String() + ")"
				wrote = true
			}
		} else {
			for channel, ch := range chsrv {
				if ch.Level == 0 && ch.Flags == 0 {
					continue
				}
				if wrote {
					str += " "
				}
				str += channel + "(" + ch.String() + ")"
				wrote = true
			}
		}
	}

	if !wrote {
		return none
	}
	return
}
