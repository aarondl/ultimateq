package config

import (
	"bytes"
	. "launchpad.net/gocheck"
	"log"
	"os"
	"testing"
)

func Test(t *testing.T) { TestingT(t) } //Hook into testing package
type s struct{}

var _ = Suite(&s{})

func init() {
	setLogger() // This had to be done for DisplayErrors' test
}

func setLogger() {
	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		log.Println("Could not set logger:", err)
	} else {
		log.SetOutput(f)
	}
}

func reqErr(name string) string {
	return `.*Requires.*` + name + `.*`
}

func invErr(name string) string {
	return `.*Invalid.*` + name + `.*`
}

var srv1 = &Server{
	Name:                "irc1",
	Host:                "irc.gamesurge.net",
	Port:                5555,
	Ssl:                 "true",
	VerifyCert:          "false",
	NoState:             "false",
	FloodProtectBurst:   "5",
	FloodProtectTimeout: "3.5",
	FloodProtectStep:    "5.5",
	NoReconnect:         "false",
	ReconnectTimeout:    "10",
	Nick:                "n1",
	Altnick:             "a1",
	Username:            "u1",
	Userhost:            "h1",
	Realname:            "r1",
	Prefix:              "p1",
	Channels:            []string{"#chan1", "#chan2"},
}

var srv2 = &Server{
	Name:                "irc2",
	Host:                "irc.gamesurge.com",
	Port:                6666,
	Ssl:                 "false",
	VerifyCert:          "true",
	NoState:             "true",
	FloodProtectBurst:   "6",
	FloodProtectTimeout: "4.5",
	FloodProtectStep:    "6.5",
	NoReconnect:         "true",
	ReconnectTimeout:    "100",
	Nick:                "n2",
	Altnick:             "a2",
	Username:            "u2",
	Userhost:            "h2",
	Realname:            "r2",
	Prefix:              "p2",
	Channels:            []string{"#chan2"},
}

func (s *s) TestConfig(c *C) {
	config := CreateConfig()
	c.Check(config.Servers, NotNil)
	c.Check(config.Global, NotNil)
}

func (s *s) TestConfig_Fallbacks(c *C) {
	config := CreateConfig()

	host, name := "irc.gamesurge.net", "gamesurge"

	srv := *srv1
	config.Global = &srv

	server := &Server{parent: config, Name: name, Host: host}
	config.Servers[name] = server

	c.Check(server.GetHost(), Equals, host)
	c.Check(server.GetName(), Equals, name)
	c.Check(server.GetPort(), Equals, config.Global.GetPort())
	c.Check(server.GetSsl(), Equals, config.Global.GetSsl())
	c.Check(server.GetVerifyCert(), Equals, config.Global.GetVerifyCert())
	c.Check(server.GetNoState(), Equals, config.Global.GetNoState())
	c.Check(server.GetFloodProtectBurst(), Equals,
		config.Global.GetFloodProtectBurst())
	c.Check(server.GetFloodProtectTimeout(), Equals,
		config.Global.GetFloodProtectTimeout())
	c.Check(server.GetFloodProtectStep(), Equals,
		config.Global.GetFloodProtectStep())
	c.Check(server.GetNoReconnect(), Equals, config.Global.GetNoReconnect())
	c.Check(server.GetReconnectTimeout(), Equals,
		config.Global.GetReconnectTimeout())
	c.Check(server.GetNick(), Equals, config.Global.GetNick())
	c.Check(server.GetAltnick(), Equals, config.Global.GetAltnick())
	c.Check(server.GetUsername(), Equals, config.Global.GetUsername())
	c.Check(server.GetUserhost(), Equals, config.Global.GetUserhost())
	c.Check(server.GetRealname(), Equals, config.Global.GetRealname())
	c.Check(server.GetPrefix(), Equals, config.Global.GetPrefix())
	c.Check(len(server.GetChannels()), Equals, len(config.Global.Channels))
	for i, v := range server.GetChannels() {
		c.Check(v, Equals, config.Global.Channels[i])
	}
}

