package config

import (
	"bytes"
	"io"
	"strings"
	"testing"
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

func (d *dyingReader) Read(b []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

type dyingWriter struct {
}

func (d *dyingWriter) Write(b []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

/*var configuration = `global:
    port: 5555
    nick: nick
    username: username
    realname: realname
    exts:
        awesome:
            config:
                friend: bob
            exec: /some/path/goes/here
            isserver: true
networks:
    myserver:
        servers:
        - irc.gamesurge.net
        nick: nickoverride
    irc.gamesurge.net:
        port: 3333
`*/

var configuration = `
nick = "nick"
username = "username"
realname = "realname"
port = 5555

[exts.awesome]
exec = "/some/path/goes/here"
isserver = "true"

[exts.awesome.config]
friend = "bob"

[networks.myserver]
servers = ["irc.gamesurge.net"]
nick = "nickoverride"

[networks.gamesurge]
servers = ["irc.gamesurge.com"]
port = 3333
`

func verifyFakeConfig(t *testing.T, conf *Config) {
	net1 := conf.Networks["myserver"]

	if exp, got := "nickoverride", net1.Nick(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := uint16(5555), net1.Port(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := "username", net1.Username(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := "realname", net1.Realname(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	if exp, got := "myserver", net1.Name(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := "irc.gamesurge.net", net1.Servers()[0]; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	net2 := conf.Networks["gamesurge"]

	if exp, got := "nick", net2.Nick(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := "irc.gamesurge.com", net2.Servers()[0]; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	ext := conf.InExts["awesome"]
	if ext == nil {
		t.Error("There should be an extension called awesome.")
	}

	if ext.InConfig["friend"] != "bob" {
		t.Error("It should load the configuration.")
	}

	if ext.InExec != "/some/path/goes/here" {
		t.Error("It should allow setting exec paths.")
	}

	if ext.InIsServer != "true" {
		t.Error("It should allow setting boolean strings.")
	}
}

func TestConfig_FromReader(t *testing.T) {
	t.Parallel()
	c := NewConfig().FromString(configuration)

	if !c.IsValid() {
		t.Error(c.errors)
		t.Fatal("It should be a valid configuration.")
	}

	verifyFakeConfig(t, c)
}

func TestConfig_FromReaderErrors(t *testing.T) {
	t.Parallel()
	c := NewConfig().FromReader(&dyingReader{})

	ers := c.Errors()
	if exp, got := 1, len(ers); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	err := ers[0].Error()
	errMsg := errMsgInvalidConfigFile[:len(errMsgInvalidConfigFile)-4]
	if !strings.Contains(err, errMsg) {
		t.Errorf(`"Expected: "%v" to contain: "%v"`, err, errMsg)
	}

	buf := bytes.NewBufferString("defaults:\n\tport: 5555")
	c = NewConfig().FromReader(buf)

	ers = c.Errors()
	if exp, got := 1, len(ers); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	err = ers[0].Error()
	if !strings.Contains(err, errMsg) {
		t.Errorf(`"Expected: "%v" to contain: "%v"`, err, errMsg)
	}
}

func TestConfig_fromFile(t *testing.T) {
	t.Parallel()
	buf := &testBuffer{bytes.NewBufferString(configuration), false}

	name := "check.yaml"
	c := NewConfig().fromFile(name, func(f string) (io.ReadCloser, error) {
		return buf, nil
	})

	if exp, got := name, c.filename; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if !buf.closed {
		t.Error("It should close the file.")
	}

	verifyFakeConfig(t, c)

	name = ""
	buf = &testBuffer{bytes.NewBufferString(configuration), false}
	c = NewConfig().fromFile(name, func(f string) (io.ReadCloser, error) {
		return buf, nil
	})

	if c.filename != defaultConfigFileName {
		t.Error("Expected it to use the default file name, got:", c.filename)
	}
}

func TestConfig_fromFileErrors(t *testing.T) {
	t.Parallel()
	errMsg := errMsgFileError[:len(errMsgFileError)-4]

	c := NewConfig().fromFile("", func(_ string) (io.ReadCloser, error) {
		return nil, io.EOF
	})
	ers := c.Errors()
	if exp, got := 1, len(ers); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	err := ers[0].Error()
	if !strings.Contains(err, errMsg) {
		t.Errorf(`"Expected: "%v" to contain: "%v"`, err, errMsg)
	}
}

func TestConfig_ToWriter(t *testing.T) {
	t.Parallel()
	c := NewConfig().FromString(configuration)

	buf := &bytes.Buffer{}
	if err := c.ToWriter(buf); err != nil {
		t.Error("Unexpected error:", err)
	}

	c = NewConfig().FromReader(buf)

	verifyFakeConfig(t, c)
}

func TestConfig_ToWriterErrors(t *testing.T) {
	t.Parallel()

	err := NewConfig().ToWriter(&dyingWriter{})
	if err == nil || err == io.EOF {
		t.Error("Expected to see an unconventional error.")
	}
}

func TestConfig_toFile(t *testing.T) {
	t.Parallel()

	c := NewConfig()
	buf := &testBuffer{&bytes.Buffer{}, false}

	filename := ""
	c.toFile("a.txt", func(fn string) (io.WriteCloser, error) {
		filename = fn
		return buf, nil
	})
	if filename != "a.txt" {
		t.Error("Expected it to set the filename to what we asked for.")
	}

	filename = ""
	c.toFile("", func(fn string) (io.WriteCloser, error) {
		filename = fn
		return buf, nil
	})
	if filename != defaultConfigFileName {
		t.Error("Expected it to set the filename to the default.")
	}

	filename = ""
	c.filename = "b.txt"
	c.toFile("", func(fn string) (io.WriteCloser, error) {
		filename = fn
		return buf, nil
	})
	if filename != "b.txt" {
		t.Error("Expected it to set the filename to the config's filename.")
	}
}

func TestConfig_toFileErrors(t *testing.T) {
	t.Parallel()
	err := NewConfig().toFile("", func(_ string) (io.WriteCloser, error) {
		return nil, io.EOF
	})

	if err == nil {
		t.Error("Expected an error.")
	}
}

func TestConfig_fixReferenceAndNames(t *testing.T) {
	t.Parallel()

	c := Config{Networks: make(map[string]*Network)}
	c.Networks["test"] = nil
	c.fixReferencesAndNames()

	if c.Network == nil {
		t.Error("It should set the network to empty not nil.")
	}

	if c.Network.InName != "global" {
		t.Error("It should set the name.")
	}

	if c.Network.protect == nil {
		t.Error("It should hook up the mutex.")
	}

	net := c.Networks["test"]
	if net == nil {
		t.Error("It should instantiate empty networks.")
	}

	if net.protect == nil {
		t.Error("It should hook up the mutex.")
	}

	if net.InName != "test" {
		t.Error("It should set the name.")
	}
}

func TestConfig_fixReferenceAndNamesExts(t *testing.T) {
	t.Parallel()

	c := NewConfig()
	c.Network.InExts = map[string]*Ext{
		"ext": {},
	}
	c.Networks["net"] = &Network{
		InExts: map[string]*Ext{
			"ext": nil,
		},
	}
	c.fixReferencesAndNames()

	ext := c.Network.InExts["ext"]

	if ext == nil {
		t.Error("Expected it to instantiate empty extensions.")
	}

	if ext.InName != "ext" {
		t.Error("It should set the name.")
	}

	if ext.protect == nil {
		t.Error("It should hook up the mutex.")
	}

	n := c.Networks["net"]
	ext = n.InExts["ext"]

	if ext == nil {
		t.Error("Expected it to instantiate empty extensions.")
	}

	if ext.InName != "ext" {
		t.Error("It should set the name.")
	}

	if ext.protect == nil {
		t.Error("It should hook up the mutex.")
	}
}
