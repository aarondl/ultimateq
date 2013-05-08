package config

import (
	"regexp"
	"errors"
	"fmt"
)

const (
	// nAssumedServers is the typcial number of configured servers for a bot
	nAssumedServers = 1
	// ircDefaultPort is IRC Server's default tcp port.
	ircDefaultPort = 6667
	// botDefaultPrefix
	defaultPrefix = "."
	// maxHostSize is the biggest hostname possible
	maxHostSize = 255

	errInvalidFormatString = "config(%v): Invalid %v, given: %v"
	errMissingFormatString = "config(%v): Requires %v, but nothing was given."
	errHost = "port"
	errPort = "port"
	errNick = "nickname"
	errAltnick = "alternate nickname"
	errRealname = "realname"
	errUserhost = "userhost"
	errPrefix = "prefix"
	errChannel = "channel"
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
	nicknameRegex = regexp.MustCompile(`^(?i)[a-z\[\]{}|^_\\` + "`]" +
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
	channelRegex = regexp.MustCompile(
		`^(?i)[#&+!][^\s\000\007,]{1,49}$`)

	// hostRegex matches hostnames
	hostRegex = regexp.MustCompile(
		`(?i)^[0-9a-z](?:(?:[0-9a-z]|-){0,61}[0-9a-z])?` +
		`(?:\.[0-9a-z](?:(?:[0-9a-z]|-){0,61}[0-9a-z])?)*\.?$`)
)

// Config holds all the information related to the bot including global settings
// default settings, and server specific settings.
type Config struct {
	defaults *irc
	context  *Server
	Errors   []error

	Servers map[string]*Server
}

// Configure starts a configuration by calling CreateConfig. Alias for
// CreateConfig
func Configure() *Config {
	return CreateConfig()
}

// CreateConfig initializes a Config object.
func CreateConfig() *Config {
	return &Config{
		defaults: &irc{},
		Servers:  make(map[string]*Server, nAssumedServers),
		Errors: make([]error, 0),
	}
}

// addError builds an error object and returns it using Sprintf.
func (c *Config) addError(format string, args ...interface{}) {
	c.Errors = append(c.Errors,
		errors.New(fmt.Sprintf(format, args...)),
	)
}

// Validates the required fields in the configuration and ensures that they're
// set to something.
func (c *Config) ValidateRequired() {
	for srv, s := range c.Servers {
		if s.GetHost() == "" {
			c.addError(errMissingFormatString, srv, errHost)
		}
		if s.GetNick() == "" {
			c.addError(errMissingFormatString, srv, errNick)
		}
		if s.GetRealname() == "" {
			c.addError(errMissingFormatString, srv, errRealname)
		}
		if s.GetUserhost() == "" {
			c.addError(errMissingFormatString, srv, errUserhost)
		}
	}
}

// Gets the current configuration context, if no context has been set, returns
// the default instance.
func (c *Config) GetContext() *irc {
	if c.context != nil {
		return c.context.irc
	}
	return c.defaults
}

func (c *Config) GetContextName() string {
	if c.context != nil {
		return c.context.host
	}
	return "defaults"
}

// Server fluently creates a server object and sets the context on the Config to
// the current instance.
func (c *Config) Server(host string) *Config {
	if !hostRegex.MatchString(host) && len(host) <= maxHostSize {
		c.addError(errInvalidFormatString)
	} else {
		c.context = &Server{c, host, &irc{}}
		c.Servers[host] = c.context
	}
	return c
}

// Port fluently sets the port for the current config context
func (c *Config) Port(port uint16) *Config {
	if port != 0 {
		c.GetContext().port = port
	} else {
		c.addError(errInvalidFormatString, c.GetContextName(), errPort, port)
	}
	return c
}

// Ssl fluently sets the ssl for the current config context
func (c *Config) Ssl(ssl bool) *Config {
	irc := c.GetContext()
	irc.ssl = ssl
	irc.isSslSet = true
	return c
}

// VerifyCert fluently sets the verifyCert for the current config context
func (c *Config) VerifyCert(verifyCert bool) *Config {
	i := c.GetContext()
	i.verifyCert = verifyCert
	i.isVerifyCertSet = true
	return c
}

// Nick fluently sets the nick for the current config context
func (c *Config) Nick(nick string) *Config {
	if nicknameRegex.MatchString(nick) {
		c.GetContext().nick = nick
	} else {
		c.addError(errInvalidFormatString, c.GetContextName(), errNick, nick)
	}
	return c
}

// Altnick fluently sets the altnick for the current config context
func (c *Config) Altnick(altnick string) *Config {
	if nicknameRegex.MatchString(altnick) {
		c.GetContext().altnick = altnick
	} else {
		c.addError(errInvalidFormatString, c.GetContextName(),
			errAltnick, altnick)
	}
	return c
}

// Realname fluently sets the realname for the current config context
func (c *Config) Realname(realname string) *Config {
	c.GetContext().realname = realname
	return c
}

// Userhost fluently sets the userhost for the current config context
func (c *Config) Userhost(userhost string) *Config {
	c.GetContext().userhost = userhost
	return c
}

// Prefix fluently sets the prefix for the current config context
func (c *Config) Prefix(prefix string) *Config {
	c.GetContext().prefix = prefix
	return c
}

// Channels fluently sets the channels for the current config context
func (c *Config) Channels(channels ...string) *Config {
	irc := c.GetContext()
	irc.channels = make([]string, len(channels))
	for i := 0; i < len(channels); i++ {
		if channelRegex.MatchString(channels[i]) {
			irc.channels[i] = channels[i]
		} else {
			c.addError(errInvalidFormatString, c.GetContextName(),
				errChannel, channels[i])
		}
	}
	return c
}

// ServerConfig stores the all the details necessary to connect to an irc server
type Server struct {
	parent *Config

	host string

	irc *irc
}

// irc config contains the options surrounding an irc server connection. But not
// the location of the server, there cannot be a default host and so we have
// this division between the Server and irc structs.
type irc struct {
	// Irc Server connection info
	port            uint16
	ssl             bool
	isSslSet        bool
	verifyCert      bool
	isVerifyCertSet bool

	// Irc User data
	nick     string
	altnick  string
	realname string
	userhost string

	// Dispatching options
	prefix   string
	channels []string
}

// GetHost gets s.host
func (s *Server) GetHost() string {
	return s.host
}

// GetPort returns port of the irc config, if it hasn't been set, returns the
// value of the default, if that hasn't been set returns ircDefaultPort.
func (s *Server) GetPort() uint16 {
	if s.irc.port != 0 {
		return s.irc.port
	} else if s.parent != nil && s.parent.defaults != nil &&
		s.parent.defaults.port != 0 {

		return s.parent.defaults.port
	}
	return ircDefaultPort
}

// GetSsl returns ssl of the irc config, if it hasn't been set, returns the
// value of the default, if that hasn't been set returns false.
func (s *Server) GetSsl() bool {
	if s.irc.isSslSet {
		return s.irc.ssl
	} else if s.parent != nil && s.parent.defaults != nil {
		return s.parent.defaults.isSslSet && s.parent.defaults.ssl
	}
	return false
}

// GetSsl returns verifyCert of the irc config, if it hasn't been set, returns
// the value of the default, if that hasn't been set returns false.
func (s *Server) GetVerifyCert() bool {
	if s.irc.isVerifyCertSet {
		return s.irc.verifyCert
	} else if s.parent != nil && s.parent.defaults != nil {
		return s.parent.defaults.isVerifyCertSet && s.parent.defaults.verifyCert
	}
	return false
}

// GetNick returns the nickname of the irc config, if it's empty, it returns the
// value of the default configuration.
func (s *Server) GetNick() string {
	if len(s.irc.nick) == 0 &&
		s.parent != nil && s.parent.defaults != nil {

		return s.parent.defaults.nick
	}
	return s.irc.nick
}

// GetAltnick returns the altnick of the irc config, if it's empty, it returns
// the value of the default configuration.
func (s *Server) GetAltnick() string {
	if len(s.irc.altnick) == 0 &&
		s.parent != nil && s.parent.defaults != nil {

		return s.parent.defaults.altnick
	}
	return s.irc.altnick
}

// GetRealname returns the realname of the irc config, if it's empty, it returns
// the value of the default configuration.
func (s *Server) GetRealname() string {
	if len(s.irc.realname) == 0 &&
		s.parent != nil && s.parent.defaults != nil {

		return s.parent.defaults.realname
	}
	return s.irc.realname
}

// GetUserhost returns the userhost of the irc config, if it's empty, it returns
// the value of the default configuration.
func (s *Server) GetUserhost() string {
	if len(s.irc.userhost) == 0 &&
		s.parent != nil && s.parent.defaults != nil {

		return s.parent.defaults.userhost
	}
	return s.irc.userhost
}

// GetPrefix returns the prefix of the irc config, if it's empty, it returns
// the value of the default configuration.
func (s *Server) GetPrefix() string {
	if len(s.irc.prefix) == 0 && s.parent != nil &&
		s.parent.defaults != nil && s.parent.defaults.prefix != "" {

		return s.parent.defaults.prefix
	} else {
		return s.irc.prefix
	}
	return defaultPrefix
}

// GetChannels returns the channels of the irc config, if it's empty, it returns
// the value of the default configuration.
func (s *Server) GetChannels() []string {
	if len(s.irc.channels) == 0 &&
		s.parent != nil && s.parent.defaults != nil {

		return s.parent.defaults.channels
	}
	return s.irc.channels
}
