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
	stringVals: []string{"storefile", "loglevel", "logfile", "secret_key"},
	mapVals:    []string{"ext", "exts", "networks"},
	boolVals:   []string{"nocorecmds"},
}

var networkValidator = validatorRules{
	stringVals: []string{
		"nick", "altnick", "username", "realname", "password",
		"tls_cert", "prefix",
	},
	stringSliceVals: []string{"servers"},
	boolVals: []string{
		"nostate", "nostore", "noautojoin",
		"noreconnect", "noverifycert",
	},
	floatVals:  []string{"floodtimeout", "floodstep", "keepalive"},
	uintVals:   []string{"reconnecttimeout", "floodlenpenalty", "joindelay"},
	mapArrVals: []string{"channels"},
}

var channelValidator = validatorRules{
	stringVals: []string{"name", "prefix", "password"},
}

var extCommonValidator = validatorRules{
	boolVals: []string{
		"noreconnect",
	},
	uintVals: []string{"reconnecttimeout"},
	mapVals:  []string{"active"},
}

var extGlobalValidator = validatorRules{
	stringVals: []string{"execdir", "listen", "tls_cert", "tls_key"},
	mapVals:    []string{"config"},
}

var extNormalValidator = validatorRules{
	stringVals: []string{"server", "exec", "tls_cert"},
	boolVals:   []string{"noreconnect", "noverifycert"},
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

// Validate checks to see if the configuration is valid. If errors are found in
// the config the Config.Errors() will return the validation errors.
// These can be used to display to the user. See DisplayErrors for a display
// helper.
func (c *Config) Validate() bool {
	ers := make(errList, 0)

	c.protect.RLock()
	c.validateTypes(&ers)
	c.protect.RUnlock()

	if len(ers) > 0 {
		c.protect.Lock()
		c.errors = ers
		c.protect.Unlock()
		return false
	}

	c.protect.RLock()
	c.validateRequired(&ers)
	c.protect.RUnlock()

	if len(ers) > 0 {
		c.protect.Lock()
		c.errors = ers
		c.protect.Unlock()
		return false
	}

	return true
}

// validateRequired checks that all required fields are present.
func (c *Config) validateRequired(ers *errList) {
	var nets mp
	if nets = c.values.get("networks"); nets == nil {
		ers.addError("Expected at least one network.")
		return
	}

	for name, netval := range nets {
		if ok := intfToMp(netval); ok == nil {
			ers.addError("(%s) Expected network to be a map, got %T", name,
				netval)
		} else {
			ctx := c.Network(name)
			if srvs, ok := ctx.Servers(); !ok || len(srvs) == 0 {
				ers.addError("(%s) Expected at least one server.", name)
			}

			if n, ok := ctx.Nick(); !ok || len(n) == 0 {
				ers.addError("(%s) Nickname is required.", name)
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

	if nets := c.values.get("networks"); nets != nil {
		for name, netVal := range nets {
			if net := intfToMp(netVal); net == nil {
				ers.addError(
					"(global networks) %s is %T but expected map [%v]",
					name, netVal, netVal)
			} else {
				networkValidator.validateMap(name, net, ers)

				if chans := net.getArr("channels"); chans != nil {
					for _, ch := range chans {
						channelValidator.validateMap(name+" channels", ch, ers)
					}
				}
			}
		}
	}

	if ext := c.values.get("ext"); ext != nil {
		extCommonValidator.validateMap("ext", ext, ers)
		extGlobalValidator.validateMap("ext", ext, ers)

		if active := ext.get("active"); active != nil {
			validateActive("ext", active, ers)
		}

		if config := ext.get("config"); config != nil {
			validateExtConfig(config, ers)
		}
	}

	if exts := c.values.get("exts"); exts != nil {
		for name, extVal := range exts {
			if ext := intfToMp(extVal); ext == nil {
				ers.addError("(exts) %s is %T but expected map [%v]",
					name, extVal, extVal)
			} else {
				extCommonValidator.validateMap(name, ext, ers)
				extNormalValidator.validateMap(name, ext, ers)

				if active := ext.get("active"); active != nil {
					validateActive(name, active, ers)
				}
			}
		}
	}
}

func validateExtConfig(m map[string]interface{}, ers *errList) {
	addErr := func(kind, key string, val interface{}) {
		ers.addError(
			"(ext config) %s is %T but expected %s [%v]",
			key, val, kind, val)
	}

	for key, val := range m {
		switch v := val.(type) {
		case map[string]interface{}:
			switch key {
			case "networks":
				validateExtNetConfig(v, ers)
			case "channels":
				validateExtChanConfig("", v, ers)
			default:
				addErr("string", key, val)
			}
		case string:
			switch key {
			case "networks":
				fallthrough
			case "channels":
				addErr("map", key, val)
			}
		default:
			switch key {
			case "networks":
				fallthrough
			case "channels":
				addErr("map", key, val)
			default:
				addErr("string", key, val)
			}
		}
	}
}

func validateExtNetConfig(m map[string]interface{}, ers *errList) {
	addErr := func(net, kind, key string, val interface{}) {
		ers.addError(
			"(ext config%s) %s is %T but expected %s [%v]",
			net, key, val, kind, val)
	}

	for key, val := range m {
		switch v := val.(type) {
		case map[string]interface{}:
			for keyname, val := range v {
				switch keyname {
				case "channels":
					if chs, ok := val.(map[string]interface{}); ok {
						validateExtChanConfig(key, chs, ers)
					} else {
						addErr(" "+key, "map", keyname, val)
					}
				default:
					if _, ok := val.(string); !ok {
						addErr(" "+key, "string", keyname, val)
					}
				}
			}
		default:
			addErr("", "map", key, val)
		}
	}
}

func validateExtChanConfig(net string, m map[string]interface{}, ers *errList) {
	addErr := func(ch, kind, key string, val interface{}) {
		var ctx string
		if len(net) > 0 {
			ctx = fmt.Sprintf(
				"ext config %s%s", net, ch)
		} else {
			ctx = fmt.Sprintf("ext config%s", ch)
		}
		ers.addError("(%s) %s is %T but expected %s [%v]",
			ctx, key, val, kind, val)
	}

	for key, val := range m {
		switch v := val.(type) {
		case map[string]interface{}:
			for keyname, val := range v {
				if _, ok := val.(string); !ok {
					addErr(" "+key, "string", keyname, val)
				}
			}
		default:
			addErr("", "map", key, val)
		}
	}
}

func validateActive(ext string, m map[string]interface{}, ers *errList) {
	var acts []interface{}
	var ok bool
	for activeNet, actVal := range m {
		if acts, ok = actVal.([]interface{}); !ok {
			ers.addError(
				"(%s active) %s is %T but expected array [%v]",
				ext, activeNet, actVal, actVal)
			return
		}

		for i, chval := range acts {
			if _, ok = chval.(string); !ok {
				ers.addError(
					"(%s active %s) %v %d is %T but expected string [%v]",
					ext, activeNet, "channel", i+1, chval, chval)
			}
		}
	}
}

// validateMap checks map's values for correct types based on the validatorRules
func (v validatorRules) validateMap(name string,
	m map[string]interface{}, ers *errList) {

	addErr := func(name, key, kind string, val interface{}) {
		ers.addError("(%s) %s is %T but expected %s [%v]",
			name, key, val, kind, val)
	}

	for _, key := range v.stringVals {
		if val, ok := m[key]; !ok {
			continue
		} else if _, ok = val.(string); !ok {
			addErr(name, key, "string", val)
		}
	}
	for _, key := range v.stringSliceVals {
		if val, ok := m[key]; ok {
			switch v := val.(type) {
			case []interface{}:
				for i, val := range v {
					if _, ok := val.(string); !ok {
						indexErr := fmt.Sprintf("%s %d", key, i+1)
						addErr(name, indexErr, "string", val)
					}
				}
			case []string:
			default:
				addErr(name, key, "array", val)
			}
		}
	}
	for _, key := range v.boolVals {
		if val, ok := m[key]; !ok {
			continue
		} else if _, ok = val.(bool); !ok {
			addErr(name, key, "bool", val)
		}
	}
	for _, key := range v.floatVals {
		if val, ok := m[key]; !ok {
			continue
		} else if _, ok = val.(float64); !ok {
			addErr(name, key, "float64", val)
		}
	}
	/* These are not used for the time being
	for _, key := range v.intVals {
		if val, ok := m[key]; !ok {
			continue
		} else if _, ok = val.(int64); !ok {
			addErr(name, key, "int", val)
		}
	}*/
	for _, key := range v.uintVals {
		if val, ok := m[key]; !ok {
			continue
		} else if _, ok = val.(int64); !ok {
			addErr(name, key, "int", val)
		}
	}
	for _, key := range v.mapVals {
		if val, ok := m[key]; !ok {
			continue
		} else if _, ok = val.(map[string]interface{}); !ok {
			addErr(name, key, "map", val)
		}
	}
	for _, key := range v.mapArrVals {
		if val, ok := m[key]; !ok {
			continue
		} else if _, ok = val.([]map[string]interface{}); !ok {
			addErr(name, key, "map array", val)
		}
	}
}
