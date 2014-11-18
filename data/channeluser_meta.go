package data

// channelUser represents a user that's on a channel.
type channelUser struct {
	User *User
	*UserModes
}

// newChannelUser creates a channel user that represents a channel that
// contains a user.
func newChannelUser(u *User, m *UserModes) channelUser {
	return channelUser{
		User:      u,
		UserModes: m,
	}
}

// userChannel represents a user that's on a channel.
type userChannel struct {
	Channel *Channel
	*UserModes
}

// newUserChannel creates a user channel that represents a user that is
// on a channel.
func newUserChannel(c *Channel, m *UserModes) userChannel {
	return userChannel{
		Channel:   c,
		UserModes: m,
	}
}
