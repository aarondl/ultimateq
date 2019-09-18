/*
Package config creates a configuration using toml.

An example configuration looks like this:
	# Anything defined here provides fallback defaults for all networks.
	# except the immediately following fields which are global-only.
	# In other words, all values you see in the network definition can be
	# defined here and all servers will use those values unless they have their
	# own defined.
	storefile = "/path/to/store/file.db"
	nocorecmds = false
	loglevel = "debug"
	logfile = "/path/to/file.log"
	secret_key = "myunbelievablylongandsecrettoken"
	ignores = ["hostsuffix"]

	# Most of the configuration values below have healthy defaults which means
	# you don't have to set any of them. servers, nick, username, realname is
	# enough!
	[networks.ircnet]
		servers = ["localhost:3333", "server.com:6667"]

		nick = "Nick"
		altnick = "Altnick"
		username = "Username"
		realname = "Realname"
		password = "Password"

		# TLS Options
		# If tls is on it will connect with tls.
		# If tls_cert, tls_key are set it will send it in an attempt to perform
		# mutual TLS with the irc server.
		# If tls_ca_cert is present it will be used as a CA when connecting
		tls         = true
		tls_ca_cert = "/path/to/ca.crt"
		tls_cert    = "/path/to/a.crt"
		tls_key     = "/path/to/a.key"
		tls_insecure_skip_verify = false

		# Bot Internal Database Options
		nostate = false
		nostore = false

		# Auto(Re)Join controls.
		noautojoin = false
		# How many seconds after connect or while banned to wait to rejoin.
		joindelay = 5

		# Flood control fine tuning knobs.
		floodlenpenalty = 120
		floodtimeout = 10.0
		floodstep = 2.0

		# Send a ping to the server every X seconds.
		keepalive = 60.0

		# Reconnection controls.
		noreconnect = false
		reconnecttimeout = 20

		# For fallback of channels below.
		prefix = "."

		[[networks.ircnet.channels]]
			name     = "#channel1"
			password = "pass1"
			prefix   = "!"
		[[networks.ircnet.channels]]
			name     = "&channel2"
			password = "pass2"
			prefix   = "@"

	# Ext provides defaults for all exts, much as the global definitions provide
	# defaults for all networks.
	[ext]
		# Define listen to create a extension server for extensions to connect
		listen = "localhost:3333"
		# OR
		listen = "/path/to/unix.sock"

		# If tls key & cert are present the remote extensions will require
		# MUTUAL tls, meaning the client will have to present a certificate as
		# well, you can use tls_client_ca to control which ca cert is used
		# to verify the client (otherwise the system ca cert pool is used).
		# If tls_insecure_skip_verify is set, the client's certificate
		# will still be required but will not be verified
		tls_key      = "/path/to/a.key"
		tls_cert     = "/path/to/a.crt"
		tls_client_ca  = "/path/to/ca.crt"
		# A CRL (revocation list) can be passed in, client certificates
		# can be revoked by adding them to the crl
		tls_client_revs = "/path/to/ca.crl"
		tls_insecure_skip_verify = false

		# Define the execdir to start all executables in the path.
		execdir = "/path/to/executables"

		# Ext configuration is deeply nested so we can configure it globally
		# based on the network, or based on the channel on that network, or even
		# on all channels on that network.
		[ext.config] # Global config value
			key = "stringvalue"
		[[ext.config.channels]] # All networks for #channel
			name = "#channel"
			key  = "stringvalue"
		[ext.config.networks.ircnet] # All channels on ircnet network
			key = "stringvalue"
		[[ext.config.networks.ircnet.channels]] # Freenode's #channel
			name = "#channel1"
			key  = "stringvalue"

	[exts.myext]
		# Define exec to specify a path to the executable to launch.
		# NOTE: Currently NOT USED
		exec = "/path/to/executable"

		# Defining this means that the bot will try to connect to this extension
		# rather than expecting it to connect to the listen server in the
		# global configuration. Server can also be unix:/path/to/sock
		#
		# NOTE: Currently NOT USED
		server       = "localhost:44"
		tls_cert     = "/path/to/a.crt"
		noverifycert = false

		[exts.myext.active]
			ircnet = ["#channel1", "#channel2"]


Once again note the fallback mechanisms between network and the "global scope"
as well as the exts and ext. This can save you lots of repetitive typing.
*/
package config

import (
	"sync"

	"gopkg.in/inconshreveable/log15.v2"
)

