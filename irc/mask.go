package irc

import (
	"regexp"
	"strings"
)

var (
	// rgxMask validates and splits masks.
	rgxMask = regexp.MustCompile(
		`(?i)^` +
			`([\w\x5B-\x60][\w\d\x5B-\x60]*)` + // nickname
			`!([^\0@\s]+)` + // username
			`@([^\0\s]+)` + // host
			`$`,
	)

	// rgxWildMask validates and splits wildmasks.
	rgxWildMask = regexp.MustCompile(
		`(?i)^` +
			`([\w\x5B-\x60\?\*][\w\d\x5B-\x60\?\*]*)` + // nickname
			`!([^\0@\s]+)` + // username
			`@([^\0\s]+)` + // host
			`$`,
	)
)

// Mask is a type that represents an irc hostmask. nickname!mask@hostname
type Mask string

// WildMask is an irc hostmask that contains wildcard characters ? and *
type WildMask string

// Match checks if the WildMask satisfies the given normal mask.
func (w WildMask) Match(m Mask) bool {
	return isMatch(string(m), string(w))
}

// IsValid checks to ensure the mask is in valid format.
func (m WildMask) IsValid() bool {
	return rgxWildMask.MatchString(string(m))
}

// Split splits a wildmask into it's fragments: nick, user, and host. If the
// format is not acceptable empty string is returned for everything.
func (w WildMask) Split() (nick, user, host string) {
	fragments := rgxWildMask.FindStringSubmatch(string(w))
	if len(fragments) == 0 {
		return
	}
	return fragments[1], fragments[2], fragments[3]
}

// Match checks if a given wildmask is satisfied by the mask.
func (m Mask) Match(w WildMask) bool {
	return isMatch(string(m), string(w))
}

// isMatch is a matching function for a string, and a string with the wildcards
// * and ? in it.
func isMatch(ms, ws string) bool {
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

// GetUsername returns the maskname of this mask.
func (m Mask) GetUsername() string {
	_, mask, _ := m.Split()
	return mask
}

// GetHost returns the host of this mask.
func (m Mask) GetHost() string {
	_, _, host := m.Split()
	return host
}

// GetFullhost returns the fullhost of this mask.
func (m Mask) GetFullhost() string {
	return string(m)
}

// IsValid checks to ensure the mask is in valid format.
func (m Mask) IsValid() bool {
	return rgxMask.MatchString(string(m))
}

// Split splits a mask into it's fragments: nick, user, and host. If the
// format is not acceptable empty string is returned for everything.
func (m Mask) Split() (nick, user, host string) {
	fragments := rgxMask.FindStringSubmatch(string(m))
	if len(fragments) == 0 {
		return
	}
	return fragments[1], fragments[2], fragments[3]
}
