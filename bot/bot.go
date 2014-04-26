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
	"github.com/aarondl/ultimateq/dispatch/cmd"
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
	errServerKilledReconn = errors.New("bot: Server reconnection aborted.")
)

type (
	// ConnProvider transforms a "server:port" string into a net.Conn
	ConnProvider func(string) (net.Conn, error)
	// StoreProvider transforms an optional path into a store.
	StoreProvider func(string) (*data.Store, error)
	// serverOp represents a server operation (starting/stopping).
	serverOp struct {
		server   *Server
		starting bool
		err      error
	}
)

// Bot is a main type that joins together all the packages into a functioning
// irc bot. It should be able to carry out most major functions that a bot would
// need through it's exported functions.
type Bot struct {
	servers map[string]*Server
	conf    *config.Config
	store   *data.Store

	// Bot-server synchronization.
	botEnd        chan error
	serverControl chan serverOp
	serverStart   chan bool
	serverStop    chan bool
	serverEnd     chan serverOp

	// Dispatching
	caps         *irc.ProtoCaps
	dispatchCore *dispatch.DispatchCore
	dispatcher   *dispatch.Dispatcher
	cmds         *cmd.Cmds
	coreCommands *coreCmds

	// IoC and DI components mostly for testing.
	attachHandlers bool
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
	return createBot(conf, nil, nil, true, true)
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
		select {
		case op := <-b.serverControl:
			isStarted := op.server.started
			if op.starting {
				op.server.killable = make(chan int)
				b.serverStart <- !isStarted
				if !isStarted {
					op.server.started = true
					servers++
				}
			} else {
				if isStarted {
					op.server.started = false
					_, isStarted = <-op.server.killable
				}
				b.serverStop <- isStarted
			}
		case op := <-b.serverEnd:
			op.server.started = false
			b.botEnd <- op.err
			servers--
		}

		if servers == 0 {
			close(b.botEnd)
			break
		}
	}
}

// StartServer starts a server by name. Start() should have been called prior
// to this.
func (b *Bot) StartServer(server string) (started bool) {
	b.protectServers.RLock()
	defer b.protectServers.RUnlock()

	var srv *Server
	if srv, started = b.servers[server]; started {
		go b.startServer(srv, true, true)
	}
	return
}

// startServer starts up a server. When it has finished (permanently
// disconnected) it will send it's disconnection error to serverEnd.
func (b *Bot) startServer(srv *Server, writing, reading bool) {
	var err error
	var disconnect bool

	b.serverControl <- serverOp{srv, true, nil}
	if !<-b.serverStart {
		return
	}

	for err == nil {
		srv.setStatus(STATUS_CONNECTING)
		err = srv.createIrcClient()
		disconnect = err != nil && err != errServerKilledConn

		if err == nil {
			srv.setStatus(STATUS_STARTED)

			srv.client.SpawnWorkers(writing, reading)
			disconnect, err = b.dispatch(srv)
			if err != nil {
				break
			}
		}

		b.protectConfig.RLock()
		if !disconnect || srv.conf.GetNoReconnect() {
			b.protectConfig.RUnlock()
			break
		}

		err = nil
		srv.Close()
		wait := time.Duration(srv.conf.GetReconnectTimeout()) * srv.reconnScale
		b.protectConfig.RUnlock()

		srv.setStatus(STATUS_RECONNECTING)
		select {
		case srv.killable <- 0:
			err = errServerKilledReconn
		case <-time.After(wait):
		}
	}

	srv.Close()
	close(srv.killable)
	b.serverEnd <- serverOp{srv, false, err}
	srv.setStatus(STATUS_STOPPED)
}

// dispatch starts dispatch loops on the server.
func (b *Bot) dispatch(srv *Server) (disconnect bool, err error) {
	var ircMsg *irc.Message
	var parseErr error
	readCh := srv.client.ReadChannel()

	b.dispatchMessage(srv, irc.NewMessage(irc.CONNECT, srv.name))
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
			srv.protectState.Lock()
			if srv.state != nil {
				srv.state.Update(ircMsg)
			}
			srv.protectState.Unlock()
			b.dispatchMessage(srv, ircMsg)
		case srv.killable <- 0:
			err = errServerKilled
			break
		}
	}

	b.dispatchMessage(srv, irc.NewMessage(irc.DISCONNECT, srv.name))
	return
}

// dispatch sends a message to both the bot's dispatcher and the given servers
func (b *Bot) dispatchMessage(s *Server, msg *irc.Message) {
	b.dispatcher.Dispatch(msg, s.endpoint)
	s.dispatcher.Dispatch(msg, s.endpoint)
	b.cmds.Dispatch(s.name, s.cmds.GetPrefix(), msg,
		s.endpoint.DataEndpoint)
	s.cmds.Dispatch(s.name, 0, msg, s.endpoint.DataEndpoint)
}

