package irc

import (
	"regexp"
	"strings"
)

var (
	// rgxHost validates and splits hosts.
	rgxHost = regexp.MustCompile(
		`(?i)^` +
			`([\w\x5B-\x60][\w\d\x5B-\x60]*)` + // nickname
			`!([^\0@\s]+)` + // username
			`@([^\0\s]+)` + // host
			`$`,
	)

	// rgxMask validates and splits masks.
	rgxMask = regexp.MustCompile(
		`(?i)^` +
			`([\w\x5B-\x60\?\*][\w\d\x5B-\x60\?\*]*)` + // nickname
			`!([^\0@\s]+)` + // username
			`@([^\0\s]+)` + // host
			`$`,
	)
)

// Host is a type that represents an irc hostname. nickname!username@hostname
type Host string

// Nick returns the nick of the host.
func (h Host) Nick() string {
	return Nick(string(h))
}

// Username returns the username of the host.
func (h Host) Username() string {
	return Username(string(h))
}

// Hostname returns the host of the host.
func (h Host) Hostname() string {
	return Hostname(string(h))
}

// Split splits a host into it's fragments: nick, user, and hostname. If the
// format is not acceptable empty string is returned for everything.
func (h Host) Split() (nick, user, hostname string) {
	return Split(string(h))
}

// String returns the fullhost of this host.
func (h Host) String() string {
	return string(h)
}

// IsValid checks to ensure the host is in valid format.
func (h Host) IsValid() bool {
	return rgxHost.MatchString(string(h))
}

// Mask is an irc hostmask that contains wildcard characters ? and *
type Mask string

// Match checks if the mask satisfies the given host.
func (m Mask) Match(h Host) bool {
	return isMatch(string(h), string(m))
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

// Match checks if a given mask is satisfied by the host.
func (h Host) Match(m Mask) bool {
	return isMatch(string(h), string(m))
}

// isMatch is a matching function for a string, and a string with the wildcards
// * and ? in it.
func isMatch(hs, ms string) bool {
	ml, hl := len(ms), len(hs)

	if ml == 0 {
		return hl == 0
	}

	var i, j, consume = 0, 0, 0
	for i < ml && j < hl {

		switch ms[i] {
		case '?', '*':
			star := false
			consume = 0

			for i < ml && (ms[i] == '*' || ms[i] == '?') {
				star = star || ms[i] == '*'
				i++
				consume++
			}

			if star {
				consume = -1
			}
		case hs[j]:
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

	for i < ml && (ms[i] == '?' || ms[i] == '*') {
		i++
	}

	if consume < 0 {
		consume = hl - j
	}
	j += consume

	if i < ml || j < hl {
		return false
	}

	return true
}

// Nick returns the nick of the host.
func Nick(host string) string {
	index := strings.IndexAny(host, "!@")
	if index >= 0 {
		return host[:index]
	}
	return host
}

// Username returns the username of the host.
func Username(host string) string {
	_, user, _ := Split(host)
	return user
}

// Hostname returns the host of the host.
func Hostname(host string) string {
	_, _, hostname := Split(host)
	return hostname
}

// Split splits a host into it's fragments: nick, user, and hostname. If the
// format is not acceptable empty string is returned for everything.
func Split(host string) (nick, user, hostname string) {
	fragments := rgxHost.FindStringSubmatch(string(host))
	if len(fragments) == 0 {
		return
	}
	return fragments[1], fragments[2], fragments[3]
}
