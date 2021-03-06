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

func (n *NetCTX) SetNick(val string) *NetCTX {
	setVal(n, "nick", val)
	return n
}

func (n *NetCTX) Altnick() (string, bool) {
	return getStr(n, "altnick", true)
}

func (n *NetCTX) SetAltnick(val string) *NetCTX {
	setVal(n, "altnick", val)
	return n
}
func (n *NetCTX) Username() (string, bool) {
	return getStr(n, "username", true)
}

func (n *NetCTX) SetUsername(val string) *NetCTX {
	setVal(n, "username", val)
	return n
}

func (n *NetCTX) Realname() (string, bool) {
	return getStr(n, "realname", true)
}

func (n *NetCTX) SetRealname(val string) *NetCTX {
	setVal(n, "realname", val)
	return n
}

func (n *NetCTX) Password() (string, bool) {
	return getStr(n, "password", true)
}

func (n *NetCTX) SetPassword(val string) *NetCTX {
	setVal(n, "password", val)
	return n
}

func (n *NetCTX) TLS() (bool, bool) {
	return getBool(n, "tls", true)
}

func (n *NetCTX) SetTLS(val bool) *NetCTX {
	setVal(n, "tls", val)
	return n
}

func (n *NetCTX) TLSCACert() (string, bool) {
	return getStr(n, "tls_ca_cert", true)
}

func (n *NetCTX) SetTLSCACert(val string) *NetCTX {
	setVal(n, "tls_ca_cert", val)
	return n
}

func (n *NetCTX) TLSCert() (string, bool) {
	return getStr(n, "tls_cert", true)
}

func (n *NetCTX) SetTLSCert(val string) *NetCTX {
	setVal(n, "tls_cert", val)
	return n
}

func (n *NetCTX) TLSKey() (string, bool) {
	return getStr(n, "tls_key", true)
}

func (n *NetCTX) SetTLSKey(val string) *NetCTX {
	setVal(n, "tls_key", val)
	return n
}

func (n *NetCTX) TLSInsecureSkipVerify() (bool, bool) {
	return getBool(n, "tls_insecure_skip_verify", true)
}

func (n *NetCTX) SetTLSInsecureSkipVerify(val bool) *NetCTX {
	setVal(n, "tls_insecure_skip_verify", val)
	return n
}

func (n *NetCTX) NoState() (bool, bool) {
	return getBool(n, "nostate", true)
}

func (n *NetCTX) SetNoState(val bool) *NetCTX {
	setVal(n, "nostate", val)
	return n
}

func (n *NetCTX) NoStore() (bool, bool) {
	return getBool(n, "nostore", true)
}

func (n *NetCTX) SetNoStore(val bool) *NetCTX {
	setVal(n, "nostore", val)
	return n
}

func (n *NetCTX) NoAutoJoin() (bool, bool) {
	return getBool(n, "noautojoin", true)
}

func (n *NetCTX) SetNoAutoJoin(val bool) *NetCTX {
	setVal(n, "noautojoin", val)
	return n
}

func (n *NetCTX) JoinDelay() (uint, bool) {
	if floodLenPenalty, ok := getUint(n, "joindelay", true); ok {
		return floodLenPenalty, true
	}
	return defaultJoinDelay, false
}

func (n *NetCTX) SetJoinDelay(val uint) *NetCTX {
	setVal(n, "joindelay", val)
	return n
}

func (n *NetCTX) FloodLenPenalty() (uint, bool) {
	if floodLenPenalty, ok := getUint(n, "floodlenpenalty", true); ok {
		return floodLenPenalty, true
	}
	return defaultFloodLenPenalty, false
}

func (n *NetCTX) SetFloodLenPenalty(val uint) *NetCTX {
	setVal(n, "floodlenpenalty", val)
	return n
}

func (n *NetCTX) FloodTimeout() (float64, bool) {
	if floodTimeout, ok := getFloat64(n, "floodtimeout", true); ok {
		return floodTimeout, ok
	}
	return defaultFloodTimeout, false
}

func (n *NetCTX) SetFloodTimeout(val float64) *NetCTX {
	setVal(n, "floodtimeout", val)
	return n
}

func (n *NetCTX) FloodStep() (float64, bool) {
	if floodStep, ok := getFloat64(n, "floodstep", true); ok {
		return floodStep, ok
	}
	return defaultFloodStep, false
}

