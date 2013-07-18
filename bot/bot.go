/*
Package bot implements the top-level package that any non-extension
will use to start a bot instance.
*/
package bot

import (
	"errors"
	"github.com/aarondl/ultimateq/config"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/dispatch"
	"github.com/aarondl/ultimateq/dispatch/commander"
	"github.com/aarondl/ultimateq/irc"
	"github.com/aarondl/ultimateq/parse"
	"log"
	"net"
	"sync"
	"time"
)

const (
	// nAssumedServers is how many servers a bot typically connects to.
	nAssumedServers = 1
	// defaultReconnScale is how the config's ReconnTimeout is scaled.
	defaultReconnScale = time.Second

	// errFmtParsingIrcMessage is when the bot fails to parse a message
	// during it's dispatch loop.
	errFmtParsingIrcMessage = "bot: Failed to parse irc message (%v) (%s)\n"
	// errFmtReaderClosed is when a write fails due to a closed socket or
	// a shutdown on the client.
	errFmtReaderClosed = "bot: %v reader closed\n"
	// errFmtClosingServer is when a IrcClient.Close returns an error.
	errFmtClosingServer = "bot: Error closing server (%v)\n"
	// fmtFailedConnecting shows when the bot is unable to connect to a server.
	fmtFailedConnecting = "bot: %v failed to connect (%v)"
	// fmtDisconnected shows when the bot is disconnected
	fmtDisconnected = "bot: %v disconnected"
	// fmtReconnecting shows when the bot is reconnecting
	fmtReconnecting = "bot: %v reconnecting in %v..."
)

var (
	// errInvalidConfig is when CreateBot was given an invalid configuration.
	errInvalidConfig = errors.New("bot: Invalid Configuration")
	// errInvalidServerId occurs when the user passes in an unknown
	// server id to a method requiring a server id.
	errUnknownServerID = errors.New("bot: Unknown Server id.")
	// errServerKilled occurs when the server is killed during the running state
	errServerKilled = errors.New("bot: Server killed.")
	// errServerKilledReconn occurs when the server is killed during a
	// reconnection pause.
	errServerKilledReconn = errors.New("bot: Server killed.")

	// connMessage is a pseudo message sent to servers upon connect.
	connMessage = &irc.Message{Name: irc.CONNECT}
	// discMessage is a pseudo message sent to servers upon disconnect.
	discMessage = &irc.Message{Name: irc.DISCONNECT}
)

type (
	// CapsProvider returns a usable ProtoCaps to start the bot with
	CapsProvider func() *irc.ProtoCaps
	// ConnProvider transforms a "server:port" string into a net.Conn
	ConnProvider func(string) (net.Conn, error)
	// StoreProvider transforms an optional path into a store.
	StoreProvider func(string) (*data.Store, error)
)

// Bot is a main type that joins together all the packages into a functioning
// irc bot. It should be able to carry out most major functions that a bot would
// need through it's exported functions.
type Bot struct {
	servers map[string]*Server
	conf    *config.Config
	store   *data.Store

	// Bot-server synchronization.
	botEnd chan error
	serverStart chan int
	serverEnd chan error

	// Dispatching
	caps         *irc.ProtoCaps
	dispatchCore *dispatch.DispatchCore
	dispatcher   *dispatch.Dispatcher
	commander    *commander.Commander
	coreCommands *coreCommands

	// IoC and DI components mostly for testing.
	attachHandlers bool
	capsProvider   CapsProvider
	connProvider   ConnProvider
	storeProvider  StoreProvider

	// Synchronization
	msgDispatchers sync.WaitGroup
	protectStore   sync.RWMutex
	protectServers sync.RWMutex
	// protectConfig also provides locking for the server's config variables
	// since they are the same config, just pointers to internal chunks.
	protectConfig sync.RWMutex
}

// Configure starts a configuration by calling CreateConfig. Alias for
// config.CreateConfig
func Configure() *config.Config {
	return config.CreateConfig()
}

// ConfigureFile starts a configuration by reading in a file. Alias for
// config.CreateConfigFromFile
func ConfigureFile(filename string) *config.Config {
	return config.CreateConfigFromFile(filename)
}

// ConfigureFunction creates a blank configuration and passes it into a function
func ConfigureFunction(cnf func(*config.Config) *config.Config) *config.Config {
	return cnf(config.CreateConfig())
}

// CheckConfig checks a bots config for validity.
func CheckConfig(c *config.Config) bool {
	if !c.IsValid() {
		c.DisplayErrors()
		return false
	}
	return true
}

// CreateBot simplifies the call to createBotFull by using default
// caps and conn provider functions.
func CreateBot(conf *config.Config) (*Bot, error) {
	if !CheckConfig(conf) {
		return nil, errInvalidConfig
	}
	return createBot(conf, nil, nil, nil, true, true)
}

