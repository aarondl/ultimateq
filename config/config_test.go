package config

import (
	"bytes"
	"log"
	"testing"
)

func reqErr(name string) string {
	return `.*Requires.*` + name + `.*`
}

func invErr(name string) string {
	return `.*Invalid.*` + name + `.*`
}

var net1 = &Network{
	Name:             "irc1",
	Host:             "irc.gamesurge.net",
	Port:             5555,
	Ssl:              "true",
	SslCert:          "file1",
	NoVerifyCert:     "false",
	NoState:          "false",
	NoStore:          "true",
	FloodLenPenalty:  "5",
	FloodTimeout:     "3.5",
	FloodStep:        "5.5",
	KeepAlive:        "7.5",
	NoReconnect:      "false",
	ReconnectTimeout: "10",
	Nick:             "n1",
	Altnick:          "a1",
	Username:         "u1",
	Userhost:         "h1",
	Realname:         "r1",
	Prefix:           "1",
	Channels:         []string{"#chan1", "#chan2"},
}

var net2 = &Network{
	Name:             "irc2",
	Host:             "irc.gamesurge.com",
	Port:             6666,
	Ssl:              "false",
	SslCert:          "file2",
	NoVerifyCert:     "true",
	NoState:          "true",
	NoStore:          "true",
	FloodLenPenalty:  "6",
	FloodTimeout:     "4.5",
	FloodStep:        "6.5",
	KeepAlive:        "8.5",
	NoReconnect:      "true",
	ReconnectTimeout: "100",
	Nick:             "n2",
	Altnick:          "a2",
	Username:         "u2",
	Userhost:         "h2",
	Realname:         "r2",
	Prefix:           "2",
	Channels:         []string{"#chan2"},
}

func TestConfig(t *testing.T) {
	config := NewConfig()
	c.Check(config.Networks, NotNil)
	c.Check(config.Global, NotNil)
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

func TestConfig_SetContext(t *testing.T) {
	t.Parallel()
}

func TestConfig_Clone(t *testing.T) {
	t.Parallel()
}

func TestValidNames(t *testing.T) {
	t.Parallel()
	goodNicks := []string{`a1bc`, `a5bc`, `a9bc`, `MyNick`, `[MyNick`,
		`My[Nick`, `]MyNick`, `My]Nick`, `\MyNick`, `My\Nick`, "MyNick",
		"My`Nick", `_MyNick`, `My_Nick`, `^MyNick`, `My^Nick`, `{MyNick`,
		`My{Nick`, `|MyNick`, `My|Nick`, `}MyNick`, `My}Nick`,
	}

	badNicks := []string{`My Name`, `My!Nick`, `My"Nick`, `My#Nick`, `My$Nick`,
		`My%Nick`, `My&Nick`, `My'Nick`, `My/Nick`, `My(Nick`, `My)Nick`,
		`My*Nick`, `My+Nick`, `My,Nick`, `My-Nick`, `My.Nick`, `My/Nick`,
		`My;Nick`, `My:Nick`, `My<Nick`, `My=Nick`, `My>Nick`, `My?Nick`,
		`My@Nick`, `1abc`, `5abc`, `9abc`, `@ChanServ`,
	}

	for i := 0; i < len(goodNicks); i++ {
		if !rgxNickname.MatchString(goodNicks[i]) {
			t.Errorf("Good nick failed regex: %v\n", goodNicks[i])
		}
	}
	for i := 0; i < len(badNicks); i++ {
		if rgxNickname.MatchString(badNicks[i]) {
			t.Errorf("Bad nick passed regex: %v\n", badNicks[i])
		}
	}
}

func TestValidChannels(t *testing.T) {
	t.Parallel()
	// Check that the first letter must be {#+!&}
	goodChannels := []string{"#ValidChannel", "+ValidChannel", "&ValidChannel",
		"!12345", "#c++"}

	badChannels := []string{"#Invalid Channel", "#Invalid,Channel",
		"#Invalid\aChannel", "#", "+", "&", "InvalidChannel"}

	for i := 0; i < len(goodChannels); i++ {
		if !rgxChannel.MatchString(goodChannels[i]) {
			t.Errorf("Good chan failed regex: %v\n", goodChannels[i])
		}
	}
	for i := 0; i < len(badChannels); i++ {
		if rgxChannel.MatchString(badChannels[i]) {
			t.Errorf("Bad chan passed regex: %v\n", badChannels[i])
		}
	}
}
