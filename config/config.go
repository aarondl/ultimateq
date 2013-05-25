/*
config package provides inline fluent configuration, validation, file
input/output, as well as configuration diffs.
*/
package config

import (
	"errors"
	"fmt"
	"log"
	"regexp"
)

const (
	// nAssumedServers is the typcial number of configured servers for a bot
	nAssumedServers = 1
	// defaultIrcPort is IRC Server's default tcp port.
	defaultIrcPort = 6667
	// defaultReconnectTimeout is how many seconds to wait between reconns.
	defaultReconnectTimeout = 20
	// botDefaultPrefix is the command prefix by default
	defaultPrefix = "."
	// maxHostSize is the biggest hostname possible
	maxHostSize = 255

	// The following format strings are for formatting various config errors.
	fmtErrInvalid         = "config(%v): Invalid %v, given: %v"
	fmtErrMissing         = "config(%v): Requires %v, but nothing was given."
	errMsgServersRequired = "config: At least one server is required."
	errMsgDuplicateServer = "config: Server names must be unique, use .Host()"

	// The following is for mapping config setting names to strings
	errHost     = "host"
	errPort     = "port"
	errNick     = "nickname"
	errAltnick  = "alternate nickname"
	errRealname = "realname"
	errUsername = "username"
	errUserhost = "userhost"
	errPrefix   = "prefix"
	errChannel  = "channel"
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
	Servers map[string]*Server
	Global  *Server
	context *Server
	Errors  []error "-"
}

// CreateConfig initializes a Config object.
func CreateConfig() *Config {
	return &Config{
		Global:  &Server{},
		Servers: make(map[string]*Server, nAssumedServers),
		Errors:  make([]error, 0),
	}
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

	for _, s := range c.Servers {
		name := s.GetName()
		if host := s.GetHost(); len(host) == 0 {
			c.addError(fmtErrMissing, name, errHost)
		} else if !rgxHost.MatchString(host) || len(host) > maxHostSize {
			c.addError(fmtErrInvalid, name, errHost, host)
		}

		if nick := s.GetNick(); len(nick) == 0 {
			c.addError(fmtErrMissing, name, errNick)
		} else if !rgxNickname.MatchString(nick) {
			c.addError(fmtErrInvalid, name, errNick, nick)
		}

		if username := s.GetUsername(); len(username) == 0 {
			c.addError(fmtErrMissing, name, errUsername)
		} else if !rgxUsername.MatchString(username) {
			c.addError(fmtErrInvalid, name, errUsername, username)
		}

		if userhost := s.GetUserhost(); len(userhost) == 0 {
			c.addError(fmtErrMissing, name, errUserhost)
		} else if !rgxHost.MatchString(userhost) {
			c.addError(fmtErrInvalid, name, errUserhost, userhost)
		}

		if realname := s.GetRealname(); len(realname) == 0 {
			c.addError(fmtErrMissing, name, errRealname)
		} else if !rgxRealname.MatchString(realname) {
			c.addError(fmtErrInvalid, name, errRealname, realname)
		}

		for _, channel := range s.GetChannels() {
			if !rgxChannel.MatchString(channel) {
				c.addError(fmtErrInvalid, name, errChannel, channel)
			}
		}
	}
	return len(c.Errors) == 0
}

// DisplayErrors is a helper function to log the output of all config to the
// standard logger.
func (c *Config) DisplayErrors() {
	for _, e := range c.Errors {
		log.Println(e.Error())
	}
}

// Gets the current configuration context, if no context has been set, returns
// the global instance.
func (c *Config) GetContext() *Server {
	if c.context != nil {
		return c.context
	}
	return c.Global
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
	context := c.GetContext()
	context.Ssl = ssl
	context.IsSslSet = true
	return c
}

// VerifyCert fluently sets the verifyCert for the current config context
func (c *Config) VerifyCert(verifyCert bool) *Config {
	i := c.GetContext()
	i.VerifyCert = verifyCert
	i.IsVerifyCertSet = true
	return c
}

// NoReconnect fluently sets reconnection for the current config context
func (c *Config) NoReconnect(noreconnect bool) *Config {
	context := c.GetContext()
	context.NoReconnect = noreconnect
	context.IsNoReconnectSet = true
	return c
}

