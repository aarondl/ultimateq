package data

import (
	"strings"
)

// Mask is a type that represents an irc hostmask. nickname!mask@hostname
type Mask string

// WildMask is a mask that contains wildcards.
// TODO: DO SOMETHING WITH THIS
type WildMask string

// GetNick returns the nick of this mask.
func (m Mask) GetNick() string {
	nick := string(m)
	index := strings.IndexAny(nick, "!@")
	if index >= 0 {
		return nick[:index]
	}
	return nick
}

// GetUser returns the maskname of this mask.
func (m Mask) GetUsername() string {
	_, mask, _ := m.SplitFullhost()
	return mask
}

// GetHost returns the host of this mask.
func (m Mask) GetHost() string {
	_, _, host := m.SplitFullhost()
	return host
}

// GetFullhost returns the fullhost of this mask.
func (m Mask) GetFullhost() string {
	return string(m)
}

// String returns a string representation of this mask.
func (m Mask) String() string {
	return string(m)
}

func (m Mask) SplitFullhost() (nick, user, host string) {
	fullhost := string(m)
	if len(fullhost) == 0 {
		return
	}

	userIndex := strings.IndexRune(fullhost, '!')
	hostIndex := strings.IndexRune(fullhost, '@')

	if userIndex <= 0 || hostIndex <= 0 || hostIndex < userIndex {
		min := len(fullhost)
		if userIndex < min && userIndex > 0 {
			min = userIndex
		}
		if hostIndex < min && hostIndex > 0 {
			min = hostIndex
		}
		nick = fullhost[:min]
		return
	}

	nick = fullhost[:userIndex]
	user = fullhost[userIndex+1 : hostIndex]
	host = fullhost[hostIndex+1:]
	return
}
