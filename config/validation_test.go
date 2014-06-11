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
	servers = [10]

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
	name = 5
	password = 5
	prefix = 5

	[[networks.ircnet.channels]]
	name = "#channel2"
	password = "pass2"
	prefix = "@"

[ext]
	listen = 5
	execdir = 20

	noreconnect = "true"
	reconnecttimeout = 40.0

	usejson = "what"

	[ext.config]
		key = 5
	[ext.config.channels.#channel]
		key = 5
	[ext.config.networks.ircnet]
		key = 5
	[ext.config.networks.ircnet.channels.#channel]
		key = 5

	[ext.active]
		ircnet = [5, 6]

[exts.myext]
	exec = 5

	server = 5
	ssl = "hello"
	sslcert = true
	noverifycert = "what"

	unix = 5

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

		{"noirc", "servers", "array", "string"},

		{"ircnet", "servers 1", "string", "int64"},
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
		{"ircnet channels", "name", "string", "int64"},
		{"ircnet channels", "password", "string", "int64"},
		{"ircnet channels", "prefix", "string", "int64"},

		{"globalext", "listen", "string", "int64"},
		{"globalext", "execdir", "string", "int64"},
		{"globalext", "noreconnect", "bool", "string"},
		{"globalext", "reconnecttimeout", "int", "float64"},
		{"globalext", "usejson", "bool", "string"},
		{"globalext config", "key", "string", "int64"},
		{"globalext config #channel", "key", "string", "int64"},
		{"globalext config ircnet", "key", "string", "int64"},
		{"globalext config ircnet #channel", "key", "string", "int64"},
		{"globalext active ircnet", "channel 1", "string", "int64"},
		{"globalext active ircnet", "channel 2", "string", "int64"},

		{"myext", "exec", "string", "int64"},
		{"myext", "server", "string", "int64"},
		{"myext", "ssl", "bool", "string"},
		{"myext", "sslcert", "string", "bool"},
		{"myext", "noverifycert", "bool", "string"},
		{"myext", "unix", "string", "int64"},
		{"myext", "usejson", "bool", "string"},
		{"myext active ircnet", "channel 1", "string", "int64"},
		{"myext active ircnet", "channel 2", "string", "int64"},
	}

	if len(expect) != len(ers) {
		for _, e := range ers {
			t.Error(e)
		}
		t.Errorf("Expected %d errors, but got %d", len(expect), len(ers))
	}

	founds := make([]bool, len(ers))

	for _, expErr := range expect {
		found := false
		for i, e := range ers {
			er := fmt.Sprintf("(%s) %s is %s but expected %s",
				expErr.object, expErr.key, expErr.foundKind, expErr.kind)
			if strings.HasPrefix(e.Error(), er) {
				found = true
				founds[i] = true
				break
			}
		}
		if !found {
			t.Error("Expected to find error concerning:",
				expErr.object, expErr.key, expErr.foundKind, expErr.kind)
		}
	}

	for i, found := range founds {
		if !found {
			t.Error("Unexpected error occurred:", ers[i])
		}
	}
}
