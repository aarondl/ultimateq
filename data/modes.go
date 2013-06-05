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
func CreateModeKindsCSV(kinds string) (*ModeKinds, error) {
	if len(kinds) == 0 {
		return nil, csvParseError
	}

	kindSplits := strings.Split(kinds, ",")
	if len(kindSplits) != 4 {
		return nil, csvParseError
	}

	return CreateModeKinds(kindSplits[0], kindSplits[1], kindSplits[2]), nil
}

// CreateModeKinds creates a mode kinds structure taking in a string, one for
// each kind of mode.
func CreateModeKinds(address, always, onset string) *ModeKinds {
	size := len(always) + len(onset) + len(address)
	m := &ModeKinds{make(map[rune]int, size)}

	for _, mode := range always {
		m.kinds[mode] = ARGS_ALWAYS
	}
	for _, mode := range onset {
		m.kinds[mode] = ARGS_ONSET
	}
	for _, mode := range address {
		m.kinds[mode] = ARGS_ADDRESS
	}

	return m
}

// getKind gets the kind of mode and returns it.
func (m *ModeKinds) getKind(mode rune) int {
	return m.kinds[mode]
}

// ModeDiff encapsulates a difference of modes, a combination of both positive
// change modes, and negative change modes.
type ModeDiff struct {
	*ModeKinds
	pos *Modeset
	neg *Modeset
}

// CreateModeDiff creates an empty ModeDiff.
func CreateModeDiff(kinds *ModeKinds) *ModeDiff {
	return &ModeDiff{
		ModeKinds: kinds,
		pos:       CreateModeset(kinds),
		neg:       CreateModeset(kinds),
	}
}

// Checks if applying this diff will set the given simple modestrs.
func (d *ModeDiff) IsSet(modestrs ...string) bool {
	return d.pos.IsSet(modestrs...)
}

// Checks if applying this diff will unset the given simple modestrs.
func (d *ModeDiff) IsUnset(modestrs ...string) bool {
	return d.neg.IsSet(modestrs...)
}

// Apply takes a complex modestring and transforms it into a diff.
func (d *ModeDiff) Apply(modestring string) {
	apply(d, modestring)
}

// String turns a ModeDiff into a complex string representation.
func (d *ModeDiff) String() string {
	modes := ""
	args := ""
	pos, neg := d.pos.String(), d.neg.String()
	if len(pos) > 0 {
		pspace := strings.Index(pos, " ")
		if pspace < 0 {
			pspace = len(pos)
		} else {
			args += " " + pos[pspace+1:]
		}
		modes += "+" + pos[:pspace]
	}
	if len(neg) > 0 {
		nspace := strings.Index(neg, " ")
		if nspace < 0 {
			nspace = len(neg)
		} else {
			args += " " + neg[nspace+1:]
		}
		modes += "-" + neg[:nspace]
	}

	return modes + args
}

// setMode adds this mode to the positive modes and removes it from the
// negative modes.
func (d *ModeDiff) setMode(mode rune) {
	d.pos.setMode(mode)
	d.neg.unsetMode(mode)
}

// unsetMode adds this mode to the negative modes and removes it from the
// positive modes.
func (d *ModeDiff) unsetMode(mode rune) {
	d.pos.unsetMode(mode)
	d.neg.setMode(mode)
}

// setArg adds this mode + argument to the positive modes and removes it
// from the negative modes.
func (d *ModeDiff) setArg(mode rune, arg string) {
	d.pos.setArg(mode, arg)
	d.neg.unsetArg(mode, arg)
}

// unsetArg adds this mode + argument to the negative modes and removes it
// from the positive modes.
func (d *ModeDiff) unsetArg(mode rune, arg string) {
	d.pos.unsetArg(mode, arg)
	d.neg.setArg(mode, arg)
}

// setAddress adds this mode + argument to the positive modes and removes it
// from the negative modes.
func (d *ModeDiff) setAddress(mode rune, address string) {
	d.pos.setAddress(mode, address)
	d.neg.unsetAddress(mode, address)
}

