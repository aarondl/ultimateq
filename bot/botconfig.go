package bot

import (
	"github.com/aarondl/ultimateq/config"
)

type configCallback func(*config.Config)

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

// ReplaceConfig replaces the current configuration for the bot, it produces
// a diff that is then used to alter the running state of the bot.
func (b *Bot) ReplaceConfig(newConfig *config.Config) {
	b.configsProtect.Lock()
	defer b.configsProtect.Unlock()
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
