package config

import "sync"

type extCtx struct {
	mutex  *sync.RWMutex
	parent map[string]interface{}
	ext    map[string]interface{}
}

func (e *extCtx) lock()    { e.mutex.Lock() }
func (e *extCtx) unlock()  { e.mutex.Unlock() }
func (e *extCtx) rlock()   { e.mutex.RLock() }
func (e *extCtx) runlock() { e.mutex.RUnlock() }

func (e *extCtx) get(key string) (interface{}, bool) {
	v, ok := e.ext[key]
	return v, ok
}

func (e *extCtx) getParent(key string) (interface{}, bool) {
	if e.parent == nil {
		return nil, false
	}

	v, ok := e.parent[key]
	return v, ok
}

func (e *extCtx) set(key string, value interface{}) {
	e.ext[key] = value
}

func (e *extCtx) UseJson() (bool, bool) {
	return getBool(e, "usejson", true)
}

func (e *extCtx) SetUseJson(val bool) {
	setVal(e, "usejson", val)
}

func (e *extCtx) NoReconnect() (bool, bool) {
	return getBool(e, "noreconnect", true)
}

func (e *extCtx) SetNoReconnect(val bool) {
	setVal(e, "noreconnect", val)
}

func (e *extCtx) ReconnectTimeout() (uint, bool) {
	reconnTimeout, ok := getUint(e, "reconnecttimeout", true)
	if !ok {
		return defaultReconnectTimeout, ok
	}
	return reconnTimeout, ok
}

func (e *extCtx) SetReconnectTimeout(val uint) {
	setVal(e, "reconnecttimeout", val)
}

func (e *extCtx) Active(network string) ([]string, bool) {
	e.rlock()
	defer e.runlock()

	var actives map[string]interface{}
	var arrVal interface{}
	var ok bool

	copyArr := func(arrVal interface{}) []string {
		newActives := make([]string, 0)

		switch v := arrVal.(type) {
		case []interface{}:
			if len(v) == 0 {
				return nil
			}

			for _, strVal := range v {
				if str, ok := strVal.(string); ok {
					newActives = append(newActives, str)
				}
			}
		case []string:
			if len(v) == 0 {
				return nil
			}

			for _, str := range v {
				newActives = append(newActives, str)
			}
		}

		return newActives
	}

	if actives = mp(e.ext).get("active"); actives != nil {
		if arrVal, ok = actives[network]; ok {
			arr := copyArr(arrVal)
			if len(arr) != 0 {
				return arr, true
			}
		}
	}

	if e.parent != nil {
		if actives = mp(e.parent).get("active"); actives != nil {
			if arrVal, ok = actives[network]; ok {
				arr := copyArr(arrVal)
				if len(arr) != 0 {
					return arr, true
				}
			}
		}
	}

	return nil, false
}

func (e *extCtx) SetActive(network string, value []string) {
	e.lock()
	defer e.unlock()

	if acts := mp(e.ext).get("active"); acts != nil {
		acts[network] = value
	} else {
		e.ext["active"] = map[string]interface{}{
			network: value,
		}
	}
}

type extGlobalCtx struct {
	*extCtx
}

func (e *extGlobalCtx) ExecDir() (string, bool) {
	return getStr(e, "execdir", false)
}

func (e *extGlobalCtx) SetExecDir(val string) {
	setVal(e, "execdir", val)
}

func (e *extGlobalCtx) Listen() (string, bool) {
	return getStr(e, "listen", false)
}

func (e *extGlobalCtx) SetListen(val string) {
	setVal(e, "listen", val)
}

/*
Config returns a map of config values for the given network and channel.
Global values are overidden by more specific ones, and all more-global values
are picked up.
*/
func (e *extGlobalCtx) Config(network, channel string) map[string]string {
	ret := make(map[string]string)
	nEmpty := len(network) == 0
	cEmpty := len(channel) == 0

	var m = mp(e.ext)
	cfg := m.get("config")
	net := cfg.get("networks").get(network)
	ch := cfg.get("channels").get(channel)
	netch := net.get("channels").get(channel)

	for k, v := range cfg {
		if str, ok := v.(string); ok {
			ret[k] = str
		}
	}

	if !nEmpty && cEmpty && net != nil {
		for k, v := range net {
			if str, ok := v.(string); ok {
				ret[k] = str
			}
		}
	}

	if !cEmpty && nEmpty && ch != nil {
		for k, v := range ch {
			if str, ok := v.(string); ok {
				ret[k] = str
			}
		}
	}

	if !nEmpty && !cEmpty && netch != nil {
		for k, v := range netch {
			if str, ok := v.(string); ok {
				ret[k] = str
			}
		}
	}

	return ret
}

/*
SetConfig sets a key value pair for a given network and channel.
If you leave either network or channel empty, then it's set at the global
level for that portion.

	[ext.config]
		# SetConfig("", "", "key", "val")
		key = "val"
	[ext.config.channels.#channel]
		# SetConfig("", "#channel", "key", "val")
		key = "val"
	[ext.config.networks.ircnet]
		# SetConfig("ircnet", "", "key", "val")
		key = "val"
	[ext.config.networks.ircnet.channels.#channel]
		# SetConfig("ircnet", "#channel", "key", "val")
		key = "val"
*/
func (e *extGlobalCtx) SetConfig(network, channel, key, value string) {
	var setMap map[string]interface{}
	var m = mp(e.ext)

	nEmpty := len(network) == 0
	cEmpty := len(channel) == 0

	switch {
	case nEmpty && cEmpty:
		setMap = m.ensure("config")
	case !nEmpty && cEmpty:
		setMap = m.ensure("config").ensure("networks").ensure(network)
	case nEmpty && !cEmpty:
		setMap = m.ensure("config").ensure("channels").ensure(channel)
	case !nEmpty && !cEmpty:
		setMap = m.ensure("config").ensure("networks").ensure(network).
			ensure("channels").ensure(channel)
	}

	setMap[key] = value
}

type extNormalCtx struct {
	*extCtx
}

func (e *extNormalCtx) Exec() (string, bool) {
	return getStr(e, "exec", false)
}

func (e *extNormalCtx) SetExec(val string) {
	setVal(e, "exec", val)
}

func (e *extNormalCtx) Server() (string, bool) {
	return getStr(e, "server", false)
}

func (e *extNormalCtx) SetServer(val string) {
	setVal(e, "server", val)
}

func (e *extNormalCtx) SSL() (bool, bool) {
	return getBool(e, "ssl", false)
}

func (e *extNormalCtx) SetSSL(val bool) {
	setVal(e, "ssl", val)
}

func (e *extNormalCtx) SSLCert() (string, bool) {
	return getStr(e, "sslcert", false)
}

func (e *extNormalCtx) SetSSLCert(val string) {
	setVal(e, "sslcert", val)
}

func (e *extNormalCtx) NoVerifyCert() (bool, bool) {
	return getBool(e, "noverifycert", false)
}

func (e *extNormalCtx) SetNoVerifyCert(val bool) {
	setVal(e, "noverifycert", val)
}

func (e *extNormalCtx) Unix() (string, bool) {
	return getStr(e, "unix", false)
}

func (e *extNormalCtx) SetUnix(val string) {
	setVal(e, "unix", val)
}
