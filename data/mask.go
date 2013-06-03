package data

import (
	"strings"
)

// Mask is a type that represents an irc hostmask. nickname!mask@hostname
type Mask string

// WildMask is an irc hostmask that contains wildcard characters ? and *
type WildMask string

func (w WildMask) Match(m Mask) bool {
	ws, ms := string(w), string(m)
	wl, ml := len(ws), len(ms)

	if wl == 0 {
		return ml == 0
	}

	var i, j, consume = 0, 0, 0
	for i < wl && j < ml {

		switch ws[i] {
		case '?', '*':
			star := false
			consume = 0

			for i < wl && (ws[i] == '*' || ws[i] == '?') {
				star = star || ws[i] == '*'
				i++
				consume++
			}

			if star {
				consume = -1
			}
		case ms[j]:
			consume = 0
			i++
			j++
		default:
			if consume != 0 {
				consume--
				j++
			} else {
				return false
			}
		}
	}

	for i < wl && (ws[i] == '?' || ws[i] == '*') {
		i++
	}

	if consume < 0 {
		consume = ml - j
	}
	j += consume

	if i < wl || j < ml {
		return false
	}

	return true
}

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
