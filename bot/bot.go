/*
Package bot implements the top-level package that any non-extension
will use to start a bot instance.
*/
package bot

import (
	"bufio"
	"errors"
	"fmt"
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
	"github.com/inconshreveable/log15"
)

const (
	// defaultReconnScale is how the config's ReconnTimeout is scaled.
	defaultReconnScale = time.Second

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
	// errParsingIrcMessage is when the bot fails to parse a message
	// during it's dispatch loop.
	errParsingIrcMessage = "Failed to parse irc message"
	// errLogFile is when the bot can't open the log file.
	errLogFile = errors.New("bot: Could not open log file")
	// errInvalidConfig is when New was given an invalid configuration.
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
	// LoggerProvider returns a log15.Handler suitable for logging.
	LoggerProvider func() log15.Handler
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

	// Logging
	log15.Logger

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
	logProvider    LoggerProvider

	// Synchronization
	msgDispatchers sync.WaitGroup
	protectServers sync.RWMutex
}

// CheckConfig checks a bots config for validity.
func CheckConfig(c *config.Config) bool {
	if ers := c.Errors(); len(ers) > 0 || !c.Validate() {
		c.DisplayErrors(log15.Root())
		return false
	}
	return true
}

// New simplifies the call to createBotFull by using default
// caps and conn provider functions.
func New(conf *config.Config) (*Bot, error) {
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
			srv.Info("Connected")
			srv.setStatus(STATUS_STARTED)

			srv.client.SpawnWorkers(writing, reading)
			disconnect, err = b.dispatch(srv)
			if err != nil {
				break
			}
		}
		srv.Info("Disconnected", "err", err)

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
		srv.Info("Reconnecting", "timeout", wait)
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
				b.Warn(errParsingIrcMessage, "err", parseErr, "ev", ev)
				break
			}
			ircMsg.NetworkID = srv.networkID
			ircMsg.NetworkInfo = srv.netInfo

			if srv.state != nil {
				srv.state.Update(ircMsg)
			}
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
	b.cmds.Dispatch(s.writer, ev, b)
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
	if b.store != nil {
		err := b.store.Close()
		b.store = nil
		return err
	}
	return nil
}

// Register an event handler to the bot in global space. Returns an identifier
// that can be used to unregister the event.
func (b *Bot) Register(event string, handler interface{}) uint64 {
	return b.RegisterFiltered("", "", event, handler)
}

// RegisterFiltered event handlers to the specified network and channel.
// Leave either blank to create a filter based on that field alone. Returns
// an identifier that can be used to unregister the event.
func (b *Bot) RegisterFiltered(network, channel, event string,
	handler interface{}) uint64 {

	return b.dispatcher.Register(network, channel, event, handler)
}

// Unregister an event handler from the bot.
func (b *Bot) Unregister(id uint64) bool {
	return b.dispatcher.Unregister(id)
}

// RegisterCmd registers a command with the bot.
// See Cmder.Register for in-depth documentation.
func (b *Bot) RegisterCmd(command *cmd.Cmd) error {
	return b.RegisterFilteredCmd("", "", command)
}

// RegisterFilteredCmd registers a command with the bot filtered based on the
// network and channel. Leave either field blank to create a filter based on
// that field alone.
func (b *Bot) RegisterFilteredCmd(network, channel string,
	command *cmd.Cmd) error {

	return b.cmds.Register(network, channel, command)
}

// UnregisterCmd from the bot. Leaving ext blank will cause all commands with
// this name from all extensions to be unregistered.
func (b *Bot) UnregisterCmd(ext, command string) bool {
	return b.UnregisterFilteredCmd("", "", ext, command)
}

// UnregisterFilteredCmd from the bot. All parameters can be blank except for
// cmd. Leaving ext blank wipes out other extension's commands with the same
// name.
func (b *Bot) UnregisterFilteredCmd(network, channel, ext, cmd string) bool {
	return b.cmds.Unregister(network, channel, ext, cmd)
}

// State returns the state db for that network id. If the server doesn't exist
// or state is disabled, returns nil.
func (b *Bot) State(networkID string) *data.State {
	s := b.getServer(networkID)
	if s == nil {
		return nil
	}

	return s.state
}

