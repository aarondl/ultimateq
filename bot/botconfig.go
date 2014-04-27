package bot

import (
	"fmt"

	"github.com/aarondl/ultimateq/config"
	"github.com/aarondl/ultimateq/irc"
)

const (
	// errFmtNewServer occurs when a server that was created by RehashConfig
	// fails to initialize for some reason.
	errFmtNewServer = "bot: The new server (%v) could not be instantiated: %v"
)

type configCallback func(*config.Config)

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
// new servers will be connected to and started. Updates updateable attributes
// from the new configuration for each server. Returns false if the config
// had an error.
func (b *Bot) ReplaceConfig(newConfig *config.Config) bool {
	if !newConfig.IsValid() {
		return false
	}

	b.protectServers.Lock()
	b.protectConfig.Lock()
	defer b.protectServers.Unlock() // LIFO
	defer b.protectConfig.Unlock()

	b.startNewServers(newConfig)

	for k, s := range b.servers {
		if serverConf := newConfig.GetServer(k); nil == serverConf {
			b.stopServer(s)
			delete(b.servers, k)
			continue
		} else {
			s.rehashConfig(serverConf)
		}
	}

	if !contains(b.conf.Global.GetChannels(), newConfig.Global.GetChannels()) {
		b.dispatchCore.SetChannels(newConfig.Global.GetChannels())
	}

	b.conf = newConfig
	return true
}

// startNewServers adds non-existing servers to the bot and starts them.
func (b *Bot) startNewServers(newConfig *config.Config) {
	for k, s := range newConfig.Servers {
		if serverConf := b.conf.GetServer(k); nil == serverConf {
			server, err := b.createServer(s)
			if err != nil {
				b.botEnd <- fmt.Errorf(errFmtNewServer, k, err)
				continue
			}
			b.servers[k] = server

			go b.startServer(server, true, true)
		}
	}
}

// rehashConfig updates the server's config values from the new configuration.
func (s *Server) rehashConfig(srvConfig *config.Server) {
	setNick := s.conf.GetNick() != srvConfig.GetNick()
	setChannels := !contains(s.conf.GetChannels(), srvConfig.GetChannels())

	s.conf = srvConfig

	if setNick {
		s.Write([]byte(irc.NICK + " :" + s.conf.GetNick()))
	}
	if setChannels {
		s.dispatchCore.SetChannels(s.conf.GetChannels())
	}
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

// contains checks that the string arrays contain the same elements.
func contains(a, b []string) bool {
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