// ReconnectTimeout fluently sets the port for the current config context
func (c *Config) ReconnectTimeout(seconds uint) *Config {
	c.GetContext().ReconnectTimeout = seconds
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

// ServerConfig stores the all the details necessary to connect to an irc server
// Although all of these are exported so they can be deserialized into a yaml
// file, they are not for direct reading and the helper methods should ALWAYS
// be used to preserve correct global-value resolution.
type Server struct {
	parent *Config

	// Name of this connection
	Name string

	// Irc Server connection info
	Host            string
	Port            uint16
	Ssl             bool
	IsSslSet        bool
	VerifyCert      bool
	IsVerifyCertSet bool

	// Auto reconnection
	NoReconnect      bool
	IsNoReconnectSet bool
	ReconnectTimeout uint

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

// GetHost gets s.host
func (s *Server) GetHost() string {
	return s.Host
}

// GetName gets s.name
func (s *Server) GetName() string {
	return s.Name
}

// GetPort returns port of the irc config, if it hasn't been set, returns the
// value of the global, if that hasn't been set returns ircDefaultPort.
func (s *Server) GetPort() uint16 {
	if s.Port != 0 {
		return s.Port
	} else if s.parent != nil && s.parent.Global != nil &&
		s.parent.Global.Port != 0 {

		return s.parent.Global.Port
	}
	return defaultIrcPort
}

// GetSsl returns ssl of the irc config, if it hasn't been set, returns the
// value of the global, if that hasn't been set returns false.
func (s *Server) GetSsl() bool {
	if s.IsSslSet {
		return s.Ssl
	} else if s.parent != nil && s.parent.Global != nil {
		return s.parent.Global.IsSslSet && s.parent.Global.Ssl
	}
	return false
}

// GetSsl returns verifyCert of the irc config, if it hasn't been set, returns
// the value of the global, if that hasn't been set returns false.
func (s *Server) GetVerifyCert() bool {
	if s.IsVerifyCertSet {
		return s.VerifyCert
	} else if s.parent != nil && s.parent.Global != nil {
		return s.parent.Global.IsVerifyCertSet && s.parent.Global.VerifyCert
	}
	return false
}

// GetNoReconnect returns verifyCert of the irc config, if it hasn't been
// set, returns the value of the global, if that hasn't been set returns false.
func (s *Server) GetNoReconnect() bool {
	if s.IsNoReconnectSet {
		return s.NoReconnect
	} else if s.parent != nil && s.parent.Global != nil {
		return s.parent.Global.IsNoReconnectSet && s.parent.Global.NoReconnect
	}
	return false
}

// GetPort returns port of the irc config, if it hasn't been set, returns the
// value of the global, if that hasn't been set returns ircDefaultPort.
func (s *Server) GetReconnectTimeout() uint {
	if s.ReconnectTimeout != 0 {
		return s.ReconnectTimeout
	} else if s.parent != nil && s.parent.Global != nil &&
		s.parent.Global.ReconnectTimeout != 0 {

		return s.parent.Global.ReconnectTimeout
	}
	return defaultReconnectTimeout
}

// GetNick returns the nickname of the irc config, if it's empty, it returns the
// value of the global configuration.
func (s *Server) GetNick() string {
	if len(s.Nick) == 0 &&
		s.parent != nil && s.parent.Global != nil {

		return s.parent.Global.Nick
	}
	return s.Nick
}

// GetAltnick returns the altnick of the irc config, if it's empty, it returns
// the value of the global configuration.
func (s *Server) GetAltnick() string {
	if len(s.Altnick) == 0 &&
		s.parent != nil && s.parent.Global != nil {

		return s.parent.Global.Altnick
	}
	return s.Altnick
}

// GetUsername returns the username of the irc config, if it's empty, it returns
// the value of the global configuration.
func (s *Server) GetUsername() string {
	if len(s.Username) == 0 &&
		s.parent != nil && s.parent.Global != nil {

		return s.parent.Global.Username
	}
	return s.Username
}

// GetUserhost returns the userhost of the irc config, if it's empty, it returns
// the value of the global configuration.
func (s *Server) GetUserhost() string {
	if len(s.Userhost) == 0 &&
		s.parent != nil && s.parent.Global != nil {

		return s.parent.Global.Userhost
	}
	return s.Userhost
}

// GetRealname returns the realname of the irc config, if it's empty, it returns
// the value of the global configuration.
func (s *Server) GetRealname() string {
	if len(s.Realname) == 0 &&
		s.parent != nil && s.parent.Global != nil {

		return s.parent.Global.Realname
	}
	return s.Realname
}

// GetPrefix returns the prefix of the irc config, if it's empty, it returns
// the value of the global configuration.
func (s *Server) GetPrefix() string {
	if len(s.Prefix) > 0 {
		return s.Prefix
	} else if s.parent != nil && s.parent.Global != nil &&
		len(s.parent.Global.Prefix) > 0 {

		return s.parent.Global.Prefix
	}
	return defaultPrefix
}

// GetChannels returns the channels of the irc config, if it's empty, it returns
// the value of the global configuration.
func (s *Server) GetChannels() []string {
	if len(s.Channels) == 0 &&
		s.parent != nil && s.parent.Global != nil {

		return s.parent.Global.Channels
	}
	return s.Channels
}
