/*
Package config creates a configuration using toml. It also does configuration
validation.
*/
package config

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"sync"
)

const (
	// defaultIrcPort is IRC Network'n default tcp port.
	defaultIrcPort = uint16(6667)
	// defaultStoreFile is where the bot will store it'n Store database if not
	// overridden.
	defaultStoreFile = "./store.db"
	// defaultFloodLenPenalty is how many characters in a message by default
	// warrant an extra second wait time.
	defaultFloodLenPenalty = uint(120)
	// defaultFloodTimeout is how many seconds worth of penalty must accumulate
	// before setting penalties.
	defaultFloodTimeout = 10.0
	// defaultFloodStep is the default number of seconds between messages once
	// flood protection has been activated.
	defaultFloodStep = 2.0
	// defaultKeepAlive is the default number of seconds to wait on an idle
	// connection before sending a ping.
	defaultKeepAlive = 60.0
	// defaultReconnectTimeout is how many seconds to wait between reconns.
	defaultReconnectTimeout = uint(20)
	// botDefaultPrefix is the command prefix by default
	defaultPrefix = '.'
	// maxHostSize is the biggest hostname possible
	maxHostSize = 255
)

// The following format strings are for formatting various config errors.
const (
	fmtErrInvalid          = "config(%v): Invalid %v, given: %v"
	fmtErrMissing          = "config(%v): Requires %v, but nothing was given."
	fmtErrNetworkNotFound  = "config: Network not found, given: %v"
	errMsgNetworksRequired = "config: At least one network is required."
	errMsgDuplicateNetwork = "config: Network names must be unique, use .Host()"
)

// The following is for mapping config setting names to strings
const (
	errServers          = "servers"
	errPort             = "port"
	errSsl              = "ssl"
	errNoVerifyCert     = "noverifycert"
	errNoState          = "nostate"
	errNoStore          = "nostore"
	errStoreFile        = "storefile"
	errFloodLenPenalty  = "floodprotectlenPenalty"
	errFloodTimeout     = "floodprotecttimeout"
	errFloodStep        = "floodprotectstep"
	errKeepAlive        = "keepalive"
	errNoReconnect      = "noreconnect"
	errReconnectTimeout = "reconnecttimeout"
	errNick             = "nickname"
	errAltnick          = "alternate nickname"
	errRealname         = "realname"
	errUsername         = "username"
	errUserhost         = "userhost"
	errPrefix           = "prefix"
	errChannel          = "channel"
)

var (
	// From the RFC:
	// nickname   =  ( letter / special ) *8( letter / digit / special / "-" )
	// letter     =  %x41-5A / %x61-7A  ; A-Z / a-z
	// digit      =  %x30-39            ; 0-9
	// special    =  %x5B-60 / %x7B-7D  ; [ ] \ ` _ ^ { | }
	// We make an excemption to the 9 char limit since few networks today
	// enforce it, and the RFC also states that clients should handle longer
	// names.
	// Test that the name is a valid IRC nickname
	rgxNickname = regexp.MustCompile(`^(?i)[a-z\[\]{}|^_\\` + "`]" +
		`[a-z0-9\[\]{}|^_\\` + "`" + `]{0,30}$`)

	/* Channels names are strings (beginning with a '&', '#', '+' or '!'
	character) of length up to fifty (50) characters.  Apart from the
	requirement that the first character is either '&', '#', '+' or '!',
	the only restriction on a channel name is that it SHALL NOT contain
	any spaces (' '), a control G (^G or ASCII 7), a comma (',').  Space
	is used as parameter separator and command is used as a list item
	separator by the protocol).  A colon (':') can also be used as a
	delimiter for the channel mask.  Channel names are case insensitive.

	Grammar:
	channelid  = 5( %x41-5A / digit )   ; 5( A-Z / 0-9 )
	chanstring = any octet except NUL, BELL, CR, LF, " ", "," and ":"
	channel    =  ( "#" / "+" / ( "!" channelid ) / "&" ) chanstring
					[ ":" chanstring ] */
	rgxChannel = regexp.MustCompile(
		`^(?i)[#&+!][^\s\000\007,]{1,49}$`)

	// rgxHost matches hostnames
	rgxHost = regexp.MustCompile(
		`(?i)^[0-9a-z](?:(?:[0-9a-z]|-){0,61}[0-9a-z])?` +
			`(?:\.[0-9a-z](?:(?:[0-9a-z]|-){0,61}[0-9a-z])?)*\.?$`)

	// rgxUsername matches usernames, insensitive all chars without spaces.
	rgxUsername = regexp.MustCompile(`^[A-Za-z0-9]+$`)
	// rgxRealname matches real names, insensitive all chars with spaces.
	rgxRealname = regexp.MustCompile(`^[A-Za-z0-9 ]+$`)
)

