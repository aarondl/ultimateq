package config

import (
	"bytes"
	"io"
	"os"

	"github.com/BurntSushi/toml"
)

const (
	// defaultConfigFileName specifies a config file name in the event that
	// none was given, but a write to the file is requested with no name given.
	defaultConfigFileName = "config.toml"
	// errMsgInvalidConfigFile is when the toml does not successfully parse
	errMsgInvalidConfigFile = "config: Failed to load config file (%v)"
	// errMsgFileError occurs if the file could not be opened.
	errMsgFileError = "config: Failed to open config file (%v)"
)

type (
	wrFileCallback func(string) (io.WriteCloser, error)
	roFileCallback func(string) (io.ReadCloser, error)
)

// FromFile overwrites the current config with the contents of the file.
// It will use defaultConfigFileName if filename is the empty string.
func (c *Config) FromFile(filename string) *Config {
	provider := func(name string) (io.ReadCloser, error) {
		return os.Open(name)
	}

	c.fromFile(filename, provider)
	return c
}

// fromFile reads the file provided by the callback and turns it
// into a config. The file provided is closed by this function. It overrides
// filename with defaultConfigFileName if it's the empty string.
func (c *Config) fromFile(filename string, fn roFileCallback) *Config {
	if filename == "" {
		filename = defaultConfigFileName
	}

	file, err := fn(filename)
	if err != nil {
		c.addError(errMsgFileError, err)
	} else {
		defer file.Close()
		c.FromReader(file)

		c.protect.Lock()
		defer c.protect.Unlock()
		c.filename = filename
	}

	return c
}

// FromString overwrites the current config with the contents of the string.
func (c *Config) FromString(config string) *Config {
	buf := bytes.NewBufferString(config)
	c.FromReader(buf)
	return c
}

// FromReader overwrites the current config with the contents of the reader.
func (c *Config) FromReader(reader io.Reader) *Config {
	c.protect.Lock()
	defer c.protect.Unlock()
	c.clear()

	_, err := toml.DecodeReader(reader, c.values)
	if err != nil {
		c.addError(errMsgInvalidConfigFile, err)
	}

	return c
}

// ToFile writes a config out to a writer. If the filename is empty
// it will write to the file that this config was loaded from, or it will
// write to the defaultConfigFileName.
func (c *Config) ToFile(filename string) error {
	provider := func(f string) (io.WriteCloser, error) {
		return os.Create(filename)
	}

	return c.toFile(filename, provider)
}

// toFile uses a callback to get a ReadWriter to write to. It also
// manages resolving the filename properly and writing the config to the Writer.
// The file provided by the callback is closed in this function.
func (c *Config) toFile(filename string, getFile wrFileCallback) error {
	if filename == "" {
		filename = c.Filename()
	}

	writer, err := getFile(filename)
	if err != nil {
		return err
	}
	defer writer.Close()

	return c.ToWriter(writer)
}

// ToWriter writes a config out to a writer.
func (c *Config) ToWriter(writer io.Writer) error {
	c.protect.RLock()
	defer c.protect.RUnlock()

	encoder := toml.NewEncoder(writer)
	err := encoder.Encode(c.values)
	if err != nil {
		return err
	}

	return err
}
