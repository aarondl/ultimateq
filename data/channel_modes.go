package data

import (
	"strings"
)

// ChannelModes encapsulates flag-based modestrings, setting and getting any
// modes and potentially using arguments as well. Some functions work with full
// modestrings containing both + and - characters, and some commands work with
// simple modestrings with are only positive or negative with the leading +/-
// omitted.
type ChannelModes struct {
	modes        map[rune]bool
	argModes     map[rune]string
	addressModes map[rune][]string

	*ChannelModeKinds

	addresses int
}

// CreateChannelModes creates an empty ChannelModes.
func CreateChannelModes(kinds *ChannelModeKinds) *ChannelModes {
	return &ChannelModes{
		modes:        make(map[rune]bool),
		argModes:     make(map[rune]string),
		addressModes: make(map[rune][]string),

		ChannelModeKinds: kinds,
	}
}

// Apply takes a complex modestring and applies it to a an existing modeset.
// Assumes any modes not declared as part of ChannelModeKinds were not intended
// for channel and are user-targeted (therefore taking an argument)
// and returns them in two arrays, positive and negative modes respectively.
func (m *ChannelModes) Apply(modestring string) ([]UnknownMode, []UnknownMode) {
	return apply(m, modestring)
}

// ApplyDiff applies a ModeDiff to the current modeset instance.
func (m *ChannelModes) ApplyDiff(d *ModeDiff) {
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

// String turns a ChannelModes into a simple string representation.
func (m *ChannelModes) String() string {
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
func (m *ChannelModes) IsSet(modestrs ...string) bool {
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
func (m *ChannelModes) Set(modestrs ...string) {
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
func (m *ChannelModes) Unset(modestrs ...string) {
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
func (m *ChannelModes) GetArg(mode rune) string {
	return m.argModes[mode]
}

// GetArg returns the addresses for the current mode. Nil if the mode is not
// set.
func (m *ChannelModes) GetAddresses(mode rune) []string {
	return m.addressModes[mode]
}

// isModeSet checks to see if a mode has been set.
func (m *ChannelModes) isModeSet(mode rune) bool {
	return m.modes[mode]
}

// setMode sets a mode.
func (m *ChannelModes) setMode(mode rune) {
	m.modes[mode] = true
}

// unsetMode unsets a mode.
func (m *ChannelModes) unsetMode(mode rune) {
	delete(m.modes, mode)
}

// isArgSet checks to see if a specific arg has been set for a mode, if arg is
// empty string simply checks for the modes existence.
func (m *ChannelModes) isArgSet(mode rune, arg string) bool {
	if check, has := m.argModes[mode]; has &&
		(len(arg) == 0 || arg == check) {

		return true
	}
	return false
}

// setArg sets an argument for a mode.
func (m *ChannelModes) setArg(mode rune, arg string) {
	m.argModes[mode] = arg
}

// unsetArg unsets an argument mode. If arg is not empty string, it will
// ensure the arg matches as well in order to unset.
func (m *ChannelModes) unsetArg(mode rune, arg string) {
	if check, has := m.argModes[mode]; has &&
		(len(arg) == 0 || arg == check) {

		delete(m.argModes, mode)
	}
}

// isAddressSet checks to see if a specific address is set in a mode, if address
// is empty string, simply checks for the modes existence.
func (m *ChannelModes) isAddressSet(mode rune, address string) bool {
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
func (m *ChannelModes) setAddress(mode rune, address string) {
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
func (m *ChannelModes) unsetAddress(mode rune, address string) {
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
