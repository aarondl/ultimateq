package config

import (
	"strconv"
	"sync"
)

// Network states the all the details necessary to connect to an irc network
// Although all of these are exported so they can be deserialized into a yaml
// file. The fields are all marked with "In" as in "Internal" and should not
// be accessed directly, but through their appropriately named helper methods.
type Network struct {
	parent  *Config       `yaml:"-" json:"-"`
	protect *sync.RWMutex `yaml:"-" json:"-"`

	// Name of this network
	InName string `yaml:"-" json:"-"`

	// Irc Network connection info
	InServers []string `yaml:"servers" json:"servers"`
	InPort    uint16   `yaml:"port" json:"port"`

	// Ssl configuration
	InSsl          string `yaml:"ssl" json:"ssl"`
	InSslCert      string `yaml:"sslcert" json:"sslcert"`
	InNoVerifyCert string `yaml:"noverifycert" json:"noverifycert"`

	// State tracking
	InNoState string `yaml:"nostate" json:"nostate"`
	InNoStore string `yaml:"nostore" json:"nostore"`

	// Flood Protection
	InFloodLenPenalty string `yaml:"floodlenpenalty" json:"floodlenpenalty"`
	InFloodTimeout    string `yaml:"floodtimeout" json:"floodtimeout"`
	InFloodStep       string `yaml:"floodstep" json:"floodstep"`

	// Keep alive
	InKeepAlive string `yaml:"keepalive" json:"keepalive"`

	// Auto reconnectiong
	InNoReconnect      string `yaml:"noreconnect" json:"noreconnect"`
	InReconnectTimeout string `yaml:"reconnecttimeout" json:"reconnecttimeout"`

	// Irc User data
	InNick     string `yaml:"nick" json:"nick"`
	InAltnick  string `yaml:"altnick" json:"altnick"`
	InUsername string `yaml:"username" json:"username"`
	InUserhost string `yaml:"userhost" json:"userhost"`
	InRealname string `yaml:"realname" json:"realname"`
	InPassword string `yaml:"password" json:"password"`

	// Dispatching options
	InPrefix   string   `yaml:"prefix" json:"prefix"`
	InChannels []string `yaml:"channels" json:"channels"`
}

// Clone clones a network.
func (n *Network) Clone() *Network {
	n.protect.RLock()
	defer n.protect.RUnlock()

	newNet := *n
	newNet.InChannels = make([]string, len(n.InChannels))
	newNet.InServers = make([]string, len(n.InServers))
	copy(newNet.InChannels, n.InChannels)
	copy(newNet.InServers, n.InServers)

	return &newNet
}

// Name gets the network's name.
func (n *Network) Name() string {
	n.protect.RLock()
	defer n.protect.RUnlock()

	return n.InName
}

// Servers gets the network's server list.
func (n *Network) Servers() []string {
	n.protect.RLock()
	defer n.protect.RUnlock()

	if len(n.InServers) == 0 {
		return []string{n.InName}
	}

	servers := make([]string, len(n.InServers))
	copy(servers, n.InServers)
	return servers
}

// Port returns gets Port of the network, or the global port, or
// ircDefaultPort
func (n *Network) Port() (port uint16) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	port = defaultIrcPort
	if n.InPort != 0 {
		port = n.InPort
	} else if n.parent != nil && n.parent.InPort != 0 {
		port = n.parent.InPort
	}
	return
}

// Ssl returns Ssl of the network, or the global ssl, or false
func (n *Network) Ssl() (ssl bool) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	var err error
	if len(n.InSsl) != 0 {
		ssl, err = strconv.ParseBool(n.InSsl)
	} else if n.parent != nil && len(n.parent.InSsl) != 0 {
		ssl, err = strconv.ParseBool(n.parent.InSsl)
	}

	if err != nil {
		ssl = false
	}
	return
}

// SslCert returns the path to the certificate used when connecting.
func (n *Network) SslCert() (cert string) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	if len(n.InSslCert) > 0 {
		cert = n.InSslCert
	} else if n.parent != nil && len(n.parent.InSslCert) > 0 {
		cert = n.parent.InSslCert
	}
	return
}

