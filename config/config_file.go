package config

import (
	"io"
	"io/ioutil"
	"launchpad.net/goyaml"
	"os"
)

const (
	errMsgInvalidConfigFile = "config: Failed to load config file (%v)"
)

// CreateConfigFromFile initializes a Config object from a file.
func CreateConfigFromFile(filename string) *Config {
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		c := CreateConfig()
		c.addError(errMsgInvalidConfigFile, err)
		return c
	} else {
		return CreateConfigFromReader(file)
	}
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
	for s, v := range c.Servers {
		v.parent = c
		v.Name = s
		if len(v.Host) == 0 {
			v.Host = s
		}
	}
}
