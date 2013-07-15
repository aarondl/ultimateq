/*
Package config provides several ways to configure an irc bot. Some methods are
inline fluent configuration, yaml reading from any io.Reader. It also provides
config validation.
*/
package config

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
)

const (
	// nAssumedServers is the typcial number of configured servers for a bot
	nAssumedServers = 1
	// defaultIrcPort is IRC Server's default tcp port.
	defaultIrcPort = uint16(6667)
	// defaultStoreFile is where the bot will store it's Store database if not
	// overridden.
	defaultStoreFile = "./store.db"
	// defaultFloodProtectBurst is how many messages can be sent before spam
	// filters set in.
	defaultFloodProtectBurst = uint(3)
	// defaultFloodProtectTimeout is how many seconds between messages before
	// the flood protection resets itself.
	defaultFloodProtectTimeout = float64(3)
	// defaultFloodProtectStep is the number of seconds between messages once
	// flood protection has been activated.
	defaultFloodProtectStep = float64(3)
	// defaultReconnectTimeout is how many seconds to wait between reconns.
	defaultReconnectTimeout = uint(20)
	// botDefaultPrefix is the command prefix by default
	defaultPrefix = '.'
	// maxHostSize is the biggest hostname possible
	maxHostSize = 255
)

// The following format strings are for formatting various config errors.
const (
	fmtErrInvalid         = "config(%v): Invalid %v, given: %v"
	fmtErrMissing         = "config(%v): Requires %v, but nothing was given."
	fmtErrServerNotFound  = "config: Server not found, given: %v"
	errMsgServersRequired = "config: At least one server is required."
	errMsgDuplicateServer = "config: Server names must be unique, use .Host()"
)

// The following is for mapping config setting names to strings
const (
	errHost                = "host"
	errPort                = "port"
	errSsl                 = "ssl"
	errNoVerifyCert        = "noverifycert"
	errNoState             = "nostate"
	errNoStore             = "nostore"
	errStoreFile           = "storefile"
	errFloodProtectBurst   = "floodprotectburst"
	errFloodProtectTimeout = "floodprotecttimeout"
	errFloodProtectStep    = "floodprotectstep"
	errNoReconnect         = "noreconnect"
	errReconnectTimeout    = "reconnecttimeout"
	errNick                = "nickname"
	errAltnick             = "alternate nickname"
	errRealname            = "realname"
	errUsername            = "username"
	errUserhost            = "userhost"
	errPrefix              = "prefix"
	errChannel             = "channel"
)

