package data

import (
	"strings"
)

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
