package config

import "sync"

// NetCTX is a context for network parts of the config, allowing querying and
// setting of network related values. If this context belongs to a specific
// network and not the global network configuration, all the values will
// fallback to the global configuration if not set.
type NetCTX struct {
	mutex   *sync.RWMutex
	parent  map[string]interface{}
	network map[string]interface{}
}

func (n *NetCTX) lock()    { n.mutex.Lock() }
func (n *NetCTX) unlock()  { n.mutex.Unlock() }
func (n *NetCTX) rlock()   { n.mutex.RLock() }
func (n *NetCTX) runlock() { n.mutex.RUnlock() }

func (n *NetCTX) get(key string) (interface{}, bool) {
	v, ok := n.network[key]
	return v, ok
}

func (n *NetCTX) getParent(key string) (interface{}, bool) {
	if n.parent == nil {
		return nil, false
	}

	v, ok := n.parent[key]
	return v, ok
}

func (n *NetCTX) set(key string, value interface{}) {
	n.network[key] = value
}

func (n *NetCTX) Nick() (string, bool) {
	return getStr(n, "nick", true)
}

func (n *NetCTX) SetNick(val string) {
	setVal(n, "nick", val)
}

func (n *NetCTX) Altnick() (string, bool) {
	return getStr(n, "altnick", true)
}

func (n *NetCTX) SetAltnick(val string) {
	setVal(n, "altnick", val)
}
func (n *NetCTX) Username() (string, bool) {
	return getStr(n, "username", true)
}

func (n *NetCTX) SetUsername(val string) {
	setVal(n, "username", val)
}

func (n *NetCTX) Realname() (string, bool) {
	return getStr(n, "realname", true)
}

func (n *NetCTX) SetRealname(val string) {
	setVal(n, "realname", val)
}

func (n *NetCTX) Password() (string, bool) {
	return getStr(n, "password", true)
}

func (n *NetCTX) SetPassword(val string) {
	setVal(n, "password", val)
}

func (n *NetCTX) SSL() (bool, bool) {
	return getBool(n, "ssl", true)
}

func (n *NetCTX) SetSSL(val bool) {
	setVal(n, "ssl", val)
}

func (n *NetCTX) SSLCert() (string, bool) {
	return getStr(n, "sslcert", true)
}

func (n *NetCTX) SetSSLCert(val string) {
	setVal(n, "sslcert", val)
}

func (n *NetCTX) NoVerifyCert() (bool, bool) {
	return getBool(n, "noverifycert", true)
}

func (n *NetCTX) SetNoVerifyCert(val bool) {
	setVal(n, "noverifycert", val)
}

func (n *NetCTX) NoState() (bool, bool) {
	return getBool(n, "nostate", true)
}

func (n *NetCTX) SetNoState(val bool) {
	setVal(n, "nostate", val)
}

func (n *NetCTX) NoStore() (bool, bool) {
	return getBool(n, "nostore", true)
}

func (n *NetCTX) SetNoStore(val bool) {
	setVal(n, "nostore", val)
}

func (n *NetCTX) FloodLenPenalty() (uint, bool) {
	if floodLenPenalty, ok := getUint(n, "floodlenpenalty", true); ok {
		return floodLenPenalty, true
	}
	return defaultFloodLenPenalty, false
}

func (n *NetCTX) SetFloodLenPenalty(val uint) {
	setVal(n, "floodlenpenalty", val)
}

func (n *NetCTX) FloodTimeout() (float64, bool) {
	if floodTimeout, ok := getFloat64(n, "floodtimeout", true); ok {
		return floodTimeout, ok
	}
	return defaultFloodTimeout, false
}

func (n *NetCTX) SetFloodTimeout(val float64) {
	setVal(n, "floodtimeout", val)
}

func (n *NetCTX) FloodStep() (float64, bool) {
	if floodStep, ok := getFloat64(n, "floodstep", true); ok {
		return floodStep, ok
	}
	return defaultFloodStep, false
}

func (n *NetCTX) SetFloodStep(val float64) {
	setVal(n, "floodstep", val)
}

func (n *NetCTX) KeepAlive() (float64, bool) {
	if keepAlive, ok := getFloat64(n, "keepalive", true); ok {
		return keepAlive, ok
	}
	return defaultKeepAlive, false
}

func (n *NetCTX) SetKeepAlive(val float64) {
	setVal(n, "keepalive", val)
}

func (n *NetCTX) NoReconnect() (bool, bool) {
	return getBool(n, "noreconnect", true)
}

func (n *NetCTX) SetNoReconnect(val bool) {
	setVal(n, "noreconnect", val)
}

func (n *NetCTX) ReconnectTimeout() (uint, bool) {
	if reconnTimeout, ok := getUint(n, "reconnecttimeout", true); ok {
		return reconnTimeout, ok
	}
	return defaultReconnectTimeout, false
}

func (n *NetCTX) SetReconnectTimeout(val uint) {
	setVal(n, "reconnecttimeout", val)
}

func (n *NetCTX) Prefix() (string, bool) {
	if prefix, ok := getStr(n, "prefix", true); ok {
		return prefix, ok
	}
	return string(defaultPrefix), false
}

func (n *NetCTX) SetPrefix(val string) {
	setVal(n, "prefix", val)
}

// Channel is the configuration for a single channel.
type Channel struct {
	Name     string
	Password string
	Prefix   string
}

func (n *NetCTX) Channels() ([]Channel, bool) {
	n.rlock()
	defer n.runlock()

	var val interface{}
	var ok bool

	if val, ok = n.network["channels"]; !ok {
		val, ok = n.parent["channels"]
	}

	if !ok {
		return nil, false
	}

	if arr, ok := val.([]map[string]interface{}); ok {
		ret := make([]Channel, len(arr))
		for i, ch := range arr {
			if nameVal, ok := ch["name"]; ok {
				if name, ok := nameVal.(string); ok {
					ret[i].Name = name
				}
			}
			if passwordVal, ok := ch["password"]; ok {
				if password, ok := passwordVal.(string); ok {
					ret[i].Password = password
				}
			}
			if prefixVal, ok := ch["prefix"]; ok {
				if prefix, ok := prefixVal.(string); ok {
					ret[i].Prefix = prefix
				}
			}
		}

		return ret, true
	} else if arr, ok := val.([]Channel); ok {
		ret := make([]Channel, len(arr))
		copy(ret, arr)
		return ret, true
	}

	return nil, false
}

func (n *NetCTX) SetChannels(val []Channel) {
	setVal(n, "channels", val)
}

func (n *NetCTX) Servers() ([]string, bool) {
	return getStrArr(n, "servers", false)
}

func (n *NetCTX) SetServers(val []string) {
	setVal(n, "servers", val)
}
