package data

import (
	"github.com/aarondl/ultimateq/irc"
)

// User encapsulates all the data associated with a user.
type User struct {
	host irc.Host
	name string
}

// NewUser creates a user object from a nickname or fullhost.
func NewUser(nickorhost string) *User {
	if len(nickorhost) == 0 {
		return nil
	}

	return &User{
		host: irc.Host(nickorhost),
	}
}

// Nick returns the nick of this user.
func (u User) Nick() string {
	return u.host.Nick()
}

// Username returns the username of this user.
func (u User) Username() string {
	return u.host.Username()
}

// Hostname returns the hostname of this user.
func (u User) Hostname() string {
	return u.host.Hostname()
}

// Host returns the string version of the user's host.
func (u User) Host() string {
	return u.host.String()
}

// SetRealname sets the real name of this user.
func (u *User) SetRealname(realname string) {
	u.name = realname
}

// Realname returns the real name of this user.
func (u *User) Realname() string {
	return u.name
}

// String returns a one-line representation of this user.
func (u *User) String() string {
	str := u.host.Nick()
	if fh := u.host.String(); len(fh) > 0 && str != fh {
		str += " " + fh
	}
	if len(u.name) > 0 {
		str += " " + u.name
	}

	return str
}
