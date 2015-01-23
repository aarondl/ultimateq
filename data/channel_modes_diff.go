package data

import (
	"strings"
)

// ModeDiff encapsulates a difference of modes, a combination of both positive
// change modes, and negative change modes.
type ModeDiff struct {
	pos ChannelModes
	neg ChannelModes

	*modeKinds
}

// NewModeDiff creates an empty ModeDiff.
func NewModeDiff(m *modeKinds) ModeDiff {
	return ModeDiff{
		modeKinds: m,
		pos:       NewChannelModes(m),
		neg:       NewChannelModes(m),
	}
}

// Clone deep copies the ModeDiff.
func (d *ModeDiff) Clone() ModeDiff {
	return ModeDiff{
		modeKinds: d.modeKinds,
		pos:       d.pos.Clone(),
		neg:       d.neg.Clone(),
	}
}

// IsSet checks if applying this diff will set the given simple modestrs.
func (d *ModeDiff) IsSet(modestrs ...string) bool {
	return d.pos.IsSet(modestrs...)
}

// IsUnset checks if applying this diff will unset the given simple modestrs.
func (d *ModeDiff) IsUnset(modestrs ...string) bool {
	return d.neg.IsSet(modestrs...)
}

// Apply takes a complex modestring and transforms it into a diff.
// Assumes any modes not declared as part of ChannelModeKinds were not intended
// for channel and are user-targeted (therefore taking an argument)
// and returns them in two arrays, positive and negative modes respectively.
func (d *ModeDiff) Apply(modestring string) ([]userMode, []userMode) {
	return apply(d, modestring)
}

// String turns a ModeDiff into a complex string representation.
func (d *ModeDiff) String() string {
	modes := ""
	args := ""
	pos, neg := d.pos.String(), d.neg.String()
	if len(pos) > 0 {
		pspace := strings.IndexRune(pos, ' ')
		if pspace < 0 {
			pspace = len(pos)
		} else {
			args += " " + pos[pspace+1:]
		}
		modes += "+" + pos[:pspace]
	}
	if len(neg) > 0 {
		nspace := strings.IndexRune(neg, ' ')
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

// isUserMode checks if the given mode belongs to the user mode kinds.
func (d ModeDiff) isUserMode(mode rune) (is bool) {
	if d.userPrefixes != nil {
		is = d.modeBit(mode) > 0
	}
	return
}