var (
	// From the RFC:
	// nickname   =  ( letter / special ) *8( letter / digit / special / "-" )
	// letter     =  %x41-5A / %x61-7A  ; A-Z / a-z
	// digit      =  %x30-39            ; 0-9
	// special    =  %x5B-60 / %x7B-7D  ; [ ] \ ` _ ^ { | }
	// We make an excemption to the 9 char limit since few servers today
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
// default settings, and server specific settings.
type Config struct {
	Servers   map[string]*Server
	Global    *Server
	context   *Server
	filename  string
	Storefile string
	Errors    []error "-"
}

// CreateConfig initializes a Config object.
func CreateConfig() *Config {
	return &Config{
		Global:  &Server{},
		Servers: make(map[string]*Server, nAssumedServers),
		Errors:  make([]error, 0),
	}
}

// Clone deep copies a configuration object.
func (c *Config) Clone() *Config {
	global := *c.Global
	newconf := &Config{
		Global:   &global,
		Servers:  make(map[string]*Server, len(c.Servers)),
		Errors:   make([]error, 0),
		filename: c.filename,
	}
	for name, srv := range c.Servers {
		newsrv := *srv
		newsrv.parent = newconf
		newconf.Servers[name] = &newsrv
	}
	return newconf
}

// addError builds an error object and returns it using Sprintf.
func (c *Config) addError(format string, args ...interface{}) {
	c.Errors = append(
		c.Errors,
		errors.New(fmt.Sprintf(format, args...)),
	)
}

// IsValid checks to see if the configuration is valid. If errors are found in
// the config the Config.Errors property is filled with the validation errors.
// These can be used to display to the user. See DisplayErrors for a display
// helper.
func (c *Config) IsValid() bool {
	if len(c.Servers) == 0 {
		c.addError(errMsgServersRequired)
		return false
	}

	c.validateServer(c.Global, false)
	for _, s := range c.Servers {
		c.validateServer(s, true)
	}

	return len(c.Errors) == 0
}

// validateServer checks a server for errors and adds to the error collection
// if any are found.
func (c *Config) validateServer(s *Server, missingIsError bool) {
	name := s.GetName()
	if len(s.Ssl) != 0 {
		if _, err := strconv.ParseBool(s.Ssl); err != nil {
			c.addError(fmtErrInvalid, name, errSsl, s.Ssl)
		}
	}

	if len(s.NoVerifyCert) != 0 {
		if _, err := strconv.ParseBool(s.NoVerifyCert); err != nil {
			c.addError(fmtErrInvalid, name, errNoVerifyCert, s.NoVerifyCert)
		}
	}

	if len(s.NoState) != 0 {
		if _, err := strconv.ParseBool(s.NoState); err != nil {
			c.addError(fmtErrInvalid, name, errNoState, s.NoState)
		}
	}

	if len(s.NoStore) != 0 {
		if _, err := strconv.ParseBool(s.NoStore); err != nil {
			c.addError(fmtErrInvalid, name, errNoStore, s.NoStore)
		}
	}

	if len(s.FloodProtectBurst) != 0 {
		if _, err :=
			strconv.ParseUint(s.FloodProtectBurst, 10, 32); err != nil {
			c.addError(fmtErrInvalid, name, errFloodProtectBurst,
				s.FloodProtectBurst)
		}
	}

	if len(s.FloodProtectTimeout) != 0 {
		if _, err := strconv.ParseFloat(s.FloodProtectTimeout, 32); err != nil {
			c.addError(fmtErrInvalid, name, errFloodProtectTimeout,
				s.FloodProtectTimeout)
		}
	}

	if len(s.FloodProtectStep) != 0 {
		if _, err := strconv.ParseFloat(s.FloodProtectStep, 32); err != nil {
			c.addError(fmtErrInvalid, name, errFloodProtectStep,
				s.FloodProtectStep)
		}
	}

	if len(s.NoReconnect) != 0 {
		if _, err := strconv.ParseBool(s.NoReconnect); err != nil {
			c.addError(fmtErrInvalid, name, errNoReconnect,
				s.NoReconnect)
		}
	}

	if len(s.ReconnectTimeout) != 0 {
		if _, err := strconv.ParseUint(s.ReconnectTimeout, 10, 32); err != nil {
			c.addError(fmtErrInvalid, name, errReconnectTimeout,
				s.ReconnectTimeout)
		}
	}

	if host := s.GetHost(); len(host) == 0 {
		if missingIsError {
			c.addError(fmtErrMissing, name, errHost)
		}
	} else if !rgxHost.MatchString(host) || len(host) > maxHostSize {
		c.addError(fmtErrInvalid, name, errHost, host)
	}

	if nick := s.GetNick(); len(nick) == 0 {
		if missingIsError {
			c.addError(fmtErrMissing, name, errNick)
		}
	} else if !rgxNickname.MatchString(nick) {
		c.addError(fmtErrInvalid, name, errNick, nick)
	}

	if username := s.GetUsername(); len(username) == 0 {
		if missingIsError {
			c.addError(fmtErrMissing, name, errUsername)
		}
	} else if !rgxUsername.MatchString(username) {
		c.addError(fmtErrInvalid, name, errUsername, username)
	}

	if userhost := s.GetUserhost(); len(userhost) == 0 {
		if missingIsError {
			c.addError(fmtErrMissing, name, errUserhost)
		}
	} else if !rgxHost.MatchString(userhost) {
		c.addError(fmtErrInvalid, name, errUserhost, userhost)
	}

	if realname := s.GetRealname(); len(realname) == 0 {
		if missingIsError {
			c.addError(fmtErrMissing, name, errRealname)
		}
	} else if !rgxRealname.MatchString(realname) {
		c.addError(fmtErrInvalid, name, errRealname, realname)
	}

	for _, channel := range s.GetChannels() {
		if !rgxChannel.MatchString(channel) {
			c.addError(fmtErrInvalid, name, errChannel, channel)
		}
	}
}

// DisplayErrors is a helper function to log the output of all config to the
// standard logger.
func (c *Config) DisplayErrors() {
	for _, e := range c.Errors {
		log.Println(e.Error())
	}
}

// GlobalContext clears the configs server context
func (c *Config) GlobalContext() *Config {
	c.context = nil
	return c
}

// ServerContext the configs server context, adds an error if the if server
// key is not found.
func (c *Config) ServerContext(name string) *Config {
	if srv, ok := c.Servers[name]; ok {
		c.context = srv
	} else {
		c.addError(fmtErrServerNotFound, name)
	}
	return c
}

// GetContext retrieves the current configuration context, if no context has
// been set, returns the global setting object.
func (c *Config) GetContext() *Server {
	if c.context != nil {
		return c.context
	}
	return c.Global
}

// GetServer retrieves the server by name if it exists, nil if not.
func (c *Config) GetServer(name string) *Server {
	return c.Servers[name]
}

// Server fluently creates a server object and sets the context on the Config to
// the current instance. This automatically sets the Host() parameter to the
// same thing. If you have multiple servers connecting to the same host, you
// will have to use this to name the server, and Host() to set the host.
func (c *Config) Server(name string) *Config {
	if len(name) != 0 {
		if _, ok := c.Servers[name]; !ok {
			c.context = &Server{parent: c, Name: name, Host: name}
			c.Servers[name] = c.context
		} else {
			c.addError(errMsgDuplicateServer)
		}
	} else {
		c.addError(fmtErrMissing, "<NONE>", errHost)
	}
	return c
}

// RemoveServer removes a server by name. Note that this does not work on
// host if a name has been set on the server.
func (c *Config) RemoveServer(name string) (deleted bool) {
	if _, deleted := c.Servers[name]; deleted {
		delete(c.Servers, name)
		c.context = nil
	}
	return
}

// Host fluently sets the host for the current config context
func (c *Config) Host(host string) *Config {
	if c.context != nil {
		c.context.Host = host
	}
	return c
}

// Port fluently sets the port for the current config context
func (c *Config) Port(port uint16) *Config {
	c.GetContext().Port = port
	return c
}

// Ssl fluently sets the ssl for the current config context
func (c *Config) Ssl(ssl bool) *Config {
	c.GetContext().Ssl = strconv.FormatBool(ssl)
	return c
}

// SslCert sets a filename that will be read in (pem format)
// to verify the server's certificate.
func (c *Config) SslCert(cert string) *Config {
	c.GetContext().SslCert = cert
	return c
}

// NoVerifyCert fluently sets the noverifyCert for the current config context
func (c *Config) NoVerifyCert(noverifycert bool) *Config {
	c.GetContext().NoVerifyCert = strconv.FormatBool(noverifycert)
	return c
}

// NoState fluently sets reconnection for the current config context,
// this turns off the irc state database (data package).
func (c *Config) NoState(nostate bool) *Config {
	c.GetContext().NoState = strconv.FormatBool(nostate)
	return c
}

// NoStore fluently sets reconnection for the current config context,
// this turns off the irc store database (data package).
func (c *Config) NoStore(nostore bool) *Config {
	c.GetContext().NoStore = strconv.FormatBool(nostore)
	return c
}

// FloodProtectBurst fluently sets flood burst for the current config context,
// this is how many messages will be bursted through without enabling flood
// protection.
func (c *Config) FloodProtectBurst(floodburst uint) *Config {
	c.GetContext().FloodProtectBurst =
		strconv.FormatUint(uint64(floodburst), 10)
	return c
}

// FloodProtectTimeout fluently sets flood timeout for the current config
// context, this is how long flood protect will stay enabled after being
// enabled.
func (c *Config) FloodProtectTimeout(floodtimeout float64) *Config {
	c.GetContext().FloodProtectTimeout =
		strconv.FormatFloat(floodtimeout, 'e', -1, 64)
	return c
}

// FloodProtectStep fluently sets flood protect step for the current config
// context, this is how many seconds to put in between messages after flood
// protect has been activated (after FloodProtectBurst messages).
func (c *Config) FloodProtectStep(floodstep float64) *Config {
	c.GetContext().FloodProtectStep =
		strconv.FormatFloat(floodstep, 'e', -1, 64)
	return c
}

// NoReconnect fluently sets reconnection for the current config context
func (c *Config) NoReconnect(noreconnect bool) *Config {
	c.GetContext().NoReconnect = strconv.FormatBool(noreconnect)
	return c
}

// ReconnectTimeout fluently sets the port for the current config context
func (c *Config) ReconnectTimeout(seconds uint) *Config {
	c.GetContext().ReconnectTimeout = strconv.FormatUint(uint64(seconds), 10)
	return c
}

// Nick fluently sets the nick for the current config context
func (c *Config) Nick(nick string) *Config {
	c.GetContext().Nick = nick
	return c
}

// Altnick fluently sets the altnick for the current config context
func (c *Config) Altnick(altnick string) *Config {
	c.GetContext().Altnick = altnick
	return c
}

// Username fluently sets the username for the current config context
func (c *Config) Username(username string) *Config {
	c.GetContext().Username = username
	return c
}

// Userhost fluently sets the userhost for the current config context
func (c *Config) Userhost(userhost string) *Config {
	c.GetContext().Userhost = userhost
	return c
}

// Realname fluently sets the realname for the current config context
func (c *Config) Realname(realname string) *Config {
	c.GetContext().Realname = realname
	return c
}

// Prefix fluently sets the prefix for the current config context
func (c *Config) Prefix(prefix string) *Config {
	c.GetContext().Prefix = prefix
	return c
}

// Channels fluently sets the channels for the current config context
func (c *Config) Channels(channels ...string) *Config {
	if len(channels) > 0 {
		context := c.GetContext()
		context.Channels = make([]string, len(channels))
		copy(context.Channels, channels)
	}
	return c
}

// Server states the all the details necessary to connect to an irc server
// Although all of these are exported so they can be deserialized into a yaml
// file, they are not for direct reading and the helper methods should ALWAYS
// be used to preserve correct global-value resolution.
type Server struct {
	parent *Config

	// Name of this connection
	Name string

	// Irc Server connection info
	Host string
	Port uint16

	// Ssl configuration
	Ssl          string
	SslCert      string
	NoVerifyCert string

	// State tracking
	NoState string
	NoStore string

	// Flood Protection
	FloodProtectBurst   string
	FloodProtectTimeout string
	FloodProtectStep    string

	// Auto reconnection
	NoReconnect      string
	ReconnectTimeout string

	// Irc User data
	Nick     string
	Altnick  string
	Username string
	Userhost string
	Realname string

	// Dispatching options
	Prefix   string
	Channels []string
}

// GetFilename returns fileName of the configuration, or the default.
func (c *Config) GetFilename() (filename string) {
	filename = defaultConfigFileName
	if len(c.filename) > 0 {
		filename = c.filename
	}
	return
}

// StoreFile fluently sets the storefile for the global context.
func (c *Config) StoreFile(storefile string) *Config {
	c.Storefile = storefile
	return c
}

// GetStoreFile gets the global storefile or defaultStoreFile.
func (c *Config) GetStoreFile() (storefile string) {
	storefile = defaultStoreFile
	if len(c.Storefile) > 0 {
		storefile = c.Storefile
	}
	return
}

// GetHost gets s.host
func (s *Server) GetHost() string {
	return s.Host
}

// GetName gets s.name
func (s *Server) GetName() string {
	return s.Name
}

// GetPort returns gets Port of the server, or the global port, or
// ircDefaultPort
func (s *Server) GetPort() (port uint16) {
	port = defaultIrcPort
	if s.Port != 0 {
		port = s.Port
	} else if s.parent != nil && s.parent.Global.Port != 0 {
		port = s.parent.Global.Port
	}
	return
}

// GetSsl returns Ssl of the server, or the global ssl, or false
func (s *Server) GetSsl() (ssl bool) {
	var err error
	if len(s.Ssl) != 0 {
		ssl, err = strconv.ParseBool(s.Ssl)
	} else if s.parent != nil && len(s.parent.Global.Ssl) != 0 {
		ssl, err = strconv.ParseBool(s.parent.Global.Ssl)
	}

	if err != nil {
		ssl = false
	}
	return
}

// GetSslCert returns the path to the certificate used when connecting.
func (s *Server) GetSslCert() (cert string) {
	if len(s.SslCert) > 0 {
		cert = s.SslCert
	} else if s.parent != nil && len(s.parent.Global.SslCert) > 0 {
		cert = s.parent.Global.SslCert
	}
	return
}

// GetNoVerifyCert gets NoVerifyCert of the server, or the global verifyCert, or
// false
func (s *Server) GetNoVerifyCert() (noverifyCert bool) {
	var err error
	if len(s.NoVerifyCert) != 0 {
		noverifyCert, err = strconv.ParseBool(s.NoVerifyCert)
	} else if s.parent != nil && len(s.parent.Global.NoVerifyCert) != 0 {
		noverifyCert, err = strconv.ParseBool(s.parent.Global.NoVerifyCert)
	}

	if err != nil {
		noverifyCert = false
	}
	return
}

// GetNoState gets NoState of the server, or the global nostate, or
// false
func (s *Server) GetNoState() (nostate bool) {
	var err error
	if len(s.NoState) != 0 {
		nostate, err = strconv.ParseBool(s.NoState)
	} else if s.parent != nil && len(s.parent.Global.NoState) != 0 {
		nostate, err = strconv.ParseBool(s.parent.Global.NoState)
	}

	if err != nil {
		nostate = false
	}
	return
}

// GetNoStore gets NoStore of the server, or the global nostore, or
// false
func (s *Server) GetNoStore() (nostore bool) {
	var err error
	if len(s.NoStore) != 0 {
		nostore, err = strconv.ParseBool(s.NoStore)
	} else if s.parent != nil && len(s.parent.Global.NoStore) != 0 {
		nostore, err = strconv.ParseBool(s.parent.Global.NoStore)
	}

	if err != nil {
		nostore = false
	}
	return
}

// GetNoReconnect gets NoReconnect of the server, or the global noReconnect, or
// false
func (s *Server) GetNoReconnect() (noReconnect bool) {
	var err error
	if len(s.NoReconnect) != 0 {
		noReconnect, err = strconv.ParseBool(s.NoReconnect)
	} else if s.parent != nil && len(s.parent.Global.NoReconnect) != 0 {
		noReconnect, err = strconv.ParseBool(s.parent.Global.NoReconnect)
	}

	if err != nil {
		noReconnect = false
	}
	return
}

// GetFloodProtectBurst gets FloodProtectBurst of the server, or the global
// floodProtectBurst, or defaultFloodProtectBurst
func (s *Server) GetFloodProtectBurst() (floodBurst uint) {
	var err error
	var u uint64
	var notset bool
	floodBurst = defaultFloodProtectBurst
	if len(s.FloodProtectBurst) != 0 {
		u, err = strconv.ParseUint(s.FloodProtectBurst, 10, 32)
	} else if s.parent != nil && len(s.parent.Global.FloodProtectBurst) != 0 {
		u, err = strconv.ParseUint(s.parent.Global.FloodProtectBurst, 10, 32)
	} else {
		notset = true
	}

	if err != nil {
		floodBurst = defaultFloodProtectBurst
	} else if !notset {
		floodBurst = uint(u)
	}
	return
}

// GetFloodProtectTimeout gets FloodProtectTimeout of the server, or the global
// floodProtectTimeout, or defaultFloodProtectTimeout
func (s *Server) GetFloodProtectTimeout() (floodTimeout float64) {
	var err error
	floodTimeout = defaultFloodProtectTimeout
	if len(s.FloodProtectTimeout) != 0 {
		floodTimeout, err =
			strconv.ParseFloat(s.FloodProtectTimeout, 32)
	} else if s.parent != nil && len(s.parent.Global.FloodProtectTimeout) != 0 {
		floodTimeout, err =
			strconv.ParseFloat(s.parent.Global.FloodProtectTimeout, 32)
	}

	if err != nil {
		floodTimeout = defaultFloodProtectTimeout
	}
	return
}

// GetFloodProtectStep gets FloodProtectStep of the server, or the global
// floodProtectStep, or defaultFloodProtectStep
func (s *Server) GetFloodProtectStep() (floodStep float64) {
	var err error
	floodStep = defaultFloodProtectStep
	if len(s.FloodProtectStep) != 0 {
		floodStep, err =
			strconv.ParseFloat(s.FloodProtectStep, 32)
	} else if s.parent != nil && len(s.parent.Global.FloodProtectStep) != 0 {
		floodStep, err =
			strconv.ParseFloat(s.parent.Global.FloodProtectStep, 32)
	}

	if err != nil {
		floodStep = defaultFloodProtectStep
	}
	return
}

// GetReconnectTimeout gets ReconnectTimeout of the server, or the global
// reconnectTimeout, or defaultReconnectTimeout
func (s *Server) GetReconnectTimeout() (reconnTimeout uint) {
	var notset bool
	var err error
	var u uint64
	reconnTimeout = defaultReconnectTimeout
	if len(s.ReconnectTimeout) != 0 {
		u, err = strconv.ParseUint(s.ReconnectTimeout, 10, 32)
	} else if s.parent != nil && len(s.parent.Global.ReconnectTimeout) != 0 {
		u, err = strconv.ParseUint(s.parent.Global.ReconnectTimeout, 10, 32)
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

// GetNick gets Nick of the server, or the global nick, or empty string.
func (s *Server) GetNick() (nick string) {
	if len(s.Nick) > 0 {
		nick = s.Nick
	} else if s.parent != nil && len(s.parent.Global.Nick) > 0 {
		nick = s.parent.Global.Nick
	}
	return
}

// GetAltnick gets Altnick of the server, or the global altnick, or empty
// string.
func (s *Server) GetAltnick() (altnick string) {
	if len(s.Altnick) > 0 {
		altnick = s.Altnick
	} else if s.parent != nil && len(s.parent.Global.Altnick) > 0 {
		altnick = s.parent.Global.Altnick
	}
	return
}

// GetUsername gets Username of the server, or the global username, or empty
// string.
func (s *Server) GetUsername() (username string) {
	if len(s.Username) > 0 {
		username = s.Username
	} else if s.parent != nil && len(s.parent.Global.Username) > 0 {
		username = s.parent.Global.Username
	}
	return
}

// GetUserhost gets Userhost of the server, or the global userhost, or empty
// string.
func (s *Server) GetUserhost() (userhost string) {
	if len(s.Userhost) > 0 {
		userhost = s.Userhost
	} else if s.parent != nil && len(s.parent.Global.Userhost) > 0 {
		userhost = s.parent.Global.Userhost
	}
	return
}

// GetRealname gets Realname of the server, or the global realname, or empty
// string.
func (s *Server) GetRealname() (realname string) {
	if len(s.Realname) > 0 {
		realname = s.Realname
	} else if s.parent != nil && len(s.parent.Global.Realname) > 0 {
		realname = s.parent.Global.Realname
	}
	return
}

// GetPrefix gets Prefix of the server, or the global prefix, or defaultPrefix.
func (s *Server) GetPrefix() (prefix rune) {
	prefix = defaultPrefix
	if len(s.Prefix) > 0 {
		prefix = rune(s.Prefix[0])
	} else if s.parent != nil && len(s.parent.Global.Prefix) > 0 {
		prefix = rune(s.parent.Global.Prefix[0])
	}
	return
}

// GetChannels gets Channels of the server, or the global channels, or nil
// slice of string (check the length!).
func (s *Server) GetChannels() (channels []string) {
	if len(s.Channels) > 0 {
		channels = s.Channels
	} else if s.parent != nil && len(s.parent.Global.Channels) > 0 {
		channels = s.parent.Global.Channels
	}
	return
}