// Config holds all the information related to the bot including global settings
// default settings, and network specific settings.
type Config struct {
	*Network  `json:"global"`
	Networks  map[string]*Network `toml:"networks" json:"networks"`
	Storefile string              `toml:"storefile" json:"storefile"`

	errors   errList      `toml:"-" json:"-"`
	filename string       `toml:"-" json:"-"`
	protect  sync.RWMutex `toml:"-" json:"-"`
}

// NewConfig initializes a Config object.
func NewConfig() *Config {
	c := &Config{}
	c.clear()

	return c
}

// Clear re-initializes all memory in the configuration.
func (c *Config) Clear() {
	c.protect.Lock()
	defer c.protect.Unlock()

	c.clear()
}

// clear re-initializes all memory in the configuration without locking first.
func (c *Config) clear() {
	c.Network = &Network{InName: "global", protect: &c.protect}
	c.Networks = make(map[string]*Network)
	c.Storefile = ""

	c.errors = make(errList, 0)
	c.filename = ""
}

// Clone deep copies a configuration object.
func (c *Config) Clone() *Config {
	c.protect.RLock()
	defer c.protect.RUnlock()

	newconf := &Config{
		Network:  c.Network.Clone(),
		Networks: make(map[string]*Network, len(c.Networks)),
		errors:   make([]error, 0),
		filename: c.filename,
	}
	for name, net := range c.Networks {
		newnet := net.Clone()
		newnet.parent = newconf
		newconf.Networks[name] = newnet
	}
	return newconf
}

// errList is an array of errors.
type errList []error

// addError builds an error object and appends it to this instances errors.
func (c *Config) addError(format string, args ...interface{}) {
	c.errors.addError(format, args...)
}

// addError builds an error object and appends it to this instances errors.
func (l *errList) addError(format string, args ...interface{}) {
	*l = append(*l, fmt.Errorf(format, args...))
}

// Errors returns the errors encountered during validation.
func (c *Config) Errors() []error {
	c.protect.RLock()
	c.protect.RUnlock()

	ers := make([]error, len(c.errors))
	copy(ers, c.errors)
	return ers
}

// IsValid checks to see if the configuration is valid. If errors are found in
// the config the Config.Errors() will return the validation errors.
// These can be used to display to the user. See DisplayErrors for a display
// helper.
func (c *Config) IsValid() bool {
	list := make(errList, 0)
	c.protect.RLock()

	if len(c.Networks) == 0 {
		c.addError(errMsgNetworksRequired)
		c.protect.RUnlock()
		return false
	}

	c.validateNetwork(c.Network, &list, false)
	for _, n := range c.Networks {
		c.validateNetwork(n, &list, true)
	}

	c.protect.RUnlock()
	c.protect.Lock()
	defer c.protect.Unlock()
	c.errors = list

	return len(c.errors) == 0
}