// NoVerifyCert gets NoVerifyCert of the network, or the global verifyCert, or
// false
func (n *Network) NoVerifyCert() (noverifyCert bool) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	var err error
	if len(n.InNoVerifyCert) != 0 {
		noverifyCert, err = strconv.ParseBool(n.InNoVerifyCert)
	} else if n.parent != nil && len(n.parent.InNoVerifyCert) != 0 {
		noverifyCert, err = strconv.ParseBool(n.parent.InNoVerifyCert)
	}

	if err != nil {
		noverifyCert = false
	}
	return
}

// NoState gets NoState of the network, or the global nostate, or
// false
func (n *Network) NoState() (nostate bool) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	var err error
	if len(n.InNoState) != 0 {
		nostate, err = strconv.ParseBool(n.InNoState)
	} else if n.parent != nil && len(n.parent.InNoState) != 0 {
		nostate, err = strconv.ParseBool(n.parent.InNoState)
	}

	if err != nil {
		nostate = false
	}
	return
}

// NoStore gets NoStore of the network, or the global nostore, or
// false
func (n *Network) NoStore() (nostore bool) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	var err error
	if len(n.InNoStore) != 0 {
		nostore, err = strconv.ParseBool(n.InNoStore)
	} else if n.parent != nil && len(n.parent.InNoStore) != 0 {
		nostore, err = strconv.ParseBool(n.parent.InNoStore)
	}

	if err != nil {
		nostore = false
	}
	return
}

// NoReconnect gets NoReconnect of the network, or the global noReconnect, or
// false
func (n *Network) NoReconnect() (noReconnect bool) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	var err error
	if len(n.InNoReconnect) != 0 {
		noReconnect, err = strconv.ParseBool(n.InNoReconnect)
	} else if n.parent != nil && len(n.parent.InNoReconnect) != 0 {
		noReconnect, err = strconv.ParseBool(n.parent.InNoReconnect)
	}

	if err != nil {
		noReconnect = false
	}
	return
}

// FloodLenPenalty gets FloodLenPenalty of the network, or the global
// floodLenPenalty, or defaultFloodLenPenalty
func (n *Network) FloodLenPenalty() (floodLenPenalty uint) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	var err error
	var u uint64
	var notset bool
	floodLenPenalty = defaultFloodLenPenalty
	if len(n.InFloodLenPenalty) != 0 {
		u, err = strconv.ParseUint(n.InFloodLenPenalty, 10, 32)
	} else if n.parent != nil && len(n.parent.InFloodLenPenalty) != 0 {
		u, err = strconv.ParseUint(n.parent.InFloodLenPenalty, 10, 32)
	} else {
		notset = true
	}

	if err != nil {
		floodLenPenalty = defaultFloodLenPenalty
	} else if !notset {
		floodLenPenalty = uint(u)
	}
	return
}

// FloodTimeout gets FloodTimeout of the network, or the global
// floodTimeout, or defaultFloodTimeout
func (n *Network) FloodTimeout() (floodTimeout float64) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	var err error
	floodTimeout = defaultFloodTimeout
	if len(n.InFloodTimeout) != 0 {
		floodTimeout, err = strconv.ParseFloat(n.InFloodTimeout, 32)
	} else if n.parent != nil && len(n.parent.InFloodTimeout) != 0 {
		floodTimeout, err = strconv.ParseFloat(n.parent.InFloodTimeout, 32)
	}

	if err != nil {
		floodTimeout = defaultFloodTimeout
	}
	return
}

// FloodStep gets FloodStep of the network, or the global
// floodStep, or defaultFloodStep
func (n *Network) FloodStep() (floodStep float64) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	var err error
	floodStep = defaultFloodStep
	if len(n.InFloodStep) != 0 {
		floodStep, err = strconv.ParseFloat(n.InFloodStep, 32)
	} else if n.parent != nil && len(n.parent.InFloodStep) != 0 {
		floodStep, err = strconv.ParseFloat(n.parent.InFloodStep, 32)
	}

	if err != nil {
		floodStep = defaultFloodStep
	}
	return
}

