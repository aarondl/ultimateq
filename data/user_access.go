package data

import (
	"bytes"
	"code.google.com/p/go.crypto/bcrypt"
	"encoding/gob"
	"errors"
	"github.com/aarondl/ultimateq/irc"
	"sync"
)

var (
	buffer     = &bytes.Buffer{}
	encoder    = gob.NewEncoder(buffer)
	decoder    = gob.NewDecoder(buffer)
	bufferLock = sync.Mutex{}

	errMissingUnameOrPwd = errors.New("data: Missing username or password.")
)

// UserAccess provides access for a user to the bot, servers, and channels.
// This information is protected by a username and crypted password combo.
type UserAccess struct {
	Username string
	Password []byte
	Masks    []string
	Global   *Access
	Server   map[string]*Access
	Channel  map[string]map[string]*Access
}

// UserAccessPwdCost is the cost factor for bcrypt. It should not be set
// unless the reasoning is good and the consequences are known.
var UserAccessPwdCost = bcrypt.DefaultCost

// CreateUserAccess initializes an access user. Requires username and password,
// but masks are optional.
func CreateUserAccess(un, pw string,
	masks ...string) (*UserAccess, error) {

	if len(un) == 0 || len(pw) == 0 {
		return nil, errMissingUnameOrPwd
	}

	a := &UserAccess{
		Username: un,
	}

	err := a.SetPassword(pw)
	if err != nil {
		return nil, err
	}

	a.Masks = make([]string, len(masks))
	copy(a.Masks, masks)

	return a, nil
}

// createUserAccess creates a useraccess that doesn't care about security.
// Mostly for testing.
func createUserAccess(masks ...string) *UserAccess {
	a := &UserAccess{}
	a.Masks = make([]string, len(masks))
	copy(a.Masks, masks)

	return a
}

// ensureServer that the server access object is created.
func (a *UserAccess) ensureServer(server string) (access *Access) {
	if a.Server == nil {
		a.Server = make(map[string]*Access)
	}
	if access = a.Server[server]; access == nil {
		access = CreateAccess(0)
		a.Server[server] = access
	}
	return
}

// ensureChannel ensures that the server access object is created.
func (a *UserAccess) ensureChannel(server, channel string) (access *Access) {
	var chans map[string]*Access
	if a.Channel == nil {
		a.Channel = make(map[string]map[string]*Access)
	}
	if chans = a.Channel[server]; chans == nil {
		a.Channel[server] = make(map[string]*Access)
		chans = a.Channel[server]
	}
	if access = chans[channel]; access == nil {
		access = CreateAccess(0)
		chans[channel] = access
	}
	return
}

// SetPassword encrypts the password string, and sets the Password property.
func (a *UserAccess) SetPassword(password string) (err error) {
	var pwd []byte
	pwd, err = bcrypt.GenerateFromPassword([]byte(password), UserAccessPwdCost)
	if err != nil {
		return
	}
	a.Password = pwd
	return
}

// VerifyPassword checks to see if the given password matches the stored
// password.
func (a *UserAccess) VerifyPassword(password string) bool {
	return nil == bcrypt.CompareHashAndPassword(a.Password, []byte(password))
}

// Serialize turns the useraccess into bytes for storage.
func (a *UserAccess) Serialize() ([]byte, error) {
	bufferLock.Lock()
	defer bufferLock.Unlock()
	buffer.Reset()
	err := encoder.Encode(a)
	if err != nil {
		return nil, err
	}

	cpy := make([]byte, buffer.Len())
	copy(cpy, buffer.Bytes())
	return cpy, nil
}

// deserialize reverses the Serialize process.
func deserialize(serialized []byte) (*UserAccess, error) {
	bufferLock.Lock()
	defer bufferLock.Unlock()
	buffer.Reset()
	if _, err := buffer.Write(serialized); err != nil {
		return nil, err
	}

	dec := &UserAccess{}
	err := decoder.Decode(dec)
	return dec, err
}

// AddMasks adds masks to this users list of wildcard masks. If a duplicate is
// given, the entire update set is rejected.
func (a *UserAccess) AddMasks(masks ...string) bool {
	for i := 0; i < len(a.Masks); i++ {
		for j := 0; j < len(masks); j++ {
			if masks[j] == a.Masks[i] {
				return false
			}
		}
	}
	a.Masks = append(a.Masks, masks...)
	return true
}

