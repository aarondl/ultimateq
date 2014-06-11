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

	[networks.ircnet]
		servers = ["localhost:3333", "server.com:6667"]

		nick = "Nick"
		altnick = "Altnick"
		username = "Username"
		realname = "Realname"
		password = "Password"

		ssl = true
		sslcert = "/path/to/a.crt"
		noverifycert = false

		nostate = false
		nostore = false

		floodlenpenalty = 120
		floodtimeout = 10.0
		floodstep = 2.0

		keepalive = 60.0

		noreconnect = false
		reconnecttimeout = 20

		# Optional, this is the hardcoded default value, you can set it if
		# you don't feel like writing prefix in the channels all the time.
		prefix = "."

		[[networks.ircnet.channels]]
			name = "#channel1"
			password = "pass1"
			prefix = "!"
		[[networks.ircnet.channels]]
			name = "#channel2"
			password = "pass2"
			prefix = "@"

	# Ext provides defaults for all exts, much as the global definitions provide
	# defaults for all networks.
	[ext]
		# Define listen to create a extension server for extensions to connect
		listen = "localhost:3333"
		# OR
		listen = "/path/to/unix.sock"

		# Define the execdir to start all executables in the path.
		execdir = "/path/to/executables"

		# Control reconnection for remote extensions.
		noreconnect = false
		reconnecttimeout = 20

		# Ext configuration is deeply nested so we can configure it globally
		# based on the network, or based on the channel on that network, or even
		# on all channels on that network.
		[ext.config] # Global config value
			key = "stringvalue"
		[ext.config.channels.#channel] # All networks for #channel
			key = "stringvalue"
		[ext.config.networks.ircnet.config] # All channels on ircnet network
			key = "stringvalue"
		[ext.config.networks.ircnet.channels.#channel] # Freenode's #channel
			key = "stringvalue"

	[exts.myext]
		# Define exec to specify a path to the executable to launch.
		exec = "/path/to/executable"

		# Defining this means that the bot will try to connect to this extension
		# rather than expecting it to connect to the listen server above.
		server = "localhost:44"
		ssl = true
		sslcert = "/path/to/a.crt"
		noverifycert = false

		# Define the above connection properties, or simply this one property.
		unix = "/path/to/sock.sock"

		# Use json not gob.
		usejson = false

		[exts.myext.active]
			ircnet = ["#channel1", "#channel2"]

Once again note the fallback mechanisms between network and the "global scope"
as well as the exts and ext. This can save you lots of repetitive typing.
*/
package config

import (
	"fmt"
	"log"
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
	values map[string]interface{}

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
	c.values = make(map[string]interface{})
	c.errors = nil
	c.filename = ""
}

// Clone deep copies a configuration object.
func (c *Config) Clone() *Config {
	c.protect.RLock()
	defer c.protect.RUnlock()

	// ? :D
	return nil
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
	ers := make(errList, 0)

	c.protect.RLock()
	c.validateRequired(&ers)
	c.protect.RUnlock()

	if len(ers) > 0 {
		c.protect.Lock()
		c.errors = ers
		c.protect.Unlock()
		return false
	}

	return true
}

// validateRequired checks that all required fields are present.
func (c *Config) validateRequired(ers *errList) {
	var nets map[string]interface{}
	if val, ok := c.values["networks"]; !ok {
		ers.addError("At least one network must be defined.")
	} else if nets, ok = val.(map[string]interface{}); !ok {
		ers.addError("Expected networks to be a map, got %T", val)
	}

	if len(nets) == 0 {
		ers.addError("Expected at least 1 network.")
		return
	}

	for name, netval := range nets {
		if net, ok := netval.(map[string]interface{}); !ok {
			ers.addError("(%s) Expected network to be a map, got %T", name,
				netval)
		} else {
			ctx := netCtx{&c.protect, c.values, net}
			if srvs, ok := ctx.Servers(); !ok || len(srvs) == 0 {
				ers.addError("(%s) Need at least one server defined", name)
			}

			if n, ok := ctx.Nick(); !ok || len(n) == 0 {
				ers.addError("(%s) Nickname is required.", name)
			}
			if n, ok := ctx.Altnick(); !ok || len(n) == 0 {
				ers.addError("(%s) Altnick is required.", name)
			}
			if n, ok := ctx.Username(); !ok || len(n) == 0 {
				ers.addError("(%s) Username is required.", name)
			}
			if n, ok := ctx.Realname(); !ok || len(n) == 0 {
				ers.addError("(%s) Realname is required.", name)
			}
		}
	}
}

// validatorRules is used internally to validate a map.
type validatorRules struct {
	stringVals      []string
	stringSliceVals []string
	boolVals        []string
	floatVals       []string
	intVals         []string
	uintVals        []string
	mapVals         []string
}

var networkValidator = validatorRules{
	stringVals: []string{
		"nick", "altnick", "username", "realname", "password",
		"sslcert", "prefix",
	},
	stringSliceVals: []string{"servers"},
	boolVals: []string{
		"ssl", "nostate", "nostore", "noreconnect",
	},
	floatVals: []string{"floodtimeout", "floodstep", "keepalive"},
	uintVals:  []string{"reconnecttimeout", "floodlenpenalty"},
	mapVals:   []string{"channels"},
}

// validateNetwork checks a network for errors and adds to the error collection
// if any are found.
func (c *Config) validateNetwork(network map[string]interface{}, ers *errList) {
}

// validateMap checks map's values for correct types based on the validatorRules
func (v validatorRules) validateMap(name string,
	m map[string]interface{}, ers *errList) {

	for _, key := range v.stringVals {
		if val, ok := m[key]; !ok {
			continue
		} else if _, ok = val.(string); !ok {
			ers.addError("(%s) value is type %T but expected string [%v]",
				name, val, val)
		}
	}
}

// Network returns the network context useable to get/set the fields for that.
// Leave name blank to return the global network context.
func (c *Config) Network(name string) *netCtx {
	c.protect.RLock()
	defer c.protect.RUnlock()

	globalCtx := &netCtx{&c.protect, nil, c.values}
	if len(name) == 0 {
		return globalCtx
	} else {
		if network, ok := getMap(globalCtx, "networks", false); ok {
			if netval, ok := network[name]; ok {
				if net, ok := netval.(map[string]interface{}); ok {
					return &netCtx{&c.protect, c.values, net}
				}
			}
		}
		return nil
	}
}

// Ext returns the extension context useable to get/set fields for the given
// extension name.
func (c *Config) Ext(name string) *extNormalCtx {
	c.protect.RLock()
	defer c.protect.RUnlock()

	var parent map[string]interface{}
	if val, ok := c.values["ext"]; ok {
		if ext, ok := val.(map[string]interface{}); ok {
			parent = ext
		}
	}

	if val, ok := c.values["exts"]; ok {
		if exts, ok := val.(map[string]interface{}); ok {
			if val, ok := exts[name]; ok {
				if ext, ok := val.(map[string]interface{}); ok {
					return &extNormalCtx{&extCtx{&c.protect, parent, ext}}
				}
			}
		}
	}

	return nil
}

// ExtGlobal returns the global extension context useable to get/set fields for
// all extensions.
func (c *Config) ExtGlobal() *extGlobalCtx {
	c.protect.RLock()
	defer c.protect.RUnlock()

	if val, ok := c.values["ext"]; ok {
		if ext, ok := val.(map[string]interface{}); ok {
			return &extGlobalCtx{&extCtx{&c.protect, nil, ext}}
		}
	}

	return nil
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

	storefile := defaultStoreFile
	ctx := &netCtx{&c.protect, nil, c.values}
	if val, ok := getStr(ctx, "storefile", false); ok {
		return val, true
	}
	return storefile, false
}

func (c *Config) NoCoreCmds() (bool, bool) {
	c.protect.RLock()
	defer c.protect.RUnlock()

	ctx := &netCtx{&c.protect, nil, c.values}
	val, ok := getBool(ctx, "nocorecmds", false)
	return val, ok
}
