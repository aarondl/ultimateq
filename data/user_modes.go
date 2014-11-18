package data

// UserModes provides basic modes for users.
type UserModes struct {
	modes byte
	*modeKinds
}

// NewUserModes creates a new usermodes using the metadata instance for
// reference information.
func NewUserModes(m *modeKinds) UserModes {
	return UserModes{
		modeKinds: m,
	}
}

// SetMode sets the mode given.
func (u *UserModes) SetMode(mode rune) {
	u.modes |= u.modeBit(mode)
}

// HasMode checks if the user has the given mode.
func (u *UserModes) HasMode(mode rune) bool {
	bit := u.modeBit(mode)
	return bit != 0 && (bit == u.modes&bit)
}

// UnsetMode unsets the mode given.
func (u *UserModes) UnsetMode(mode rune) {
	u.modes &= ^u.modeBit(mode)
}

// String turns user modes into a string.
func (u *UserModes) String() string {
	ret := ""
	for i := 0; i < len(u.userPrefixes); i++ {
		if u.HasMode(u.userPrefixes[i][0]) {
			ret += string(u.userPrefixes[i][0])
		}
	}
	return ret
}

// StringSymbols turns user modes into a string but uses mode chars instead.
func (u *UserModes) StringSymbols() string {
	ret := ""
	for i := 0; i < len(u.userPrefixes); i++ {
		if u.HasMode(u.userPrefixes[i][0]) {
			ret += string(u.userPrefixes[i][1])
		}
	}
	return ret
}
