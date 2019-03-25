package bot

import (
	"bufio"
	"os"
	"os/signal"
	"time"

	"github.com/aarondl/ultimateq/config"
)

// Run makes a very typical bot. It will call the cb function passed in
// before starting to allow registration of extensions etc. Returns error
// if the bot could not be created. Does NOT return until dead.
// The following are featured behaviors:
// Reads configuration file from ./config.toml
// Watches for Keyboard Input OR SIGTERM OR SIGKILL and shuts down normally.
// Pauses after death to allow all goroutines to come to a graceful shutdown.
func Run(cb func(b *Bot)) error {
	cfg := config.New().FromFile("config.toml")
	b, err := New(cfg)
	if err != nil {
		return err
	}
	defer b.Close()

	cb(b)

	end := b.Start()

	_, ok := cfg.ExtGlobal().Listen()
	if ok {
		api := NewAPIServer(b)
		go func() {
			err := api.Start()
			if err != nil {
				b.Logger.Error("failed to start apiserver", "err", err)
			}
		}()
	}

	input, quit := make(chan int), make(chan os.Signal, 2)

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		input <- 0
	}()

	signal.Notify(quit, os.Interrupt, os.Kill)

	stop := false
	for !stop {
		select {
		case <-input:
			b.Stop()
			stop = true
		case <-quit:
			b.Stop()
			stop = true
		case err, ok := <-end:
			if ok {
				b.Info("Server death", "err", err)
			}
			stop = !ok
		}
	}

	b.Cleanup()
	b.Info("Shutting down...")
	<-time.After(1 * time.Second)

	return nil
}