func (s *s) TestConfig_Fluent(c *C) {
	srv2host := "znc.gamesurge.net"

	conf := CreateConfig().
		// Setting Globals
		Host(""). // Should not break anything
		Port(srv2.GetPort()).
		Ssl(srv2.GetSsl()).
		VerifyCert(srv2.GetVerifyCert()).
		NoState(srv2.GetNoState()).
		FloodProtectBurst(srv2.GetFloodProtectBurst()).
		FloodProtectTimeout(srv2.GetFloodProtectTimeout()).
		FloodProtectStep(srv2.GetFloodProtectStep()).
		NoReconnect(srv2.GetNoReconnect()).
		ReconnectTimeout(srv2.GetReconnectTimeout()).
		Nick(srv2.GetNick()).
		Altnick(srv2.GetAltnick()).
		Username(srv2.GetUsername()).
		Userhost(srv2.GetUserhost()).
		Realname(srv2.GetRealname()).
		Prefix(srv2.GetPrefix()).
		Channels(srv2.GetChannels()...).
		// Server 1
		Server(srv1.GetName()).
		Host(srv1.GetHost()).
		Port(srv1.GetPort()).
		Ssl(srv1.GetSsl()).
		VerifyCert(srv1.GetVerifyCert()).
		NoState(srv1.GetNoState()).
		FloodProtectBurst(srv1.GetFloodProtectBurst()).
		FloodProtectTimeout(srv1.GetFloodProtectTimeout()).
		FloodProtectStep(srv1.GetFloodProtectStep()).
		NoReconnect(srv1.GetNoReconnect()).
		ReconnectTimeout(srv1.GetReconnectTimeout()).
		Nick(srv1.GetNick()).
		Altnick(srv1.GetAltnick()).
		Username(srv1.GetUsername()).
		Userhost(srv1.GetUserhost()).
		Realname(srv1.GetRealname()).
		Prefix(srv1.GetPrefix()).
		Channels(srv1.GetChannels()...).
		// Server 2 using defaults
		Server(srv2host)

	server := conf.GetServer(srv1.Name)
	server2 := conf.GetServer(srv2host)
	c.Check(server.GetHost(), Equals, srv1.GetHost())
	c.Check(server.GetName(), Equals, srv1.GetName())
	c.Check(server.GetPort(), Equals, srv1.GetPort())
	c.Check(server.GetSsl(), Equals, srv1.GetSsl())
	c.Check(server.GetVerifyCert(), Equals, srv1.GetVerifyCert())
	c.Check(server.GetNoState(), Equals, srv1.GetNoState())
	c.Check(server.GetFloodProtectBurst(), Equals, srv1.GetFloodProtectBurst())
	c.Check(server.GetFloodProtectTimeout(), Equals,
		srv1.GetFloodProtectTimeout())
	c.Check(server.GetFloodProtectStep(), Equals, srv1.GetFloodProtectStep())
	c.Check(server.GetNoReconnect(), Equals, srv1.GetNoReconnect())
	c.Check(server.GetReconnectTimeout(), Equals, srv1.GetReconnectTimeout())
	c.Check(server.GetNick(), Equals, srv1.GetNick())
	c.Check(server.GetAltnick(), Equals, srv1.GetAltnick())
	c.Check(server.GetUsername(), Equals, srv1.GetUsername())
	c.Check(server.GetUserhost(), Equals, srv1.GetUserhost())
	c.Check(server.GetRealname(), Equals, srv1.GetRealname())
	c.Check(server.GetPrefix(), Equals, srv1.GetPrefix())
	c.Check(len(server.GetChannels()), Equals, len(srv1.Channels))
	for i, v := range server.GetChannels() {
		c.Check(v, Equals, srv1.Channels[i])
	}

	c.Check(server2.GetHost(), Equals, srv2host)
	c.Check(server2.GetPort(), Equals, srv2.GetPort())
	c.Check(server2.GetSsl(), Equals, srv2.GetSsl())
	c.Check(server2.GetVerifyCert(), Equals, srv2.GetVerifyCert())
	c.Check(server2.GetNoState(), Equals, srv2.GetNoState())
	c.Check(server2.GetFloodProtectBurst(), Equals, srv2.GetFloodProtectBurst())
	c.Check(server2.GetFloodProtectTimeout(), Equals,
		srv2.GetFloodProtectTimeout())
	c.Check(server2.GetFloodProtectStep(), Equals, srv2.GetFloodProtectStep())
	c.Check(server2.GetNoReconnect(), Equals, srv2.GetNoReconnect())
	c.Check(server2.GetReconnectTimeout(), Equals, srv2.GetReconnectTimeout())
	c.Check(server2.GetNick(), Equals, srv2.GetNick())
	c.Check(server2.GetAltnick(), Equals, srv2.GetAltnick())
	c.Check(server2.GetUsername(), Equals, srv2.GetUsername())
	c.Check(server2.GetUserhost(), Equals, srv2.GetUserhost())
	c.Check(server2.GetRealname(), Equals, srv2.GetRealname())
	c.Check(server2.GetPrefix(), Equals, srv2.GetPrefix())
	c.Check(len(server2.GetChannels()), Equals, len(srv2.Channels))
	for i, v := range server2.GetChannels() {
		c.Check(v, Equals, srv2.Channels[i])
	}
}

