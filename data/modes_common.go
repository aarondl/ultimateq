package data

import (
	"fmt"
	"strings"
)

// The various kinds of mode-argument behavior during parsing.
const (
	ARGS_NONE    = 0x1
	ARGS_ALWAYS  = 0x2
	ARGS_ONSET   = 0x3
	ARGS_ADDRESS = 0x4
)

const (
	// fmtErrCsvParse is thrown when a csv string is not in the correct format.
	fmtErrCsvParse = "data: Could not parse csv string (%v)"
	// fmtErrCouldNotParsePrefix is when the prefix string from 005 raw is not
	// in the correct format.
	fmtErrCouldNotParsePrefix = "data: Could not parse prefix (%v)"
	// errMsgMoreThanEight happens when there is more than 8 in a prefix.
	errMsgMoreThanEight = "data: UserModeKinds supports maximum 8 modes (%v)"
	// BITS_IN_BYTE is to avoid pulling in unsafe and magic numbers.
	BITS_IN_BYTE = 8
)

// UserMode is returned by apply helper when it encounters a user mode
type UserMode struct {
	Mode rune
	Arg  string
}

// UserModeKinds maps modes applied to a user via a channel to a mode character
// or display symobol.
type UserModeKinds struct {
	modeInfo [][2]rune
}

// CreateUserModeKinds creates an object that can be used to get/set user
// channel modes on a user. Prefix should be in IRC PREFIX style string. Of the
// form (ov)@+ where the letters map to symbols
func CreateUserModeKinds(prefix string) (*UserModeKinds, error) {
	modes, err := parsePrefixString(prefix)
	if err != nil {
		return nil, err
	}

	return &UserModeKinds{modeInfo: modes}, nil
}

// UpdateModes updates the internal lookup table. This will invalidate all the
// modes that were set previously so they should all be wiped out as well.
func (u *UserModeKinds) UpdateModes(prefix string) error {
	update, err := parsePrefixString(prefix)
	if err != nil {
		return err
	}
	u.modeInfo = update
	return nil
}

// parsePrefixString parses a prefix string into an slice of arrays depicting
// the mapping from symbol to char, as well as providing an index/bit to set
// and unset.
func parsePrefixString(prefix string) ([][2]rune, error) {
	if len(prefix) == 0 || prefix[0] != '(' {
		return nil, fmt.Errorf(fmtErrCouldNotParsePrefix, prefix)
	}

	split := strings.IndexRune(prefix, ')')
	if split < 0 {
		return nil, fmt.Errorf(fmtErrCouldNotParsePrefix, prefix)
	}

	if split-1 > BITS_IN_BYTE {
		return nil, fmt.Errorf(errMsgMoreThanEight, prefix)
	}

	modes := make([][2]rune, split-1)

	for i := 1; i < split; i++ {
		modes[i-1][0], modes[i-1][1] =
			rune(prefix[i]), rune(prefix[split+i])
	}

	return modes, nil
}

// GetSymbol returns the symbol character of the mode given.
func (u *UserModeKinds) GetSymbol(mode rune) rune {
	for i := 0; i < len(u.modeInfo); i++ {
		if u.modeInfo[i][0] == mode {
			return u.modeInfo[i][1]
		}
	}
	return 0
}

// GetMode returns the mode character of the symbol given.
func (u *UserModeKinds) GetMode(symbol rune) rune {
	for i := 0; i < len(u.modeInfo); i++ {
		if u.modeInfo[i][1] == symbol {
			return u.modeInfo[i][0]
		}
	}
	return 0
}

// GetModeBit returns the bit of the mode character to set.
func (u *UserModeKinds) GetModeBit(mode rune) byte {
	for i := uint(0); i < uint(len(u.modeInfo)); i++ {
		if u.modeInfo[i][0] == mode {
			return 1 << i
		}
	}
	return 0
}

// ChannelModeKinds contains mode type information, ModeDiff and Modeset
// require this information to parse correctly.
type ChannelModeKinds struct {
	kinds map[rune]int
}

// CreateChannelModeKindsCSV creates ChannelModeKinds from an IRC CHANMODES csv
// string. The format of which is ARGS_ADDRESS,ARGS_ALWAYS,ARGS_ONSET,ARGS_NONE
func CreateChannelModeKindsCSV(kindstr string) (*ChannelModeKinds, error) {
	kinds, err := parseChannelModeKindsCSV(kindstr)
	if err != nil {
		return nil, err
	}
	return &ChannelModeKinds{kinds}, nil
}

// CreateChannelModeKinds creates a mode kinds structure taking in a string,
// one for each kind of mode.
func CreateChannelModeKinds(
	address, always, onset, none string) *ChannelModeKinds {

	return &ChannelModeKinds{
		parseChannelModeKinds(address, always, onset, none),
	}
}

