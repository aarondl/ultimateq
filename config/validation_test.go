package config

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"gopkg.in/inconshreveable/log15.v2"
)

func TestValidation(t *testing.T) {
	t.Parallel()

	c := New().FromString(`
	[networks.net]
	`)

	if c.Validate() {
		t.Error("Expected it to be invalid.")
	}

	expErr := "(net) Expected at least one server."
	if ers := c.Errors(); len(ers) == 0 {
		t.Error("Expected one error.")
	} else if ers[0].Error() != expErr {
		t.Error("Expected a particular error message, got:", ers[0])
	}

	c = New().FromString(`
	nick = "n"
	altnick = "n"
	realname = "n"
	username = "n"
	prefix = 5

	[networks.net]
		servers = ["n"]
	`)

	if c.Validate() {
		t.Error("Expected it to be invalid.")
	}

	expErr = "(global) prefix is int64 but expected string [5]"
	if ers := c.Errors(); len(ers) == 0 {
		t.Error("Expected one error.")
	} else if ers[0].Error() != expErr {
		t.Error("Expected a particular error message, got:", ers[0])
	}
}

func TestValidation_DisplayErrors(t *testing.T) {
	t.Parallel()
	b := &bytes.Buffer{}

	logger := log15.New()
	logger.SetHandler(log15.StreamHandler(b, log15.LogfmtFormat()))

	c := New().FromString(`
	nick = "n"
	altnick = "n"
	realname = "n"
	username = "n"
	prefix = 5

	[networks.net]
	`)

	if c.Validate() {
		t.Error("Expected it to be invalid.")
	}

	exp := "(global) prefix is int64 but expected string [5]"
	c.DisplayErrors(logger)
	if !strings.Contains(b.String(), exp) {
		t.Error("Expected a particular error message, got:", b.String())
	}
}

type rexpect struct {
	context, message string
}

type texpect struct {
	context, key, kind, foundKind string
}

func TestValidation_RequiredNoServers(t *testing.T) {
	t.Parallel()

	expects := []rexpect{
		{"", "Expected at least one network."},
	}

	requiredTestHelper("", expects, t)
}

func TestValidation_RequiredServers(t *testing.T) {
	t.Parallel()

	cfg := `[networks.hello]`

	expects := []rexpect{
		{"hello", "Nickname is required."},
		{"hello", "Username is required."},
		{"hello", "Realname is required."},
		{"hello", "Expected at least one server."},
	}

	requiredTestHelper(cfg, expects, t)
}

func TestValidation_RequiredTypes(t *testing.T) {
	t.Parallel()

	cfg := `networks = 5`
	expects := []rexpect{{"", "Expected at least one network."}}

	requiredTestHelper(cfg, expects, t)

	cfg = "[networks]\nserver = 5"
	expects = []rexpect{{"server", "Expected network to be a map, got int64"}}

	requiredTestHelper(cfg, expects, t)
}

func requiredTestHelper(cfg string, expects []rexpect, t *testing.T) {
	ers := make(errList, 0)

	c := New().FromString(cfg)
	c.validateRequired(&ers)

	if len(expects) != len(ers) {
		for _, e := range ers {
			t.Error(e)
		}
		t.Errorf("Expected %d errors, but got %d", len(expects), len(ers))
	}

	founds := make([]bool, len(ers))

	for _, expErr := range expects {
		found := false
		for i, e := range ers {
			var er string
			if len(expErr.context) == 0 {
				er = fmt.Sprintf("%s", expErr.message)
			} else {
				er = fmt.Sprintf("(%s) %s", expErr.context, expErr.message)
			}
			if strings.HasPrefix(e.Error(), er) {
				found = true
				founds[i] = true
				break
			}
		}
		if !found {
			t.Error("Expected to find error concerning:",
				expErr.context, expErr.message)
		}
	}

	for i, found := range founds {
		if !found {
			t.Error("Unexpected error occurred:", ers[i])
		}
	}
}

func TestValidation_TypesTopLevel(t *testing.T) {
	t.Parallel()
	cfg := `
		networks = 5
		ext = 5
		exts = 5`

	exps := []texpect{
		{"global", "networks", "map", "int64"},
		{"global", "ext", "map", "int64"},
		{"global", "exts", "map", "int64"},
	}

	typesTestHelper(cfg, exps, t)
}

func TestValidation_TypesMidLevel(t *testing.T) {
	t.Parallel()

	cfg := `
	[networks]
	noirc = 5

	[networks.ircnet]
	channels = 5

	[ext]
	config = 5
	active = 5

	[exts]
	myext = 5

	[exts.extension]
	active = 5`

	exps := []texpect{
		{"global networks", "noirc", "map", "int64"},
		{"ircnet", "channels", "map", "int64"},
		{"ext", "config", "map", "int64"},
		{"ext", "active", "map", "int64"},
		{"exts", "myext", "map", "int64"},
		{"extension", "active", "map", "int64"},
	}

	typesTestHelper(cfg, exps, t)
}

func TestValidation_TypesConfig(t *testing.T) {
	t.Parallel()

	cfg := `
	[ext.active]
	list = 5
	[ext.config]
	networks = 5
	channels = 5
	[ext.config.more]
	list = 5
	[exts.extname.active]
	list = 5`

	exps := []texpect{
		{"ext active", "list", "array", "int64"},
		{"extname active", "list", "array", "int64"},
		{"ext config", "networks", "map", "int64"},
		{"ext config", "channels", "map", "int64"},
		{"ext config", "more", "string", "map[string]interface {}"},
	}

	typesTestHelper(cfg, exps, t)

	cfg = `
	[ext.config]
	networks = "5"
	channels = "5"`

	exps = []texpect{
		{"ext config", "networks", "map", "string"},
		{"ext config", "channels", "map", "string"},
	}

	typesTestHelper(cfg, exps, t)
}