func (s *s) TestConfig_Defaults(c *C) {
	conf := CreateConfig().
		Nick(srv1.Nick).
		Realname(srv1.Realname).
		Username(srv1.Username).
		Userhost(srv1.Userhost).
		Server(srv1.GetName())
	srv := conf.GetServer(srv1.GetName())

	c.Check(srv.GetPort(), Equals, defaultIrcPort)
	c.Check(srv.GetSsl(), Equals, false)
	c.Check(srv.GetVerifyCert(), Equals, false)
	c.Check(srv.GetNoState(), Equals, false)
	c.Check(srv.GetFloodProtectBurst(), Equals, defaultFloodProtectBurst)
	c.Check(srv.GetFloodProtectTimeout(), Equals, defaultFloodProtectTimeout)
	c.Check(srv.GetFloodProtectStep(), Equals, defaultFloodProtectStep)
	c.Check(srv.GetNoReconnect(), Equals, false)
	c.Check(srv.GetReconnectTimeout(), Equals, defaultReconnectTimeout)
}

func (s *s) TestConfig_InvalidValues(c *C) {
	conf := CreateConfig().
		Nick(srv1.Nick).
		Realname(srv1.Realname).
		Username(srv1.Username).
		Userhost(srv1.Userhost).
		Server(srv1.GetName())
	srv := conf.GetServer(srv1.GetName())
	srv.Ssl = "x"
	srv.FloodProtectBurst = "x"
	srv.FloodProtectStep = "x"
	srv.FloodProtectTimeout = "x"
	srv.VerifyCert = "x"
	srv.NoState = "x"
	srv.NoReconnect = "x"
	srv.ReconnectTimeout = "x"

	c.Check(srv.GetSsl(), Equals, false)
	c.Check(srv.GetVerifyCert(), Equals, false)
	c.Check(srv.GetNoState(), Equals, false)
	c.Check(srv.GetFloodProtectBurst(), Equals, defaultFloodProtectBurst)
	c.Check(srv.GetFloodProtectTimeout(), Equals, defaultFloodProtectTimeout)
	c.Check(srv.GetFloodProtectStep(), Equals, defaultFloodProtectStep)
	c.Check(srv.GetNoReconnect(), Equals, false)
	c.Check(srv.GetReconnectTimeout(), Equals, defaultReconnectTimeout)

	c.Check(conf.IsValid(), Equals, false)
	c.Check(len(conf.Errors), Equals, 8)
	c.Check(conf.Errors[0].Error(), Matches, invErr(errSsl))
	c.Check(conf.Errors[1].Error(), Matches, invErr(errVerifyCert))
	c.Check(conf.Errors[2].Error(), Matches, invErr(errNoState))
	c.Check(conf.Errors[3].Error(), Matches, invErr(errFloodProtectBurst))
	c.Check(conf.Errors[4].Error(), Matches, invErr(errFloodProtectTimeout))
	c.Check(conf.Errors[5].Error(), Matches, invErr(errFloodProtectStep))
	c.Check(conf.Errors[6].Error(), Matches, invErr(errNoReconnect))
	c.Check(conf.Errors[7].Error(), Matches, invErr(errReconnectTimeout))
}

