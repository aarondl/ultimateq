package irc

import (
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// These constants are the mappings from the 004 and 005 events to their
// respective spots inside the NetworkInfo type.
const (
	INFO_RFC         = "RFC"
	INFO_IRCD        = "IRCD"
	INFO_CASEMAPPING = "CASEMAPPING"
	INFO_PREFIX      = "PREFIX"
	INFO_CHANTYPES   = "CHANTYPES"
	INFO_CHANMODES   = "CHANMODES"
	INFO_CHANLIMIT   = "CHANLIMIT"
	INFO_CHANNELLEN  = "CHANNELLEN"
	INFO_NICKLEN     = "NICKLEN"
	INFO_TOPICLEN    = "TOPICLEN"
	INFO_AWAYLEN     = "AWAYLEN"
	INFO_KICKLEN     = "KICKLEN"
	INFO_MODES       = "MODES"
)

// These constants are healthy defaults for a NetworkInfo type. They were
// taken from ngircd.
const (
	INFO_DEFAULT_SERVERNAME  = "unknown"
	INFO_DEFAULT_IRCDVERSION = "unknown"
	INFO_DEFAULT_USERMODES   = "acCiorRswx"
	INFO_DEFAULT_LCHANMODES  = "beiIklmnoOPrRstvz"

	INFO_DEFAULT_RFC         = "RFC2812"
	INFO_DEFAULT_IRCD        = "unknown"
	INFO_DEFAULT_CASEMAPPING = "ascii"
	INFO_DEFAULT_PREFIX      = "(ov)@+"
	INFO_DEFAULT_CHANTYPES   = "#&~"
	INFO_DEFAULT_CHANMODES   = "beI,k,l,imnOPRstz"
	INFO_DEFAULT_CHANLIMIT   = 20
	INFO_DEFAULT_CHANNELLEN  = 50
	INFO_DEFAULT_NICKLEN     = 9
	INFO_DEFAULT_TOPICLEN    = 490
	INFO_DEFAULT_AWAYLEN     = 127
	INFO_DEFAULT_KICKLEN     = 400
	INFO_DEFAULT_MODES       = 5
)

var (
	capsRegexp = regexp.MustCompile(`^(?i)([A-Z0-9]+)(?:=([^\s]+))?$`)
)

// NetworkInfo is used to record the server capabilities, this later aids in
// parsing irc protocol.
type NetworkInfo struct {
	// The server's self-defined name.
	serverName string
	// The ircd's version.
	ircdVersion string
	// The user modes
	usermodes string
	// The legacy chanmodes, chanmodes should be used instead.
	lchanmodes string

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
	// The length of kick events
	kicklen int
	// The number of modes allowed per mode set
	modes int

	// The other flags sent in.
	extras map[string]string

	protect *sync.RWMutex
}

// NewNetworkInfo initializes a networkinfo struct.
func NewNetworkInfo() *NetworkInfo {
	p := &NetworkInfo{
		serverName:  INFO_DEFAULT_SERVERNAME,
		ircdVersion: INFO_DEFAULT_IRCDVERSION,
		usermodes:   INFO_DEFAULT_USERMODES,
		lchanmodes:  INFO_DEFAULT_LCHANMODES,
		rfc:         INFO_DEFAULT_RFC,
		ircd:        INFO_DEFAULT_IRCD,
		casemapping: INFO_DEFAULT_CASEMAPPING,
		prefix:      INFO_DEFAULT_PREFIX,
		chantypes:   INFO_DEFAULT_CHANTYPES,
		chanmodes:   INFO_DEFAULT_CHANMODES,
		chanlimit:   INFO_DEFAULT_CHANLIMIT,
		channellen:  INFO_DEFAULT_CHANNELLEN,
		nicklen:     INFO_DEFAULT_NICKLEN,
		topiclen:    INFO_DEFAULT_TOPICLEN,
		awaylen:     INFO_DEFAULT_AWAYLEN,
		kicklen:     INFO_DEFAULT_KICKLEN,
		modes:       INFO_DEFAULT_MODES,
		extras:      make(map[string]string),

		protect: new(sync.RWMutex),
	}
	return p
}

// Clone safely clones this networkinfo instance.
func (p *NetworkInfo) Clone() *NetworkInfo {
	p.protect.RLock()
	defer p.protect.RUnlock()
	clone := *p
	clone.extras = make(map[string]string)
	for k, v := range p.extras {
		clone.extras[k] = v
	}
	clone.protect = new(sync.RWMutex)
	return &clone
}

// ServerName gets the servername from the NetworkInfo.
func (p *NetworkInfo) ServerName() string {
	p.protect.RLock()
	defer p.protect.RUnlock()
	return p.serverName
}

// IrcdVersion gets the irc version from the NetworkInfo.
func (p *NetworkInfo) IrcdVersion() string {
	p.protect.RLock()
	defer p.protect.RUnlock()
	return p.ircdVersion
}

// Usermodes gets the usermodes from the NetworkInfo.
func (p *NetworkInfo) Usermodes() string {
	p.protect.RLock()
	defer p.protect.RUnlock()
	return p.usermodes
}

// LegacyChanmodes gets the legacy channel modes from the NetworkInfo.
func (p *NetworkInfo) LegacyChanmodes() string {
	p.protect.RLock()
	defer p.protect.RUnlock()
	return p.lchanmodes
}

// RFC gets the rfc from the NetworkInfo.
func (p *NetworkInfo) RFC() string {
	p.protect.RLock()
	defer p.protect.RUnlock()
	return p.rfc
}

// IRCD gets the ircd from the NetworkInfo.
func (p *NetworkInfo) IRCD() string {
	p.protect.RLock()
	defer p.protect.RUnlock()
	return p.ircd
}

// Casemapping gets the casemapping from the NetworkInfo.
func (p *NetworkInfo) Casemapping() string {
	p.protect.RLock()
	defer p.protect.RUnlock()
	return p.casemapping
}

// Prefix gets the prefix from the NetworkInfo.
func (p *NetworkInfo) Prefix() string {
	p.protect.RLock()
	defer p.protect.RUnlock()
	return p.prefix
}

// Chantypes gets the chantypes from the NetworkInfo.
func (p *NetworkInfo) Chantypes() string {
	p.protect.RLock()
	defer p.protect.RUnlock()
	return p.chantypes
}

// Chanmodes gets the chanmodes from the NetworkInfo.
func (p *NetworkInfo) Chanmodes() string {
	p.protect.RLock()
	defer p.protect.RUnlock()
	return p.chanmodes
}

// Chanlimit gets the chanlimit from the NetworkInfo.
func (p *NetworkInfo) Chanlimit() int {
	p.protect.RLock()
	defer p.protect.RUnlock()
	return p.chanlimit
}

// Channellen gets the channellen from the NetworkInfo.
func (p *NetworkInfo) Channellen() int {
	p.protect.RLock()
	defer p.protect.RUnlock()
	return p.channellen
}

// Nicklen gets the nicklen from the NetworkInfo.
func (p *NetworkInfo) Nicklen() int {
	p.protect.RLock()
	defer p.protect.RUnlock()
	return p.nicklen
}

// Topiclen gets the topiclen from the NetworkInfo.
func (p *NetworkInfo) Topiclen() int {
	p.protect.RLock()
	defer p.protect.RUnlock()
	return p.topiclen
}

// Awaylen gets the awaylen from the NetworkInfo.
func (p *NetworkInfo) Awaylen() int {
	p.protect.RLock()
	defer p.protect.RUnlock()
	return p.awaylen
}

// Kicklen gets the kicklen from the NetworkInfo.
func (p *NetworkInfo) Kicklen() int {
	p.protect.RLock()
	defer p.protect.RUnlock()
	return p.kicklen
}

// Modes gets the modes from the NetworkInfo.
func (p *NetworkInfo) Modes() int {
	p.protect.RLock()
	defer p.protect.RUnlock()
	return p.modes
}

// Extra gets any non-hardcoded modes from the NetworkInfo.
func (p *NetworkInfo) Extra(key string) string {
	p.protect.RLock()
	defer p.protect.RUnlock()
	return p.extras[key]
}

// Extras clones the internal map and returns it
func (p *NetworkInfo) Extras() map[string]string {
	p.protect.RLock()
	defer p.protect.RUnlock()

	cloned := make(map[string]string, len(p.extras))
	for k, v := range p.extras {
		cloned[k] = v
	}

	return cloned
}

// ParseISupport adds all values in a 005 to the current networkinfo object.
func (p *NetworkInfo) ParseISupport(e *Event) {
	p.protect.Lock()
	defer p.protect.Unlock()

	for _, arg := range e.Args[1:] {
		if strings.Contains(arg, " ") {
			continue
		}

		regexResult := capsRegexp.FindStringSubmatch(arg)
		name, value := regexResult[1], regexResult[2]

		if strings.HasPrefix(name, INFO_RFC) {
			p.rfc = name
			continue
		}

		switch name {
		case INFO_IRCD:
			p.ircd = value
		case INFO_CASEMAPPING:
			p.casemapping = value
		case INFO_PREFIX:
			p.prefix = value
		case INFO_CHANTYPES:
			p.chantypes = value
		case INFO_CHANMODES:
			p.chanmodes = value
		case INFO_CHANLIMIT:
			if strings.Contains(value, ":") {
				value = strings.Split(value, ":")[1]
			}
			i, e := strconv.Atoi(value)
			if e == nil {
				p.chanlimit = i
			}
		case INFO_CHANNELLEN:
			i, e := strconv.Atoi(value)
			if e == nil {
				p.channellen = i
			}
		case INFO_NICKLEN:
			i, e := strconv.Atoi(value)
			if e == nil {
				p.nicklen = i
			}
		case INFO_TOPICLEN:
			i, e := strconv.Atoi(value)
			if e == nil {
				p.topiclen = i
			}
		case INFO_AWAYLEN:
			i, e := strconv.Atoi(value)
			if e == nil {
				p.awaylen = i
			}
		case INFO_KICKLEN:
			i, e := strconv.Atoi(value)
			if e == nil {
				p.kicklen = i
			}
		case INFO_MODES:
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

// IsChannel checks to see if the target is a channel based on this instances
// chantypes.
func (p *NetworkInfo) IsChannel(target string) (isChan bool) {
	if len(target) > 0 {
		p.protect.RLock()
		isChan = strings.ContainsRune(p.chantypes, rune(target[0]))
		p.protect.RUnlock()
	}
	return
}

// ParseMyInfo adds all values in a 005 to the current networkinfo object.
func (p *NetworkInfo) ParseMyInfo(e *Event) {
	p.protect.Lock()
	defer p.protect.Unlock()

	p.serverName = e.Args[1]
	p.ircdVersion = e.Args[2]
	p.usermodes = e.Args[3]
	p.lchanmodes = e.Args[4]
}
