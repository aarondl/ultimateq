package config

import (
	"strconv"
	"sync"
)

// Ext represents an extension.
type Ext struct {
	parent  *Ext          `toml:"-" json:"-"`
	protect *sync.RWMutex `toml:"-" json:"-"`

	// All Extensions
	InName   string            `toml:"-" json:"-"`
	InConfig map[string]string `toml:"config" json:"config"`

	// Local Only
	InLocal string `toml:"local" json:"local"`

	// Remote Only
	InExec string `toml:"exec" json:"exec"`
	// If set it will use json not gob.
	InUseJSON string `toml:"usejson" json:"usejson"`

	// If the extension is the server
	// not the bot. The bot will connect.
	InIsServer string `toml:"isserver" json:"isserver"`

	// TCP/IP
	InAddress       string `toml:"address" json:"address"`
	InSsl           string `toml:"ssl" json:"ssl"`
	InSslClientCert string `toml:"sslclientcert" json:"sslclientcert"`
	InNoVerifyCert  string `toml:"noverifycert" json:"noverifycert"`

	// UNIX
	InSock string `toml:"sock" json:"sock"`

	// Auto Connections
	InNoReconnect      string `toml:"noreconnect" json:"noreconnect"`
	InReconnectTimeout string `toml:"reconnecttimeout" json:"reconnecttimeout"`
}

// Clone deep copies the Extension.
func (e *Ext) Clone() *Ext {
	e.protect.RLock()
	defer e.protect.RUnlock()

	newExt := *e
	newExt.InConfig = make(map[string]string)
	for k, v := range e.InConfig {
		newExt.InConfig[k] = v
	}

	return &newExt
}

// Name returns the extensions name.
func (e *Ext) Name() string {
	e.protect.RLock()
	defer e.protect.RUnlock()

	return e.InName
}

// Config returns the extensions name.
func (e *Ext) Config() map[string]string {
	e.protect.RLock()
	defer e.protect.RUnlock()

	var conf map[string]string
	if e.InConfig != nil {
		conf = e.InConfig
	} else if e.parent != nil && e.parent.InConfig != nil {
		conf = e.parent.InConfig
	}
	m := make(map[string]string, len(conf))
	for k, v := range conf {
		m[k] = v
	}
	return m
}

// Local returns Local of the network, or the global local, or false
func (e *Ext) Local() (local bool) {
	e.protect.RLock()
	defer e.protect.RUnlock()

	var err error
	if len(e.InLocal) != 0 {
		local, err = strconv.ParseBool(e.InLocal)
	} else if e.parent != nil && len(e.parent.InLocal) != 0 {
		local, err = strconv.ParseBool(e.parent.InLocal)
	}

	if err != nil {
		local = false
	}
	return
}

// Exec gets Exec of the network, or the global exec, or empty string.
func (e *Ext) Exec() (exec string) {
	e.protect.RLock()
	defer e.protect.RUnlock()

	if len(e.InExec) > 0 {
		exec = e.InExec
	} else if e.parent != nil && len(e.parent.InExec) > 0 {
		exec = e.parent.InExec
	}
	return
}

// UseJSON returns UseJSON of the network, or the global usejson, or false
func (e *Ext) UseJSON() (usejson bool) {
	e.protect.RLock()
	defer e.protect.RUnlock()

	var err error
	if len(e.InUseJSON) != 0 {
		usejson, err = strconv.ParseBool(e.InUseJSON)
	} else if e.parent != nil && len(e.parent.InUseJSON) != 0 {
		usejson, err = strconv.ParseBool(e.parent.InUseJSON)
	}

	if err != nil {
		usejson = false
	}
	return
}

// IsServer gets IsServer of the network, or the global isserver, or
// false
func (e *Ext) IsServer() (isserver bool) {
	e.protect.RLock()
	defer e.protect.RUnlock()

	var err error
	if len(e.InIsServer) != 0 {
		isserver, err = strconv.ParseBool(e.InIsServer)
	} else if e.parent != nil && len(e.parent.InIsServer) != 0 {
		isserver, err = strconv.ParseBool(e.parent.InIsServer)
	}

	if err != nil {
		isserver = false
	}
	return
}

