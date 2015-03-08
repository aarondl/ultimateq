package data

import (
	"strconv"
)

const (
	ascA               = 65
	ascZ               = 90
	asca               = 97
	ascz               = 122
	nAlphabet          = 26
	none               = "none"
	allFlags           = `-ALL-`
	allFlagsNum uint64 = 0xFFFFFFFFFFFFF

	wholeAlphabet = `ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz`
)

// Access defines an access level and flags a-zA-Z for a user.
type Access struct {
	Level uint8
	Flags uint64
}

// NewAccess creates an access type with the permissions.
func NewAccess(level uint8, flags ...string) *Access {
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
	a.Flags |= getFlagBits(flags...)
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

// HasFlags checks many flags at once. Flags are or'd together.
func (a *Access) HasFlags(flags ...string) bool {
	for i := 0; i < len(flags); i++ {
		for _, f := range flags[i] {
			if a.HasFlag(f) {
				return true
			}
		}
	}
	return false
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

// IsZero checks if this instance of access has no flags and no level.
func (a *Access) IsZero() bool {
	return a.Flags == 0 && a.Level == 0
}

// String transforms the Access into a human-readable format.
func (a Access) String() (str string) {
	hasLevel := a.Level != 0
	hasFlags := a.Flags != 0
	if !hasLevel && !hasFlags {
		return none
	}
	if hasLevel {
		str += strconv.Itoa(int(a.Level))
	}
	if hasFlags {
		if hasLevel {
			str += " "
		}
		str += getFlagString(a.Flags)
	}

	return
}

// getFlagBits creates a mask containing all the modes.
func getFlagBits(flags ...string) (bits uint64) {
	for i := 0; i < len(flags); i++ {
		for _, f := range flags[i] {
			bits |= getFlagBit(f)
		}
	}
	return
}

// getFlagBit maps A-Za-z to bits in a uint64
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

// getFlagString maps the bits in a uint64 to A-Za-z
func getFlagString(bits uint64) (flags string) {
	var bit uint64 = 1
	var n = nAlphabet * 2

	if (bits & allFlagsNum) == allFlagsNum {
		return allFlags
	}

	for i := 0; i < n; i, bit = i+1, bit<<1 {
		if bit&bits != bit {
			continue
		}

		if i < nAlphabet {
			flags += string(i + ascA)
		} else {
			flags += string(i - nAlphabet + asca)
		}
	}
	return
}