// unsetAddress adds this mode + argument to the negative modes and removes it
// from the positive modes.
func (d *ModeDiff) unsetAddress(mode rune, address string) {
	d.pos.unsetAddress(mode, address)
	d.neg.setAddress(mode, address)
}

// Modeset encapsulates flag-based modestrings, setting and getting any modes
// and potentially using arguments as well. Some functions work with full
// modestrings containing both + and - characters, and some commands work with
// simple modestrings with are only positive or negative with the leading +/-
// omitted.
type Modeset struct {
	modes        map[rune]bool
	argModes     map[rune]string
	addressModes map[rune][]string

	*ModeKinds

	addresses int
}

// CreateModeset creates an empty Modeset.
func CreateModeset(kinds *ModeKinds) *Modeset {
	return &Modeset{
		modes:        make(map[rune]bool),
		argModes:     make(map[rune]string),
		addressModes: make(map[rune][]string),

		ModeKinds: kinds,
	}
}

// Apply takes a complex modestring and applies it to a an existing modeset
func (m *Modeset) Apply(modestring string) {
	apply(m, modestring)
}

// ApplyDiff applies a ModeDiff to the current modeset instance.
func (m *Modeset) ApplyDiff(d *ModeDiff) {
	for mode, _ := range d.pos.modes {
		m.setMode(mode)
	}
	for mode, arg := range d.pos.argModes {
		m.setArg(mode, arg)
	}
	for mode, args := range d.pos.addressModes {
		for i := 0; i < len(args); i++ {
			m.setAddress(mode, args[i])
		}
	}

	for mode, _ := range d.neg.modes {
		m.unsetMode(mode)
	}
	for mode, arg := range d.neg.argModes {
		m.unsetArg(mode, arg)
	}
	for mode, args := range d.neg.addressModes {
		for i := 0; i < len(args); i++ {
			m.unsetAddress(mode, args[i])
		}
	}
}

// String turns a Modeset into a simple string representation.
func (m *Modeset) String() string {
	length := len(m.modes)
	arglength := len(m.argModes) + m.addresses
	modes := make([]rune, length+arglength)
	args := make([]string, arglength)

	index := 0
	argIndex := 0

	for mode, _ := range m.modes {
		modes[index] = mode
		index++
	}
	for mode, arg := range m.argModes {
		modes[index] = mode
		args[argIndex] = arg
		argIndex++
		index++
	}
	for mode, arglist := range m.addressModes {
		for j := 0; j < len(arglist); j++ {
			modes[index] = mode
			args[argIndex] = arglist[j]
			argIndex++
			index++
		}
	}

	if argIndex == 0 {
		return string(modes)
	}
	return string(modes) + " " + strings.Join(args, " ")
}

// IsSet checks to see if the given modes are set using simple mode strings.
func (m *Modeset) IsSet(modestrs ...string) bool {
	modes, args := parseSimpleModestrings(modestrs...)
	if len(modes) == 0 {
		return false
	}

	used := 0

	for _, mode := range modes {
		kind := m.getKind(mode)
		switch kind {
		case ARGS_ALWAYS, ARGS_ONSET, ARGS_ADDRESS:
			arg, found := "", false
			if used < len(args) {
				arg = args[used]
				used++
			}
			if kind == ARGS_ADDRESS {
				found = m.isAddressSet(mode, arg)
			} else {
				found = m.isArgSet(mode, arg)
			}
			if !found {
				return false
			}
		case ARGS_NONE:
			if !m.isModeSet(mode) {
				return false
			}
		}
	}

	return true
}

// Set sets modes using a simple mode string.
func (m *Modeset) Set(modestrs ...string) {
	modes, args := parseSimpleModestrings(modestrs...)
	if len(modes) == 0 {
		return
	}

	used := 0

	for _, mode := range modes {
		switch m.getKind(mode) {
		case ARGS_ALWAYS, ARGS_ONSET:
			if used >= len(args) {
				break
			}
			m.setArg(mode, args[used])
			used++
		case ARGS_ADDRESS:
			if used >= len(args) {
				break
			}
			m.setAddress(mode, args[used])
			used++
		case ARGS_NONE:
			m.setMode(mode)
		}
	}
}