func (s *s) TestConfig_ValidationEmpty(c *C) {
	conf := CreateConfig()
	c.Check(conf.IsValid(), Equals, false)
	c.Check(len(conf.Errors), Equals, 1)
	c.Check(conf.Errors[0].Error(), Equals, errMsgServersRequired)
}

func (s *s) TestConfig_ValidationNoHost(c *C) {
	conf := CreateConfig().
		Server("").
		Port(srv1.Port)
	c.Check(len(conf.Servers), Equals, 0)
	c.Check(conf.Global.Port, Equals, uint16(srv1.Port))
	c.Check(conf.IsValid(), Equals, false)
	c.Check(len(conf.Errors), Equals, 2)
	c.Check(conf.Errors[0].Error(), Matches, reqErr(errHost))
	c.Check(conf.Errors[1].Error(), Equals, errMsgServersRequired)
}

func (s *s) TestConfig_ValidationInvalidHost(c *C) {
	conf := CreateConfig().
		Nick(srv1.Nick).
		Realname(srv1.Realname).
		Username(srv1.Username).
		Userhost(srv1.Userhost).
		Server("%")
	c.Check(conf.IsValid(), Equals, false)
	c.Check(len(conf.Errors), Equals, 1)
	c.Check(conf.Errors[0].Error(), Matches, invErr(errHost))
}

func (s *s) TestConfig_ValidationNoHostInternal(c *C) {
	conf := CreateConfig().
		Server(srv1.Host).
		Nick(srv1.Nick).
		Channels(srv1.Channels...).
		Username(srv1.Username).
		Userhost(srv1.Userhost).
		Realname(srv1.Realname)
	conf.Servers[srv1.Host].Host = ""
	c.Check(conf.IsValid(), Equals, false)
	c.Check(len(conf.Errors), Equals, 1) // Internal No host
	c.Check(conf.Errors[0].Error(), Matches, reqErr(errHost))
}

func (s *s) TestConfig_ValidationDuplicateName(c *C) {
	conf := CreateConfig().
		Nick(srv1.Nick).
		Realname(srv1.Realname).
		Username(srv1.Username).
		Userhost(srv1.Userhost).
		Server("a.com").
		Server("a.com")
	c.Check(len(conf.Servers), Equals, 1)
	c.Check(conf.IsValid(), Equals, false)
	c.Check(len(conf.Errors), Equals, 1)
	c.Check(conf.Errors[0].Error(), Equals, errMsgDuplicateServer)
}

func (s *s) TestConfig_ValidationMissing(c *C) {
	conf := CreateConfig().
		Server(srv1.Host)
	c.Check(conf.IsValid(), Equals, false)
	// Missing: Nick, Username, Userhost, Realname
	c.Check(len(conf.Errors), Equals, 4)
	c.Check(conf.Errors[0].Error(), Matches, reqErr(errNick))
	c.Check(conf.Errors[1].Error(), Matches, reqErr(errUsername))
	c.Check(conf.Errors[2].Error(), Matches, reqErr(errUserhost))
	c.Check(conf.Errors[3].Error(), Matches, reqErr(errRealname))
}

