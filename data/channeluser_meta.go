package data

// ChannelUser represents a user that's on a channel.
type ChannelUser struct {
	User *User
	*UserModes
}

// NewChannelUser creates a channel user that represents a channel that
// contains a user.
func NewChannelUser(u *User, m *UserModes) *ChannelUser {
	return &ChannelUser{
		User:      u,
		UserModes: m,
	}
}

// UserChannel represents a user that's on a channel.
type UserChannel struct {
	Channel *Channel
	*UserModes
}

// NewUserChannel creates a user channel that represents a user that is
// on a channel.
func NewUserChannel(c *Channel, m *UserModes) *UserChannel {
	return &UserChannel{
		Channel:   c,
		UserModes: m,
	}
}