// validateNetwork checks a network for errors and adds to the error collection
// if any are found.
func (c *Config) validateNetwork(n *Network, ers *errList, missingIsErr bool) {
	name := n.InName
	if len(n.InSsl) != 0 {
		if _, err := strconv.ParseBool(n.InSsl); err != nil {
			ers.addError(fmtErrInvalid, name, errSsl, n.InSsl)
		}
	}

	if len(n.InNoVerifyCert) != 0 {
		if _, err := strconv.ParseBool(n.InNoVerifyCert); err != nil {
			ers.addError(fmtErrInvalid, name, errNoVerifyCert, n.InNoVerifyCert)
		}
	}

	if len(n.InNoState) != 0 {
		if _, err := strconv.ParseBool(n.InNoState); err != nil {
			ers.addError(fmtErrInvalid, name, errNoState, n.InNoState)
		}
	}

	if len(n.InNoStore) != 0 {
		if _, err := strconv.ParseBool(n.InNoStore); err != nil {
			ers.addError(fmtErrInvalid, name, errNoStore, n.InNoStore)
		}
	}

	if len(n.InFloodLenPenalty) != 0 {
		if _, err :=
			strconv.ParseUint(n.InFloodLenPenalty, 10, 32); err != nil {
			ers.addError(fmtErrInvalid, name, errFloodLenPenalty,
				n.InFloodLenPenalty)
		}
	}

	if len(n.InFloodTimeout) != 0 {
		if _, err := strconv.ParseFloat(n.InFloodTimeout, 32); err != nil {
			ers.addError(fmtErrInvalid, name, errFloodTimeout,
				n.InFloodTimeout)
		}
	}

	if len(n.InFloodStep) != 0 {
		if _, err := strconv.ParseFloat(n.InFloodStep, 32); err != nil {
			ers.addError(fmtErrInvalid, name, errFloodStep,
				n.InFloodStep)
		}
	}

	if len(n.InKeepAlive) != 0 {
		if _, err := strconv.ParseFloat(n.InKeepAlive, 32); err != nil {
			ers.addError(fmtErrInvalid, name, errKeepAlive,
				n.InKeepAlive)
		}
	}

	if len(n.InNoReconnect) != 0 {
		if _, err := strconv.ParseBool(n.InNoReconnect); err != nil {
			ers.addError(fmtErrInvalid, name, errNoReconnect,
				n.InNoReconnect)
		}
	}

	if len(n.InReconnectTimeout) != 0 {
		_, err := strconv.ParseUint(n.InReconnectTimeout, 10, 32)
		if err != nil {
			ers.addError(fmtErrInvalid, name, errReconnectTimeout,
				n.InReconnectTimeout)
		}
	}

	if s := n.Servers(); len(s) == 0 {
		if missingIsErr {
			ers.addError(fmtErrMissing, name, errServers)
		}
	} else {
		for _, srv := range s {
			if !rgxHost.MatchString(srv) || len(srv) > maxHostSize {
				ers.addError(fmtErrInvalid, name, errServers, srv)
			}
		}
	}

	if nick := n.Nick(); len(nick) == 0 {
		if missingIsErr {
			ers.addError(fmtErrMissing, name, errNick)
		}
	} else if !rgxNickname.MatchString(nick) {
		ers.addError(fmtErrInvalid, name, errNick, nick)
	}

	if username := n.Username(); len(username) == 0 {
		if missingIsErr {
			ers.addError(fmtErrMissing, name, errUsername)
		}
	} else if !rgxUsername.MatchString(username) {
		ers.addError(fmtErrInvalid, name, errUsername, username)
	}

	if realname := n.Realname(); len(realname) == 0 {
		if missingIsErr {
			ers.addError(fmtErrMissing, name, errRealname)
		}
	} else if !rgxRealname.MatchString(realname) {
		ers.addError(fmtErrInvalid, name, errRealname, realname)
	}

	for _, channel := range n.Channels() {
		if !rgxChannel.MatchString(channel) {
			ers.addError(fmtErrInvalid, name, errChannel, channel)
		}
	}
}

// DisplayErrors is a helper function to log the output of all config to the
// standard logger.
func (c *Config) DisplayErrors() {
	c.protect.RLock()
	defer c.protect.RUnlock()

	for _, e := range c.errors {
		log.Println(e.Error())
	}
}

// GetNetwork retrieves the network by name if it exists, nil if not.
func (c *Config) GetNetwork(name string) *Network {
	c.protect.RLock()
	defer c.protect.RUnlock()

	return c.Networks[name]
}

// Filename returns fileName of the configuration, or the default.
func (c *Config) Filename() (filename string) {
	c.protect.RLock()
	defer c.protect.RUnlock()

	filename = defaultConfigFileName
	if len(c.filename) > 0 {
		filename = c.filename
	}
	return
}

// StoreFile gets the global storefile or defaultStoreFile.
func (c *Config) StoreFile() (storefile string) {
	c.protect.RLock()
	defer c.protect.RUnlock()

	storefile = defaultStoreFile
	if len(c.Storefile) > 0 {
		storefile = c.Storefile
	}
	return
}
