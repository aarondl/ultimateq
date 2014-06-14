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
	// network id to a method requiring a network id.
	errUnknownServerID = errors.New("bot: Unknown Network id")
	// errServerKilled occurs when the server is killed during the running state
	errServerKilled = errors.New("bot: Server killed")
	// errServerKilledReconn occurs when the server is killed during a
	// reconnection pause.
	errServerKilledReconn = errors.New("bot: Server reconnection aborted")
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
}

// CheckConfig checks a bots config for validity.
func CheckConfig(c *config.Config) bool {
	if ers := c.Errors(); len(ers) > 0 || !c.Validate() {
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

// StartNetwork starts a network by name. Start() should have been called prior
// to this.
func (b *Bot) StartNetwork(networkID string) (started bool) {
	b.protectServers.RLock()
	defer b.protectServers.RUnlock()

	var srv *Server
	if srv, started = b.servers[networkID]; started {
		go b.startServer(srv, true, true)
	}
	return
}

// startServer starts up a server. When it has finished (permanently
// disconnected) it will send it's disconnection error to serverEnd.
func (b *Bot) startServer(srv *Server, writing, reading bool) {
	var err error
	var disconnect bool
	var temporary bool

	b.serverControl <- serverOp{srv, true, nil}
	if !<-b.serverStart {
		return
	}

	for err == nil {
		srv.setStatus(STATUS_CONNECTING)
		err, temporary = srv.createIrcClient()
		disconnect = err != nil && temporary

		if err == nil {
			srv.setStatus(STATUS_STARTED)

			srv.client.SpawnWorkers(writing, reading)
			disconnect, err = b.dispatch(srv)
			if err != nil {
				break
			}
		}
		log.Printf("(%s) Disconnected", srv.networkID)

		cfg := srv.conf.Network(srv.networkID)
		noReconn, _ := cfg.NoReconnect()
		reconnTime, _ := cfg.ReconnectTimeout()

		if !disconnect || noReconn {
			break
		}

		err = nil
		srv.Close()
		wait := time.Duration(reconnTime) * srv.reconnScale

		srv.setStatus(STATUS_RECONNECTING)
		log.Printf("(%s) Reconnecting in %v", srv.networkID, wait)
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
		irc.NewEvent(srv.networkID, srv.netInfo, irc.CONNECT, srv.networkID))
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
			ircMsg.NetworkID = srv.networkID
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
		irc.NewEvent(srv.networkID, srv.netInfo, irc.DISCONNECT, srv.networkID))
	return
}

// dispatch sends a message to both the bot's dispatcher and the given servers
func (b *Bot) dispatchMessage(s *Server, ev *irc.Event) {
	b.dispatcher.Dispatch(s.writer, ev)
	s.dispatcher.Dispatch(s.writer, ev)
	b.cmds.Dispatch(s.networkID, s.cmds.GetPrefix(), s.writer, ev, b)
	s.cmds.Dispatch(s.networkID, 0, s.writer, ev, b)
}

// Stop shuts down all connections and exits.
func (b *Bot) Stop() {
	b.protectServers.RLock()
	defer b.protectServers.RUnlock()

	for _, srv := range b.servers {
		b.stopServer(srv)
	}
}

// StopNetwork stops a network by name.
func (b *Bot) StopNetwork(networkID string) (stopped bool) {
	b.protectServers.RLock()
	defer b.protectServers.RUnlock()

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

// RegisterNetwork adds an event handler to a network specific dispatcher.
func (b *Bot) RegisterNetwork(
	networkID string, event string, handler interface{}) (int, error) {

	if s := b.getServer(networkID); s != nil {
		return s.dispatcher.Register(event, handler), nil
	}
	return 0, errUnknownServerID
}

// Unregister removes an event handler from the bot's global dispatcher
func (b *Bot) Unregister(event string, id int) bool {
	return b.dispatcher.Unregister(event, id)
}

// UnregisterNetwork removes an event handler from a network specific
// dispatcher.
func (b *Bot) UnregisterNetwork(
	networkID string, event string, id int) (bool, error) {

	if s := b.getServer(networkID); s != nil {
		return s.dispatcher.Unregister(event, id), nil
	}
	return false, errUnknownServerID
}

// RegisterCmd registers a command with the bot.
// See Cmder.Register for in-depth documentation.
func (b *Bot) RegisterCmd(command *cmd.Cmd) error {
	return b.cmds.Register(cmd.GLOBAL, command)
}

// RegisterNetworkCmd registers a command with the network.
// See Cmder.Register for in-depth documentation.
func (b *Bot) RegisterNetworkCmd(networkID string, command *cmd.Cmd) error {
	if s := b.getServer(networkID); s != nil {
		return s.cmds.Register(networkID, command)
	}
	return errUnknownServerID
}

// UnregisterCmd unregister's a command from the bot.
func (b *Bot) UnregisterCmd(command string) bool {
	return b.cmds.Unregister(cmd.GLOBAL, command)
}

// UnregisterNetworkCmd unregister's a command from the network.
func (b *Bot) UnregisterNetworkCmd(networkID, command string) bool {
	if s := b.getServer(networkID); s != nil {
		return s.cmds.Unregister(networkID, command)
	}
	return false
}

// UsingState calls a callback if the requested network can present a state db.
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

// UsingStore calls a callback if the bot can present a store db.
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

// NetworkWriter retrieves a network's writer. Will be nil if the network does
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
		servers:        make(map[string]*Server, 0),
		connProvider:   connProv,
		storeProvider:  storeProv,
		attachHandlers: attachHandlers,
		botEnd:         make(chan error),
		serverControl:  make(chan serverOp),
		serverStart:    make(chan bool),
		serverStop:     make(chan bool),
		serverEnd:      make(chan serverOp),
	}

	networks := conf.Networks()
	cfg := conf.Network("")
	pfx, _ := cfg.Prefix()
	b.createDispatching(pfx, nil)

	makeStore := false
	for _, net := range networks {
		nostore, _ := conf.Network(net).NoStore()
		makeStore = makeStore || !nostore
	}

	var err error
	if makeStore {
		sfile, _ := conf.StoreFile()
		if err = b.createStore(sfile); err != nil {
			return nil, err
		}
	}

	for _, net := range networks {
		server, err := b.createServer(net, conf)
		if err != nil {
			return nil, err
		}
		b.servers[net] = server
	}

	nostore, _ := cfg.NoStore()
	if attachCommands && !nostore {
		b.coreCommands, err = NewCoreCmds(b)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

// createServer creates a dispatcher, and an irc client to connect to this
// server.
func (b *Bot) createServer(netID string, conf *config.Config) (*Server, error) {
	s := &Server{
		bot:         b,
		networkID:   netID,
		netInfo:     irc.NewNetworkInfo(),
		conf:        conf,
		killable:    make(chan int),
		reconnScale: defaultReconnScale,
	}

	cfg := conf.Network(netID)
	pfx, _ := cfg.Prefix()
	s.createDispatching(pfx, nil)

	nostate, _ := cfg.NoState()
	if !nostate {
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

	cfg := config.NewConfig().FromFile("config.toml")
	b, err := NewBot(cfg)
	if err != nil {
		return err
	}
	defer b.Close()

	cb(b)

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
			if ok {
				log.Println("Server death:", err)
			}
			stop = !ok
		}
	}

	log.Println("Shutting down...")
	<-time.After(1 * time.Second)

	return nil
}
