package data

// ChannelUser represents a user that's on a channel.
type ChannelUser struct {
	User *User
	*UserModes
}

// CreateChannelUser creates a channel user that represents a channel that
// contains a user.
func CreateChannelUser(u *User, m *UserModes) *ChannelUser {
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

// CreateUserChannel creates a user channel that represents a user that is
// on a channel.
func CreateUserChannel(c *Channel, m *UserModes) *UserChannel {
	return &UserChannel{
		Channel:   c,
		UserModes: m,
	}
}
