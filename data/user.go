package data

import (
	"github.com/aarondl/ultimateq/api"
	"github.com/aarondl/ultimateq/irc"
)

// User encapsulates all the data associated with a user.
type User struct {
	irc.Host `json:"host"`
	Realname string `json:"realname"`
}

// NewUser creates a user object from a nickname or fullhost.
func NewUser(nickorhost string) *User {
	if len(nickorhost) == 0 {
		return nil
	}

	return &User{
		Host: irc.Host(nickorhost),
	}
}

// String returns a one-line representation of this user.
func (u *User) String() string {
	str := u.Host.Nick()
	if fh := u.Host.String(); len(fh) > 0 && str != fh {
		str += " " + fh
	}
	if len(u.Realname) > 0 {
		str += " " + u.Realname
	}

	return str
}

// ToProto converts stateuser to a protocol buffer
func (u *User) ToProto() *api.StateUser {
	user := new(api.StateUser)
	user.Host = string(u.Host)
	user.Realname = u.Realname

	return user
}
