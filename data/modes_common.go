package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/aarondl/ultimateq/api"
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

// modeKinds is a lookup structure that uses the CHANMODES and USERMODES
// capabilities to discern what modes are what types.
type modeKinds struct {
	userPrefixes [][2]rune
	channelModes map[rune]int
	sync.RWMutex
}

type modeKindJSON struct {
	UserPrefixes [][]string     `json:"user_prefixes"`
	ChannelModes map[string]int `json:"channel_modes"`
}

// userMode is returned by apply helper when it encounters a user mode
type userMode struct {
	Mode rune
	Arg  string
}

// newModeKinds creates a new modekinds lookup struct using the strings passed
// in from a networkinfo object.
func newModeKinds(prefix, chanModes string) (*modeKinds, error) {
	userPrefixes, err := parsePrefixString(prefix)
	if err != nil {
		return nil, err
	}
	channelModes, err := parseChannelModeKindsCSV(chanModes)
	if err != nil {
		return nil, err
	}

	return &modeKinds{
		userPrefixes: userPrefixes,
		channelModes: channelModes,
	}, nil
}

// MarshalJSON turns modeKinds -> JSON
func (m *modeKinds) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte(`null`), nil
	}

	var toJSON modeKindJSON

	m.RLock()
	defer m.RUnlock()

	if m.userPrefixes != nil {
		toJSON.UserPrefixes = make([][]string, len(m.userPrefixes))
		for i, prefix := range m.userPrefixes {
			toJSON.UserPrefixes[i] = []string{
				string(prefix[0]), string(prefix[1]),
			}
		}
	}
	if m.channelModes != nil {
		toJSON.ChannelModes = make(map[string]int, len(m.channelModes))
		for k, v := range m.channelModes {
			toJSON.ChannelModes[string(k)] = v
		}
	}

	return json.Marshal(toJSON)
}

// UnmarshalJSON turns JSON -> modeKinds
func (m *modeKinds) UnmarshalJSON(b []byte) error {
	var fromJSON modeKindJSON

	if err := json.Unmarshal(b, &fromJSON); err != nil {
		return err
	}

	if fromJSON.UserPrefixes != nil {
		m.userPrefixes = make([][2]rune, len(fromJSON.UserPrefixes))
		for i, prefix := range fromJSON.UserPrefixes {
			if len(prefix) != 2 || len(prefix[0]) != 1 || len(prefix[1]) != 1 {
				return errors.New("user_prefixes is an array of length 2 of 1 character strings")
			}

			m.userPrefixes[i] = [2]rune{
				rune(prefix[0][0]),
				rune(prefix[1][0]),
			}
		}
	}
	if fromJSON.ChannelModes != nil {
		m.channelModes = make(map[rune]int, len(fromJSON.ChannelModes))
		for k, v := range fromJSON.ChannelModes {
			if len(k) != 1 {
				return errors.New("channel_modes is a map of characters to integers")
			}
			m.channelModes[rune(k[0])] = v
		}
	}

	return nil
}

// ToProto turns modeKinds -> API Struct
func (m *modeKinds) ToProto() *api.ModeKinds {
	if m == nil {
		return nil
	}

	var proto api.ModeKinds

	m.RLock()
	defer m.RUnlock()

	if m.userPrefixes != nil {
		proto.UserPrefixes = make([]*api.ModeKinds_UserPrefix, len(m.userPrefixes))
		for i, prefix := range m.userPrefixes {
			proto.UserPrefixes[i] = &api.ModeKinds_UserPrefix{
				Symbol: string(prefix[0]),
				Char:   string(prefix[1]),
			}
		}
	}
	if m.channelModes != nil {
		proto.ChannelModes = make(map[string]int32, len(m.channelModes))
		for k, v := range m.channelModes {
			proto.ChannelModes[string(k)] = int32(v)
		}
	}

	return &proto
}

// FromProto turns API struct -> modeKinds
func (m *modeKinds) FromProto(proto *api.ModeKinds) error {
	if proto.UserPrefixes != nil {
		m.userPrefixes = make([][2]rune, len(proto.UserPrefixes))
		for i, prefix := range proto.UserPrefixes {
			if len(prefix.Char) != 1 || len(prefix.Symbol) != 1 {
				return errors.New("user_prefixes is an array of length 2 of 1 character strings")
			}

			m.userPrefixes[i] = [2]rune{
				rune(prefix.Symbol[0]),
				rune(prefix.Char[0]),
			}
		}
	}
	if proto.ChannelModes != nil {
		m.channelModes = make(map[rune]int, len(proto.ChannelModes))
		for k, v := range proto.ChannelModes {
			if len(k) != 1 {
				return errors.New("channel_modes is a map of characters to integers")
			}
			m.channelModes[rune(k[0])] = int(v)
		}
	}

	return nil
}

// updateModes updates lookup structures in place.
func (m *modeKinds) update(prefix, chanModes string) error {
	m.Lock()
	defer m.Unlock()

	var userPrefixes [][2]rune
	var channelModes map[rune]int
	var err error

	if len(prefix) > 0 {
		userPrefixes, err = parsePrefixString(prefix)
		if err != nil {
			return err
		}
	}
	if len(chanModes) > 0 {
		channelModes, err = parseChannelModeKindsCSV(chanModes)
		if err != nil {
			return err
		}
	}

	if len(prefix) > 0 {
		m.userPrefixes = userPrefixes
	}
	if len(chanModes) > 0 {
		m.channelModes = channelModes
	}
	return nil
}

// Symbol returns the symbol character of the mode given.
func (m modeKinds) Symbol(mode rune) rune {
	m.RLock()
	defer m.RUnlock()

	for i := 0; i < len(m.userPrefixes); i++ {
		if m.userPrefixes[i][0] == mode {
			return m.userPrefixes[i][1]
		}
	}
	return 0
}

// Mode returns the mode character of the symbol given.
func (m modeKinds) Mode(symbol rune) rune {
	m.RLock()
	defer m.RUnlock()

	for i := 0; i < len(m.userPrefixes); i++ {
		if m.userPrefixes[i][1] == symbol {
			return m.userPrefixes[i][0]
		}
	}
	return 0
}

// ModeBit returns the bit of the mode character to set.
func (m modeKinds) modeBit(mode rune) byte {
	m.RLock()
	defer m.RUnlock()

	for i := uint(0); i < uint(len(m.userPrefixes)); i++ {
		if m.userPrefixes[i][0] == mode {
			return 1 << i
		}
	}
	return 0
}

// kind gets the kind of a mode and returns it.
func (m modeKinds) kind(mode rune) int {
	m.RLock()
	defer m.RUnlock()

	return m.channelModes[mode]
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

	return kinds
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
		kindSplits[0], kindSplits[1], kindSplits[2], kindSplits[3]), nil
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
	kind(rune) int
	isUserMode(rune) bool
}

// apply parses a complex modestring and applies it to a moder interface. All
// non-recognized modes are given back a positive and negative array of
// UnknownMode.
func apply(m moder, modestring string) (pos, neg []userMode) {
	adding := true
	used := 0
	pos = make([]userMode, 0)
	neg = make([]userMode, 0)

	splits := strings.Split(strings.TrimSpace(modestring), " ")
	args := splits[1:]

	for _, mode := range splits[0] {
		if add, sub := mode == '+', mode == '-'; add || sub {
			adding = add
			continue
		}

		kind := m.kind(mode)
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
					pos = append(pos, userMode{mode, args[used]})
				} else {
					neg = append(neg, userMode{mode, args[used]})
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
