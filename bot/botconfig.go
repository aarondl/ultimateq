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
	b.configsProtect.RLock()
	defer b.configsProtect.RUnlock()
	fn(b.conf)
}

// WriteConfig opens the config for writing, for the duration of the callback
// the config is synchronized.
func (b *Bot) WriteConfig(fn configCallback) {
	b.configsProtect.Lock()
	defer b.configsProtect.Unlock()
	fn(b.conf)
}

// ReplaceConfig replaces the current configuration for the bot. Running
// servers not present in the new config will be shut down immediately, while
// new servers will be created and returned, ready to start.
func (b *Bot) ReplaceConfig(newConfig *config.Config) []NewServer {
	if !newConfig.IsValid() {
		return nil
	}

	servers := make([]NewServer, 0)

	b.serversProtect.Lock()
	b.configsProtect.Lock()
	defer b.serversProtect.Unlock() // LIFO
	defer b.configsProtect.Unlock()

	for k, s := range b.servers {
		if serverConf := newConfig.GetServer(k); nil == serverConf {
			b.stopServer(s)
			b.disconnectServer(s)
			delete(b.servers, k)
		} else {
			setNick := s.conf.GetNick() != serverConf.GetNick()
			s.conf = serverConf

			if setNick {
				s.Writeln(irc.NICK + " :" + s.conf.GetNick())
			}
		}
	}

	for k, s := range newConfig.Servers {
		if serverConf := b.conf.GetServer(k); nil == serverConf {
			server, err := b.createServer(s)
			b.servers[k] = server
			servers = append(servers, NewServer{k, server, err})
		}
	}

	b.conf = newConfig

	return servers
}

// StartNewServers starts the servers in the regular way for the bot. If there
// is any error starting the bot it will write back to the NewServer array any
// errors.
func (b *Bot) StartNewServers(servers []NewServer) {
	for i := 0; i < len(servers); i++ {
		err := b.connectServer(servers[i].server)
		if err != nil {
			servers[i].Err = err
			continue
		}
		b.startServer(servers[i].server, true, true)
	}
}

// Rehash loads the config from a file. It attempts to use the previously read
// config file name if loaded from a file... If not it will use a default file
// name. It then calls Bot.ReplaceConfig.
func (b *Bot) Rehash() error {
	conf := config.CreateConfigFromFile(confName)
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
	b.configsProtect.RLock()
	defer b.configsProtect.RUnlock()
	err = config.FlushConfigToFile(b.conf, confName)
	return
}
