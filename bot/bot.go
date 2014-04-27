/*
Package bot implements the top-level package that any non-extension
will use to start a bot instance.
*/
package bot

import (
	"bufio"
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/aarondl/ultimateq/config"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/dispatch"
	"github.com/aarondl/ultimateq/dispatch/cmd"
	"github.com/aarondl/ultimateq/irc"
	"github.com/aarondl/ultimateq/parse"
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
	// errInvalidConfig is when NewBot was given an invalid configuration.
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

// Configure starts a configuration by calling NewConfig. Alias for
// config.NewConfig
func Configure() *config.Config {
	return config.NewConfig()
}

// ConfigureFile starts a configuration by reading in a file. Alias for
// config.NewConfigFromFile
func ConfigureFile(filename string) *config.Config {
	return config.NewConfigFromFile(filename)
}

// ConfigureFunction creates a blank configuration and passes it into a function
func ConfigureFunction(cnf func(*config.Config) *config.Config) *config.Config {
	return cnf(config.NewConfig())
}

// CheckConfig checks a bots config for validity.
func CheckConfig(c *config.Config) bool {
	if !c.IsValid() {
		c.DisplayErrors()
		return false
	}
	return true
}

// NewBot simplifies the call to createBotFull by using default
// caps and conn provider functions.
func NewBot(conf *config.Config) (*Bot, error) {
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
	var ircMsg *irc.Event
	var parseErr error
	readCh := srv.client.ReadChannel()

	b.dispatchMessage(srv,
		irc.NewEvent(srv.name, srv.netInfo, irc.CONNECT, srv.name))
	for err == nil && !disconnect {
		select {
		case ev, ok := <-readCh:
			if !ok {
				disconnect = true
				break
			}

			ircMsg, parseErr = parse.Parse(ev)
			if parseErr != nil {
				log.Printf(errFmtParsingIrcMessage, parseErr, ev)
				break
			}
			ircMsg.NetworkID = srv.name
			ircMsg.NetworkInfo = srv.netInfo

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

	b.dispatchMessage(srv,
		irc.NewEvent(srv.name, srv.netInfo, irc.DISCONNECT, srv.name))
	return
}

// dispatch sends a message to both the bot's dispatcher and the given servers
func (b *Bot) dispatchMessage(s *Server, ev *irc.Event) {
	b.dispatcher.Dispatch(ev, s.writer)
	s.dispatcher.Dispatch(ev, s.writer)
	b.cmds.Dispatch(s.name, s.cmds.GetPrefix(), ev, s.writer, b)
	s.cmds.Dispatch(s.name, 0, ev, s.writer, b)
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
func (b *Bot) StopServer(networkID string) (stopped bool) {
	if srv := b.getServer(networkID); srv != nil {
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

	if s := b.getServer(server); s != nil {
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

	if s := b.getServer(server); s != nil {
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
	if s := b.getServer(srv); s != nil {
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
	if s := b.getServer(server); s != nil {
		return s.cmds.Unregister(server, command)
	}
	return false
}

// UsingState calls a callback if the requested server can present a state db.
// The returned boolean is whether or not the function was called.
func (b *Bot) UsingState(networkID string, fn func(*data.State)) (called bool) {
	s := b.getServer(networkID)
	if s == nil {
		return false
	}

	s.protectState.RLock()
	defer s.protectState.RUnlock()
	if s.state != nil {
		fn(s.state)
		called = true
	}
	return
}

// OpenState locks the state db, and returns it. CloseState must be called or
// the lock will never be released and the bot will sieze up. The state must
// be checked for nil in case the state is disabled.
func (b *Bot) OpenState(networkID string) *data.State {
	s := b.getServer(networkID)
	if s == nil {
		return nil
	}

	s.protectState.RLock()
	return s.state
}

// CloseState unlocks the data state after use by OpenState.
func (b *Bot) CloseState(networkID string) {
	s := b.getServer(networkID)
	if s != nil {
		s.protectState.RUnlock()
	}
}

// UsingStore calls a callback if the requested server can present a data store.
// The returned boolean is whether or not the function was called.
func (b *Bot) UsingStore(fn func(*data.Store)) (called bool) {
	b.protectStore.RLock()
	defer b.protectStore.RUnlock()
	if b.store != nil {
		fn(b.store)
		called = true
	}
	return
}

// OpenStore locks the store db, and returns it. CloseStore must be called or
// the lock will never be released and the bot will sieze up. The store must
// be checked for nil.
func (b *Bot) OpenStore() *data.Store {
	b.protectStore.RLock()
	return b.store
}

// CloseStore unlocks the data store after use by OpenState.
func (b *Bot) CloseStore() {
	b.protectStore.RUnlock()
}

// GetEndpoint retrieves a servers endpoint. Will be nil if the server does
// not exist.
func (b *Bot) NetworkWriter(networkID string) (w irc.Writer) {
	if s := b.getServer(networkID); s != nil {
		w = s.writer
	}
	return w
}

// createBot creates a bot from the given configuration, using the providers
// given to create connections and protocol.
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
		b.coreCommands, err = NewCoreCmds(b)
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
		netInfo:     irc.NewNetworkInfo(),
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

	if b.attachHandlers {
		s.handler = &coreHandler{bot: b}
		s.handlerID = s.dispatcher.Register(irc.RAW, s.handler)
	}

	s.writer = &irc.Helper{s}

	return s, nil
}

// createDispatcher uses the bot's current ProtoCaps to create a dispatcher.
func (b *Bot) createDispatching(prefix rune, channels []string) {
	b.dispatchCore = dispatch.NewDispatchCore(channels...)
	b.dispatcher = dispatch.NewDispatcher(b.dispatchCore)
	b.cmds = cmd.NewCmds(prefix, b.dispatchCore)
}

// createStore creates a store from a filename.
func (b *Bot) createStore(filename string) (err error) {
	if b.storeProvider == nil {
		b.store, err = data.NewStore(data.MakeFileStoreProvider(filename))
	} else {
		b.store, err = b.storeProvider(filename)
	}
	return
}

// getServer safely retrieves a server.
func (b *Bot) getServer(networkID string) *Server {
	b.protectServers.RLock()
	defer b.protectServers.RUnlock()

	if srv, ok := b.servers[networkID]; ok {
		return srv
	}
	return nil
}

// Run makes a very typical bot. It will call the cb function passed in
// before starting to allow registration of extensions etc. Returns error
// if the bot could not be created. Does NOT return until dead.
// The following are featured behaviors:
// Watches for Keyboard Input OR SIGTERM OR SIGKILL and shuts down normally.
// Creates a logger on stdout.
func Run(cb func(b *Bot)) error {
	log.SetOutput(os.Stdout)

	b, err := NewBot(ConfigureFile("config.yaml"))
	if err != nil {
		return err
	}
	defer b.Close()

	end := b.Start()

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
			log.Println("Server death:", err)
			stop = !ok
		}
	}

	log.Println("Shutting down...")
	<-time.After(1 * time.Second)

	return nil
}