const (
	// defaultIrcPort is IRC Network'n default tcp port.
	defaultIrcPort = uint16(6667)
	// defaultStoreFile is where the bot will store it'n Store database if not
	// overridden.
	defaultStoreFile = "./store.db"
	// defaultLogLevel is the log level of the bot.
	defaultLogLevel = "info"
	// defaultJoinDelay is how many seconds to wait before auto (re)joining a
	// channel.
	defaultJoinDelay = uint(5)
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
	// defaultPrefix is the command prefix by default
	defaultPrefix = '.'
)

// The following format strings are for formatting various config errors.
const (
	fmtErrInvalid          = "config(%v): Invalid %v, given: %v"
	fmtErrMissing          = "config(%v): Requires %v, but nothing was given."
	fmtErrNetworkNotFound  = "config: Network not found, given: %v"
	errMsgNetworksRequired = "config: At least one network is required."
	errMsgDuplicateNetwork = "config: Network names must be unique, use .Host()"
)

// Config holds all the information related to the bot including global settings
// default settings, and network specific settings.
type Config struct {
	values mp

	errors   errList
	filename string
	protect  sync.RWMutex
}

// New initializes a Config object.
func New() *Config {
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
	c.values = make(map[string]interface{})
	c.errors = nil
	c.filename = ""
}

// Clone deep copies a configuration object.
func (c *Config) Clone() *Config {
	c.protect.RLock()
	defer c.protect.RUnlock()

	nc := New()
	copyMap(nc.values, c.values)

	return nc
}

// Replace replaces the configuration with a new one.
func (c *Config) Replace(newConfig *Config) *Config {
	c.protect.Lock()
	defer c.protect.Unlock()

	newConfig.protect.RLock()
	defer newConfig.protect.RUnlock()

	c.values = make(map[string]interface{})
	copyMap(c.values, newConfig.values)

	return c
}

// Network returns the network context useable to get/set the fields for that.
// Leave name blank to return the global network context.
func (c *Config) Network(name string) *NetCTX {
	c.protect.RLock()
	defer c.protect.RUnlock()

	if len(name) == 0 {
		return &NetCTX{&c.protect, nil, c.values}
	} else {
		if nets := c.values.get("networks"); nets != nil {
			if net := nets.get(name); net != nil {
				return &NetCTX{&c.protect, c.values, net}
			}
		}
		return nil
	}
}

// Networks returns a list of configured networks.
func (c *Config) Networks() []string {
	c.protect.RLock()
	defer c.protect.RUnlock()

	nets := c.values.get("networks")
	if nets == nil {
		return nil
	}

	rets := make([]string, 0)
	for key, _ := range nets {
		rets = append(rets, key)
	}

	return rets
}

// Ext returns the extension context useable to get/set fields for the given
// extension name.
func (c *Config) Ext(name string) *ExtNormalCTX {
	c.protect.RLock()
	defer c.protect.RUnlock()

	parent := c.values.get("ext")

	if exts := c.values.get("exts"); exts != nil {
		if ext := exts.get(name); ext != nil {
			return &ExtNormalCTX{&ExtCTX{&c.protect, parent, ext}}
		}
	}

	return nil
}

// ExtGlobal returns the global extension context useable to get/set fields for
// all extensions.
func (c *Config) ExtGlobal() *ExtGlobalCTX {
	c.protect.RLock()
	defer c.protect.RUnlock()

	if ext := c.values.get("ext"); ext != nil {
		return &ExtGlobalCTX{&ExtCTX{&c.protect, nil, ext}}
	}

	ext := make(map[string]interface{})
	c.values["ext"] = ext
	return &ExtGlobalCTX{&ExtCTX{&c.protect, nil, ext}}
}

// Exts returns a list of configured extensions.
func (c *Config) Exts() []string {
	c.protect.RLock()
	defer c.protect.RUnlock()

	exts := c.values.get("exts")
	if exts == nil {
		return nil
	}

	rets := make([]string, 0)
	for key := range exts {
		rets = append(rets, key)
	}

	return rets
}

// DisplayErrors is a helper function to log the output of all config errors to
// the standard logger.
func (c *Config) DisplayErrors(logger log15.Logger) {
	c.protect.RLock()
	defer c.protect.RUnlock()

	for _, e := range c.errors {
		logger.Error(e.Error())
	}
}

// Filename returns fileName of the configuration, or the default.
func (c *Config) Filename() string {
	c.protect.RLock()
	defer c.protect.RUnlock()

	filename := defaultConfigFileName
	if len(c.filename) > 0 {
		filename = c.filename
	}
	return filename
}

// StoreFile gets the global storefile or defaultStoreFile.
func (c *Config) StoreFile() (string, bool) {
	c.protect.RLock()
	defer c.protect.RUnlock()

	if val, ok := c.values["storefile"]; ok {
		if storefile, ok := val.(string); ok {
			return storefile, true
		}
	}
	return defaultStoreFile, false
}