// Store returns the store for the bot. Returns nil if store is disabled.
func (b *Bot) Store() *data.Store {
	return b.store
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
	storeProv StoreProvider, logProv LoggerProvider,
	attachHandlers, attachCommands bool) (*Bot, error) {

	b := &Bot{
		conf:           conf,
		Logger:         log15.New(),
		servers:        make(map[string]*Server, 0),
		connProvider:   connProv,
		storeProvider:  storeProv,
		logProvider:    logProv,
		attachHandlers: attachHandlers,
		botEnd:         make(chan error),
		serverControl:  make(chan serverOp),
		serverStart:    make(chan bool),
		serverStop:     make(chan bool),
		serverEnd:      make(chan serverOp),
	}

	var err error
	var logHandler log15.Handler

	if logProv != nil {
		logHandler = logProv()
	} else {
		if file, ok := conf.LogFile(); ok {
			logHandler, err = log15.FileHandler(file, log15.LogfmtFormat())
			if err != nil {
				return nil, err
			}
		} else {
			logHandler = log15.StdoutHandler
		}
		if level, ok := conf.LogLevel(); ok {
			lvl, _ := log15.LvlFromString(level)
			logHandler = log15.LvlFilterHandler(lvl, logHandler)
		} else {
			logHandler = log15.LvlFilterHandler(log15.LvlInfo, logHandler)
		}
	}
	b.Logger.SetHandler(logHandler)

	networks := conf.Networks()
	cfg := conf.Network("")
	b.createDispatching()

	makeStore := false
	for _, net := range networks {
		nostore, _ := conf.Network(net).NoStore()
		makeStore = makeStore || !nostore
	}

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
		Logger:      b.Logger.New("net", netID),
		networkID:   netID,
		netInfo:     irc.NewNetworkInfo(),
		conf:        conf,
		killable:    make(chan int),
		reconnScale: defaultReconnScale,
	}

	cfg := conf.Network(netID)

	nostate, _ := cfg.NoState()
	if !nostate {
		if err := s.createState(); err != nil {
			return nil, err
		}
	}

	if b.attachHandlers {
		s.handler = &coreHandler{bot: b, untilJoinScale: time.Second}
		s.handlerID = b.dispatcher.Register(netID, "", irc.RAW, s.handler)
	}

	s.writer = &irc.Helper{Writer: s}

	return s, nil
}

// createDispatcher uses the bot's current ProtoCaps to create a dispatcher.
func (b *Bot) createDispatching(channels ...string) {
	b.dispatchCore = dispatch.NewDispatchCore(b.Logger, channels...)
	b.dispatcher = dispatch.NewDispatcher(b.dispatchCore)
	b.cmds = cmd.NewCmds(b.mkPrefixFetcher(), b.dispatchCore)
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

// mkPrefixFetcher creates a function that can fetch the prefix for a given
// network or channel (or if both are omitted global).
func (b *Bot) mkPrefixFetcher() func(network, channel string) rune {
	prefixii := make(map[string]rune)

	net := b.conf.Network("")
	prefixii[":"], _ = net.Prefix()

	chans, _ := net.Channels()
	for _, ch := range chans {
		if len(ch.Prefix) > 0 {
			prefixii[":"+ch.Name] = rune(ch.Prefix[0])
		}
	}

	nets := b.conf.Networks()
	for _, netName := range nets {
		net = b.conf.Network(netName)
		if pfx, ok := net.Prefix(); ok {
			prefixii[netName+":"] = pfx
		}

		chans, _ = net.Channels()
		for _, ch := range chans {
			if len(ch.Prefix) > 0 {
				prefixii[netName+":"+ch.Name] = rune(ch.Prefix[0])
			}
		}
	}

	return func(n, c string) rune {
		var pfx rune
		var ok bool
		key := fmt.Sprintf("%s:%s", n, c)
		if pfx, ok = prefixii[key]; ok {
			return pfx
		}
		key = fmt.Sprintf(":%s", c)
		if pfx, ok = prefixii[key]; ok {
			return pfx
		}
		key = fmt.Sprintf("%s:", n)
		if pfx, ok = prefixii[key]; ok {
			return pfx
		}
		pfx, _ = prefixii[":"]
		return pfx
	}
}

// Run makes a very typical bot. It will call the cb function passed in
// before starting to allow registration of extensions etc. Returns error
// if the bot could not be created. Does NOT return until dead.
// The following are featured behaviors:
// Reads configuration file from ./config.toml
// Watches for Keyboard Input OR SIGTERM OR SIGKILL and shuts down normally.
// Pauses after death to allow all goroutines to come to a graceful shutdown.
func Run(cb func(b *Bot)) error {
	cfg := config.NewConfig().FromFile("config.toml")
	b, err := New(cfg)
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
				b.Info("Server death", "err", err)
			}
			stop = !ok
		}
	}

	b.Info("Shutting down...")
	<-time.After(1 * time.Second)

	return nil
}
