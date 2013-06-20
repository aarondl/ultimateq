package data

import (
	"github.com/aarondl/ultimateq/irc"
)

// User encapsulates all the data associated with a user.
type User struct {
	mask irc.Mask
	name string
}

// CreateUser creates a user object from a nickname or fullhost.
func CreateUser(nickorhost string) *User {
	if len(nickorhost) == 0 {
		return nil
	}

	return &User{
		mask: irc.Mask(nickorhost),
	}
}

// GetNick returns the nick of this user.
func (u *User) GetNick() string {
	return u.mask.GetNick()
}

// GetUser returns the username of this user.
func (u *User) GetUsername() string {
	return u.mask.GetUsername()
}

// GetHost returns the host of this user.
func (u *User) GetHost() string {
	return u.mask.GetHost()
}

// GetFullhost returns the fullhost of this user.
func (u *User) GetFullhost() string {
	return u.mask.GetFullhost()
}

// Realname sets the real name of this user.
func (u *User) Realname(realname string) {
	u.name = realname
}

// GetRealname returns the real name of this user.
func (u *User) GetRealname() string {
	return u.name
}

// String returns a one-line representation of this user.
func (u *User) String() string {
	str := u.mask.GetNick()
	if fh := u.mask.GetFullhost(); len(fh) > 0 && str != fh {
		str += " " + fh
	}
	if len(u.name) > 0 {
		str += " " + u.name
	}

	return str
}