// SetStoreFile sets the global storefile or defaultStoreFile.
func (c *Config) SetStoreFile(val string) *Config {
	c.protect.Lock()
	defer c.protect.Unlock()

	c.values["storefile"] = interface{}(val)
	return c
}

// LogFile gets the global logfile or defaultLogFile.
func (c *Config) LogFile() (string, bool) {
	c.protect.RLock()
	defer c.protect.RUnlock()

	if val, ok := c.values["logfile"]; ok {
		if logfile, ok := val.(string); ok {
			return logfile, true
		}
	}
	return "", false
}

// SetLogFile sets the global logfile or defaultLogFile.
func (c *Config) SetLogFile(val string) *Config {
	c.protect.Lock()
	defer c.protect.Unlock()

	c.values["logfile"] = interface{}(val)
	return c
}

// LogLevel gets the global loglevel or defaultLogLevel.
func (c *Config) LogLevel() (string, bool) {
	c.protect.RLock()
	defer c.protect.RUnlock()

	if val, ok := c.values["loglevel"]; ok {
		if loglevel, ok := val.(string); ok {
			return loglevel, true
		}
	}
	return defaultLogLevel, false
}

// SetLogLevel sets the global loglevel or defaultLogLevel.
func (c *Config) SetLogLevel(val string) *Config {
	c.protect.Lock()
	defer c.protect.Unlock()

	c.values["loglevel"] = interface{}(val)
	return c
}

// NoCoreCmds gets the value of the corecmds variable.
func (c *Config) NoCoreCmds() (bool, bool) {
	c.protect.RLock()
	defer c.protect.RUnlock()

	if val, ok := c.values["nocorecmds"]; ok {
		if corecmds, ok := val.(bool); ok {
			return corecmds, true
		}
	}
	return false, false
}

// SetNoCoreCmds gets the value of the corecmds variable.
func (c *Config) SetNoCoreCmds(val bool) *Config {
	c.protect.Lock()
	defer c.protect.Unlock()

	c.values["nocorecmds"] = interface{}(val)
	return c
}

// SecretKey gets the value of the secretKey variable
func (c *Config) SecretKey() (string, bool) {
	c.protect.RLock()
	defer c.protect.RUnlock()

	if val, ok := c.values["secret_key"]; ok {
		if skey, ok := val.(string); ok {
			return skey, true
		}
	}

	return "", false
}

// SetSecretKey value in the config.
func (c *Config) SetSecretKey(key string) *Config {
	c.protect.Lock()
	defer c.protect.Unlock()

	c.values["secret_key"] = interface{}(key)
	return c
}

// Ignores returns the global set ignores
func (c *Config) Ignores() ([]string, bool) {
	c.protect.RLock()
	defer c.protect.RUnlock()

	var ignores []string
	val, ok := c.values["ignores"]
	if !ok {
		return nil, false
	}

	islice, ok := val.([]interface{})
	if !ok {
		return nil, false
	}

	for _, i := range islice {
		if s, ok := i.(string); ok {
			ignores = append(ignores, s)
		}
	}

	return ignores, true
}

// SetIgnores sets the ignores array
func (c *Config) SetIgnores(ignores []string) *Config {
	c.protect.Lock()
	defer c.protect.Unlock()

	intfIgnores := make([]interface{}, len(ignores))
	for i, s := range ignores {
		intfIgnores[i] = s
	}

	c.values["ignores"] = intfIgnores
	return c
}

// NewNetwork creates a network and returns the network's context.
// If the network exists, the context will be nil.
func (c *Config) NewNetwork(name string) *NetCTX {
	c.protect.RLock()
	defer c.protect.RUnlock()

	var net, nets map[string]interface{}

	if nets = c.values.get("networks"); nets == nil {
		nets = make(map[string]interface{})
		c.values["networks"] = interface{}(nets)
	}

	if net = mp(nets).get(name); net != nil {
		return nil
	}

	net = make(mp)
	nets[name] = net

	return &NetCTX{&c.protect, c.values, net}
}

// NewExt creates a extension and returns the extension's context.
// If the extension exists, the context will be nil.
func (c *Config) NewExt(name string) *ExtNormalCTX {
	c.protect.RLock()
	defer c.protect.RUnlock()

	var ext, exts mp

	if exts = c.values.get("exts"); exts == nil {
		exts = make(mp)
		c.values["exts"] = interface{}(exts)
	}

	if ext = exts.get(name); ext == nil {
		ext = make(mp)
		exts[name] = ext
	} else {
		return nil
	}

	parent := c.values.get("ext")

	return &ExtNormalCTX{&ExtCTX{&c.protect, parent, ext}}
}
