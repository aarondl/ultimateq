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

var srv1 = &Server{
	nil, "irc", "irc.gamesurge.net",
	5555, true, true, false, true, false, true, false, true, 10,
	"n1", "a1", "u1", "h1", "r1", "p1",
	[]string{"#chan", "#chan2"},
}

var srv2 = &Server{
	nil, "irc2", "nuclearfallout.gamesurge.net",
	7777, false, false, true, false, false, false, false, false, 10,
	"n2", "a2", "u2", "h2", "r2", "p2",
	[]string{"#chan2"},
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
	c.Check(server.GetPort(), Equals, config.Global.Port)
	c.Check(server.GetSsl(), Equals, config.Global.Ssl)
	c.Check(server.GetVerifyCert(), Equals, config.Global.VerifyCert)
	c.Check(server.GetNoState(), Equals, config.Global.NoState)
	c.Check(server.GetNoReconnect(), Equals, config.Global.NoReconnect)
	c.Check(server.GetReconnectTimeout(), Equals,
		config.Global.ReconnectTimeout)
	c.Check(server.GetNick(), Equals, config.Global.Nick)
	c.Check(server.GetAltnick(), Equals, config.Global.Altnick)
	c.Check(server.GetUsername(), Equals, config.Global.Username)
	c.Check(server.GetUserhost(), Equals, config.Global.Userhost)
	c.Check(server.GetRealname(), Equals, config.Global.Realname)
	c.Check(server.GetPrefix(), Equals, config.Global.Prefix)
	c.Check(len(server.GetChannels()), Equals, len(config.Global.Channels))
	for i, v := range server.GetChannels() {
		c.Check(v, Equals, config.Global.Channels[i])
	}

	//Check bools more throughly
	server.IsSslSet = true
	server.IsVerifyCertSet = true
	server.IsNoStateSet = true
	server.IsNoReconnectSet = true
	c.Check(server.GetSsl(), Equals, false)
	c.Check(server.GetVerifyCert(), Equals, false)
	c.Check(server.GetNoState(), Equals, false)
	c.Check(server.GetNoReconnect(), Equals, false)

	server.IsSslSet = false
	server.IsVerifyCertSet = false
	server.IsNoStateSet = false
	server.IsNoReconnectSet = false
	config.Global.Ssl = false
	config.Global.VerifyCert = false
	config.Global.NoState = false
	config.Global.NoReconnect = false
	c.Check(server.GetSsl(), Equals, false)
	c.Check(server.GetVerifyCert(), Equals, false)
	c.Check(server.GetNoState(), Equals, false)
	c.Check(server.GetNoReconnect(), Equals, false)

	//Check default values more thoroughly
	config.Global.Port = 0
	c.Check(server.GetPort(), Equals, uint16(defaultIrcPort))
	config.Global.Prefix = ""
	c.Check(server.GetPrefix(), Equals, ".")
	config.Global.ReconnectTimeout = 0
	c.Check(server.GetReconnectTimeout(), Equals,
		uint(defaultReconnectTimeout))
}

