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

// StoredUser provides access for a user to the bot, networks, and channels.
// This information is protected by a username and crypted password combo.
type StoredUser struct {
	Username string
	Password []byte
	Masks    []string
	Global   *Access
	Network  map[string]*Access
	Channel  map[string]map[string]*Access
	JSONStorer
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
		Username:   strings.ToLower(un),
		JSONStorer: make(JSONStorer),
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

// ensureNetwork that the network access object is created.
func (a *StoredUser) ensureNetwork(network string) (access *Access) {
	network = strings.ToLower(network)
	if a.Network == nil {
		a.Network = make(map[string]*Access)
	}
	if access = a.Network[network]; access == nil {
		access = NewAccess(0)
		a.Network[network] = access
	}
	return
}

// ensureChannel ensures that the network access object is created.
func (a *StoredUser) ensureChannel(network, channel string) (access *Access) {
	network = strings.ToLower(network)
	channel = strings.ToLower(channel)
	var chans map[string]*Access
	if a.Channel == nil {
		a.Channel = make(map[string]map[string]*Access)
	}
	if chans = a.Channel[network]; chans == nil {
		a.Channel[network] = make(map[string]*Access)
		chans = a.Channel[network]
	}
	if access = chans[channel]; access == nil {
		access = NewAccess(0)
		chans[channel] = access
	}
	return
}

// doNetwork get's the network access, calls a callback if the network exists.
func (a *StoredUser) doNetwork(network string, do func(string, *Access)) {
	network = strings.ToLower(network)
	if access, ok := a.Network[network]; ok {
		do(network, access)
	}
}

// doChannel get's the network access, calls a callback if the channel exists.
func (a *StoredUser) doChannel(network, channel string,
	do func(string, string, *Access)) {

	network = strings.ToLower(network)
	channel = strings.ToLower(channel)
	if chanMap, ok := a.Channel[network]; ok {
		if access, ok := chanMap[channel]; ok {
			do(network, channel, access)
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
// overridden thusly: Global > Network > Channel
func (a *StoredUser) Has(network, channel string,
	level uint8, flags ...string) bool {

	network = strings.ToLower(network)
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
	if check(a.Network[network]) {
		return true
	}
	if chans, ok := a.Channel[network]; ok {
		if check(chans[channel]) {
			return true
		}
	}

	return false
}

// HasLevel checks if a user has a given level of access. Where his access is
// overridden thusly: Global > Network > Channel
func (a *StoredUser) HasLevel(network, channel string, level uint8) bool {
	if a.HasGlobalLevel(level) {
		return true
	}
	if a.HasNetworkLevel(network, level) {
		return true
	}
	if a.HasChannelLevel(network, channel, level) {
		return true
	}
	return false
}

// HasFlags checks if a user has a given level of access. Where his access is
// overridden thusly: Global > Network > Channel
func (a *StoredUser) HasFlags(network, channel string, flags ...string) bool {
	var searchBits = getFlagBits(flags...)

	network = strings.ToLower(network)
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
	if check(a.Network[network]) {
		return true
	}
	if chans, ok := a.Channel[network]; ok {
		if check(chans[channel]) {
			return true
		}
	}

	return false
}

// HasFlag checks if a user has a given flag. Where his access is
// overridden thusly: Global > Network > Channel
func (a *StoredUser) HasFlag(network, channel string, flag rune) bool {
	if a.HasGlobalFlag(flag) {
		return true
	}
	if a.HasNetworkFlag(network, flag) {
		return true
	}
	if a.HasChannelFlag(network, channel, flag) {
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

// GrantNetwork sets both Level and Flags at the same time.
func (a *StoredUser) GrantNetwork(network string, level uint8,
	flags ...string) {

	a.ensureNetwork(network).SetAccess(level, flags...)
}

// GrantNetworkFlags sets network flags.
func (a *StoredUser) GrantNetworkFlags(network string, flags ...string) {
	a.ensureNetwork(network).SetFlags(flags...)
}

// GrantNetworkLevel sets network level.
func (a *StoredUser) GrantNetworkLevel(network string, level uint8) {
	a.ensureNetwork(network).Level = level
}

// RevokeNetwork removes a user's network access.
func (a *StoredUser) RevokeNetwork(network string) {
	a.doNetwork(network, func(srv string, _ *Access) {
		delete(a.Network, srv)
	})
}

// RevokeNetworkLevel removes network access.
func (a *StoredUser) RevokeNetworkLevel(network string) {
	a.doNetwork(network, func(_ string, access *Access) {
		access.Level = 0
	})
}

// RevokeNetworkFlags removes flags from the network level.
func (a *StoredUser) RevokeNetworkFlags(network string, flags ...string) {
	a.doNetwork(network, func(_ string, access *Access) {
		access.ClearFlags(flags...)
	})
}

// GetNetwork gets the network access for the given network.
func (a *StoredUser) GetNetwork(network string) (access *Access) {
	a.doNetwork(network, func(_ string, acc *Access) {
		access = acc
	})
	return
}

// HasNetworkLevel checks a user to see if their network level access is equal
// or above the specified access.
func (a *StoredUser) HasNetworkLevel(network string, level uint8) (has bool) {
	a.doNetwork(network, func(_ string, access *Access) {
		has = access.HasLevel(level)
	})
	return
}

// HasNetworkFlags checks a user to see if their network level flags contain the
// given flags.
func (a *StoredUser) HasNetworkFlags(network string,
	flags ...string) (has bool) {

	a.doNetwork(network, func(_ string, access *Access) {
		has = access.HasFlags(flags...)
	})
	return
}

// HasNetworkFlag checks a user to see if their network level flags contain the
// given flag.
func (a *StoredUser) HasNetworkFlag(network string, flag rune) (has bool) {
	a.doNetwork(network, func(_ string, access *Access) {
		has = access.HasFlag(flag)
	})
	return
}

// GrantChannel sets both Level and Flags at the same time.
func (a *StoredUser) GrantChannel(network, channel string, level uint8,
	flags ...string) {

	a.ensureChannel(network, channel).SetAccess(level, flags...)
}

// GrantChannelFlags sets channel flags.
func (a *StoredUser) GrantChannelFlags(network, channel string,
	flags ...string) {

	a.ensureChannel(network, channel).SetFlags(flags...)
}

// GrantChannelLevel sets channel level.
func (a *StoredUser) GrantChannelLevel(network, channel string, level uint8) {
	a.ensureChannel(network, channel).Level = level
}

// RevokeChannel removes a user's channel access.
func (a *StoredUser) RevokeChannel(network, channel string) {
	a.doChannel(network, channel, func(srv, ch string, _ *Access) {
		delete(a.Channel[srv], ch)
	})
}

// RevokeChannelLevel removes channel access.
func (a *StoredUser) RevokeChannelLevel(network, channel string) {
	a.doChannel(network, channel, func(_, _ string, access *Access) {
		access.Level = 0
	})
}

// RevokeChannelFlags removes flags from the channel level.
func (a *StoredUser) RevokeChannelFlags(network, channel string,
	flags ...string) {
	a.doChannel(network, channel, func(_, _ string, access *Access) {
		access.ClearFlags(flags...)
	})
}

// GetChannel gets the network access for the given channel.
func (a *StoredUser) GetChannel(network, channel string) (access *Access) {
	a.doChannel(network, channel, func(_, _ string, acc *Access) {
		access = acc
	})
	return
}

// HasChannelLevel checks a user to see if their channel level access is equal
// or above the specified access.
func (a *StoredUser) HasChannelLevel(network, channel string,
	level uint8) (has bool) {

	a.doChannel(network, channel, func(_, _ string, access *Access) {
		has = access.HasLevel(level)
	})
	return
}

// HasChannelFlags checks a user to see if their channel level flags contain the
// given flags.
func (a *StoredUser) HasChannelFlags(network, channel string,
	flags ...string) (has bool) {

	a.doChannel(network, channel, func(_, _ string, access *Access) {
		has = access.HasFlags(flags...)
	})
	return
}

// HasChannelFlag checks a user to see if their channel level flags contain the
// given flag.
func (a *StoredUser) HasChannelFlag(network, channel string,
	flag rune) (has bool) {

	a.doChannel(network, channel, func(_, _ string, access *Access) {
		has = access.HasFlag(flag)
	})
	return
}

// String turns StoredUser into a user consumable format.
func (a *StoredUser) String(network, channel string) (str string) {
	var wrote bool

	if a.Global != nil && (a.Global.Level > 0 || a.Global.Flags > 0) {
		str += "G(" + a.Global.String() + ")"
		wrote = true
	}

	a.doNetwork(network, func(_ string, srv *Access) {
		if wrote {
			str += " "
		}
		str += "S(" + srv.String() + ")"
		wrote = true
	})

	network = strings.ToLower(network)
	channel = strings.ToLower(channel)
	if chsrv, ok := a.Channel[network]; ok {
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
