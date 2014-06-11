package config

import "sync"

// netCtx is a context for network parts of the config, allowing querying and
// setting of network related values.
type netCtx struct {
	mutex   *sync.RWMutex
	parent  map[string]interface{}
	network map[string]interface{}
}

func (n *netCtx) lock()    { n.mutex.Lock() }
func (n *netCtx) unlock()  { n.mutex.Unlock() }
func (n *netCtx) rlock()   { n.mutex.RLock() }
func (n *netCtx) runlock() { n.mutex.RUnlock() }

func (n *netCtx) get(key string) (interface{}, bool) {
	v, ok := n.network[key]
	return v, ok
}

func (n *netCtx) getParent(key string) (interface{}, bool) {
	if n.parent == nil {
		return nil, false
	}

	v, ok := n.parent[key]
	return v, ok
}

func (n *netCtx) set(key string, value interface{}) {
	n.network[key] = value
}

func (n *netCtx) Nick() (string, bool) {
	return getStr(n, "nick", true)
}

func (n *netCtx) SetNick(val string) {
	setVal(n, "nick", val)
}

func (n *netCtx) Altnick() (string, bool) {
	return getStr(n, "altnick", true)
}

func (n *netCtx) SetAltnick(val string) {
	setVal(n, "altnick", val)
}
func (n *netCtx) Username() (string, bool) {
	return getStr(n, "username", true)
}

func (n *netCtx) SetUsername(val string) {
	setVal(n, "username", val)
}

func (n *netCtx) Realname() (string, bool) {
	return getStr(n, "realname", true)
}

func (n *netCtx) SetRealname(val string) {
	setVal(n, "realname", val)
}

func (n *netCtx) Password() (string, bool) {
	return getStr(n, "password", true)
}

func (n *netCtx) SetPassword(val string) {
	setVal(n, "password", val)
}

func (n *netCtx) SSL() (bool, bool) {
	return getBool(n, "ssl", true)
}

func (n *netCtx) SetSSL(val bool) {
	setVal(n, "ssl", val)
}

func (n *netCtx) SSLCert() (string, bool) {
	return getStr(n, "sslcert", true)
}

func (n *netCtx) SetSSLCert(val string) {
	setVal(n, "sslcert", val)
}

func (n *netCtx) NoVerifyCert() (bool, bool) {
	return getBool(n, "noverifycert", true)
}

func (n *netCtx) SetNoVerifyCert(val bool) {
	setVal(n, "noverifycert", val)
}

func (n *netCtx) NoState() (bool, bool) {
	return getBool(n, "nostate", true)
}

func (n *netCtx) SetNoState(val bool) {
	setVal(n, "nostate", val)
}

func (n *netCtx) NoStore() (bool, bool) {
	return getBool(n, "nostore", true)
}

func (n *netCtx) SetNoStore(val bool) {
	setVal(n, "nostore", val)
}

func (n *netCtx) FloodLenPenalty() (uint, bool) {
	return getUint(n, "floodlenpenalty", true)
}

func (n *netCtx) SetFloodLenPenalty(val uint) {
	setVal(n, "floodlenpenalty", val)
}

func (n *netCtx) FloodTimeout() (float64, bool) {
	return getFloat64(n, "floodtimeout", true)
}

func (n *netCtx) SetFloodTimeout(val float64) {
	setVal(n, "floodtimeout", val)
}

func (n *netCtx) FloodStep() (float64, bool) {
	return getFloat64(n, "floodstep", true)
}

func (n *netCtx) SetFloodStep(val float64) {
	setVal(n, "floodstep", val)
}

func (n *netCtx) KeepAlive() (float64, bool) {
	return getFloat64(n, "keepalive", true)
}

func (n *netCtx) SetKeepAlive(val float64) {
	setVal(n, "keepalive", val)
}

func (n *netCtx) NoReconnect() (bool, bool) {
	return getBool(n, "noreconnect", true)
}

func (n *netCtx) SetNoReconnect(val bool) {
	setVal(n, "noreconnect", val)
}

func (n *netCtx) ReconnectTimeout() (uint, bool) {
	return getUint(n, "reconnecttimeout", true)
}

func (n *netCtx) SetReconnectTimeout(val uint) {
	setVal(n, "reconnecttimeout", val)
}

func (n *netCtx) Prefix() (string, bool) {
	return getStr(n, "prefix", true)
}

func (n *netCtx) SetPrefix(val string) {
	setVal(n, "prefix", val)
}

// Channel is the configuration for a single channel.
type Channel struct {
	Name     string
	Password string
	Prefix   string
}

func (n *netCtx) Channels() ([]Channel, bool) {
	n.rlock()
	defer n.runlock()

	var val interface{}
	var arr []map[string]interface{}
	var ok bool

	if val, ok = n.network["channels"]; !ok {
		val, ok = n.parent["channels"]
	}

	if !ok {
		return nil, false
	}

	if arr, ok = val.([]map[string]interface{}); !ok || len(arr) == 0 {
		return nil, false
	}

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
}

func (n *netCtx) SetChannels(val []Channel) {
	setVal(n, "channels", val)
}

// Servers returns the list of servers for the network.
func (n *netCtx) Servers() ([]string, bool) {
	return getStrArr(n, "servers", false)
}
