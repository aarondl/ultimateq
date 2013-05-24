package irc

import (
	"regexp"
	"strconv"
	"strings"
)

const (
	CAPS_RFC         = "RFC"
	CAPS_IRCD        = "IRCD"
	CAPS_CASEMAPPING = "CASEMAPPING"
	CAPS_PREFIX      = "PREFIX"
	CAPS_CHANTYPES   = "CHANTYPES"
	CAPS_CHANMODES   = "CHANMODES"
	CAPS_CHANLIMIT   = "CHANLIMIT"
	CAPS_CHANNELLEN  = "CHANNELLEN"
	CAPS_NICKLEN     = "NICKLEN"
	CAPS_TOPICLEN    = "TOPICLEN"
	CAPS_AWAYLEN     = "AWAYLEN"
	CAPS_KICKLEN     = "KICKLEN"
	CAPS_MODES       = "MODES"

	CAPS_DEFAULT_RFC         = "RFC2812"
	CAPS_DEFAULT_IRCD        = "none"
	CAPS_DEFAULT_CASEMAPPING = "ascii"
	CAPS_DEFAULT_PREFIX      = "(ov)@+"
	CAPS_DEFAULT_CHANTYPES   = "#&~"
	CAPS_DEFAULT_CHANMODES   = "beI,k,l,imnOPRstz"
	CAPS_DEFAULT_CHANLIMIT   = 20
	CAPS_DEFAULT_CHANNELLEN  = 50
	CAPS_DEFAULT_NICKLEN     = 9
	CAPS_DEFAULT_TOPICLEN    = 490
	CAPS_DEFAULT_AWAYLEN     = 127
	CAPS_DEFAULT_KICKLEN     = 400
	CAPS_DEFAULT_MODES       = 5
)

var (
	capsRegexp = regexp.MustCompile(`^(?i)([A-Z0-9]+)(?:=([^\s]+))?$`)
)

// Used to record the server settings, aids in parsing irc protocol.
type ProtoCaps struct {
	// The RFC referred to.
	rfc string
	// The IRC name
	ircd string
	// The string casemapping
	casemapping string
	// The prefix for user modes
	prefix string
	// The channel types supported by the server, usually &#~
	chantypes string
	// The channel modes allowed to be set by the server.
	chanmodes string
	// The max amount of channels we're allowed to join.
	chanlimit int
	// The max length of channel names
	channellen int
	// The max length of nicknames
	nicklen int
	// The max length of topics
	topiclen int
	// The time before away is set
	awaylen int
	// The length of kick messages
	kicklen int
	// The number of modes allowed per mode set
	modes int

	// The other flags sent in.
	extras map[string]string
}

// CreateProtoCaps initializes a protocaps struct.
func CreateProtoCaps() *ProtoCaps {
	p := &ProtoCaps{
		rfc:         CAPS_DEFAULT_RFC,
		ircd:        CAPS_DEFAULT_IRCD,
		casemapping: CAPS_DEFAULT_CASEMAPPING,
		prefix:      CAPS_DEFAULT_PREFIX,
		chantypes:   CAPS_DEFAULT_CHANTYPES,
		chanmodes:   CAPS_DEFAULT_CHANMODES,
		chanlimit:   CAPS_DEFAULT_CHANLIMIT,
		channellen:  CAPS_DEFAULT_CHANNELLEN,
		nicklen:     CAPS_DEFAULT_NICKLEN,
		topiclen:    CAPS_DEFAULT_TOPICLEN,
		awaylen:     CAPS_DEFAULT_AWAYLEN,
		kicklen:     CAPS_DEFAULT_KICKLEN,
		modes:       CAPS_DEFAULT_MODES,
		extras:      make(map[string]string),
	}
	return p
}

// RFC gets the rfc from the ProtoCaps.
func (p *ProtoCaps) RFC() string {
	return p.rfc
}

// IRCD gets the ircd from the ProtoCaps.
func (p *ProtoCaps) IRCD() string {
	return p.ircd
}

// Casemapping gets the casemapping from the ProtoCaps.
func (p *ProtoCaps) Casemapping() string {
	return p.casemapping
}

// Prefix gets the prefix from the ProtoCaps.
func (p *ProtoCaps) Prefix() string {
	return p.prefix
}

// Chantypes gets the chantypes from the ProtoCaps.
func (p *ProtoCaps) Chantypes() string {
	return p.chantypes
}

// Chanmodes gets the chanmodes from the ProtoCaps.
func (p *ProtoCaps) Chanmodes() string {
	return p.chanmodes
}

// Chanlimit gets the chanlimit from the ProtoCaps.
func (p *ProtoCaps) Chanlimit() int {
	return p.chanlimit
}

// Channellen gets the channellen from the ProtoCaps.
func (p *ProtoCaps) Channellen() int {
	return p.channellen
}

// Nicklen gets the nicklen from the ProtoCaps.
func (p *ProtoCaps) Nicklen() int {
	return p.nicklen
}

// Topiclen gets the topiclen from the ProtoCaps.
func (p *ProtoCaps) Topiclen() int {
	return p.topiclen
}

// Awaylen gets the awaylen from the ProtoCaps.
func (p *ProtoCaps) Awaylen() int {
	return p.awaylen
}

// Kicklen gets the kicklen from the ProtoCaps.
func (p *ProtoCaps) Kicklen() int {
	return p.kicklen
}

// Modes gets the modes from the ProtoCaps.
func (p *ProtoCaps) Modes() int {
	return p.modes
}

// Extra gets any non-hardcoded modes from the ProtoCaps.
func (p *ProtoCaps) Extra(key string) string {
	return p.extras[key]
}

// ParseProtoCaps adds all values in a 005 to the current protocaps object.
func (p *ProtoCaps) ParseProtoCaps(m *IrcMessage) {
	for _, arg := range m.Args {
		if strings.Contains(arg, " ") {
			continue
		}

		regexResult := capsRegexp.FindStringSubmatch(arg)
		name, value := regexResult[1], regexResult[2]

		if strings.HasPrefix(name, CAPS_RFC) {
			p.rfc = name
			continue
		}

		switch name {
		case CAPS_IRCD:
			p.ircd = value
		case CAPS_CASEMAPPING:
			p.casemapping = value
		case CAPS_PREFIX:
			p.prefix = value
		case CAPS_CHANTYPES:
			p.chantypes = value
		case CAPS_CHANMODES:
			p.chanmodes = value
		case CAPS_CHANLIMIT:
			if strings.Contains(value, ":") {
				value = strings.Split(value, ":")[1]
			}
			i, e := strconv.Atoi(value)
			if e == nil {
				p.chanlimit = i
			}
		case CAPS_CHANNELLEN:
			i, e := strconv.Atoi(value)
			if e == nil {
				p.channellen = i
			}
		case CAPS_NICKLEN:
			i, e := strconv.Atoi(value)
			if e == nil {
				p.nicklen = i
			}
		case CAPS_TOPICLEN:
			i, e := strconv.Atoi(value)
			if e == nil {
				p.topiclen = i
			}
		case CAPS_AWAYLEN:
			i, e := strconv.Atoi(value)
			if e == nil {
				p.awaylen = i
			}
		case CAPS_KICKLEN:
			i, e := strconv.Atoi(value)
			if e == nil {
				p.kicklen = i
			}
		case CAPS_MODES:
			i, e := strconv.Atoi(value)
			if e == nil {
				p.modes = i
			}
		default:
			if value == "" {
				value = "true"
			}
			p.extras[name] = value
		}
	}
}
