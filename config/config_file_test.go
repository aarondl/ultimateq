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
nocorecmds = false

nick = "Nick"
altnick = "Altnick"
username = "Username"
realname = "Realname"
password = "Password"

[networks.noirc]
	servers = ["lol:3"]
[networks.ircnet]
	servers = ["localhost:3333", "server.com:6667"]

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
	prefix = "."

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
	server = "localhost:44"
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
	var exps string
	var expb bool
	var expu uint
	var expf float64

	exps = "/path/to/store/file.db"
	if got, ok := conf.StoreFile(); !ok || exps != got {
		t.Errorf("Expected: %s, got: %s", exps, got)
	}

	expb = false
	if got, ok := conf.NoCoreCmds(); !ok || expb != got {
		t.Errorf("Expected: %v, got: %v", expb, got)
	}

	net1 := conf.Network("ircnet")
	if net1 == nil {
		t.Error("Expected ircnet to be configured.")
	}

	exps = "Nick"
	if got, ok := net1.Nick(); !ok || exps != got {
		t.Errorf("Expected: %s, got: %s", exps, got)
	}

	exps = "Altnick"
	if got, ok := net1.Altnick(); !ok || exps != got {
		t.Errorf("Expected: %s, got: %s", exps, got)
	}

	exps = "Username"
	if got, ok := net1.Username(); !ok || exps != got {
		t.Errorf("Expected: %s, got: %s", exps, got)
	}

	exps = "Realname"
	if got, ok := net1.Realname(); !ok || exps != got {
		t.Errorf("Expected: %s, got: %s", exps, got)
	}

	exps = "Password"
	if got, ok := net1.Password(); !ok || exps != got {
		t.Errorf("Expected: %s, got: %s", exps, got)
	}

	expb = true
	if got, ok := net1.SSL(); !ok || expb != got {
		t.Errorf("Expected: %v, got: %v", expb, got)
	}

	exps = "/path/to/a.crt"
	if got, ok := net1.SSLCert(); !ok || exps != got {
		t.Errorf("Expected: %s, got: %s", exps, got)
	}

	expb = false
	if got, ok := net1.NoVerifyCert(); !ok || expb != got {
		t.Errorf("Expected: %v, got: %v", expb, got)
	}

	expb = false
	if got, ok := net1.NoState(); !ok || expb != got {
		t.Errorf("Expected: %v, got: %v", expb, got)
	}

	expb = false
	if got, ok := net1.NoStore(); !ok || expb != got {
		t.Errorf("Expected: %v, got: %v", expb, got)
	}

	expu = 120
	if got, ok := net1.FloodLenPenalty(); !ok || expu != got {
		t.Errorf("Expected: %v, got: %v", expu, got)
	}

	expf = 10.0
	if got, ok := net1.FloodTimeout(); !ok || expf != got {
		t.Errorf("Expected: %v, got: %v", expf, got)
	}

	expf = 2.0
	if got, ok := net1.FloodStep(); !ok || expf != got {
		t.Errorf("Expected: %v, got: %v", expf, got)
	}

	expf = 60.0
	if got, ok := net1.KeepAlive(); !ok || expf != got {
		t.Errorf("Expected: %v, got: %v", expf, got)
	}

	expb = false
	if got, ok := net1.NoReconnect(); !ok || expb != got {
		t.Errorf("Expected: %v, got: %v", expb, got)
	}

	expu = 20
	if got, ok := net1.ReconnectTimeout(); !ok || expu != got {
		t.Errorf("Expected: %v, got: %v", expu, got)
	}

	exps = "."
	if got, ok := net1.Prefix(); !ok || exps != got {
		t.Errorf("Expected: %v, got: %v", exps, got)
	}

	if chans, ok := net1.Channels(); ok {
		c1, c2 := chans[0], chans[1]

		if exp, got := "#channel1", c1.Name; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
		if exp, got := "pass1", c1.Password; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
		if exp, got := "!", c1.Prefix; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}

		if exp, got := "#channel2", c2.Name; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
		if exp, got := "pass2", c2.Password; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
		if exp, got := "@", c2.Prefix; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
	} else {
		t.Error("Expected to get some channels.")
	}

	if servers, ok := net1.Servers(); ok {
		if servers[0] != "localhost:3333" {
			t.Error("The first server was wrong:", servers[0])
		}
		if servers[1] != "server.com:6667" {
			t.Error("The first server was wrong:", servers[1])
		}
	} else {
		t.Error("Expected to get some servers.")
	}

	net2 := conf.Network("noirc")
	if net2 == nil {
		t.Error("Expected noirc to be configured.")
	}

	if servers, ok := net2.Servers(); ok {
		if servers[0] != "lol:3" {
			t.Error("The first server was wrong:", servers[0])
		}
	} else {
		t.Error("Expected to get some servers.")
	}

	globalExt := conf.ExtGlobal()

	exps = "/path/to/executables"
	if got, ok := globalExt.ExecDir(); !ok || exps != got {
		t.Errorf("Expected: %v, got: %v", exps, got)
	}

	exps = "localhost:3333"
	if got, ok := globalExt.Listen(); !ok || exps != got {
		t.Errorf("Expected: %v, got: %v", exps, got)
	}

	expb = false
	if got, ok := globalExt.NoReconnect(); !ok || expb != got {
		t.Errorf("Expected: %v, got: %v", expb, got)
	}

	expu = 20
	if got, ok := globalExt.ReconnectTimeout(); !ok || expu != got {
		t.Errorf("Expected: %v, got: %v", expu, got)
	}

	expb = true
	if got, ok := globalExt.UseJson(); !ok || expb != got {
		t.Errorf("Expected: %v, got: %v", expb, got)
	}

	ext := conf.Ext("myext")

	exps = "/path/to/executable"
	if got, ok := ext.Exec(); !ok || exps != got {
		t.Errorf("Expected: %v, got: %v", exps, got)
	}

	exps = "localhost:44"
	if got, ok := ext.Server(); !ok || exps != got {
		t.Errorf("Expected: %v, got: %v", exps, got)
	}

	expb = false
	if got, ok := ext.NoReconnect(); !ok || expb != got {
		t.Errorf("Expected: %v, got: %v", expb, got)
	}

	expu = 20
	if got, ok := ext.ReconnectTimeout(); !ok || expu != got {
		t.Errorf("Expected: %v, got: %v", expu, got)
	}

	expb = true
	if got, ok := ext.SSL(); !ok || expb != got {
		t.Errorf("Expected: %v, got: %v", expb, got)
	}

	exps = "/path/to/a.crt"
	if got, ok := ext.SSLCert(); !ok || exps != got {
		t.Errorf("Expected: %v, got: %v", exps, got)
	}

	expb = false
	if got, ok := ext.NoVerifyCert(); !ok || expb != got {
		t.Errorf("Expected: %v, got: %v", expb, got)
	}

	exps = "/path/to/sock.sock"
	if got, ok := ext.Unix(); !ok || exps != got {
		t.Errorf("Expected: %v, got: %v", exps, got)
	}

	expb = false
	if got, ok := ext.UseJson(); !ok || expb != got {
		t.Errorf("Expected: %v, got: %v", expb, got)
	}

	if active, ok := ext.Active("ircnet"); !ok || active == nil {
		t.Error("Expected some active channels.")
	} else {
		if active[0] != "#channel1" {
			t.Error("Expected #channel1 to be the first active channel.")
		}
		if active[1] != "#channel2" {
			t.Error("Expected #channel2 to be the first active channel.")
		}
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
	t.SkipNow()
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
