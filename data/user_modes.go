package data

// UserModes provides basic modes for users.
type UserModes struct {
	Modes     byte       `json:"modes"`
	ModeKinds *modeKinds `json:"mode_kinds"`
}

// NewUserModes creates a new usermodes using the metadata instance for
// reference information.
func NewUserModes(m *modeKinds) UserModes {
	return UserModes{
		ModeKinds: m,
	}
}

// SetMode sets the mode given.
func (u *UserModes) SetMode(mode rune) {
	u.Modes |= u.ModeKinds.modeBit(mode)
}

// HasMode checks if the user has the given mode.
func (u *UserModes) HasMode(mode rune) bool {
	bit := u.ModeKinds.modeBit(mode)
	return bit != 0 && (bit == u.Modes&bit)
}

// UnsetMode unsets the mode given.
func (u *UserModes) UnsetMode(mode rune) {
	u.Modes &= ^u.ModeKinds.modeBit(mode)
}

// String turns user modes into a string.
func (u *UserModes) String() string {
	ret := ""
	for i := 0; i < len(u.ModeKinds.userPrefixes); i++ {
		if u.HasMode(u.ModeKinds.userPrefixes[i][0]) {
			ret += string(u.ModeKinds.userPrefixes[i][0])
		}
	}
	return ret
}

// StringSymbols turns user modes into a string but uses mode chars instead.
func (u *UserModes) StringSymbols() string {
	ret := ""
	for i := 0; i < len(u.ModeKinds.userPrefixes); i++ {
		if u.HasMode(u.ModeKinds.userPrefixes[i][0]) {
			ret += string(u.ModeKinds.userPrefixes[i][1])
		}
	}
	return ret
}
