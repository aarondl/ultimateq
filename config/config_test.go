package config

import "testing"

func TestConfig_New(t *testing.T) {
	t.Parallel()

	c := NewConfig()
	if c == nil {
		t.Error("Expected a configuration to be created.")
	}
}

/*var cloneable = `global:
    nick: a
    realname: a
    username: a
networks:
    srv:
        servers: i.com
        channels: #dude
        exts:
            fun:
                config:
                    key: value
`
*/

/*var cloneable = `
nick = "a"
realname = "a"
username = "a"

[networks.srv]
servers = ["i.com"]
channels = ["#dude"]

[networks.srv.exts.fun.config]
key = "value"
`

func isReqErr(e error) bool {
	return strings.Contains(e.Error(), "Required")
}

func isInvErr(e error) bool {
	return strings.Contains(e.Error(), "Invalid")
}

func TestConfig(t *testing.T) {
	t.Parallel()
	c := NewConfig()
	if c.Network == nil || c.Network.protect == nil {
		t.Error("Expected global settings to be initialized.")
	}

	if c.Networks == nil {
		t.Error("Expected network map to be initialized.")
	}
}

func TestConfig_Clone(t *testing.T) {
	t.Parallel()
	c := NewConfig().FromString(cloneable)
	if !c.IsValid() {
		t.Error(c.errors)
		t.Error("Expected a valid configuration.")
	}

	b := c.Clone()
	if b == c || b.Network == c.Network || &b.Networks == &c.Networks {
		t.Error("It should allocate all it's own memory.")
	}

	sb, sc := b.Networks["srv"], c.Networks["srv"]
	if sb == sc ||
		&sb.InChannels == &sc.InChannels || &sb.InServers == &sc.InServers {

		t.Error("Networks should also be deep-copied.")
	}
}

func TestConfig_Clear(t *testing.T) {
	t.Parallel()
	name := "something"

	c := NewConfig()
	c.Network.InName = name
	c.filename = name
	c.Storefile = name
	c.Networks[name] = &Network{}

	c.Clear()
	if c.Network.InName == name {
		t.Error("It should wipe the network name.")
	}

	if c.filename == name {
		t.Error("It should wipe the filename.")
	}

	if c.Storefile == name {
		t.Error("It should wipe the store file.")
	}

	if _, ok := c.Networks[name]; ok {
		t.Error("It should wipe the networks.")
	}
}

var net1 = &Network{
	InName:             "irc1",
	InServers:          []string{"irc.gamesurge.net"},
	InPort:             5555,
	InSsl:              "true",
	InSslCert:          "file1",
	InNoVerifyCert:     "false",
	InNoState:          "false",
	InNoStore:          "true",
	InFloodLenPenalty:  "5",
	InFloodTimeout:     "3.5",
	InFloodStep:        "5.5",
	InKeepAlive:        "7.5",
	InNoReconnect:      "false",
	InReconnectTimeout: "10",
	InNick:             "n1",
	InAltnick:          "a1",
	InUsername:         "u1",
	InUserhost:         "h1",
	InRealname:         "r1",
	InPassword:         "p1",
	InPrefix:           "1",
	InChannels:         []string{"#chan1", "#chan2"},
	InExts: map[string]*Ext{
		"ext": {
			InName:             "ext",
			InConfig:           map[string]string{"k": "1"},
			InLocal:            "true",
			InExec:             "ee1",
			InUseJSON:          "false",
			InIsServer:         "false",
			InAddress:          "ea1",
			InSsl:              "true",
			InSslClientCert:    "escc1",
			InNoVerifyCert:     "false",
			InSock:             "eso1",
			InNoReconnect:      "true",
			InReconnectTimeout: "10",
		},
	},
}

var net2 = &Network{
	InName:             "irc2",
	InServers:          []string{"irc.gamesurge.com"},
	InPort:             6666,
	InSsl:              "false",
	InSslCert:          "file2",
	InNoVerifyCert:     "true",
	InNoState:          "true",
	InNoStore:          "true",
	InFloodLenPenalty:  "6",
	InFloodTimeout:     "4.5",
	InFloodStep:        "6.5",
	InKeepAlive:        "8.5",
	InNoReconnect:      "true",
	InReconnectTimeout: "100",
	InNick:             "n2",
	InAltnick:          "a2",
	InUsername:         "u2",
	InUserhost:         "h2",
	InRealname:         "r2",
	InPassword:         "p2",
	InPrefix:           "2",
	InChannels:         []string{"#chan2"},
	InExts: map[string]*Ext{
		"ext": {
			InName:             "ext",
			InConfig:           map[string]string{"k": "2"},
			InLocal:            "false",
			InExec:             "ee2",
			InUseJSON:          "true",
			InIsServer:         "true",
			InAddress:          "ea2",
			InSsl:              "false",
			InSslClientCert:    "escc2",
			InNoVerifyCert:     "true",
			InSock:             "eso2",
			InNoReconnect:      "false",
			InReconnectTimeout: "20",
		},
	},
}

func TestConfig_GetNetwork(t *testing.T) {
	t.Parallel()
	c := NewConfig().FromString("[networks.friend]")
	n := c.GetNetwork("friend")
	if n == nil {
		t.Error("Expected the friend network to exist and be returned.")
	}
}

func TestConfig_Storefile(t *testing.T) {
	t.Parallel()
	c := NewConfig().FromString(`storefile = "filename"`)
	if c.StoreFile() != "filename" {
		t.Error("Store file should return the filename for the config.")
	}

	c = NewConfig().FromString("")
	if c.StoreFile() != defaultStoreFile {
		t.Error("Store file when unset should be the default store file name.")
	}
}

func TestConfig_Fallbacks(t *testing.T) {
	t.Parallel()

	c := NewConfig()
	n1, n2 := *net1, *net2
	e1, e2 := *net1.InExts["ext"], *net2.InExts["ext"]
	n1.protect, n2.protect = &c.protect, &c.protect
	e1.protect, e2.protect = &c.protect, &c.protect
	c.Network = &n1
	n2.parent = c
	e2.parent = &e1

	b := strconv.FormatBool
	u := func(ui uint) string {
		return strconv.Itoa(int(ui))
	}
	f := func(f float64) string {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}

	if exp, got := n2.InName, n2.Name(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n2.InPort, n2.Port(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n2.InSsl, b(n2.Ssl()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n2.InSslCert, n2.SslCert(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n2.InNoVerifyCert, b(n2.NoVerifyCert()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n2.InNoState, b(n2.NoState()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n2.InNoStore, b(n2.NoStore()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n2.InFloodLenPenalty, u(n2.FloodLenPenalty()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n2.InFloodTimeout, f(n2.FloodTimeout()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n2.InFloodStep, f(n2.FloodStep()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n2.InKeepAlive, f(n2.KeepAlive()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n2.InNoReconnect, b(n2.NoReconnect()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n2.InReconnectTimeout, u(n2.ReconnectTimeout()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n2.InNick, n2.Nick(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n2.InAltnick, n2.Altnick(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n2.InUsername, n2.Username(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n2.InUserhost, n2.Userhost(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n2.InRealname, n2.Realname(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n2.InPassword, n2.Password(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n2.InPrefix[0], byte(n2.Prefix()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	for i, s := range n2.Servers() {
		if n2.InServers[i] != s {
			t.Error("Expected: %v, got: %v", n2.InServers[i], s)
		}
	}
	for i, s := range n2.Channels() {
		if n2.InChannels[i] != s {
			t.Error("Expected: %v, got: %v", n2.InServers[i], s)
		}
	}

	if exp, got := e2.InName, e2.Name(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := e2.InConfig["k"], e2.Config()["k"]; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := e2.InLocal, b(e2.Local()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := e2.InExec, e2.Exec(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := e2.InUseJSON, b(e2.UseJSON()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := e2.InIsServer, b(e2.IsServer()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := e2.InAddress, e2.Address(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := e2.InSsl, b(e2.Ssl()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := e2.InSslClientCert, e2.SslClientCert(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := e2.InNoVerifyCert, b(e2.NoVerifyCert()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := e2.InSock, e2.Sock(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := e2.InNoReconnect, b(e2.NoReconnect()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := e2.InReconnectTimeout, u(e2.ReconnectTimeout()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	n2 = Network{protect: &c.protect, parent: c, InName: "servername"}
	e2 = Ext{protect: &c.protect, parent: &e1}

	if exp, got := n1.InPort, n2.Port(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n1.InSsl, b(n2.Ssl()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n1.InSslCert, n2.SslCert(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n1.InNoVerifyCert, b(n2.NoVerifyCert()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n1.InNoState, b(n2.NoState()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n1.InNoStore, b(n2.NoStore()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n1.InFloodLenPenalty, u(n2.FloodLenPenalty()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n1.InFloodTimeout, f(n2.FloodTimeout()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n1.InFloodStep, f(n2.FloodStep()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n1.InKeepAlive, f(n2.KeepAlive()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n1.InNoReconnect, b(n2.NoReconnect()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n1.InReconnectTimeout, u(n2.ReconnectTimeout()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n1.InNick, n2.Nick(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n1.InAltnick, n2.Altnick(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n1.InUsername, n2.Username(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n1.InUserhost, n2.Userhost(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n1.InRealname, n2.Realname(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n1.InPassword, n2.Password(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := n1.InPrefix[0], byte(n2.Prefix()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	for i, s := range n2.Channels() {
		if n1.InChannels[i] != s {
			t.Errorf("Expected: %v, got: %v", n1.InServers[i], s)
		}
	}

	if exp, got := e1.InConfig["k"], e2.Config()["k"]; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := e1.InLocal, b(e2.Local()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := e1.InExec, e2.Exec(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := e1.InUseJSON, b(e2.UseJSON()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := e1.InIsServer, b(e2.IsServer()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := e1.InAddress, e2.Address(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := e1.InSsl, b(e2.Ssl()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := e1.InSslClientCert, e2.SslClientCert(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := e1.InNoVerifyCert, b(e2.NoVerifyCert()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := e1.InSock, e2.Sock(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := e1.InNoReconnect, b(e2.NoReconnect()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := e1.InReconnectTimeout, u(e2.ReconnectTimeout()); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestConfig_Defaults(t *testing.T) {
	t.Parallel()
	c := NewConfig()

	if exp, got := defaultStoreFile, c.StoreFile(); exp != got {
		t.Error("Expected: %v, got: %v")
	}
	if exp, got := defaultIrcPort, c.Port(); exp != got {
		t.Error("Expected: %v, got: %v")
	}
	if exp, got := false, c.Ssl(); exp != got {
		t.Error("Expected: %v, got: %v")
	}
	if exp, got := false, c.NoVerifyCert(); exp != got {
		t.Error("Expected: %v, got: %v")
	}
	if exp, got := false, c.NoState(); exp != got {
		t.Error("Expected: %v, got: %v")
	}
	if exp, got := false, c.NoStore(); exp != got {
		t.Error("Expected: %v, got: %v")
	}
	if exp, got := defaultFloodLenPenalty, c.FloodLenPenalty(); exp != got {
		t.Error("Expected: %v, got: %v")
	}
	if exp, got := defaultFloodTimeout, c.FloodTimeout(); exp != got {
		t.Error("Expected: %v, got: %v")
	}
	if exp, got := defaultFloodStep, c.FloodStep(); exp != got {
		t.Error("Expected: %v, got: %v")
	}
	if exp, got := defaultKeepAlive, c.KeepAlive(); exp != got {
		t.Error("Expected: %v, got: %v")
	}
	if exp, got := false, c.NoReconnect(); exp != got {
		t.Error("Expected: %v, got: %v")
	}
	if exp, got := defaultReconnectTimeout, c.ReconnectTimeout(); exp != got {
		t.Error("Expected: %v, got: %v")
	}
	if exp, got := defaultPrefix, c.Prefix(); exp != got {
		t.Error("Expected: %v, got: %v")
	}
}

func TestConfig_InvalidValues(t *testing.T) {
	t.Parallel()
	c := NewConfig().FromString(`
	nick = "a"
	username = "username"
	realname = "realname"
	[networks.lol]`)
	c.InSsl = "x"
	c.InFloodLenPenalty = "x"
	c.InFloodTimeout = "x"
	c.InFloodStep = "x"
	c.InKeepAlive = "x"
	c.InNoVerifyCert = "x"
	c.InNoState = "x"
	c.InNoStore = "x"
	c.InNoReconnect = "x"
	c.InReconnectTimeout = "x"
	c.InPrefix = "xx"

	if exp, got := false, c.Ssl(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := false, c.NoVerifyCert(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := false, c.NoState(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := false, c.NoStore(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := defaultFloodLenPenalty, c.FloodLenPenalty(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := defaultFloodTimeout, c.FloodTimeout(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := defaultFloodStep, c.FloodStep(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := defaultKeepAlive, c.KeepAlive(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := false, c.NoReconnect(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := defaultReconnectTimeout, c.ReconnectTimeout(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	if exp, got := false, c.IsValid(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	if exp, got := 10, len(c.errors); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	ers := make([]string, len(c.errors))
	for i, e := range c.errors {
		ers[i] = e.Error()
	}
	if exp, got := errSsl, ers[0]; !strings.Contains(got, exp) {
		t.Errorf("Expected: \"%v\" to contain \"%v\"", got, exp)
	}
	if exp, got := errNoVerifyCert, ers[1]; !strings.Contains(got, exp) {
		t.Errorf("Expected: \"%v\" to contain \"%v\"", got, exp)
	}
	if exp, got := errNoState, ers[2]; !strings.Contains(got, exp) {
		t.Errorf("Expected: \"%v\" to contain \"%v\"", got, exp)
	}
	if exp, got := errNoStore, ers[3]; !strings.Contains(got, exp) {
		t.Errorf("Expected: \"%v\" to contain \"%v\"", got, exp)
	}
	if exp, got := errFloodLenPenalty, ers[4]; !strings.Contains(got, exp) {
		t.Errorf("Expected: \"%v\" to contain \"%v\"", got, exp)
	}
	if exp, got := errFloodTimeout, ers[5]; !strings.Contains(got, exp) {
		t.Errorf("Expected: \"%v\" to contain \"%v\"", got, exp)
	}
	if exp, got := errFloodStep, ers[6]; !strings.Contains(got, exp) {
		t.Errorf("Expected: \"%v\" to contain \"%v\"", got, exp)
	}
	if exp, got := errKeepAlive, ers[7]; !strings.Contains(got, exp) {
		t.Errorf("Expected: \"%v\" to contain \"%v\"", got, exp)
	}
	if exp, got := errNoReconnect, ers[8]; !strings.Contains(got, exp) {
		t.Errorf("Expected: \"%v\" to contain \"%v\"", got, exp)
	}
	if exp, got := errReconnectTimeout, ers[9]; !strings.Contains(got, exp) {
		t.Errorf("Expected: \"%v\" to contain \"%v\"", got, exp)
	}
}

/*
func TestConfig(t *testing.T) {
	_ := NewConfig()
}

func TestConfig_Fallbacks(t *testing.T) {
	config := NewConfig()

	host, name := "irc.gamesurge.net", "gamesurge"

	net := *net1
	config.Global = &net

	network := &Network{parent: config, Name: name, Host: host}
	config.Networks[name] = network

	c.Check(network.GetHost(), Equals, host)
	c.Check(network.GetName(), Equals, name)
	c.Check(network.GetPort(), Equals, config.Network.GetPort())
	c.Check(network.GetSsl(), Equals, config.Network.GetSsl())
	c.Check(network.GetSslCert(), Equals, config.Network.GetSslCert())
	c.Check(network.GetNoVerifyCert(), Equals, config.Network.GetNoVerifyCert())
	c.Check(network.GetNoState(), Equals, config.Network.GetNoState())
	c.Check(network.GetNoStore(), Equals, config.Network.GetNoStore())
	c.Check(network.GetFloodLenPenalty(), Equals,
		config.Network.GetFloodLenPenalty())
	c.Check(network.GetFloodTimeout(), Equals, config.Network.GetFloodTimeout())
	c.Check(network.GetFloodStep(), Equals, config.Network.GetFloodStep())
	c.Check(network.GetKeepAlive(), Equals, config.Network.GetKeepAlive())
	c.Check(network.GetNoReconnect(), Equals, config.Network.GetNoReconnect())
	c.Check(network.GetReconnectTimeout(), Equals,
		config.Network.GetReconnectTimeout())
	c.Check(network.GetNick(), Equals, config.Network.GetNick())
	c.Check(network.GetAltnick(), Equals, config.Network.GetAltnick())
	c.Check(network.GetUsername(), Equals, config.Network.GetUsername())
	c.Check(network.GetUserhost(), Equals, config.Network.GetUserhost())
	c.Check(network.GetRealname(), Equals, config.Network.GetRealname())
	c.Check(network.GetPrefix(), Equals, config.Network.GetPrefix())
	c.Check(len(network.GetChannels()), Equals, len(config.Network.Channels))
	for i, v := range network.GetChannels() {
		c.Check(v, Equals, config.Network.Channels[i])
	}
}

func TestConfig_Globals(t *testing.T) {
	t.Parallel()
	conf := NewConfig().
		StoreFile("store").
		Nick(net1.Nick).
		Realname(net1.Realname).
		Username(net1.Username).
		Userhost(net1.Userhost).
		Network(net1.GetName())

	c.Check(conf.GetStoreFile(), Equals, "store")
	c.Check(conf.IsValid(), Equals, true)
}

func TestConfig_Defaults(t *testing.T) {
	t.Parallel()
	conf := NewConfig().FromString(tnaheaoeth)
	net := conf.GetNetwork(net1.GetName())

	c.Check(conf.GetStoreFile(), Equals, defaultStoreFile)
	c.Check(net.GetPort(), Equals, defaultIrcPort)
	c.Check(net.GetSsl(), Equals, false)
	c.Check(net.GetNoVerifyCert(), Equals, false)
	c.Check(net.GetNoState(), Equals, false)
	c.Check(net.GetNoStore(), Equals, false)
	c.Check(net.GetFloodLenPenalty(), Equals, defaultFloodLenPenalty)
	c.Check(net.GetFloodTimeout(), Equals, defaultFloodTimeout)
	c.Check(net.GetFloodStep(), Equals, defaultFloodStep)
	c.Check(net.GetKeepAlive(), Equals, defaultKeepAlive)
	c.Check(net.GetNoReconnect(), Equals, false)
	c.Check(net.GetReconnectTimeout(), Equals, defaultReconnectTimeout)
	c.Check(net.GetPrefix(), Equals, defaultPrefix)
}

func TestConfig_InvalidValues(t *testing.T) {
	t.Parallel()
	conf := NewConfig().
		Nick(net1.Nick).
		Realname(net1.Realname).
		Username(net1.Username).
		Userhost(net1.Userhost).
		Network(net1.GetName())
	net := conf.GetNetwork(net1.GetName())
	net.Ssl = "x"
	net.FloodLenPenalty = "x"
	net.FloodTimeout = "x"
	net.FloodStep = "x"
	net.KeepAlive = "x"
	net.NoVerifyCert = "x"
	net.NoState = "x"
	net.NoStore = "x"
	net.NoReconnect = "x"
	net.ReconnectTimeout = "x"
	net.Prefix = "xx"

	c.Check(net.GetSsl(), Equals, false)
	c.Check(net.GetNoVerifyCert(), Equals, false)
	c.Check(net.GetNoState(), Equals, false)
	c.Check(net.GetNoStore(), Equals, false)
	c.Check(net.GetFloodLenPenalty(), Equals, defaultFloodLenPenalty)
	c.Check(net.GetFloodTimeout(), Equals, defaultFloodTimeout)
	c.Check(net.GetFloodStep(), Equals, defaultFloodStep)
	c.Check(net.GetKeepAlive(), Equals, defaultKeepAlive)
	c.Check(net.GetNoReconnect(), Equals, false)
	c.Check(net.GetReconnectTimeout(), Equals, defaultReconnectTimeout)

	c.Check(conf.IsValid(), Equals, false)
	c.Check(len(conf.Errors), Equals, 10)
	c.Check(conf.Errors[0].Error(), Matches, invErr(errSsl))
	c.Check(conf.Errors[1].Error(), Matches, invErr(errNoVerifyCert))
	c.Check(conf.Errors[2].Error(), Matches, invErr(errNoState))
	c.Check(conf.Errors[3].Error(), Matches, invErr(errNoStore))
	c.Check(conf.Errors[4].Error(), Matches, invErr(errFloodLenPenalty))
	c.Check(conf.Errors[5].Error(), Matches, invErr(errFloodTimeout))
	c.Check(conf.Errors[6].Error(), Matches, invErr(errFloodStep))
	c.Check(conf.Errors[7].Error(), Matches, invErr(errKeepAlive))
	c.Check(conf.Errors[8].Error(), Matches, invErr(errNoReconnect))
	c.Check(conf.Errors[9].Error(), Matches, invErr(errReconnectTimeout))
}

func TestConfig_ValidationEmpty(t *testing.T) {
	t.Parallel()
	conf := NewConfig()
	c.Check(conf.IsValid(), Equals, false)
	c.Check(len(conf.Errors), Equals, 1)
	c.Check(conf.Errors[0].Error(), Equals, errMsgNetworksRequired)
}

func TestConfig_ValidationNoHost(t *testing.T) {
	t.Parallel()
	conf := NewConfig().
		Network("").
		Port(net1.Port)
	c.Check(len(conf.Networks), Equals, 0)
	c.Check(conf.Global.Port, Equals, uint16(net1.Port))
	c.Check(conf.IsValid(), Equals, false)
	c.Check(len(conf.Errors), Equals, 2)
	c.Check(conf.Errors[0].Error(), Matches, reqErr(errHost))
	c.Check(conf.Errors[1].Error(), Equals, errMsgNetworksRequired)
}

func TestConfig_ValidationInvalidHost(t *testing.T) {
	t.Parallel()
	conf := NewConfig().
		Nick(net1.Nick).
		Realname(net1.Realname).
		Username(net1.Username).
		Userhost(net1.Userhost).
		Network("%")
	c.Check(conf.IsValid(), Equals, false)
	c.Check(len(conf.Errors), Equals, 1)
	c.Check(conf.Errors[0].Error(), Matches, invErr(errHost))
}

func TestConfig_ValidationNoHostInternal(t *testing.T) {
	t.Parallel()
	conf := NewConfig().
		Network(net1.Host).
		Nick(net1.Nick).
		Channels(net1.Channels...).
		Username(net1.Username).
		Userhost(net1.Userhost).
		Realname(net1.Realname)
	conf.Networks[net1.Host].Host = ""
	c.Check(conf.IsValid(), Equals, false)
	c.Check(len(conf.Errors), Equals, 1) // Internal No host
	c.Check(conf.Errors[0].Error(), Matches, reqErr(errHost))
}

func TestConfig_ValidationDuplicateName(t *testing.T) {
	t.Parallel()
	conf := NewConfig().
		Nick(net1.Nick).
		Realname(net1.Realname).
		Username(net1.Username).
		Userhost(net1.Userhost).
		Network("a.com").
		Network("a.com")
	c.Check(len(conf.Networks), Equals, 1)
	c.Check(conf.IsValid(), Equals, false)
	c.Check(len(conf.Errors), Equals, 1)
	c.Check(conf.Errors[0].Error(), Equals, errMsgDuplicateNetwork)
}

func TestConfig_ValidationMissing(t *testing.T) {
	t.Parallel()
	conf := NewConfig().
		Network(net1.Host)
	c.Check(conf.IsValid(), Equals, false)
	// Missing: Nick, Username, Userhost, Realname
	c.Check(len(conf.Errors), Equals, 4)
	c.Check(conf.Errors[0].Error(), Matches, reqErr(errNick))
	c.Check(conf.Errors[1].Error(), Matches, reqErr(errUsername))
	c.Check(conf.Errors[2].Error(), Matches, reqErr(errUserhost))
	c.Check(conf.Errors[3].Error(), Matches, reqErr(errRealname))
}

func TestConfig_ValidationRegex(t *testing.T) {
	t.Parallel()
	conf := NewConfig().
		Network(net1.Host).
		Nick(`@Nick`).              // no special chars
		Channels(`chan`).           // must start with valid prefix
		Username(`spaces in here`). // no spaces
		Userhost(`~#@#$@!`).        // must be a host
		Realname(`@ !`)             // no special chars
	c.Check(conf.IsValid(), Equals, false)
	c.Check(len(conf.Errors), Equals, 5)
	c.Check(conf.Errors[0].Error(), Matches, invErr(errNick))
	c.Check(conf.Errors[1].Error(), Matches, invErr(errUsername))
	c.Check(conf.Errors[2].Error(), Matches, invErr(errUserhost))
	c.Check(conf.Errors[3].Error(), Matches, invErr(errRealname))
	c.Check(conf.Errors[4].Error(), Matches, invErr(errChannel))
}

func TestConfig_DisplayErrors(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	log.SetOutput(buf)
	c.Check(buf.Len(), Equals, 0)
	conf := NewConfig().
		Network("localhost")
	c.Check(conf.IsValid(), Equals, false)
	c.Check(len(conf.Errors), Equals, 4)
	conf.DisplayErrors()
	c.Check(buf.Len(), Not(Equals), 0)
	setLogger() // Reset the logger
}

func TestConfig_GetNetwork(t *testing.T) {
	t.Parallel()
	conf := NewConfig()
	conf.Networks[net1.GetName()] = net1
	conf.Networks[net2.GetName()] = net2
	c.Check(conf.GetNetwork(net1.GetName()), Equals, net1)
	c.Check(conf.GetNetwork(net2.GetName()), Equals, net2)
}

func TestConfig_Clone(t *testing.T) {
	t.Parallel()

}
*/