// KeepAlive gets KeepAlive of the network, or the global keepAlive,
// or defaultKeepAlive.
func (n *Network) KeepAlive() (keepAlive float64) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	var err error
	keepAlive = defaultKeepAlive
	if len(n.InKeepAlive) != 0 {
		keepAlive, err = strconv.ParseFloat(n.InKeepAlive, 32)
	} else if n.parent != nil && len(n.parent.InKeepAlive) != 0 {
		keepAlive, err = strconv.ParseFloat(n.parent.InKeepAlive, 32)
	}

	if err != nil {
		keepAlive = defaultKeepAlive
	}
	return
}

// ReconnectTimeout gets ReconnectTimeout of the network, or the global
// reconnectTimeout, or defaultReconnectTimeout
func (n *Network) ReconnectTimeout() (reconnTimeout uint) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	var notset bool
	var err error
	var u uint64
	reconnTimeout = defaultReconnectTimeout
	if len(n.InReconnectTimeout) != 0 {
		u, err = strconv.ParseUint(n.InReconnectTimeout, 10, 32)
	} else if n.parent != nil && len(n.parent.InReconnectTimeout) != 0 {
		u, err = strconv.ParseUint(n.parent.InReconnectTimeout, 10, 32)
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

// Nick gets Nick of the network, or the global nick, or empty string.
func (n *Network) Nick() (nick string) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	if len(n.InNick) > 0 {
		nick = n.InNick
	} else if n.parent != nil && len(n.parent.InNick) > 0 {
		nick = n.parent.InNick
	}
	return
}

// Altnick gets Altnick of the network, or the global nick, or empty string.
func (n *Network) Altnick() (altnick string) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	if len(n.InAltnick) > 0 {
		altnick = n.InAltnick
	} else if n.parent != nil && len(n.parent.InAltnick) > 0 {
		altnick = n.parent.InAltnick
	}
	return
}

// Username gets Username of the network, or the global username, or empty
// string.
func (n *Network) Username() (username string) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	if len(n.InUsername) > 0 {
		username = n.InUsername
	} else if n.parent != nil && len(n.parent.InUsername) > 0 {
		username = n.parent.InUsername
	}
	return
}

// Userhost gets Userhost of the network, or the global userhost, or empty
// string.
func (n *Network) Userhost() (userhost string) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	if len(n.InUserhost) > 0 {
		userhost = n.InUserhost
	} else if n.parent != nil && len(n.parent.InUserhost) > 0 {
		userhost = n.parent.InUserhost
	}
	return
}

// Realname gets Realname of the network, or the global realname, or empty
// string.
func (n *Network) Realname() (realname string) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	if len(n.InRealname) > 0 {
		realname = n.InRealname
	} else if n.parent != nil && len(n.parent.InRealname) > 0 {
		realname = n.parent.InRealname
	}
	return
}

// Password gets password of the network, or the global password,
// or empty string.
func (n *Network) Password() (password string) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	if len(n.InPassword) > 0 {
		password = n.InPassword
	} else if n.parent != nil && len(n.parent.InPassword) > 0 {
		password = n.parent.InPassword
	}
	return
}

// Prefix gets Prefix of the network, or the global prefix, or defaultPrefix.
func (n *Network) Prefix() (prefix rune) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	prefix = defaultPrefix
	if len(n.InPrefix) > 0 {
		prefix = rune(n.InPrefix[0])
	} else if n.parent != nil && len(n.parent.InPrefix) > 0 {
		prefix = rune(n.parent.InPrefix[0])
	}
	return
}

// Channels gets Channels of the network, or the global channels, or nil
// slice of string (check the length!).
func (n *Network) Channels() (channels []string) {
	n.protect.RLock()
	defer n.protect.RUnlock()

	if len(n.InChannels) > 0 {
		channels = make([]string, len(n.InChannels))
		copy(channels, n.InChannels)
	} else if n.parent != nil && len(n.parent.InChannels) > 0 {
		channels = make([]string, len(n.parent.InChannels))
		copy(channels, n.parent.InChannels)
	}
	return
}
