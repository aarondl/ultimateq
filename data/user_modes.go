package data

import (
	"errors"
	"fmt"
	"strings"
)

const (
	fmtErrCouldNotParsePrefix = "data: Could not parse prefix (%v)"
)

// UserModes maps modes applied to a user via a channel to a mode character or
// display symobol.
type UserModes struct {
	modes [][2]rune
}

// CreateUserModes creates an object that can be used to get/set user
// channel modes on a user. Prefix should be in IRC PREFIX style string. Of the
// form (ov)@+ where the letters map to symbols
func CreateUserModes(prefix string) (*UserModes, error) {
	if modes, err := parsePrefixString(prefix); err != nil {
		return nil, err
	} else {
		return &UserModes{
			modes: modes,
		}, nil
	}
}

// UpdateModes updates the internal lookup table. This will invalidate all the
// modes that were set previously so they should all be wiped out as well.
func (u *UserModes) UpdateModes(prefix string) error {
	if update, err := parsePrefixString(prefix); err != nil {
		return err
	} else {
		u.modes = update
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
func (u *UserModes) GetSymbol(mode rune) rune {
	for i := 0; i < len(u.modes); i++ {
		if u.modes[i][0] == mode {
			return u.modes[i][1]
		}
	}
	return 0
}

// GetMode returns the mode character of the symbol given.
func (u *UserModes) GetMode(symbol rune) rune {
	for i := 0; i < len(u.modes); i++ {
		if u.modes[i][1] == symbol {
			return u.modes[i][0]
		}
	}
	return 0
}

// GetModeBit returns the bit of the mode character to set.
func (u *UserModes) GetModeBit(mode rune) int {
	for i := uint(0); i < uint(len(u.modes)); i++ {
		if u.modes[i][0] == mode {
			return 1 << i
		}
	}
	return 0
}
