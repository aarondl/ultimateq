package config

import "fmt"

// validatorRules is used internally to validate a map.
type validatorRules struct {
	stringVals      []string
	stringSliceVals []string
	boolVals        []string
	floatVals       []string
	intVals         []string
	uintVals        []string
	mapVals         []string
	mapArrVals      []string
}

var globalValidator = validatorRules{
	stringVals: []string{"storefile"},
	mapVals:    []string{"ext", "exts", "networks"},
	boolVals:   []string{"nocorecmds"},
}

var networkValidator = validatorRules{
	stringVals: []string{
		"nick", "altnick", "username", "realname", "password",
		"sslcert", "prefix",
	},
	stringSliceVals: []string{"servers"},
	boolVals: []string{
		"ssl", "nostate", "nostore", "noreconnect", "noverifycert",
	},
	floatVals:  []string{"floodtimeout", "floodstep", "keepalive"},
	uintVals:   []string{"reconnecttimeout", "floodlenpenalty"},
	mapArrVals: []string{"channels"},
}

var extCommonValidator = validatorRules{
	boolVals: []string{
		"usejson", "noreconnect",
	},
	uintVals: []string{"reconnecttimeout"},
	mapVals:  []string{"active"},
}

var extGlobalValidator = validatorRules{
	stringVals: []string{"execdir", "listen"},
	mapVals:    []string{"config"},
}

var extNormalValidator = validatorRules{
	stringVals: []string{"server", "exec", "sslcert", "unix"},
	boolVals:   []string{"ssl", "noreconnect"},
}

// errList is an array of errors.
type errList []error

// addError builds an error object and appends it to this instances errors.
func (c *Config) addError(format string, args ...interface{}) {
	c.errors.addError(format, args...)
}

// addError builds an error object and appends it to this instances errors.
func (l *errList) addError(format string, args ...interface{}) {
	*l = append(*l, fmt.Errorf(format, args...))
}

// Errors returns the errors encountered during validation.
func (c *Config) Errors() []error {
	c.protect.RLock()
	c.protect.RUnlock()

	ers := make([]error, len(c.errors))
	copy(ers, c.errors)
	return ers
}

// IsValid checks to see if the configuration is valid. If errors are found in
// the config the Config.Errors() will return the validation errors.
// These can be used to display to the user. See DisplayErrors for a display
// helper.
func (c *Config) IsValid() bool {
	ers := make(errList, 0)

	c.protect.RLock()
	c.validateRequired(&ers)
	c.protect.RUnlock()

	if len(ers) > 0 {
		c.protect.Lock()
		c.errors = ers
		c.protect.Unlock()
		return false
	}

	c.protect.RLock()
	c.validateTypes(&ers)
	c.protect.RUnlock()

	return true
}

// validateRequired checks that all required fields are present.
func (c *Config) validateRequired(ers *errList) {
	var nets map[string]interface{}
	if val, ok := c.values["networks"]; !ok {
		ers.addError("At least one network must be defined.")
	} else if nets, ok = val.(map[string]interface{}); !ok {
		ers.addError("Expected networks to be a map, got %T", val)
	}

	if len(nets) == 0 {
		ers.addError("Expected at least 1 network.")
		return
	}

	for name, netval := range nets {
		if _, ok := netval.(map[string]interface{}); !ok {
			ers.addError("(%s) Expected network to be a map, got %T", name,
				netval)
		} else {
			ctx := c.Network(name)
			if srvs, ok := ctx.Servers(); !ok || len(srvs) == 0 {
				ers.addError("(%s) Need at least one server defined", name)
			}

			if n, ok := ctx.Nick(); !ok || len(n) == 0 {
				ers.addError("(%s) Nickname is required.", name)
			}
			if n, ok := ctx.Altnick(); !ok || len(n) == 0 {
				ers.addError("(%s) Altnick is required.", name)
			}
			if n, ok := ctx.Username(); !ok || len(n) == 0 {
				ers.addError("(%s) Username is required.", name)
			}
			if n, ok := ctx.Realname(); !ok || len(n) == 0 {
				ers.addError("(%s) Realname is required.", name)
			}
		}
	}
}

// validateTypes checks the types of all of the map's objects.
func (c *Config) validateTypes(ers *errList) {
	globalValidator.validateMap("global", c.values, ers)
	networkValidator.validateMap("global", c.values, ers)

	if netsVal, ok := c.values["networks"]; ok {
		if nets, ok := netsVal.(map[string]interface{}); ok {
			for name, netVal := range nets {
				if net, ok := netVal.(map[string]interface{}); ok {
					networkValidator.validateMap(name, net, ers)
				}
			}
		}
	}

	if extsVal, ok := c.values["ext"]; ok {
		if exts, ok := extsVal.(map[string]interface{}); ok {
			for name, extVal := range exts {
				if ext, ok := extVal.(map[string]interface{}); ok {
					extCommonValidator.validateMap(name, ext, ers)
					extGlobalValidator.validateMap(name, ext, ers)
				}
			}
		}
	}

	if extsVal, ok := c.values["exts"]; ok {
		if exts, ok := extsVal.(map[string]interface{}); ok {
			for name, extVal := range exts {
				if ext, ok := extVal.(map[string]interface{}); ok {
					extCommonValidator.validateMap(name, ext, ers)
					extNormalValidator.validateMap(name, ext, ers)
				}
			}
		}
	}
}

// validateMap checks map's values for correct types based on the validatorRules
func (v validatorRules) validateMap(name string,
	m map[string]interface{}, ers *errList) {

	for _, key := range v.stringVals {
		if val, ok := m[key]; !ok {
			continue
		} else if _, ok = val.(string); !ok {
			ers.addError("(%s) %s is type %T but expected string [%v]",
				name, key, val, val)
		}
	}
	for _, key := range v.stringSliceVals {
		if val, ok := m[key]; !ok {
			continue
		} else if _, ok = val.([]string); !ok {
			ers.addError("(%s) %s is type %T but expected []string [%v]",
				name, key, val, val)
		}
	}
	for _, key := range v.boolVals {
		if val, ok := m[key]; !ok {
			continue
		} else if _, ok = val.(bool); !ok {
			ers.addError("(%s) %s is type %T but expected bool [%v]",
				name, key, val, val)
		}
	}
	for _, key := range v.floatVals {
		if val, ok := m[key]; !ok {
			continue
		} else if _, ok = val.(float64); !ok {
			ers.addError("(%s) %s is type %T but expected float64 [%v]",
				name, key, val, val)
		}
	}
	for _, key := range v.intVals {
		if val, ok := m[key]; !ok {
			continue
		} else if _, ok = val.(int64); !ok {
			ers.addError("(%s) %s is type %T but expected int [%v]",
				name, key, val, val)
		}
	}
	for _, key := range v.uintVals {
		if val, ok := m[key]; !ok {
			continue
		} else if _, ok = val.(int64); !ok {
			ers.addError("(%s) %s is type %T but expected int [%v]",
				name, key, val, val)
		}
	}
	for _, key := range v.mapVals {
		if val, ok := m[key]; !ok {
			continue
		} else if _, ok = val.(map[string]interface{}); !ok {
			ers.addError("(%s) %s is type %T but expected map [%v]",
				name, key, val, val)
		}
	}
	for _, key := range v.mapArrVals {
		if val, ok := m[key]; !ok {
			continue
		} else if _, ok = val.([]map[string]interface{}); !ok {
			ers.addError("(%s) %s is type %T but expected maparr [%v]",
				name, key, val, val)
		}
	}
}
