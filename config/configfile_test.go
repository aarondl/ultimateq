package config

import (
	"bytes"
	"errors"
	"io"
	. "launchpad.net/gocheck"
)

type testBuffer struct {
	io.ReadWriter
	closed bool
}

func (t *testBuffer) Close() error {
	t.closed = true
	return nil
}

type dyingReader struct {
}

func (d *dyingReader) Read(b []byte) (n int, err error) {
	err = io.ErrUnexpectedEOF
	return
}

var configuration = `global:
    port: 5555
    nick: nick
    username: username
    userhost: userhost.com
    realname: realname
servers:
    myserver:
        host: irc.gamesurge.net
        nick: nickoverride
    irc.gamesurge.net:
        port: 3333
`

func verifyFakeConfig(c *C, conf *Config) {
	srv1 := conf.Servers["myserver"]
	c.Assert(srv1.GetNick(), Equals, "nickoverride")
	c.Assert(srv1.GetPort(), Equals, uint16(5555))
	c.Assert(srv1.GetUsername(), Equals, "username")
	c.Assert(srv1.GetUserhost(), Equals, "userhost.com")
	c.Assert(srv1.GetRealname(), Equals, "realname")

	c.Assert(srv1.GetName(), Equals, "myserver")
	c.Assert(srv1.GetHost(), Equals, "irc.gamesurge.net")

	srv2 := conf.Servers["irc.gamesurge.net"]
	c.Assert(srv2.GetNick(), Equals, "nick")
	c.Assert(srv2.GetHost(), Equals, "irc.gamesurge.net")
	c.Assert(srv2.GetName(), Equals, srv2.GetHost())
}

func (s *s) TestConfig_FromReader(c *C) {
	buf := bytes.NewBufferString(configuration)
	conf := CreateConfigFromReader(buf)
	c.Assert(len(conf.Errors), Equals, 0)

	verifyFakeConfig(c, conf)

	c.Assert(conf.IsValid(), Equals, true)
}

func (s *s) TestConfig_FromReaderErrors(c *C) {
	conf := CreateConfigFromReader(&dyingReader{})
	c.Assert(len(conf.Errors), Equals, 1)
	c.Assert(conf.Errors[0].Error(), Matches,
		errMsgInvalidConfigFile[:len(errMsgInvalidConfigFile)-4]+`.*`)

	buf := bytes.NewBufferString("defaults:\n\tport: 5555")
	conf = CreateConfigFromReader(buf)
	c.Assert(len(conf.Errors), Equals, 1)
	c.Assert(conf.Errors[0].Error(), Matches,
		errMsgInvalidConfigFile[:len(errMsgInvalidConfigFile)-4]+`.*`)
}

func (s *s) TestConfig_ToWriter(c *C) {
	outbuf := bytes.NewBufferString(configuration)
	conf := CreateConfigFromReader(outbuf)
	c.Assert(len(conf.Errors), Equals, 0)

	inbuf := &bytes.Buffer{}
	FlushConfigToWriter(conf, inbuf)

	conf = CreateConfigFromReader(inbuf)
	verifyFakeConfig(c, conf)
}

func (s *s) TestConfig_FromFile(c *C) {
	buf := &testBuffer{bytes.NewBufferString(configuration), false}
	name := "check.yaml"
	conf := createConfigFromFile(name, func(f string) (io.ReadCloser, error) {
		return buf, nil
	})
	c.Assert(len(conf.Errors), Equals, 0)
	c.Assert(conf.filename, Equals, name)
	c.Assert(buf.closed, Equals, true)

	verifyFakeConfig(c, conf)
	c.Assert(conf.IsValid(), Equals, true)

	conf = createConfigFromFile(name, func(f string) (io.ReadCloser, error) {
		return nil, errors.New("")
	})
	c.Assert(len(conf.Errors), Equals, 1)
}

func (s *s) TestConfig_ToFile(c *C) {
	outbuf := bytes.NewBufferString(configuration)
	conf := CreateConfigFromReader(outbuf)
	c.Assert(len(conf.Errors), Equals, 0)

	inbuf := &testBuffer{&bytes.Buffer{}, false}
	c.Assert(inbuf.closed, Equals, false)
	err := flushConfigToFile(conf, "", func(f string) (io.WriteCloser, error) {
		c.Assert(f, Equals, defaultConfigFileName)
		return inbuf, nil
	})
	c.Assert(err, IsNil)
	c.Assert(inbuf.closed, Equals, true)

	conf.filename = "check.yaml"
	inbuf.closed = false
	err = flushConfigToFile(conf, "", func(f string) (io.WriteCloser, error) {
		c.Assert(f, Equals, conf.filename)
		return inbuf, nil
	})
	c.Assert(err, IsNil)
	c.Assert(inbuf.closed, Equals, true)

	name := "other.yaml"
	inbuf.closed = false
	flushConfigToFile(conf, name, func(f string) (io.WriteCloser, error) {
		c.Assert(f, Equals, name)
		return inbuf, nil
	})
	c.Assert(err, IsNil)
	c.Assert(inbuf.closed, Equals, true)

	inbuf.closed = false
	err = flushConfigToFile(conf, "", func(_ string) (io.WriteCloser, error) {
		return nil, errors.New("")
	})
	c.Assert(err, NotNil)
	c.Assert(inbuf.closed, Equals, false)

	conf = CreateConfigFromReader(inbuf)
	verifyFakeConfig(c, conf)
}