func (s *s) TestConfig_Fluent(c *C) {
	srv2host := "znc.gamesurge.net"

	conf := CreateConfig().
		Host(""). // Should not break anything
		Port(srv2.Port).
		Ssl(srv2.Ssl).
		VerifyCert(srv2.VerifyCert).
		ReconnectTimeout(srv2.ReconnectTimeout).
		Nick(srv2.Nick).
		Altnick(srv2.Altnick).
		Username(srv2.Username).
		Userhost(srv2.Userhost).
		Realname(srv2.Realname).
		Prefix(srv2.Prefix).
		Channels(srv2.Channels...).
		Server(srv1.Name).
		Host(srv1.Host).
		Port(srv1.Port).
		Ssl(srv1.Ssl).
		VerifyCert(srv1.VerifyCert).
		NoState(srv1.NoState).
		NoReconnect(srv1.NoReconnect).
		ReconnectTimeout(srv1.ReconnectTimeout).
		Nick(srv1.Nick).
		Altnick(srv1.Altnick).
		Username(srv1.Username).
		Userhost(srv1.Userhost).
		Realname(srv1.Realname).
		Prefix(srv1.Prefix).
		Channels(srv1.Channels...).
		Server(srv2host)

	server := conf.GetServer(srv1.Name)
	server2 := conf.GetServer(srv2host)
	c.Check(server.GetHost(), Equals, srv1.Host)
	c.Check(server.GetName(), Equals, srv1.Name)
	c.Check(server.GetPort(), Equals, srv1.Port)
	c.Check(server.GetSsl(), Equals, srv1.Ssl)
	c.Check(server.GetVerifyCert(), Equals, srv1.VerifyCert)
	c.Check(server.GetNoState(), Equals, srv1.NoState)
	c.Check(server.GetNoReconnect(), Equals, srv1.NoReconnect)
	c.Check(server.GetReconnectTimeout(), Equals, srv1.ReconnectTimeout)
	c.Check(server.GetNick(), Equals, srv1.Nick)
	c.Check(server.GetAltnick(), Equals, srv1.Altnick)
	c.Check(server.GetUsername(), Equals, srv1.Username)
	c.Check(server.GetUserhost(), Equals, srv1.Userhost)
	c.Check(server.GetRealname(), Equals, srv1.Realname)
	c.Check(server.GetPrefix(), Equals, srv1.Prefix)
	c.Check(len(server.GetChannels()), Equals, len(srv1.Channels))
	for i, v := range server.GetChannels() {
		c.Check(v, Equals, srv1.Channels[i])
	}

	c.Check(server2.GetHost(), Equals, srv2host)
	c.Check(server2.GetPort(), Equals, srv2.Port)
	c.Check(server2.GetSsl(), Equals, srv2.Ssl)
	c.Check(server2.GetVerifyCert(), Equals, srv2.VerifyCert)
	c.Check(server2.GetNoState(), Equals, srv2.NoState)
	c.Check(server2.GetNoReconnect(), Equals, srv2.NoReconnect)
	c.Check(server2.GetReconnectTimeout(), Equals, srv2.ReconnectTimeout)
	c.Check(server2.GetNick(), Equals, srv2.Nick)
	c.Check(server2.GetAltnick(), Equals, srv2.Altnick)
	c.Check(server2.GetUsername(), Equals, srv2.Username)
	c.Check(server2.GetUserhost(), Equals, srv2.Userhost)
	c.Check(server2.GetRealname(), Equals, srv2.Realname)
	c.Check(server2.GetPrefix(), Equals, srv2.Prefix)
	c.Check(len(server2.GetChannels()), Equals, len(srv2.Channels))
	for i, v := range server2.GetChannels() {
		c.Check(v, Equals, srv2.Channels[i])
	}
}

func (s *s) TestConfig_Validation(c *C) {
	conf := CreateConfig()
	c.Check(conf.IsValid(), Equals, false)
	c.Check(len(conf.Errors), Not(Equals), 0)

	conf = CreateConfig().
		Server("").
		Port(srv1.Port)
	c.Check(len(conf.Servers), Equals, 0)
	c.Check(conf.Global.Port, Equals, uint16(srv1.Port))
	c.Check(conf.IsValid(), Equals, false)
	c.Check(len(conf.Errors), Equals, 2)

	conf = CreateConfig().
		Nick(srv1.Nick).
		Realname(srv1.Realname).
		Username(srv1.Username).
		Userhost(srv1.Userhost).
		Server("a.com").
		Server("a.com")
	c.Check(len(conf.Servers), Equals, 1)
	c.Check(conf.IsValid(), Equals, false)
	c.Check(len(conf.Errors), Equals, 1)

	conf = CreateConfig().
		Nick(srv1.Nick).
		Realname(srv1.Realname).
		Username(srv1.Username).
		Userhost(srv1.Userhost).
		Server("%")
	c.Check(conf.IsValid(), Equals, false)
	// Invalid: Host
	c.Check(len(conf.Errors), Equals, 1)

	conf = CreateConfig().
		Server(srv1.Host)
	c.Check(conf.IsValid(), Equals, false)
	// Missing: Nick, Realname, Username, Userhost
	c.Check(len(conf.Errors), Equals, 4)

	conf = CreateConfig().
		Server(srv1.Host).
		Nick(`@Nick`).              // no special chars
		Channels(`chan`).           // must start with valid prefix
		Username(`spaces in here`). // no spaces
		Userhost(`~#@#$@!`).        // must be a host
		Realname(`@ !`)             // no special chars
	c.Check(conf.IsValid(), Equals, false)
	c.Check(len(conf.Errors), Equals, 5)

	conf = CreateConfig().
		Server(srv1.Host).
		Nick(srv1.Nick).
		Channels(srv1.Channels...).
		Username(srv1.Username).
		Userhost(srv1.Userhost).
		Realname(srv1.Realname)
	conf.Servers[srv1.Host].Host = ""
	c.Check(conf.IsValid(), Equals, false)
	c.Check(len(conf.Errors), Equals, 1) // No host
	conf.Errors = nil
	conf.Servers[srv1.Host].Host = "@@@"
	c.Check(conf.IsValid(), Equals, false)
	c.Check(len(conf.Errors), Equals, 1) // Bad host
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
