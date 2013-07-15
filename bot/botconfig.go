package bot

import (
	"github.com/aarondl/ultimateq/config"
	"github.com/aarondl/ultimateq/irc"
)

type configCallback func(*config.Config)

type NewServer struct {
	ServerName string
	server     *Server
	Err        error
}

// ReadConfig opens the config for reading, for the duration of the callback
// the config is synchronized.
func (b *Bot) ReadConfig(fn configCallback) {
	b.protectConfig.RLock()
	defer b.protectConfig.RUnlock()
	fn(b.conf)
}

// WriteConfig opens the config for writing, for the duration of the callback
// the config is synchronized.
func (b *Bot) WriteConfig(fn configCallback) {
	b.protectConfig.Lock()
	defer b.protectConfig.Unlock()
	fn(b.conf)
}

// ReplaceConfig replaces the current configuration for the bot. Running
// servers not present in the new config will be shut down immediately, while
// new servers will be connected to and started. Updates active channels
// for all dispatchers as well as sends nick messages to the servers with
// updates for nicknames. Returns any new servers added.
func (b *Bot) ReplaceConfig(newConfig *config.Config) []NewServer {
	if !newConfig.IsValid() {
		return nil
	}

	servers := make([]NewServer, 0)

	b.protectServers.Lock()
	b.protectConfig.Lock()
	defer b.protectServers.Unlock() // LIFO
	defer b.protectConfig.Unlock()

	for k, s := range b.servers {
		if serverConf := newConfig.GetServer(k); nil == serverConf {
			b.stopServer(s)
			b.disconnectServer(s)
			delete(b.servers, k)
		} else {
			setNick := s.conf.GetNick() != serverConf.GetNick()
			setChannels :=
				!elementsEquals(s.conf.GetChannels(), serverConf.GetChannels())

			if !s.conf.GetNoState() && serverConf.GetNoState() {
				s.protectState.Lock()
				s.state = nil
				s.protectState.Unlock()
			}

			s.conf = serverConf

			if setNick {
				s.Writeln(irc.NICK + " :" + s.conf.GetNick())
			}
			if setChannels {
				s.dispatcher.Channels(s.conf.GetChannels())
			}
		}
	}

	if !elementsEquals(b.conf.Global.GetChannels(),
		newConfig.Global.GetChannels()) {

		b.dispatcher.Channels(newConfig.Global.GetChannels())
	}

	for k, s := range newConfig.Servers {
		if serverConf := b.conf.GetServer(k); nil == serverConf {
			server, err := b.createServer(s)
			b.servers[k] = server

			if err == nil {
				err = b.connectServer(server)
				if err == nil {
					b.startServer(server, true, true)
				}
			}
			servers = append(servers, NewServer{k, server, err})
		}
	}

	b.conf = newConfig

	return servers
}

// Rehash loads the config from a file. It attempts to use the previously read
// config file name if loaded from a file... If not it will use a default file
// name. It then calls Bot.ReplaceConfig.
func (b *Bot) Rehash() error {
	b.protectConfig.RLock()
	name := b.conf.GetFilename()
	b.protectConfig.RUnlock()

	conf := config.CreateConfigFromFile(name)
	if !CheckConfig(conf) {
		return errInvalidConfig
	}
	b.ReplaceConfig(conf)
	return nil
}

// DumpConfig dumps the config to a file. It attempts to use the previously read
// config file name if loaded from a file... If not it will use a default file
// name.
func (b *Bot) DumpConfig() (err error) {
	b.protectConfig.RLock()
	defer b.protectConfig.RUnlock()
	err = config.FlushConfigToFile(b.conf, b.conf.GetFilename())
	return
}

// elementsEquals checks that the string arrays contain the same elements.
func elementsEquals(a, b []string) bool {
	lena, lenb := len(a), len(b)
	if lena != lenb {
		return false
	}

	for i := 0; i < lena; i++ {
		j := 0
		for ; j < lenb; j++ {
			if a[i] == b[j] {
				break
			}
		}
		if j >= lenb {
			return false
		}
	}

	return true
}
