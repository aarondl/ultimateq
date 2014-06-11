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
	return getUint(e, "reconnecttimeout", true)
}

func (e *extCtx) SetReconnectTimeout(val uint) {
	setVal(e, "reconnecttimeout", val)
}

func (e *extCtx) Active(network string) ([]string, bool) {
	e.rlock()
	defer e.runlock()

	var val interface{}
	var actives map[string]interface{}
	var ok bool

	if val, ok = e.ext["active"]; !ok {
		val, ok = e.parent["active"]
	}

	if !ok {
		return nil, false
	}

	if actives, ok = val.(map[string]interface{}); !ok || len(actives) == 0 {
		return nil, false
	}

	if interfaceValue, ok := actives[network]; ok {
		newActives := make([]string, 0)

		if strArr, ok := interfaceValue.([]interface{}); ok {
			for _, strVal := range strArr {
				if str, ok := strVal.(string); ok {
					newActives = append(newActives, str)
				}
			}
		}

		if len(newActives) > 0 {
			return newActives, true
		} else {
			return nil, false
		}
	}

	return nil, false
}

func (e *extCtx) SetActive(network string, value []string) {
	e.lock()
	defer e.unlock()

	var val interface{}
	var actives map[string]interface{}
	var ok bool

	if val, ok = e.ext["active"]; !ok {
		val, ok = e.parent["active"]
	}

	if !ok {
		return
	}

	if actives, ok = val.(map[string]interface{}); !ok || len(actives) == 0 {
		return
	}

	actives[network] = value
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

type extNormalCtx struct {
	*extCtx
}

func (e *extNormalCtx) Exec() (string, bool) {
	return getStr(e, "exec", false)
}

func (e *extNormalCtx) Server() (string, bool) {
	return getStr(e, "server", false)
}

func (e *extNormalCtx) SetServer(val string) {
	setVal(e, "server", val)
}

func (e *extNormalCtx) SetExec(val string) {
	setVal(e, "exec", val)
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