// Address gets Address of the network, or the global address, or empty string.
func (e *Ext) Address() (address string) {
	e.protect.RLock()
	defer e.protect.RUnlock()

	if len(e.InAddress) > 0 {
		address = e.InAddress
	} else if e.parent != nil && len(e.parent.InAddress) > 0 {
		address = e.parent.InAddress
	}
	return
}

// Ssl returns Ssl of the network, or the global ssl, or false
func (e *Ext) Ssl() (ssl bool) {
	e.protect.RLock()
	defer e.protect.RUnlock()

	var err error
	if len(e.InSsl) != 0 {
		ssl, err = strconv.ParseBool(e.InSsl)
	} else if e.parent != nil && len(e.parent.InSsl) != 0 {
		ssl, err = strconv.ParseBool(e.parent.InSsl)
	}

	if err != nil {
		ssl = false
	}
	return
}

// SslClientCert returns the path to the clientClientCertificate used when
// connecting.
func (e *Ext) SslClientCert() (clientClientCert string) {
	e.protect.RLock()
	defer e.protect.RUnlock()

	if len(e.InSslClientCert) > 0 {
		clientClientCert = e.InSslClientCert
	} else if e.parent != nil && len(e.parent.InSslClientCert) > 0 {
		clientClientCert = e.parent.InSslClientCert
	}
	return
}

// NoVerifyCert gets NoVerifyCert of the network, or the global verifyCert, or
// false
func (e *Ext) NoVerifyCert() (noverifyCert bool) {
	e.protect.RLock()
	defer e.protect.RUnlock()

	var err error
	if len(e.InNoVerifyCert) != 0 {
		noverifyCert, err = strconv.ParseBool(e.InNoVerifyCert)
	} else if e.parent != nil && len(e.parent.InNoVerifyCert) != 0 {
		noverifyCert, err = strconv.ParseBool(e.parent.InNoVerifyCert)
	}

	if err != nil {
		noverifyCert = false
	}
	return
}

// Sock gets Sock of the network, or the global sock, or empty string.
func (e *Ext) Sock() (sock string) {
	e.protect.RLock()
	defer e.protect.RUnlock()

	if len(e.InSock) > 0 {
		sock = e.InSock
	} else if e.parent != nil && len(e.parent.InSock) > 0 {
		sock = e.parent.InSock
	}
	return
}

// NoReconnect gets NoReconnect of the network, or the global noReconnect, or
// false
func (e *Ext) NoReconnect() (noReconnect bool) {
	e.protect.RLock()
	defer e.protect.RUnlock()

	var err error
	if len(e.InNoReconnect) != 0 {
		noReconnect, err = strconv.ParseBool(e.InNoReconnect)
	} else if e.parent != nil && len(e.parent.InNoReconnect) != 0 {
		noReconnect, err = strconv.ParseBool(e.parent.InNoReconnect)
	}

	if err != nil {
		noReconnect = false
	}
	return
}

// ReconnectTimeout gets ReconnectTimeout of the network, or the global
// reconnectTimeout, or defaultReconnectTimeout
func (e *Ext) ReconnectTimeout() (reconnTimeout uint) {
	e.protect.RLock()
	defer e.protect.RUnlock()

	var notset bool
	var err error
	var u uint64
	reconnTimeout = defaultReconnectTimeout
	if len(e.InReconnectTimeout) != 0 {
		u, err = strconv.ParseUint(e.InReconnectTimeout, 10, 32)
	} else if e.parent != nil && len(e.parent.InReconnectTimeout) != 0 {
		u, err = strconv.ParseUint(e.parent.InReconnectTimeout, 10, 32)
	} else {
		notset = true
	}

	if err != nil {
		reconnTimeout = defaultReconnectTimeout
	} else if !notset {
		reconnTimeout = uint(u)
	}
	return
}
