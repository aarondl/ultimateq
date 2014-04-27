/*
Package config creates a configuration using yaml. It also does configuration
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
		`^(?i)[#&+!][^\n\000\007,]{1,49}$`)

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
	*Network  `yaml:"global" json:"global"`
	Networks  map[string]*Network `yaml:"networks" json:"networks"`
	Storefile string              `yaml:"storefile" json:"storefile"`

	errors   errList      `yaml:"-" json:"-"`
	filename string       `yaml:"-" json:"-"`
	protect  sync.RWMutex `yaml:"-" json:"-"`
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

// Network states the all the details necessary to connect to an irc network
// Although all of these are exported so they can be deserialized into a yaml
// file. The fields are all marked with "In" as in "Internal" and should not
// be accessed directly, but through their appropriately named helper methods.
type Network struct {
	parent  *Config       `yaml:"-" json:"-"`
	protect *sync.RWMutex `yaml:"-" json:"-"`

	// Name of this network
	InName string `yaml:"-" json:"-"`

	// Irc Network connection info
	InServers []string `yaml:"servers" json:"servers"`
	InPort    uint16   `yaml:"port" json:"port"`

	// Ssl configuration
	InSsl          string `yaml:"ssl" json:"ssl"`
	InSslCert      string `yaml:"sslcert" json:"sslcert"`
	InNoVerifyCert string `yaml:"noverifycert" json:"noverifycert"`

	// State tracking
	InNoState string `yaml:"nostate" json:"nostate"`
	InNoStore string `yaml:"nostore" json:"nostore"`

	// Flood Protection
	InFloodLenPenalty string `yaml:"floodlenpenalty" json:"floodlenpenalty"`
	InFloodTimeout    string `yaml:"floodtimeout" json:"floodtimeout"`
	InFloodStep       string `yaml:"floodstep" json:"floodstep"`

	// Keep alive
	InKeepAlive string `yaml:"keepalive" json:"keepalive"`

	// Auto reconnectiong
	InNoReconnect      string `yaml:"noreconnect" json:"noreconnect"`
	InReconnectTimeout string `yaml:"reconnecttimeout" json:"reconnecttimeout"`

	// Irc User data
	InNick     string `yaml:"nick" json:"nick"`
	InAltnick  string `yaml:"altnick" json:"altnick"`
	InUsername string `yaml:"username" json:"username"`
	InUserhost string `yaml:"userhost" json:"userhost"`
	InRealname string `yaml:"realname" json:"realname"`
	InPassword string `yaml:"password" json:"password"`

	// Dispatching options
	InPrefix   string   `yaml:"prefix" json:"prefix"`
	InChannels []string `yaml:"channels" json:"channels"`
}

// Clone clones a network.
func (n *Network) Clone() *Network {
	n.protect.RLock()
	defer n.protect.RUnlock()

	newNet := *n
	newNet.InChannels = make([]string, len(n.InChannels))
	newNet.InServers = make([]string, len(n.InServers))
	copy(newNet.InChannels, n.InChannels)
	copy(newNet.InServers, n.InServers)

	return &newNet
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

// Name gets the network's name.
func (n *Network) Name() string {
	n.protect.RLock()
	defer n.protect.RUnlock()

	return n.InName
}

// Servers gets the network's server list.
func (n *Network) Servers() []string {
	n.protect.RLock()
	defer n.protect.RUnlock()

	if len(n.InServers) == 0 {
		return []string{n.InName}
	}

	servers := make([]string, len(n.InServers))
	copy(servers, n.InServers)
	return servers
}

// Port returns gets Port of the network, or the global port, or
// ircDefaultPort
func (n *Network) Port() (port uint16) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	port = defaultIrcPort
	if n.InPort != 0 {
		port = n.InPort
	} else if n.parent != nil && n.parent.InPort != 0 {
		port = n.parent.InPort
	}
	return
}

// Ssl returns Ssl of the network, or the global ssl, or false
func (n *Network) Ssl() (ssl bool) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	var err error
	if len(n.InSsl) != 0 {
		ssl, err = strconv.ParseBool(n.InSsl)
	} else if n.parent != nil && len(n.parent.InSsl) != 0 {
		ssl, err = strconv.ParseBool(n.parent.InSsl)
	}

	if err != nil {
		ssl = false
	}
	return
}

// SslCert returns the path to the certificate used when connecting.
func (n *Network) SslCert() (cert string) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	if len(n.InSslCert) > 0 {
		cert = n.InSslCert
	} else if n.parent != nil && len(n.parent.InSslCert) > 0 {
		cert = n.parent.InSslCert
	}
	return
}

// NoVerifyCert gets NoVerifyCert of the network, or the global verifyCert, or
// false
func (n *Network) NoVerifyCert() (noverifyCert bool) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	var err error
	if len(n.InNoVerifyCert) != 0 {
		noverifyCert, err = strconv.ParseBool(n.InNoVerifyCert)
	} else if n.parent != nil && len(n.parent.InNoVerifyCert) != 0 {
		noverifyCert, err = strconv.ParseBool(n.parent.InNoVerifyCert)
	}

	if err != nil {
		noverifyCert = false
	}
	return
}

// NoState gets NoState of the network, or the global nostate, or
// false
func (n *Network) NoState() (nostate bool) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	var err error
	if len(n.InNoState) != 0 {
		nostate, err = strconv.ParseBool(n.InNoState)
	} else if n.parent != nil && len(n.parent.InNoState) != 0 {
		nostate, err = strconv.ParseBool(n.parent.InNoState)
	}

	if err != nil {
		nostate = false
	}
	return
}

// NoStore gets NoStore of the network, or the global nostore, or
// false
func (n *Network) NoStore() (nostore bool) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	var err error
	if len(n.InNoStore) != 0 {
		nostore, err = strconv.ParseBool(n.InNoStore)
	} else if n.parent != nil && len(n.parent.InNoStore) != 0 {
		nostore, err = strconv.ParseBool(n.parent.InNoStore)
	}

	if err != nil {
		nostore = false
	}
	return
}

// NoReconnect gets NoReconnect of the network, or the global noReconnect, or
// false
func (n *Network) NoReconnect() (noReconnect bool) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	var err error
	if len(n.InNoReconnect) != 0 {
		noReconnect, err = strconv.ParseBool(n.InNoReconnect)
	} else if n.parent != nil && len(n.parent.InNoReconnect) != 0 {
		noReconnect, err = strconv.ParseBool(n.parent.InNoReconnect)
	}

	if err != nil {
		noReconnect = false
	}
	return
}

// FloodLenPenalty gets FloodLenPenalty of the network, or the global
// floodLenPenalty, or defaultFloodLenPenalty
func (n *Network) FloodLenPenalty() (floodLenPenalty uint) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	var err error
	var u uint64
	var notset bool
	floodLenPenalty = defaultFloodLenPenalty
	if len(n.InFloodLenPenalty) != 0 {
		u, err = strconv.ParseUint(n.InFloodLenPenalty, 10, 32)
	} else if n.parent != nil && len(n.parent.InFloodLenPenalty) != 0 {
		u, err = strconv.ParseUint(n.parent.InFloodLenPenalty, 10, 32)
	} else {
		notset = true
	}

	if err != nil {
		floodLenPenalty = defaultFloodLenPenalty
	} else if !notset {
		floodLenPenalty = uint(u)
	}
	return
}

// FloodTimeout gets FloodTimeout of the network, or the global
// floodTimeout, or defaultFloodTimeout
func (n *Network) FloodTimeout() (floodTimeout float64) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	var err error
	floodTimeout = defaultFloodTimeout
	if len(n.InFloodTimeout) != 0 {
		floodTimeout, err = strconv.ParseFloat(n.InFloodTimeout, 32)
	} else if n.parent != nil && len(n.parent.InFloodTimeout) != 0 {
		floodTimeout, err = strconv.ParseFloat(n.parent.InFloodTimeout, 32)
	}

	if err != nil {
		floodTimeout = defaultFloodTimeout
	}
	return
}

// FloodStep gets FloodStep of the network, or the global
// floodStep, or defaultFloodStep
func (n *Network) FloodStep() (floodStep float64) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	var err error
	floodStep = defaultFloodStep
	if len(n.InFloodStep) != 0 {
		floodStep, err = strconv.ParseFloat(n.InFloodStep, 32)
	} else if n.parent != nil && len(n.parent.InFloodStep) != 0 {
		floodStep, err = strconv.ParseFloat(n.parent.InFloodStep, 32)
	}

	if err != nil {
		floodStep = defaultFloodStep
	}
	return
}

// KeepAlive gets KeepAlive of the network, or the global keepAlive,
// or defaultKeepAlive.
func (n *Network) KeepAlive() (keepAlive float64) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	var err error
	keepAlive = defaultKeepAlive
	if len(n.InKeepAlive) != 0 {
		keepAlive, err = strconv.ParseFloat(n.InKeepAlive, 32)
	} else if n.parent != nil && len(n.parent.InKeepAlive) != 0 {
		keepAlive, err = strconv.ParseFloat(n.parent.InKeepAlive, 32)
	}

	if err != nil {
		keepAlive = defaultKeepAlive
	}
	return
}

// ReconnectTimeout gets ReconnectTimeout of the network, or the global
// reconnectTimeout, or defaultReconnectTimeout
func (n *Network) ReconnectTimeout() (reconnTimeout uint) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	var notset bool
	var err error
	var u uint64
	reconnTimeout = defaultReconnectTimeout
	if len(n.InReconnectTimeout) != 0 {
		u, err = strconv.ParseUint(n.InReconnectTimeout, 10, 32)
	} else if n.parent != nil && len(n.parent.InReconnectTimeout) != 0 {
		u, err = strconv.ParseUint(n.parent.InReconnectTimeout, 10, 32)
	} else {
		notset = true
	}

	if err != nil {
		reconnTimeout = defaultReconnectTimeout
	} else if !notset {
		reconnTimeout = uint(u)
	}
	return
}

// Nick gets Nick of the network, or the global nick, or empty string.
func (n *Network) Nick() (nick string) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	if len(n.InNick) > 0 {
		nick = n.InNick
	} else if n.parent != nil && len(n.parent.InNick) > 0 {
		nick = n.parent.InNick
	}
	return
}

// Altnick gets Altnick of the network, or the global nick, or empty string.
func (n *Network) Altnick() (altnick string) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	if len(n.InAltnick) > 0 {
		altnick = n.InAltnick
	} else if n.parent != nil && len(n.parent.InAltnick) > 0 {
		altnick = n.parent.InAltnick
	}
	return
}

// Username gets Username of the network, or the global username, or empty
// string.
func (n *Network) Username() (username string) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	if len(n.InUsername) > 0 {
		username = n.InUsername
	} else if n.parent != nil && len(n.parent.InUsername) > 0 {
		username = n.parent.InUsername
	}
	return
}

// Userhost gets Userhost of the network, or the global userhost, or empty
// string.
func (n *Network) Userhost() (userhost string) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	if len(n.InUserhost) > 0 {
		userhost = n.InUserhost
	} else if n.parent != nil && len(n.parent.InUserhost) > 0 {
		userhost = n.parent.InUserhost
	}
	return
}

// Realname gets Realname of the network, or the global realname, or empty
// string.
func (n *Network) Realname() (realname string) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	if len(n.InRealname) > 0 {
		realname = n.InRealname
	} else if n.parent != nil && len(n.parent.InRealname) > 0 {
		realname = n.parent.InRealname
	}
	return
}

// Password gets password of the network, or the global password,
// or empty string.
func (n *Network) Password() (password string) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	if len(n.InPassword) > 0 {
		password = n.InPassword
	} else if n.parent != nil && len(n.parent.InPassword) > 0 {
		password = n.parent.InPassword
	}
	return
}

// Prefix gets Prefix of the network, or the global prefix, or defaultPrefix.
func (n *Network) Prefix() (prefix rune) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	prefix = defaultPrefix
	if len(n.InPrefix) > 0 {
		prefix = rune(n.InPrefix[0])
	} else if n.parent != nil && len(n.parent.InPrefix) > 0 {
		prefix = rune(n.parent.InPrefix[0])
	}
	return
}

// Channels gets Channels of the network, or the global channels, or nil
// slice of string (check the length!).
func (n *Network) Channels() (channels []string) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	if len(n.InChannels) > 0 {
		channels = make([]string, len(n.InChannels))
		copy(channels, n.InChannels)
	} else if n.parent != nil && len(n.parent.InChannels) > 0 {
		channels = make([]string, len(n.parent.InChannels))
		copy(channels, n.parent.InChannels)
	}
	return
}