// Start runs the bot. A channel is returned, every time a server is killed
// permanently it reports the error on this channel. When the channel is closed,
// there are no more servers left to run and the program can safely exit.
func (b *Bot) Start() <-chan error {
	b.protectServers.RLock()
	for _, srv := range b.servers {
		go b.startServer(srv, true, true)
	}
	b.protectServers.RUnlock()

	go b.monitorServers()

	return b.botEnd
}

// monitorServers watches the starting and stopping of servers. It sends any
// permanent server deaths to the botEnd channel, and if all servers
// are stopped it closes the botEnd channel.
func (b *Bot) monitorServers() {
	servers := 0
	for {
		var err error
		select {
		case <-b.serverStart:
			servers++
		case err = <-b.serverEnd:
			b.botEnd <- err
			servers--
		}

		if servers == 0 {
			close(b.botEnd)
		}
	}
}

// startServer starts up a server. When it has finished (permanently
// disconnected) it will send it's disconnection error to serverEnd.
func (b *Bot) startServer(srv *Server, writing, reading bool) {
	var err error
	var disconnect bool

	b.serverStart <- 0
	for err == nil {
		srv.setConnecting(true, true)
		err = srv.createIrcClient()
		if err != nil {
			srv.setConnecting(false, true)
			break
		}

		srv.protectState.Lock()
		srv.setConnecting(false, false)
		srv.setConnected(true, false)
		srv.setStarted(true, false)
		srv.protectState.Unlock()

		srv.client.SpawnWorkers(writing, reading)
		disconnect, err = b.dispatch(srv)
		if err != nil {
			break
		}

		srv.client.Close()
		srv.client = nil
		srv.setStarted(false, true)
		srv.setConnected(false, true)

		b.protectConfig.RLock()
		if !disconnect || srv.conf.GetNoReconnect() {
			b.protectConfig.RUnlock()
			break
		}

		wait := time.Duration(srv.conf.GetReconnectTimeout()) * srv.reconnScale
		b.protectConfig.RUnlock()

		srv.setReconnecting(true, true)
		select {
		case <-srv.kill:
			err = errServerKilledReconn
		case <-time.After(wait):
		}
		srv.setReconnecting(false, true)
	}

	b.serverEnd <- err
}

// dispatch starts dispatch loops on the server.
func (b *Bot) dispatch(srv *Server) (disconnect bool, err error) {
	var ircMsg *irc.Message
	var parseErr error
	readCh := srv.client.ReadChannel()

	b.dispatchMessage(srv, connMessage)
	for err == nil && !disconnect {
		select {
		case msg, ok := <-readCh:
			if !ok {
				disconnect = true
				break
			}
			ircMsg, parseErr = parse.Parse(msg)
			if parseErr != nil {
				log.Printf(errFmtParsingIrcMessage, parseErr, msg)
				break
			}
			b.dispatchMessage(srv, ircMsg)
		case <-srv.kill:
			err = errServerKilled
			break
		}
	}

	b.dispatchMessage(srv, discMessage)
	return
}

// dispatch sends a message to both the bot's dispatcher and the given servers
func (b *Bot) dispatchMessage(s *Server, msg *irc.Message) {
	b.dispatcher.Dispatch(msg, s.endpoint)
	s.dispatcher.Dispatch(msg, s.endpoint)
	b.commander.Dispatch(s.name, msg, s.endpoint.DataEndpoint)
	s.commander.Dispatch(s.name, msg, s.endpoint.DataEndpoint)
}

// Close closes the store database.
func (b *Bot) Close() error {
	b.protectStore.Lock()
	defer b.protectStore.Unlock()
	if b.store != nil {
		err := b.store.Close()
		b.store = nil
		return err
	}
	return nil
}

// Register adds an event handler to the bot's global dispatcher.
func (b *Bot) Register(event string, handler interface{}) int {
	return b.dispatcher.Register(event, handler)
}

// RegisterServer adds an event handler to a server specific dispatcher.
func (b *Bot) RegisterServer(
	server string, event string, handler interface{}) (int, error) {

	b.protectServers.RLock()
	defer b.protectServers.RUnlock()

	if s, ok := b.servers[server]; ok {
		return s.dispatcher.Register(event, handler), nil
	}
	return 0, errUnknownServerID
}

// Unregister removes an event handler from the bot's global dispatcher
func (b *Bot) Unregister(event string, id int) bool {
	return b.dispatcher.Unregister(event, id)
}

// UnregisterServer removes an event handler from a server specific dispatcher.
func (b *Bot) UnregisterServer(
	server string, event string, id int) (bool, error) {

	b.protectServers.RLock()
	defer b.protectServers.RUnlock()

	if s, ok := b.servers[server]; ok {
		return s.dispatcher.Unregister(event, id), nil
	}
	return false, errUnknownServerID
}

