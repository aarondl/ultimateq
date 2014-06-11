package config

import (
	"fmt"
	"strings"
	"testing"
)

const badconfig = `
storefile = 5
nocorecmds = "hello"

nick = 6
altnick = 7
username = 8
realname = 9
password = 10

[networks.noirc]
	servers = "farse"
[networks.ircnet]
	servers = 10

	ssl = "destroy"
	sslcert = false
	noverifycert = "lol"

	nostate = 5
	nostore = 6

	floodlenpenalty = 20.0
	floodtimeout = "anarchy"
	floodstep = "string"

	keepalive = "what"

	noreconnect = "abc"
	reconnecttimeout = 20.0

	prefix = false

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
	# OR listen = "/path/to/unix.sock"

	# Define the execdir to start all executables in the path.
	execdir = "/path/to/executables"

	# Control reconnection for remote extensions.
	noreconnect = false
	reconnecttimeout = 20

	usejson = true

	# Ext configuration is deeply nested so we can configure it globally
	# based on the network, or based on the channel on that network, or even
	# on all channels on that network.
	[ext.config] # Global config value
		key = 5
	[ext.config.channels.#channel] # All networks for #channel
		key = 5
	[ext.config.networks.ircnet.config] # All channels on ircnet network
		key = 5
	[ext.config.networks.ircnet.channels.#channel] # Freenode's #channel
		key = 5

[exts.myext]
	# Define exec to specify a path to the executable to launch.
	exec = 5

	# Defining this means that the bot will try to connect to this extension
	# rather than expecting it to connect to the listen server above.
	server = 5
	ssl = "hello"
	sslcert = 5
	noverifycert = "what"

	# Define the above connection properties, or simply this one property.
	unix = 5

	# Use json not gob.
	usejson = "there"

	[exts.myext.active]
		ircnet = [5, 6]
`

func TestValidation(t *testing.T) {
	t.Parallel()

	ers := make(errList, 0)

	c := NewConfig().FromString(badconfig)
	c.validateTypes(&ers)

	expect := []struct {
		object    string
		key       string
		kind      string
		foundKind string
	}{
		{"global", "storefile", "string", "int64"},
		{"global", "nocorecmds", "bool", "string"},
		{"global", "nick", "string", "int64"},
		{"global", "altnick", "string", "int64"},
		{"global", "username", "string", "int64"},
		{"global", "realname", "string", "int64"},
		{"global", "password", "string", "int64"},
		{"noirc", "servers", "[]string", "string"},
		{"ircnet", "servers", "[]string", "int64"},
		{"ircnet", "ssl", "bool", "string"},
		{"ircnet", "sslcert", "string", "bool"},
		{"ircnet", "noverifycert", "bool", "string"},
		{"ircnet", "nostate", "bool", "int64"},
		{"ircnet", "nostore", "bool", "int64"},
		{"ircnet", "floodlenpenalty", "int", "float64"},
		{"ircnet", "floodtimeout", "float64", "string"},
		{"ircnet", "floodstep", "float64", "string"},
		{"ircnet", "keepalive", "float64", "string"},
		{"ircnet", "noreconnect", "bool", "string"},
		{"ircnet", "reconnecttimeout", "int", "float64"},
		{"ircnet", "prefix", "string", "bool"},
	}

	for _, expErr := range expect {
		found := false
		for _, e := range ers {
			er := fmt.Sprintf("(%s) %s is type %s but expected %s",
				expErr.object, expErr.key, expErr.foundKind, expErr.kind)
			if strings.HasPrefix(e.Error(), er) {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find error concerning:",
				expErr.object, expErr.key, expErr.foundKind, expErr.kind)
		}
	}
}
