package data

import (
	"errors"
	"strings"
)

// The various kinds of mode-argument behavior during parsing.
const (
	ARGS_NONE    = 0x0
	ARGS_ALWAYS  = 0x1
	ARGS_ONSET   = 0x2
	ARGS_ADDRESS = 0x3
)

var (
	// csvParseError is thrown when a csv string is not in the correct format.
	csvParseError = errors.New("data: Could not parse csv string")
)

// ModeKinds contains mode type information, ModeDiff and Modeset require this
// information to parse correctly.
type ModeKinds struct {
	kinds map[rune]int
}

// CreateModeKindsCSV creates ModeKinds from an IRC CHANMODES csv string. The
// format of which is ARGS_ADDRESS,ARGS_ALWAYS,ARGS_ONSET,ARGS_NONE
func CreateModeKindsCSV(kindstr string) (*ModeKinds, error) {
	if kinds, err := parseModeKindsCSV(kindstr); err != nil {
		return nil, err
	} else {
		return &ModeKinds{kinds}, nil
	}
}

// CreateModeKinds creates a mode kinds structure taking in a string, one for
// each kind of mode.
func CreateModeKinds(address, always, onset string) *ModeKinds {
	return &ModeKinds{
		parseModeKinds(address, always, onset),
	}
}

// parseModeKinds creates a map[rune]int from a set of strings, one for each
// kind of mode present.
func parseModeKinds(address, always, onset string) (kinds map[rune]int) {
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

	return
}

// parseModeKindsCSV creates a map[rune]int from an IRC CHANMODES csv string.
// The format of which is ARGS_ADDRESS,ARGS_ALWAYS,ARGS_ONSET,ARGS_NONE
func parseModeKindsCSV(kindstr string) (map[rune]int, error) {
	if len(kindstr) == 0 {
		return nil, csvParseError
	}

	kindSplits := strings.Split(kindstr, ",")
	if len(kindSplits) != 4 {
		return nil, csvParseError
	}

	return parseModeKinds(kindSplits[0], kindSplits[1], kindSplits[2]), nil
}

// Update updates the internal lookup table. This will invalidate all the
// modes that were set previously using this ModeKinds so they should be reset.
func (m *ModeKinds) Update(address, always, onset string) {
	m.kinds = parseModeKinds(address, always, onset)
}

// UpdateCSV updates the internal lookup table. This will invalidate all the
// modes that were set previously using this ModeKinds so they should be reset.
func (m *ModeKinds) UpdateCSV(kindstr string) error {
	var err error
	var kinds map[rune]int
	if kinds, err = parseModeKindsCSV(kindstr); err != nil {
		return err
	} else {
		m.kinds = kinds
	}
	return nil
}

// getKind gets the kind of mode and returns it.
func (m *ModeKinds) getKind(mode rune) int {
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
}

// apply parses a complex modestring and applies it to a moder interface.
func apply(m moder, modestring string) {
	adding := true
	used := 0

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
		case ARGS_NONE:
			if adding {
				m.setMode(mode)
			} else {
				m.unsetMode(mode)
			}
		}
	}
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
