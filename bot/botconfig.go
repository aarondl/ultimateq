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
	fn(b.conf)
}

// WriteConfig opens the config for writing, for the duration of the callback
// the config is synchronized.
func (b *Bot) WriteConfig(fn configCallback) {
	fn(b.conf)
}

// ReplaceConfig replaces the current configuration for the bot. Running
// servers not present in the new config will be shut down immediately, while
// new servers will be connected to and started. Updates updateable attributes
// from the new configuration for each server. Returns false if the config
// had an error.
func (b *Bot) ReplaceConfig(newConfig *config.Config) bool {
	if !newConfig.Validate() {
		return false
	}

	b.protectServers.Lock()
	defer b.protectServers.Unlock() // LIFO

	b.startNewServers(newConfig)

	for netID, s := range b.servers {
		if serverConf := newConfig.Network(netID); nil == serverConf {
			b.stopServer(s)
			delete(b.servers, netID)
			continue
		} else {
			s.rehashConfig(newConfig)
		}
	}

	b.conf.Replace(newConfig)
	return true
}

// startNewServers adds non-existing servers to the bot and starts them.
func (b *Bot) startNewServers(newConfig *config.Config) {
	for _, net := range newConfig.Networks() {
		if serverConf := b.conf.Network(net); nil == serverConf {
			server, err := b.createServer(net, newConfig)
			if err != nil {
				b.botEnd <- fmt.Errorf(errFmtNewServer, net, err)
				continue
			}
			b.servers[net] = server

			go b.startServer(server, true, true)
		}
	}
}

// rehashConfig updates the server's config values from the new configuration.
func (s *Server) rehashConfig(newConfig *config.Config) {
	oldNick, _ := s.conf.Network(s.networkID).Nick()
	newNick, _ := newConfig.Network(s.networkID).Nick()
	setNick := newNick != oldNick

	if setNick {
		s.Write([]byte(irc.NICK + " :" + newNick))
	}
}

// Rehash loads the config from a file. It attempts to use the previously read
// config file name if loaded from a file... If not it will use a default file
// name. It then calls Bot.ReplaceConfig.
func (b *Bot) Rehash() error {
	fname, _ := b.conf.StoreFile()

	conf := config.New().FromFile(fname)
	if !CheckConfig(conf) {
		return errInvalidConfig
	}
	b.conf.Replace(conf)
	return nil
}

// DumpConfig dumps the config to a file. It attempts to use the previously read
// config file name if loaded from a file... If not it will use a default file
// name.
func (b *Bot) DumpConfig() error {
	return b.conf.ToFile("")
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
