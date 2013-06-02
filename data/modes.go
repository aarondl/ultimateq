package data

import (
	"strings"
)

// ModeDiff encapsulates a difference of modes, a combination of both positive
// change modes, and negative change modes.
type ModeDiff struct {
	pos *Modeset
	neg *Modeset
}

// CreateModeDiff creates an empty ModeDiff.
func CreateModeDiff() *ModeDiff {
	return &ModeDiff{
		pos: CreateModeset(),
		neg: CreateModeset(),
	}
}

// CreateModeDiffFromModestring creates a ModeDiff and applies a modestring.
func CreateModeDiffFromModestring(modestring, hasargs string) *ModeDiff {
	return CreateModeDiff().Apply(modestring, hasargs)
}

// Checks if applying this diff will set the given simple modestrs.
func (m *ModeDiff) IsSet(modestrs ...string) bool {
	return m.pos.IsSet(modestrs...)
}

// Checks if applying this diff will unset the given simple modestrs.
func (m *ModeDiff) IsUnset(modestrs ...string) bool {
	return m.neg.IsSet(modestrs...)
}

// Apply takes a complex modestring and transforms it into a diff.
func (m *ModeDiff) Apply(modestring, hasargs string) *ModeDiff {
	apply(m, modestring, hasargs)
	return m
}

// String turns a ModeDiff into a complex string representation.
func (m *ModeDiff) String() string {
	lenpos, lenneg := len(m.pos.modes), len(m.neg.modes)

	pos := make([]rune, lenpos)
	neg := make([]rune, lenneg)
	posargs := make([]string, lenpos)
	negargs := make([]string, lenneg)

	posindex := 0
	negindex := 0
	posargsIndex := lenpos - 1
	negargsIndex := lenneg - 1

	for mode, arg := range m.pos.modes {
		if len(arg) > 0 {
			posargs[posargsIndex] = arg
			pos[posargsIndex] = mode
			posargsIndex--
		} else {
			pos[posindex] = mode
			posindex++
		}
	}

	for mode, arg := range m.neg.modes {
		if len(arg) > 0 {
			negargs[negargsIndex] = arg
			neg[negargsIndex] = mode
			negargsIndex--
		} else {
			neg[negindex] = mode
			negindex++
		}
	}

	return "+" + string(pos) + "-" + string(neg) + " " +
		strings.Join(posargs[posargsIndex+1:], " ") + " " +
		strings.Join(negargs[negargsIndex+1:], " ")
}

// setPositiveMode fullfills the moder interface to allow pos modes to be
// set.
func (m *ModeDiff) setPositiveMode(mode rune, arg string) {
	delete(m.neg.modes, mode)
	m.pos.modes[mode] = arg
}

// setNegativeMode fullfills the moder interface to allow neg modes to be
// set.
func (m *ModeDiff) setNegativeMode(mode rune, arg string) {
	delete(m.pos.modes, mode)
	m.neg.modes[mode] = arg
}

// Modeset encapsulates flag-based modestrings, setting and getting any modes
// and potentially using arguments as well. Some functions work with full
// modestrings containing both + and - characters, and some commands work with
// simple modestrings with are only positive or negative with the leading +/-
// omitted.
type Modeset struct {
	modes map[rune]string
}

// CreateModeset creates an empty Modeset.
func CreateModeset() *Modeset {
	return &Modeset{
		modes: make(map[rune]string),
	}
}

// CreateModesetFromModestring creates a Modeset and applies a modestring.
func CreateModesetFromModestring(modestring, hasargs string) *Modeset {
	return CreateModeset().Apply(modestring, hasargs)
}

// Apply takes a complex modestring and applies it to a an existing modeset
// instance.
func (m *Modeset) Apply(modestring, hasargs string) *Modeset {
	apply(m, modestring, hasargs)
	return m
}

// ApplyDiff applies a ModeDiff to the current modeset instance.
func (m *Modeset) ApplyDiff(d *ModeDiff) {
	for mode, arg := range d.pos.modes {
		m.modes[mode] = arg
	}

	for mode, arg := range d.neg.modes {
		if len(arg) > 0 {
			if delarg, ok := m.modes[mode]; !ok || arg != delarg {
				continue
			}
		}

		delete(m.modes, mode)
	}
}

// String turns a Modeset into a simple string representation.
func (m *Modeset) String() string {
	length := len(m.modes)
	modes := make([]rune, length)
	args := make([]string, length)

	index := 0
	argsIndex := length - 1

	for mode, arg := range m.modes {
		if len(arg) > 0 {
			args[argsIndex] = arg
			modes[argsIndex] = mode
			argsIndex--
		} else {
			modes[index] = mode
			index++
		}
	}
	return string(modes) + " " + strings.Join(args[argsIndex+1:], " ")
}

// IsSet checks to see if the given modes are set using simple mode strings.
func (m *Modeset) IsSet(modestrs ...string) bool {
	modes, args := parseSimpleModestrings(modestrs...)
	if len(modes) == 0 {
		return false
	}

	diff := len(modes) - len(args)
	for i := 0; i < len(modes); i++ {
		if i >= diff {
			if arg, found := m.modes[modes[i]]; !found || arg != args[i-diff] {
				return false
			}
		} else {
			if _, found := m.modes[modes[i]]; !found {
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

	diff := len(modes) - len(args)
	for i := 0; i < len(modes); i++ {
		if i >= diff {
			m.modes[modes[i]] = args[i-diff]
		} else {
			m.modes[modes[i]] = ""
		}
	}
}

// Unset unsets modes using a simple mode string.
func (m *Modeset) Unset(modestrs ...string) {
	modes, args := parseSimpleModestrings(modestrs...)
	if len(modes) == 0 {
		return
	}

	diff := len(modes) - len(args)
	for i := 0; i < len(modes); i++ {
		if i >= diff {
			if arg, found := m.modes[modes[i]]; found && arg == args[i-diff] {
				delete(m.modes, modes[i])
			}
		} else {
			delete(m.modes, modes[i])
		}
	}
}

// setPositiveMode fullfills the moder interface to allow positive modes to be
// set.
func (m *Modeset) setPositiveMode(mode rune, arg string) {
	m.modes[mode] = arg
}

// setNegativeMode fullfills the moder interface to allow negative modes to be
// set.
func (m *Modeset) setNegativeMode(mode rune, arg string) {
	if len(arg) > 0 {
		if delarg := m.modes[mode]; arg != delarg {
			return
		}
	}
	delete(m.modes, mode)
}

// moder is an interface that defines common behavior between all mode managing
// types.
type moder interface {
	setPositiveMode(rune, string)
	setNegativeMode(rune, string)
}

// apply parses a complex modestring and applies it to a moder interface.
func apply(m moder, modestring, hasargs string) {
	adding := true
	argsUsed := 0

	splits := strings.Split(strings.TrimSpace(modestring), " ")

	for _, c := range splits[0] {
		if add, sub := c == '+', c == '-'; add || sub {
			adding = add
			continue
		}

		if strings.ContainsRune(hasargs, c) {
			if adding {
				m.setPositiveMode(c, splits[argsUsed+1])
			} else {
				m.setNegativeMode(c, splits[argsUsed+1])
			}
			argsUsed++
		} else {
			if adding {
				m.setPositiveMode(c, "")
			} else {
				m.setNegativeMode(c, "")
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