func TestValidation_TypesConfigMidLevel(t *testing.T) {
	t.Parallel()

	cfg := `
	[ext.config.networks]
	network = 5
	[ext.config.channels]
	channel = 5
	[ext.config.networks.ircnet]
	channels = 5`

	exps := []texpect{
		{"ext config", "network", "map", "int64"},
		{"ext config", "channel", "map", "int64"},
		{"ext config ircnet", "channels", "map", "int64"},
	}

	typesTestHelper(cfg, exps, t)
}

func TestValidation_TypesLeafs(t *testing.T) {
	t.Parallel()

	cfg := `
		storefile = 5
		nocorecmds = "hello"
		logfile = 5
		loglevel = 5
		secret_key = 5

		nick = 6
		altnick = 7
		username = 8
		realname = 9
		password = 10

		[networks.noirc]
			servers = "farse"
		[networks.ircnet]
			servers = [10]

			tls = "hello"
			tls_ca_cert = false
			tls_cert = false
			tls_key = false
			tls_insecure_skip_verify = "lol"

			nostate = 5
			nostore = 6

			noautojoin = 5
			joindelay = "lol"

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

			tls_cert = true
			tls_key  = true
			tls_client_ca = true
			tls_client_revs = true

			[ext.config]
				key = 5
			[ext.config.channels."#channel"]
				key = 5
			[ext.config.networks.ircnet]
				key = 5
			[ext.config.networks.ircnet.channels."#channel"]
				key = 5

			[ext.active]
				ircnet = [5, 6]

		[exts.myext]
			exec = 5

			server = 5
			tls_cert = true
			tls_insecure_skip_verify = "what"

			unix = 5

			[exts.myext.active]
				ircnet = [5, 6]`

	exps := []texpect{
		{"global", "storefile", "string", "int64"},
		{"global", "nocorecmds", "bool", "string"},
		{"global", "loglevel", "string", "int64"},
		{"global", "logfile", "string", "int64"},
		{"global", "secret_key", "string", "int64"},
		{"global", "nick", "string", "int64"},
		{"global", "altnick", "string", "int64"},
		{"global", "username", "string", "int64"},
		{"global", "realname", "string", "int64"},
		{"global", "password", "string", "int64"},

		{"noirc", "servers", "array", "string"},

		{"ircnet", "servers 1", "string", "int64"},
		{"ircnet", "tls", "bool", "string"},
		{"ircnet", "tls_key", "string", "bool"},
		{"ircnet", "tls_cert", "string", "bool"},
		{"ircnet", "tls_ca_cert", "string", "bool"},
		{"ircnet", "tls_insecure_skip_verify", "bool", "string"},
		{"ircnet", "nostate", "bool", "int64"},
		{"ircnet", "nostore", "bool", "int64"},
		{"ircnet", "noautojoin", "bool", "int64"},
		{"ircnet", "joindelay", "int", "string"},
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

		{"ext", "listen", "string", "int64"},
		{"ext", "tls_cert", "string", "bool"},
		{"ext", "tls_key", "string", "bool"},
		{"ext", "tls_client_ca", "string", "bool"},
		{"ext", "tls_client_revs", "string", "bool"},
		{"ext", "execdir", "string", "int64"},
		{"ext", "noreconnect", "bool", "string"},
		{"ext", "reconnecttimeout", "int", "float64"},
		{"ext config", "key", "string", "int64"},
		{"ext config #channel", "key", "string", "int64"},
		{"ext config ircnet", "key", "string", "int64"},
		{"ext config ircnet #channel", "key", "string", "int64"},
		{"ext active ircnet", "channel 1", "string", "int64"},
		{"ext active ircnet", "channel 2", "string", "int64"},

		{"myext", "exec", "string", "int64"},
		{"myext", "server", "string", "int64"},
		{"myext", "tls_cert", "string", "bool"},
		{"myext", "tls_insecure_skip_verify", "bool", "string"},
		{"myext active ircnet", "channel 1", "string", "int64"},
		{"myext active ircnet", "channel 2", "string", "int64"},
	}

	typesTestHelper(cfg, exps, t)
}

func typesTestHelper(cfg string, expects []texpect, t *testing.T) {
	ers := make(errList, 0)

	c := New().FromString(cfg)
	c.validateTypes(&ers)

	if len(expects) != len(ers) {
		for _, e := range ers {
			t.Error(e)
		}
		t.Errorf("Expected %d errors, but got %d", len(expects), len(ers))
	}

	founds := make([]bool, len(ers))

	for _, expErr := range expects {
		found := false
		for i, e := range ers {
			er := fmt.Sprintf("(%s) %s is %s but expected %s",
				expErr.context, expErr.key, expErr.foundKind, expErr.kind)
			if strings.HasPrefix(e.Error(), er) {
				found = true
				founds[i] = true
				break
			}
		}
		if !found {
			t.Error("Expected to find error concerning:",
				expErr.context, expErr.key, expErr.foundKind, expErr.kind)
		}
	}

	for i, found := range founds {
		if !found {
			t.Error("Unexpected error occurred:", ers[i])
		}
	}
}
