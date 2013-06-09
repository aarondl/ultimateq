package data

// UserModes provides basic modes for channels and users.
type UserModes struct {
	modes int
	*UserModeKinds
}

// CreateUserModes creates a new usermodes using the metadata instance for
// reference information.
func CreateUserModes(u *UserModeKinds) *UserModes {
	return &UserModes{
		UserModeKinds: u,
	}
}

// SetMode sets the mode given.
func (u *UserModes) SetMode(mode rune) {
	u.modes |= u.GetModeBit(mode)
}

// HasMode checks if the user has the given mode.
func (u *UserModes) HasMode(mode rune) bool {
	bit := u.GetModeBit(mode)
	return bit != 0 && (bit == u.modes&bit)
}

// UnsetMode unsets the mode given.
func (u *UserModes) UnsetMode(mode rune) {
	u.modes &= ^u.GetModeBit(mode)
}