// RegisterCommand registers a command with the bot.
// See Commander.Register for in-depth documentation.
func (b *Bot) RegisterCommand(cmd *commander.Command) error {
	return b.commander.Register(commander.GLOBAL, cmd)
}

// RegisterServerCommand registers a command with the server.
// See Commander.Register for in-depth documentation.
func (b *Bot) RegisterServerCommand(srv string, cmd *commander.Command) error {

	b.protectServers.RLock()
	defer b.protectServers.RUnlock()

	if s, ok := b.servers[srv]; ok {
		return s.commander.Register(srv, cmd)
	}
	return errUnknownServerID
}

// UnregisterCommand unregister's a command from the bot.
func (b *Bot) UnregisterCommand(cmd string) bool {
	return b.commander.Unregister(commander.GLOBAL, cmd)
}

// UnregisterServerCommand unregister's a command from the server.
func (b *Bot) UnregisterServerCommand(server, cmd string) bool {
	b.protectServers.RLock()
	defer b.protectServers.RUnlock()

	if s, ok := b.servers[server]; ok {
		return s.commander.Unregister(server, cmd)
	}
	return false
}

// Writeln writes a string to the given server's IrcClient.
func (b *Bot) Writeln(server, message string) error {
	b.protectServers.RLock()
	defer b.protectServers.RUnlock()

	srv, ok := b.servers[server]
	if !ok {
		return errUnknownServerID
	}
	return srv.Writeln(message)
}

// createBot creates a bot from the given configuration, using the providers
// given to create connections and protocol caps.
func createBot(conf *config.Config, capsProv CapsProvider,
	connProv ConnProvider, storeProv StoreProvider,
	attachHandlers, attachCommands bool) (*Bot, error) {

	b := &Bot{
		conf:           conf,
		servers:        make(map[string]*Server, nAssumedServers),
		capsProvider:   capsProv,
		connProvider:   connProv,
		storeProvider:  storeProv,
		attachHandlers: attachHandlers,
		botEnd: make(chan error),
		serverStart: make(chan int),
		serverEnd: make(chan error),
	}

	if capsProv == nil {
		b.caps = irc.CreateProtoCaps()
	} else {
		b.caps = capsProv()
	}

	var err error
	if err = b.createDispatching(
		conf.Global.GetPrefix(), conf.Global.GetChannels()); err != nil {

		return nil, err
	}

	makeStore := false
	for _, srv := range conf.Servers {
		makeStore = makeStore || !srv.GetNoStore()
	}

	if makeStore {
		if err = b.createStore(conf.GetStoreFile()); err != nil {
			return nil, err
		}
	}

	for name, srv := range conf.Servers {
		server, err := b.createServer(srv)
		if err != nil {
			return nil, err
		}
		b.servers[name] = server
	}

	if attachCommands && !conf.Global.GetNoStore() {
		b.coreCommands, err = CreateCoreCommands(b)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

// createServer creates a dispatcher, and an irc client to connect to this
// server.
func (b *Bot) createServer(conf *config.Server) (*Server, error) {
	s := &Server{
		bot:          b,
		name:         conf.GetName(),
		caps:         b.caps.Clone(),
		conf:         conf,
		kill:         make(chan int),
		killdispatch: make(chan int),
		killreconn:   make(chan int),
		reconnScale:  defaultReconnScale,
	}

	if err := s.createDispatching(
		conf.GetPrefix(), conf.GetChannels()); err != nil {

		return nil, err
	}

	if !conf.GetNoState() {
		if err := s.createState(); err != nil {
			return nil, err
		}
	}

	s.createServerEndpoint(b.store, &b.protectStore)

	if b.attachHandlers {
		s.handler = &coreHandler{bot: b}
		s.handlerID = s.dispatcher.Register(irc.RAW, s.handler)
	}

	return s, nil
}

// createDispatcher uses the bot's current ProtoCaps to create a dispatcher.
func (b *Bot) createDispatching(prefix rune, channels []string) error {
	var err error
	b.dispatchCore, err = dispatch.CreateDispatchCore(b.caps, channels...)
	if err != nil {
		return err
	}
	b.dispatcher = dispatch.CreateDispatcher(b.dispatchCore)
	b.commander = commander.CreateCommander(prefix, b.dispatchCore)
	return nil
}

// createStore creates a store from a filename.
func (b *Bot) createStore(filename string) (err error) {
	if b.storeProvider == nil {
		b.store, err = data.CreateStore(data.MakeFileStoreProvider(filename))
	} else {
		b.store, err = b.storeProvider(filename)
	}
	return
}

// mergeProtocaps merges a protocaps with the bot's current protocaps to ensure
// the bot's main dispatcher still recognizes all channel types that the servers
// currently recognize.
func (b *Bot) mergeProtocaps(toMerge *irc.ProtoCaps) (err error) {
	b.caps.Merge(toMerge)
	err = b.dispatchCore.Protocaps(b.caps)
	return
}
