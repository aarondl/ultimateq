package extremote

import (
	"github.com/aarondl/ultimateq/api"
	"google.golang.org/grpc"
)

type Config struct {
	Name string

	BotAddress       string
	BoomerangAddress string
}

type Extension struct {
	cfg Config

	kill chan struct{}

	boomerangError chan error
}

// New creates a new remote extension
func New(config Config) *Extension {
	return &Extension{
		cfg: config,
	}
}

// Start connects to the bot/and or starts the boomerang server
func (e *Extension) Start() (*Client, error) {
	e.kill = make(chan struct{})

	if len(e.cfg.BoomerangAddress) != 0 {
		e.startBoomerang()
	}

	if len(e.cfg.BotAddress) != 0 {
		conn, err := grpc.Dial(e.cfg.BotAddress)
		if err != nil {
			return nil, err
		}

		return &Client{
			client: api.NewExtClient(conn),
		}, nil
	}

	return nil, nil
}

func (e *Extension) startBoomerang() {
	e.boomerangError = make(chan error, 1)

	// Start boomerang server
}

// Stop disconnects from the bot/and or stops the boomerang server
func (e *Extension) Stop() {
	close(e.kill)
}
