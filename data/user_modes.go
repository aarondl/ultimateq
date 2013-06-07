package data

import (
	"errors"
	"fmt"
	"strings"
)

const (
	fmtErrCouldNotParsePrefix = "data: Could not parse prefix (%v)"
)

// UserModes provides basic modes for channels and users.
type UserModes struct {
	modes int
	*UserModesMeta
}

// CreateUserModes creates a new usermodes using the metadata instance for
// reference information.
func CreateUserModes(u *UserModesMeta) *UserModes {
	return &UserModes{
		UserModesMeta: u,
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

// UserModesMeta maps modes applied to a user via a channel to a mode character
// or display symobol.
type UserModesMeta struct {
	modeInfo [][2]rune
}

// CreateUserModesMeta creates an object that can be used to get/set user
// channel modes on a user. Prefix should be in IRC PREFIX style string. Of the
// form (ov)@+ where the letters map to symbols
func CreateUserModesMeta(prefix string) (*UserModesMeta, error) {
	if modes, err := parsePrefixString(prefix); err != nil {
		return nil, err
	} else {
		return &UserModesMeta{
			modeInfo: modes,
		}, nil
	}
}

// UpdateModes updates the internal lookup table. This will invalidate all the
// modes that were set previously so they should all be wiped out as well.
func (u *UserModesMeta) UpdateModes(prefix string) error {
	if update, err := parsePrefixString(prefix); err != nil {
		return err
	} else {
		u.modeInfo = update
	}
	return nil
}

// parsePrefixString parses a prefix string into an slice of arrays depicting
// the mapping from symbol to char, as well as providing an index/bit to set
// and unset.
func parsePrefixString(prefix string) ([][2]rune, error) {
	if len(prefix) == 0 || prefix[0] != '(' {
		return nil, errors.New(fmt.Sprintf(fmtErrCouldNotParsePrefix, prefix))
	}

	split := strings.IndexRune(prefix, ')')
	if split < 0 {
		return nil, errors.New(fmt.Sprintf(fmtErrCouldNotParsePrefix, prefix))
	}

	modes := make([][2]rune, split-1)

	for i := 1; i < split; i++ {
		modes[i-1][0], modes[i-1][1] =
			rune(prefix[i]), rune(prefix[split+i])
	}

	return modes, nil
}

// GetSymbol returns the symbol character of the mode given.
func (u *UserModesMeta) GetSymbol(mode rune) rune {
	for i := 0; i < len(u.modeInfo); i++ {
		if u.modeInfo[i][0] == mode {
			return u.modeInfo[i][1]
		}
	}
	return 0
}

// GetMode returns the mode character of the symbol given.
func (u *UserModesMeta) GetMode(symbol rune) rune {
	for i := 0; i < len(u.modeInfo); i++ {
		if u.modeInfo[i][1] == symbol {
			return u.modeInfo[i][0]
		}
	}
	return 0
}

// GetModeBit returns the bit of the mode character to set.
func (u *UserModesMeta) GetModeBit(mode rune) int {
	for i := uint(0); i < uint(len(u.modeInfo)); i++ {
		if u.modeInfo[i][0] == mode {
			return 1 << i
		}
	}
	return 0
}