// Stop shuts down all connections and exits.
func (b *Bot) Stop() {
	b.protectServers.RLock()
	defer b.protectServers.RUnlock()

	for _, srv := range b.servers {
		b.stopServer(srv)
	}
}

// StopServer stops a server by name.
func (b *Bot) StopServer(server string) (stopped bool) {
	b.protectServers.RLock()
	defer b.protectServers.RUnlock()

	if srv, ok := b.servers[server]; ok {
		stopped = b.stopServer(srv)
	}
	return
}

// stopServer stops the current server if it's running.
func (b *Bot) stopServer(srv *Server) (stopped bool) {
	b.serverControl <- serverOp{srv, false, nil}
	return <-b.serverStop
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

// RegisterCmd registers a command with the bot.
// See Cmder.Register for in-depth documentation.
func (b *Bot) RegisterCmd(command *cmd.Cmd) error {
	return b.cmds.Register(cmd.GLOBAL, command)
}

// RegisterServerCmd registers a command with the server.
// See Cmder.Register for in-depth documentation.
func (b *Bot) RegisterServerCmd(srv string, command *cmd.Cmd) error {

	b.protectServers.RLock()
	defer b.protectServers.RUnlock()

	if s, ok := b.servers[srv]; ok {
		return s.cmds.Register(srv, command)
	}
	return errUnknownServerID
}

// UnregisterCmd unregister's a command from the bot.
func (b *Bot) UnregisterCmd(command string) bool {
	return b.cmds.Unregister(cmd.GLOBAL, command)
}

// UnregisterServerCmd unregister's a command from the server.
func (b *Bot) UnregisterServerCmd(server, command string) bool {
	b.protectServers.RLock()
	defer b.protectServers.RUnlock()

	if s, ok := b.servers[server]; ok {
		return s.cmds.Unregister(server, command)
	}
	return false
}

// GetEndpoint retrieves a servers endpoint. Will be nil if the server does
// not exist.
func (b *Bot) GetEndpoint(server string) (endpoint *data.DataEndpoint) {
	b.protectServers.RLock()
	defer b.protectServers.RUnlock()

	if srv, ok := b.servers[server]; ok {
		endpoint = srv.endpoint.DataEndpoint
	}
	return
}

// createBot creates a bot from the given configuration, using the providers
// given to create connections and protocol caps.
func createBot(conf *config.Config, connProv ConnProvider,
	storeProv StoreProvider,
	attachHandlers, attachCommands bool) (*Bot, error) {

	b := &Bot{
		conf:           conf,
		servers:        make(map[string]*Server, nAssumedServers),
		connProvider:   connProv,
		storeProvider:  storeProv,
		attachHandlers: attachHandlers,
		botEnd:         make(chan error),
		serverControl:  make(chan serverOp),
		serverStart:    make(chan bool),
		serverStop:     make(chan bool),
		serverEnd:      make(chan serverOp),
	}

	b.caps = irc.CreateProtoCaps()
	b.createDispatching(conf.Global.GetPrefix(), conf.Global.GetChannels())

	makeStore := false
	for _, srv := range conf.Servers {
		makeStore = makeStore || !srv.GetNoStore()
	}

	var err error
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
		b.coreCommands, err = CreateCoreCmds(b)
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
		bot:         b,
		name:        conf.GetName(),
		caps:        b.caps.Clone(),
		conf:        conf,
		killable:    make(chan int),
		reconnScale: defaultReconnScale,
	}

	s.createDispatching(conf.GetPrefix(), conf.GetChannels())

	if !conf.GetNoState() {
		if err := s.createState(); err != nil {
			return nil, err
		}
	}

	s.createEndpoint(b.store, &b.protectStore)

	if b.attachHandlers {
		s.handler = &coreHandler{bot: b}
		s.handlerID = s.dispatcher.Register(irc.RAW, s.handler)
	}

	return s, nil
}

// createDispatcher uses the bot's current ProtoCaps to create a dispatcher.
func (b *Bot) createDispatching(prefix rune, channels []string) {
	b.dispatchCore = dispatch.CreateDispatchCore(b.caps, channels...)
	b.dispatcher = dispatch.CreateDispatcher(b.dispatchCore)
	b.cmds = cmd.CreateCmds(prefix, b.dispatchCore)
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
func (b *Bot) mergeProtocaps(toMerge *irc.ProtoCaps) {
	b.caps.Merge(toMerge)
	b.dispatchCore.Protocaps(b.caps)
}