func (n *NetCTX) SetFloodStep(val float64) *NetCTX {
	setVal(n, "floodstep", val)
	return n
}

func (n *NetCTX) KeepAlive() (float64, bool) {
	if keepAlive, ok := getFloat64(n, "keepalive", true); ok {
		return keepAlive, ok
	}
	return defaultKeepAlive, false
}

func (n *NetCTX) SetKeepAlive(val float64) *NetCTX {
	setVal(n, "keepalive", val)
	return n
}

func (n *NetCTX) NoReconnect() (bool, bool) {
	return getBool(n, "noreconnect", true)
}

func (n *NetCTX) SetNoReconnect(val bool) *NetCTX {
	setVal(n, "noreconnect", val)
	return n
}

func (n *NetCTX) ReconnectTimeout() (uint, bool) {
	if reconnTimeout, ok := getUint(n, "reconnecttimeout", true); ok {
		return reconnTimeout, ok
	}
	return defaultReconnectTimeout, false
}

func (n *NetCTX) SetReconnectTimeout(val uint) *NetCTX {
	setVal(n, "reconnecttimeout", val)
	return n
}

func (n *NetCTX) Prefix() (rune, bool) {
	if prefix, ok := getStr(n, "prefix", true); ok {
		if len(prefix) > 0 {
			return rune(prefix[0]), ok
		}
	}
	return defaultPrefix, false
}

// ChannelPrefix retrieves the prefix with the correct default chain up to
// network and global space.
func (n *NetCTX) ChannelPrefix(channel string) (rune, bool) {
	n.rlock()
	defer n.runlock()

	var val interface{}
	var channelsList []map[string]interface{}
	var ok bool

	if val, ok = n.network["channels"]; !ok {
		val, ok = n.parent["channels"]
	}

	if !ok {
		return n.Prefix()
	}

	channelsList, ok = val.([]map[string]interface{})
	if !ok {
		return n.Prefix()
	}

	var foundChan map[string]interface{}
	for _, c := range channelsList {
		if name, ok := c["name"]; !ok {
			continue
		} else if nameStr, ok := name.(string); !ok {
			continue
		} else {
			if nameStr == channel {
				foundChan = c
				break
			}
		}

	}

	if foundChan == nil {
		return n.Prefix()
	}

	if pfxIntf, ok := foundChan["prefix"]; ok {
		if str, ok := pfxIntf.(string); ok {
			return rune(str[0]), true
		}
	}

	return n.Prefix()
}

func (n *NetCTX) SetPrefix(val rune) *NetCTX {
	setVal(n, "prefix", string(val))
	return n
}

// Channel is the configuration for a single channel.
type Channel struct {
	Name     string
	Password string
	Prefix   string
}

func (n *NetCTX) Channels() (map[string]Channel, bool) {
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

	if channelsList, ok := val.([]map[string]interface{}); ok {
		ret := make(map[string]Channel, len(channelsList))

		for _, chanVals := range channelsList {
			var c Channel

			if nameVal, ok := chanVals["name"]; !ok {
				// Need to have a name
				continue
			} else {
				if name, ok := nameVal.(string); ok {
					c.Name = name
				}
			}
			if passwordVal, ok := chanVals["password"]; ok {
				if password, ok := passwordVal.(string); ok {
					c.Password = password
				}
			}
			if prefixVal, ok := chanVals["prefix"]; ok {
				if prefix, ok := prefixVal.(string); ok {
					c.Prefix = prefix
				}
			}

			ret[c.Name] = c
		}

		return ret, true
	}

	return nil, false
}

func (n *NetCTX) SetChannels(val []Channel) *NetCTX {
	channelsList := make([]map[string]interface{}, len(val))
	for _, v := range val {
		chanMap := map[string]interface{}{}
		chanMap["name"] = v.Name
		if len(v.Password) > 0 {
			chanMap["password"] = v.Password
		}
		if len(v.Password) > 0 {
			chanMap["prefix"] = v.Prefix
		}
		channelsList = append(channelsList, chanMap)
	}
	setVal(n, "channels", channelsList)
	return n
}

func (n *NetCTX) Servers() ([]string, bool) {
	return getStrArr(n, "servers", false)
}

func (n *NetCTX) SetServers(val []string) *NetCTX {
	setVal(n, "servers", val)
	return n
}
