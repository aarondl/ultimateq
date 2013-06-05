package data

// ChannelUser represents a user that's on a channel.
type ChannelUser struct {
	Channel *Channel
	User    *User
	*UserModes
	modes int
}

// CreateChannelUser creates a channel user that represents a user that is
// on a channel.
func CreateChannelUser(c *Channel, u *User, m *UserModes) *ChannelUser {
	return &ChannelUser{
		Channel:   c,
		User:      u,
		UserModes: m,
	}
}

// SetMode sets the mode given.
func (u *ChannelUser) SetMode(mode rune) {
	u.modes |= u.GetModeBit(mode)
}

// HasMode checks if the user has the given mode.
func (u *ChannelUser) HasMode(mode rune) bool {
	bit := u.GetModeBit(mode)
	return bit != 0 && (bit == u.modes&bit)
}

// ClearMode unsets the mode given.
func (u *ChannelUser) ClearMode(mode rune) {
	u.modes &= ^u.GetModeBit(mode)
}