func (s *s) TestConfig_ValidationRegex(c *C) {
	conf := CreateConfig().
		Server(srv1.Host).
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

func (s *s) TestConfig_DisplayErrors(c *C) {
	buf := &bytes.Buffer{}
	log.SetOutput(buf)
	c.Check(buf.Len(), Equals, 0)
	conf := CreateConfig().
		Server("localhost")
	c.Check(conf.IsValid(), Equals, false)
	c.Check(len(conf.Errors), Equals, 4)
	conf.DisplayErrors()
	c.Check(buf.Len(), Not(Equals), 0)
	setLogger() // Reset the logger
}

func (s *s) TestConfig_GetServer(c *C) {
	conf := CreateConfig()
	conf.Servers[srv1.GetName()] = srv1
	conf.Servers[srv2.GetName()] = srv2
	c.Check(conf.GetServer(srv1.GetName()), Equals, srv1)
	c.Check(conf.GetServer(srv2.GetName()), Equals, srv2)
}

func (s *s) TestConfig_RemoveServer(c *C) {
	conf := CreateConfig()
	conf.Servers[srv1.GetName()] = srv1
	conf.Servers[srv2.GetName()] = srv2
	c.Check(conf.GetServer(srv1.GetName()), Equals, srv1)
	c.Check(conf.GetServer(srv2.GetName()), Equals, srv2)

	conf.ServerContext(srv1.GetName())
	c.Check(conf.context, NotNil)

	conf.RemoveServer(srv1.GetName())
	c.Check(conf.IsValid(), Equals, true)

	c.Check(conf.context, IsNil)

	c.Check(conf.GetServer(srv1.GetName()), IsNil)
	c.Check(conf.GetServer(srv2.GetName()), Equals, srv2)
}

func (s *s) TestConfig_SetContext(c *C) {
	conf := CreateConfig()
	srv := *srv1
	conf.Servers[srv1.GetName()] = &srv

	var p1, p2, p3 uint16 = 1, 2, 3

	conf.Port(p1) // Should set global context
	c.Check(conf.Global.GetPort(), Equals, p1)

	conf.ServerContext(srv1.GetName())
	conf.Port(p2)
	conf.GlobalContext()
	conf.Port(p3)

	c.Check(conf.GetServer(srv1.GetName()).GetPort(), Equals, p2)
	c.Check(conf.Global.GetPort(), Equals, p3)

	c.Check(len(conf.Errors), Equals, 0)
	conf.ServerContext("")
	c.Check(len(conf.Errors), Equals, 1)
	c.Check(conf.IsValid(), Equals, false)
	c.Check(conf.Errors[0].Error(), Matches, fmtErrServerNotFound[:33]+".*")
}

func (s *s) TestValidNames(c *C) {
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
			c.Errorf("Good nick failed regex: %v\n", goodNicks[i])
		}
	}
	for i := 0; i < len(badNicks); i++ {
		if rgxNickname.MatchString(badNicks[i]) {
			c.Errorf("Bad nick passed regex: %v\n", badNicks[i])
		}
	}
}

func (s *s) TestConfig_Clone(c *C) {
	conf := CreateConfig()

	srv := *srv1
	srv.parent = conf
	name := srv1.Name
	filename := "file.yaml"
	conf.filename = filename
	conf.Servers[name] = &srv

	var globalPort, serverPort uint16 = 1, 2

	newconf := conf.Clone().
		GlobalContext().
		Port(globalPort).
		ServerContext(name).
		Port(0)

	c.Check(newconf.GetFilename(), Equals, conf.GetFilename())

	newconf.GlobalContext()
	c.Check(conf.Global.Port, Not(Equals), globalPort)
	c.Check(srv1.Port, Not(Equals), globalPort)
	c.Check(newconf.GetServer(name).GetPort(), Equals, globalPort)

	newconf.
		ServerContext(srv1.Name).
		Port(serverPort)

	c.Check(conf.Global.Port, Not(Equals), serverPort)
	c.Check(srv1.Port, Not(Equals), serverPort)
	c.Check(newconf.GetServer(name).GetPort(), Equals, serverPort)
}

func (s *s) TestConfig_Filename(c *C) {
	conf := CreateConfig()
	filename := "file.yaml"
	c.Check(conf.GetFilename(), Equals, defaultConfigFileName)
	conf.filename = filename
	c.Check(conf.GetFilename(), Equals, filename)
}

func (s *s) TestValidChannels(c *C) {
	// Check that the first letter must be {#+!&}
	goodChannels := []string{"#ValidChannel", "+ValidChannel", "&ValidChannel",
		"!12345", "#c++"}

	badChannels := []string{"#Invalid Channel", "#Invalid,Channel",
		"#Invalid\aChannel", "#", "+", "&", "InvalidChannel"}

	for i := 0; i < len(goodChannels); i++ {
		if !rgxChannel.MatchString(goodChannels[i]) {
			c.Errorf("Good chan failed regex: %v\n", goodChannels[i])
		}
	}
	for i := 0; i < len(badChannels); i++ {
		if rgxChannel.MatchString(badChannels[i]) {
			c.Errorf("Bad chan passed regex: %v\n", badChannels[i])
		}
	}
}
