package config

import "sync"

type extGlobalCtx struct {
	*extCtx
}

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

func (e *extGlobalCtx) NoReconnect() (bool, bool) {
	return getBool(e, "noreconnect", false)
}

func (e *extGlobalCtx) SetNoReconnect(val bool) {
	setVal(e, "noreconnect", val)
}

func (e *extGlobalCtx) ReconnectTimeout() (uint, bool) {
	return getUint(e, "reconnecttimeout", false)
}

func (e *extGlobalCtx) SetReconnectTimeout(val uint) {
	setVal(e, "reconnecttimeout", val)
}

func (e *extGlobalCtx) UseJson() (bool, bool) {
	return getBool(e, "usejson", false)
}

func (e *extGlobalCtx) SetUseJson(val bool) {
	setVal(e, "usejson", val)
}

type extNormalCtx struct {
	*extCtx
}

func (e *extNormalCtx) Server() (string, bool) {
	return getStr(e, "server", false)
}

func (e *extNormalCtx) SetServer(val string) {
	setVal(e, "server", val)
}

func (e *extNormalCtx) Exec() (string, bool) {
	return getStr(e, "exec", false)
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

func (e *extNormalCtx) NoReconnect() (bool, bool) {
	return getBool(e, "noreconnect", true)
}

func (e *extNormalCtx) SetNoReconnect(val bool) {
	setVal(e, "noreconnect", val)
}

func (e *extNormalCtx) ReconnectTimeout() (uint, bool) {
	return getUint(e, "reconnecttimeout", true)
}

func (e *extNormalCtx) SetReconnectTimeout(val uint) {
	setVal(e, "reconnecttimeout", val)
}

func (e *extNormalCtx) UseJson() (bool, bool) {
	return getBool(e, "usejson", true)
}

func (e *extNormalCtx) SetUseJson(val bool) {
	setVal(e, "usejson", val)
}
