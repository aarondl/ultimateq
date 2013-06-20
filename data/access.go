package data

const (
	ascA = 65
	ascZ = 90
	asca = 97
	ascz = 122
)

// Access defines an access level and flags a-zA-Z for a user.
type Access struct {
	Level uint8
	Flags uint64
}

// CreateAccess creates an access type with the permissions.
func CreateAccess(level uint8, flags ...string) *Access {
	a := &Access{}
	a.SetAccess(level, flags...)
	return a
}

// SetAccess sets all facets of access.
func (a *Access) SetAccess(level uint8, flags ...string) {
	a.Level = level
	a.SetFlags(flags...)
}

// SetFlags sets many flags at once.
func (a *Access) SetFlags(flags ...string) {
	for i := 0; i < len(flags); i++ {
		for _, f := range flags[i] {
			a.SetFlag(f)
		}
	}
}

// ClearFlags clears many flags at once.
func (a *Access) ClearFlags(flags ...string) {
	for i := 0; i < len(flags); i++ {
		for _, f := range flags[i] {
			a.ClearFlag(f)
		}
	}
}

// HasLevel checks to see that the level is >= the given level.
func (a *Access) HasLevel(level uint8) bool {
	return a.Level >= level
}

// HasFlags checks many flags at once.
func (a *Access) HasFlags(flags ...string) bool {
	for i := 0; i < len(flags); i++ {
		for _, f := range flags[i] {
			if !a.HasFlag(f) {
				return false
			}
		}
	}
	return true
}

// SetFlag sets the flag given.
func (a *Access) SetFlag(flag rune) {
	a.Flags |= getFlagBit(flag)
}

// HasFlag checks if the user has the given flag.
func (a *Access) HasFlag(flag rune) bool {
	bit := getFlagBit(flag)
	return bit != 0 && (bit == a.Flags&bit)
}

// ClearFlag clears the flag given.
func (a *Access) ClearFlag(flag rune) {
	a.Flags &= ^getFlagBit(flag)
}

// ClearAllFlags clears all flags.
func (a *Access) ClearAllFlags() {
	a.Flags = 0
}

// getFlagBit maps a-zA-Z to bits in a uint64
func getFlagBit(flag rune) (bit uint64) {
	asc := uint64(flag)
	if asc >= ascA && asc <= ascZ {
		asc -= ascA
		bit = 1 << asc
	} else if asc >= asca && asc <= ascz {
		asc -= ascA + (asca - ascZ - 1)
		bit = 1 << asc
	}
	return
}