// Unset unsets modes using a simple mode string.
func (m *Modeset) Unset(modestrs ...string) {
	modes, args := parseSimpleModestrings(modestrs...)
	if len(modes) == 0 {
		return
	}

	used := 0

	for _, mode := range modes {

		switch m.getKind(mode) {
		case ARGS_ALWAYS:
			if used >= len(args) {
				break
			}
			m.unsetArg(mode, args[used])
			used++
		case ARGS_ADDRESS:
			if used >= len(args) {
				break
			}
			m.unsetAddress(mode, args[used])
			used++
		case ARGS_ONSET:
			m.unsetArg(mode, "")
		case ARGS_NONE:
			m.unsetMode(mode)
		}
	}
}

// GetArg returns the argument for the current mode. Empty string if the mode
// is not set.
func (m *Modeset) GetArg(mode rune) string {
	return m.argModes[mode]
}

// GetArg returns the addresses for the current mode. Nil if the mode is not
// set.
func (m *Modeset) GetAddresses(mode rune) []string {
	return m.addressModes[mode]
}

// isModeSet checks to see if a mode has been set.
func (m *Modeset) isModeSet(mode rune) bool {
	return m.modes[mode]
}

// setMode sets a mode.
func (m *Modeset) setMode(mode rune) {
	m.modes[mode] = true
}

// unsetMode unsets a mode.
func (m *Modeset) unsetMode(mode rune) {
	delete(m.modes, mode)
}

// isArgSet checks to see if a specific arg has been set for a mode, if arg is
// empty string simply checks for the modes existence.
func (m *Modeset) isArgSet(mode rune, arg string) bool {
	if check, has := m.argModes[mode]; has &&
		(len(arg) == 0 || arg == check) {

		return true
	}
	return false
}

// setArg sets an argument for a mode.
func (m *Modeset) setArg(mode rune, arg string) {
	m.argModes[mode] = arg
}

// unsetArg unsets an argument mode. If arg is not empty string, it will
// ensure the arg matches as well in order to unset.
func (m *Modeset) unsetArg(mode rune, arg string) {
	if check, has := m.argModes[mode]; has &&
		(len(arg) == 0 || arg == check) {

		delete(m.argModes, mode)
	}
}

// isAddressSet checks to see if a specific address is set in a mode, if address
// is empty string, simply checks for the modes existence.
func (m *Modeset) isAddressSet(mode rune, address string) bool {
	if addresses, has := m.addressModes[mode]; !has {
		return false
	} else if len(address) > 0 {
		i, lenaddr := 0, len(addresses)
		for ; i < lenaddr && addresses[i] != address; i++ {
		}
		if i >= lenaddr {
			return false
		}
	}

	return true
}

// setAddress sets an address for a mode.
func (m *Modeset) setAddress(mode rune, address string) {
	if addresses, has := m.addressModes[mode]; !has {
		m.addressModes[mode] = []string{address}
		m.addresses++
	} else {
		i, lenaddr := 0, len(addresses)
		for ; i < lenaddr && addresses[i] != address; i++ {
		}
		if i >= lenaddr {
			m.addressModes[mode] = append(addresses, address)
			m.addresses++
		}
	}
}

// unsetAddress unsets an address for a mode.
func (m *Modeset) unsetAddress(mode rune, address string) {
	if addresses, has := m.addressModes[mode]; has {
		i, lenaddr := 0, len(addresses)
		for ; i < lenaddr && addresses[i] != address; i++ {
		}
		if i < lenaddr {
			if lenaddr == 1 {
				delete(m.addressModes, mode)
				m.addresses--
			} else {
				if i < lenaddr-1 {
					addresses[i], addresses[lenaddr-1] =
						addresses[lenaddr-1], addresses[i]
				}
				m.addressModes[mode] = addresses[:lenaddr-1]
				m.addresses--
			}
		}
	}
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