// parseChannelModeKinds creates a map[rune]int from a set of strings, one for
// each kind of mode present.
func parseChannelModeKinds(address, always, onset, none string) (
	kinds map[rune]int) {
	size := len(always) + len(onset) + len(address)
	kinds = make(map[rune]int, size)

	for _, mode := range always {
		kinds[mode] = ARGS_ALWAYS
	}
	for _, mode := range onset {
		kinds[mode] = ARGS_ONSET
	}
	for _, mode := range address {
		kinds[mode] = ARGS_ADDRESS
	}
	for _, mode := range none {
		kinds[mode] = ARGS_NONE
	}

	return
}

// parseChannelModeKindsCSV creates a map[rune]int from an IRC CHANMODES csv
// string. The format of which is ARGS_ADDRESS,ARGS_ALWAYS,ARGS_ONSET,ARGS_NONE
func parseChannelModeKindsCSV(kindstr string) (map[rune]int, error) {
	if len(kindstr) == 0 {
		return nil, fmt.Errorf(fmtErrCsvParse, kindstr)
	}

	kindSplits := strings.Split(kindstr, ",")
	if len(kindSplits) != 4 {
		return nil, fmt.Errorf(fmtErrCsvParse, kindstr)
	}

	return parseChannelModeKinds(
			kindSplits[0], kindSplits[1], kindSplits[2], kindSplits[3]),
		nil
}

// Update updates the internal lookup table. This will invalidate all the
// modes that were set previously using this ChannelModeKinds so they should be
// reset.
func (m *ChannelModeKinds) Update(address, always, onset, none string) {
	m.kinds = parseChannelModeKinds(address, always, onset, none)
}

// UpdateCSV updates the internal lookup table. This will invalidate all the
// modes that were set previously using this ChannelModeKinds so they should be
// reset.
func (m *ChannelModeKinds) UpdateCSV(kindstr string) (err error) {
	var kinds map[rune]int
	kinds, err = parseChannelModeKindsCSV(kindstr)
	if err != nil {
		return
	}
	m.kinds = kinds
	return
}

// getKind gets the kind of mode and returns it.
func (m *ChannelModeKinds) getKind(mode rune) int {
	return m.kinds[mode]
}

// moder is an interface that defines common behavior between all mode managing
// kinds.
type moder interface {
	setMode(rune)
	setArg(rune, string)
	setAddress(rune, string)
	unsetMode(rune)
	unsetArg(rune, string)
	unsetAddress(rune, string)
	getKind(rune) int
	isUserMode(rune) bool
}

// apply parses a complex modestring and applies it to a moder interface. All
// non-recognized modes are given back a positive and negative array of
// UnknownMode.
func apply(m moder, modestring string) (pos, neg []UserMode) {
	adding := true
	used := 0
	pos = make([]UserMode, 0)
	neg = make([]UserMode, 0)

	splits := strings.Split(strings.TrimSpace(modestring), " ")
	args := splits[1:]

	for _, mode := range splits[0] {
		if add, sub := mode == '+', mode == '-'; add || sub {
			adding = add
			continue
		}

		kind := m.getKind(mode)
		switch kind {
		case ARGS_ALWAYS, ARGS_ONSET:
			if adding {
				if used >= len(args) {
					break
				}
				m.setArg(mode, args[used])
				used++
			} else {
				arg := ""
				if kind == ARGS_ALWAYS {
					if used >= len(args) {
						break
					}
					arg = args[used]
					used++
				}
				m.unsetArg(mode, arg)
			}
		case ARGS_ADDRESS:
			if used >= len(args) {
				break
			}
			if adding {
				m.setAddress(mode, args[used])
			} else {
				m.unsetAddress(mode, args[used])
			}
			used++
		default:
			if isUserMode := m.isUserMode(mode); isUserMode {
				if used >= len(args) {
					break
				}
				if adding {
					pos = append(pos, UserMode{mode, args[used]})
				} else {
					neg = append(neg, UserMode{mode, args[used]})
				}
				used++
			} else {
				if adding {
					m.setMode(mode)
				} else {
					m.unsetMode(mode)
				}
			}
		}
	}

	return
}

// parseSimpleModestrings morphs many simple mode strings into a single modes
// and args pair. Where the N arguments belong to the last N modes in the
// arrays.
func parseSimpleModestrings(modestrs ...string) (modes []rune, args []string) {
	modes = make([]rune, 0, len(modestrs))
	args = make([]string, 0, len(modestrs))

	for i := 0; i < len(modestrs); i++ {
		modestr := strings.TrimSpace(modestrs[i])
		if len(modestr) == 0 {
			continue
		}

		if strings.Contains(modestr, " ") {
			splits := strings.Split(modestr, " ")
			modes = append(modes, []rune(splits[0])...)
			args = append(args, splits[1:]...)
		} else {
			modes = append(modes, []rune(modestr)...)
			swap := len(modes) - 1
			for j := 0; j < len(args); j++ {
				modes[swap-j], modes[swap-j-1] = modes[swap-j-1], modes[swap-j]
			}
		}
	}

	return
}
