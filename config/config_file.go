package config

import (
	"io"
	"io/ioutil"
	"launchpad.net/goyaml"
	"os"
)

const (
	// defaultConfigFileName specifies a config file name in the event that
	// none was given, but a write to the file is requested with no name given.
	defaultConfigFileName = "config.yaml"
	// errMsgInvalidConfigFile is when the yaml does not successfully parse
	errMsgInvalidConfigFile = "config: Failed to load config file (%v)"
	// errMsgFileError occurs if the file could not be opened.
	errMsgFileError = "config: Failed to open config file (%v)"
)

type (
	wrFileCallback func(string) (io.WriteCloser, error)
	roFileCallback func(string) (io.ReadCloser, error)
)

// CreateConfigFromFile initializes a Config object from a file.
func CreateConfigFromFile(filename string) *Config {
	provider := func(name string) (io.ReadCloser, error) {
		return os.Open(name)
	}
	return createConfigFromFile(filename, provider)
}

// createConfigFromFile reads the file provided by the callback and turns it
// into a config. The file provided is closed by this function.
func createConfigFromFile(filename string, fn roFileCallback) (conf *Config) {
	file, err := fn(filename)
	if err != nil {
		conf = CreateConfig()
		conf.addError(errMsgInvalidConfigFile, err)
	} else {
		conf = CreateConfigFromReader(file)
		conf.filename = filename
		file.Close()
	}
	return
}

// CreateConfigFromReader initializes a Config object from a reader.
func CreateConfigFromReader(reader io.Reader) *Config {
	c := &Config{
		Errors: make([]error, 0),
	}
	buf, err := ioutil.ReadAll(reader)
	if err != nil {
		c.addError(errMsgInvalidConfigFile, err)
		return c
	}
	err = goyaml.Unmarshal(buf, c)
	if err != nil {
		c.addError(errMsgInvalidConfigFile, err)
	}

	c.fixReferencesAndNames()

	return c
}

// fixReferencesAndNames is called before a config-file-deserialized config
// is returned to patch up backreferences to the main config as well as check
// that the name/host are set properly.
func (c *Config) fixReferencesAndNames() {
	if c.Global == nil {
		c.Global = &Server{}
	}
	for s, v := range c.Servers {
		v.parent = c
		v.Name = s
		if len(v.Host) == 0 {
			v.Host = s
		}
	}
}

// FlushConfigToFile writes a config out to a writer. If the filename is empty
// it will write to the file that this config was loaded from, or it will
// write to the defaultConfigFileName.
func FlushConfigToFile(conf *Config, filename string) (err error) {
	provider := func(f string) (io.WriteCloser, error) {
		return os.Create(filename)
	}

	err = flushConfigToFile(conf, filename, provider)
	return
}

// flushConfigToFile uses a callback to get a ReadWriter to write to. It also
// manages resolving the filename properly and writing the config to the Writer.
// The file provided by the callback is closed in this function.
func flushConfigToFile(conf *Config, filename string,
	getFile wrFileCallback) (err error) {

	if filename == "" {
		if conf.filename != "" {
			filename = conf.filename
		} else {
			filename = defaultConfigFileName
		}
	}

	var writer io.WriteCloser
	writer, err = getFile(filename)
	if err != nil {
		return
	}
	defer writer.Close()

	err = FlushConfigToWriter(conf, writer)
	return
}

// FlushConfigToWriter writes a config out to a writer
func FlushConfigToWriter(conf *Config, writer io.Writer) (err error) {
	marshalled, err := goyaml.Marshal(conf)
	if err != nil {
		return
	}
	var n, written = 0, 0
	for err == nil && written < len(marshalled) {
		n, err = writer.Write(marshalled[written:])
		written += n
	}
	return
}