// DelMasks deletes masks to this users list of wildcard masks. If a single mask
// is deleted, it returns true, even if other masks failed to delete.
func (a *UserAccess) DelMasks(masks ...string) (deleted bool) {
	for i := 0; i < len(a.Masks); i++ {
		for j := 0; j < len(masks); j++ {
			if masks[j] == a.Masks[i] {
				a.Masks[i], a.Masks[len(a.Masks)-1] =
					a.Masks[len(a.Masks)-1], a.Masks[i]
				a.Masks = a.Masks[:len(a.Masks)-1]
				deleted = true
			}
		}
	}
	return
}

// Has checks if a user has the given level and flags. Where his access is
// overridden thusly: Global > Server > Channel
func (a *UserAccess) Has(server, channel string,
	level uint8, flags ...string) bool {

	var searchBits = getFlagBits(flags...)
	var hasFlags, hasLevel bool

	var check = func(access *Access) bool {
		if access != nil {
			hasLevel = hasLevel || access.HasLevel(level)
			searchBits &= ^access.Flags
			hasFlags = hasFlags || searchBits == 0
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
func (a *UserAccess) HasLevel(server, channel string, level uint8) bool {
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
func (a *UserAccess) HasFlags(server, channel string, flags ...string) bool {
	var searchBits = getFlagBits(flags...)

	var check = func(access *Access) bool {
		if access != nil {
			searchBits &= ^access.Flags
		}
		return searchBits == 0
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

// IsMatch checks to see if this UserAccess has a wildmask that will satisfy
// the given mask.
func (a *UserAccess) IsMatch(mask irc.Mask) bool {
	for i := 0; i < len(a.Masks); i++ {
		if irc.WildMask(a.Masks[i]).Match(mask) {
			return true
		}
	}
	return false
}

// GrantGlobal sets both Level and Flags at the same time.
func (a *UserAccess) GrantGlobal(level uint8, flags ...string) {
	a.GrantGlobalLevel(level)
	a.GrantGlobalFlags(flags...)
}

// GrantGlobalFlags sets global flags.
func (a *UserAccess) GrantGlobalFlags(flags ...string) {
	if a.Global == nil {
		a.Global = &Access{}
	}
	a.Global.SetFlags(flags...)
}

// GrantGlobalLevel sets global level.
func (a *UserAccess) GrantGlobalLevel(level uint8) {
	if a.Global == nil {
		a.Global = &Access{}
	}
	a.Global.Level = level
}

// RevokeGlobal removes a user's global access.
func (a *UserAccess) RevokeGlobal() {
	a.Global = nil
}

// RevokeGlobalLevel removes global access.
func (a *UserAccess) RevokeGlobalLevel() {
	if a.Global != nil {
		a.Global.Level = 0
	}
}

// RevokeGlobalFlags removes flags from the global level.
func (a *UserAccess) RevokeGlobalFlags(flags ...string) {
	if a.Global != nil {
		a.Global.ClearFlags(flags...)
	}
}

// GetGlobal returns the global access.
func (a *UserAccess) GetGlobal() *Access {
	return a.Global
}

// HasGlobalLevel checks a user to see if their global level access is equal
// or above the specified access.
func (a *UserAccess) HasGlobalLevel(level uint8) (has bool) {
	if a.Global != nil {
		has = a.Global.HasLevel(level)
	}
	return
}

// HasGlobalFlags checks a user to see if their global level flags contain the
// given flags.
func (a *UserAccess) HasGlobalFlags(flags ...string) (has bool) {
	if a.Global != nil {
		has = a.Global.HasFlags(flags...)
	}
	return
}

// HasGlobalFlag checks a user to see if their global level flags contain the
// given flag.
func (a *UserAccess) HasGlobalFlag(flag rune) (has bool) {
	if a.Global != nil {
		has = a.Global.HasFlag(flag)
	}
	return
}

// GrantServer sets both Level and Flags at the same time.
func (a *UserAccess) GrantServer(server string, level uint8, flags ...string) {
	a.ensureServer(server).SetAccess(level, flags...)
}

// GrantServerFlags sets server flags.
func (a *UserAccess) GrantServerFlags(server string, flags ...string) {
	a.ensureServer(server).SetFlags(flags...)
}

// GrantServerLevel sets server level.
func (a *UserAccess) GrantServerLevel(server string, level uint8) {
	a.ensureServer(server).Level = level
}

// RevokeServer removes a user's server access.
func (a *UserAccess) RevokeServer(server string) {
	delete(a.Server, server)
}

// RevokeServerLevel removes server access.
func (a *UserAccess) RevokeServerLevel(server string) {
	if access, ok := a.Server[server]; ok {
		access.Level = 0
	}
}

// RevokeServerFlags removes flags from the server level.
func (a *UserAccess) RevokeServerFlags(server string, flags ...string) {
	if access, ok := a.Server[server]; ok {
		access.ClearFlags(flags...)
	}
}

// GetServer gets the server access for the given server.
func (a *UserAccess) GetServer(server string) *Access {
	return a.Server[server]
}

// HasServerLevel checks a user to see if their server level access is equal
// or above the specified access.
func (a *UserAccess) HasServerLevel(server string, level uint8) (has bool) {
	if access, ok := a.Server[server]; ok {
		has = access.HasLevel(level)
	}
	return
}

// HasServerFlags checks a user to see if their server level flags contain the
// given flags.
func (a *UserAccess) HasServerFlags(server string, flags ...string) (has bool) {
	if access, ok := a.Server[server]; ok {
		has = access.HasFlags(flags...)
	}
	return
}

// HasServerFlag checks a user to see if their server level flags contain the
// given flag.
func (a *UserAccess) HasServerFlag(server string, flag rune) (has bool) {
	if access, ok := a.Server[server]; ok {
		has = access.HasFlag(flag)
	}
	return
}

// GrantChannel sets both Level and Flags at the same time.
func (a *UserAccess) GrantChannel(server, channel string, level uint8,
	flags ...string) {

	a.ensureChannel(server, channel).SetAccess(level, flags...)
}

// GrantChannelFlags sets channel flags.
func (a *UserAccess) GrantChannelFlags(server, channel string,
	flags ...string) {

	a.ensureChannel(server, channel).SetFlags(flags...)
}

// GrantChannelLevel sets channel level.
func (a *UserAccess) GrantChannelLevel(server, channel string, level uint8) {
	a.ensureChannel(server, channel).Level = level
}

// RevokeChannel removes a user's channel access.
func (a *UserAccess) RevokeChannel(server, channel string) {
	if chans, ok := a.Channel[server]; ok {
		delete(chans, channel)
	}
}

// RevokeChannelLevel removes channel access.
func (a *UserAccess) RevokeChannelLevel(server, channel string) {
	if chans, ok := a.Channel[server]; ok {
		if access, ok := chans[channel]; ok {
			access.Level = 0
		}
	}
}

// RevokeChannelFlags removes flags from the channel level.
func (a *UserAccess) RevokeChannelFlags(server, channel string,
	flags ...string) {
	if chans, ok := a.Channel[server]; ok {
		if access, ok := chans[channel]; ok {
			access.ClearFlags(flags...)
		}
	}
}

// GetChannel gets the server access for the given channel.
func (a *UserAccess) GetChannel(server, channel string) (access *Access) {
	if chans, ok := a.Channel[server]; ok {
		access = chans[channel]
	}
	return
}

// HasChannelLevel checks a user to see if their channel level access is equal
// or above the specified access.
func (a *UserAccess) HasChannelLevel(server, channel string,
	level uint8) (has bool) {

	if chans, ok := a.Channel[server]; ok {
		if access, ok := chans[channel]; ok {
			has = access.HasLevel(level)
		}
	}
	return
}

// HasChannelFlags checks a user to see if their channel level flags contain the
// given flags.
func (a *UserAccess) HasChannelFlags(server, channel string,
	flags ...string) (has bool) {

	if chans, ok := a.Channel[server]; ok {
		if access, ok := chans[channel]; ok {
			has = access.HasFlags(flags...)
		}
	}
	return
}

// HasChannelFlag checks a user to see if their channel level flags contain the
// given flag.
func (a *UserAccess) HasChannelFlag(server, channel string,
	flag rune) (has bool) {

	if chans, ok := a.Channel[server]; ok {
		if access, ok := chans[channel]; ok {
			has = access.HasFlag(flag)
		}
	}
	return
}

// String turns UserAccess into a user consumable format.
func (a *UserAccess) String(server, channel string) (str string) {
	var wrote bool

	if a.Global != nil && (a.Global.Level > 0 || a.Global.Flags > 0) {
		str += "G(" + a.Global.String() + ")"
		wrote = true
	}

	if srv, ok := a.Server[server]; ok && (srv.Level > 0 || srv.Flags > 0) {
		if wrote {
			str += " "
		}
		str += "S(" + srv.String() + ")"
		wrote = true
	}

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
