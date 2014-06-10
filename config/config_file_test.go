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

const configuration = `
storefile = "/path/to/store/file.db"
corecmds = false

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
	defaultprefix = "."

	[[networks.ircnet.channels]]
	name = "#channel1"
	password = "password"
	prefix = "!"

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
	server = ["localhost:44", "server.com:4444"]
	ssl = true
	sslcert = "/path/to/a.crt"
	noverifycert = false

	# Define the above connection properties, or simply this one property.
	unix = "/path/to/sock.sock"

	# Use json not gob.
	usejson = false

	[exts.myext.active]
		ircnet = ["#channel1", "#channel2"]
`

func verifyFakeConfig(t *testing.T, conf *Config) {
	/*
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
	*/
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

	err := NewConfig().FromString(configuration).ToWriter(&dyingWriter{})
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
