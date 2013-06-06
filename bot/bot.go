/*
Package bot implements the top-level package that any non-extension
will use to start a bot instance.
*/
package bot

import (
	"errors"
	"fmt"
	"github.com/aarondl/ultimateq/config"
	"github.com/aarondl/ultimateq/dispatch"
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
	errFmtParsingIrcMessage = "bot: Failed to parse irc message (%v)\n"
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
	errUnknownServerId = errors.New("bot: Unknown Server id.")
)

type (
	// CapsProvider returns a usable ProtoCaps to start the bot with
	CapsProvider func() *irc.ProtoCaps
	// ConnProvider transforms a "server:port" string into a net.Conn
	ConnProvider func(string) (net.Conn, error)
)

// Bot is a main type that joins together all the packages into a functioning
// irc bot. It should be able to carry out most major functions that a bot would
// need through it's exported functions.
type Bot struct {
	servers map[string]*Server

	conf *config.Config

	caps       *irc.ProtoCaps
	dispatcher *dispatch.Dispatcher

	// IoC and DI components mostly for testing.
	attachHandlers bool
	capsProvider   CapsProvider
	connProvider   ConnProvider

	msgDispatchers sync.WaitGroup
	// servers
	serversProtect sync.RWMutex
	// configs (including server configs)
	configsProtect sync.RWMutex
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

// Check config checks a bots config for validity.
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
	return createBot(conf, nil, nil, true)
}

// Connect creates the connections and the IrcClient objects, as well as
// connects the bot to all defined servers.
func (b *Bot) Connect() []error {
	var ers = make([]error, 0, nAssumedServers)
	b.serversProtect.RLock()
	for _, srv := range b.servers {
		err := b.connectServer(srv)
		if err != nil {
			ers = append(ers, err)
		}
	}
	b.serversProtect.RUnlock()

	if len(ers) > 0 {
		return ers
	}
	return nil
}

// ConnectServer creates the connection and IrcClient object for the given
// serverId.
func (b *Bot) ConnectServer(serverId string) (found bool, err error) {
	b.serversProtect.RLock()
	if srv, ok := b.servers[serverId]; ok {
		err = b.connectServer(srv)
		found = true
	}
	b.serversProtect.RUnlock()
	return
}

// connectServer creates the connection and IrcClient object for the given
// server.
func (b *Bot) connectServer(srv *Server) (err error) {
	if srv.IsConnected() {
		return errors.New(fmt.Sprintf(errFmtAlreadyConnected, srv.name))
	}
	srv.protect.Lock()
	err = srv.createIrcClient()
	if err == nil {
		srv.setConnected(true, false)
	}
	srv.protect.Unlock()
	return
}

// Start begins message pumps on all defined and connected servers.
func (b *Bot) Start() {
	b.start(true, true)
}

// StartServer begins message pumps on a server by id.
func (b *Bot) StartServer(serverId string) (found bool) {
	b.serversProtect.RLock()
	if srv, ok := b.servers[serverId]; ok {
		b.startServer(srv, true, true)
		found = true
	}
	b.serversProtect.RUnlock()
	return
}

// start begins the called for routines on all servers
func (b *Bot) start(writing, reading bool) {
	b.serversProtect.RLock()
	for _, srv := range b.servers {
		b.startServer(srv, writing, reading)
	}
	b.serversProtect.RUnlock()
}

// startServer begins the called for routines on the specific server
func (b *Bot) startServer(srv *Server, writing, reading bool) {
	if srv.IsStarted() {
		return
	}

	srv.protect.Lock()
	defer srv.protect.Unlock()

	if srv.isConnected() && srv.client != nil {
		if writing {
			srv.setWriting(true, false)
		}
		if reading {
			srv.setReading(true, false)
		}
		srv.client.SpawnWorkers(writing, reading)

		b.dispatchMessage(srv, &irc.IrcMessage{Name: irc.CONNECT})

		if reading {
			b.msgDispatchers.Add(1)
			go b.dispatchMessages(srv)
		}
	}
}

// Stop shuts down all dispatch routines.
func (b *Bot) Stop() {
	b.serversProtect.RLock()
	for _, srv := range b.servers {
		b.stopServer(srv)
	}
	b.serversProtect.RUnlock()
}

// StopServer shuts down the dispatch routine of the given server by id.
func (b *Bot) StopServer(serverId string) (found bool) {
	b.serversProtect.RLock()
	if srv, ok := b.servers[serverId]; ok {
		b.stopServer(srv)
		found = true
	}
	b.serversProtect.RUnlock()
	return
}

// stopServer stops dispatcher on the given server.
func (b *Bot) stopServer(srv *Server) {
	if srv.IsReading() {
		srv.killdispatch <- 0
		srv.setReading(false, true)
	}
}

// Disconnect closes all connections to the servers
func (b *Bot) Disconnect() {
	b.serversProtect.RLock()
	for _, srv := range b.servers {
		b.disconnectServer(srv)
	}
	b.serversProtect.RUnlock()
}

// DisconnectServer disconnects the given server by id.
func (b *Bot) DisconnectServer(serverId string) (found bool) {
	b.serversProtect.RLock()
	if srv, ok := b.servers[serverId]; ok {
		b.disconnectServer(srv)
		found = true
	}
	b.serversProtect.RUnlock()
	return
}

// disconnectServer disconnects the given server.
func (b *Bot) disconnectServer(srv *Server) {
	srv.protect.RLock()
	if !srv.isConnected() || srv.client == nil {
		srv.protect.RUnlock()
		return
	}
	srv.client.Close()
	srv.protect.RUnlock()

	srv.protect.Lock()
	defer srv.protect.Unlock()
	srv.client = nil
	srv.setWriting(false, false)
	srv.setConnected(false, false)
}

// InterruptReconnect stops reconnecting the given server by id.
func (b *Bot) InterruptReconnect(serverId string) (found bool) {
	b.serversProtect.RLock()
	if srv, ok := b.servers[serverId]; ok {
		b.interruptReconnect(srv)
		found = true
	}
	b.serversProtect.RUnlock()
	return
}

// interruptReconnect stops reconnecting the given server.
func (b *Bot) interruptReconnect(srv *Server) {
	if srv.IsReconnecting() {
		srv.killreconn <- 0
	}
}

// WaitForHalt waits for all servers to halt.
func (b *Bot) WaitForHalt() {
	b.msgDispatchers.Wait()
	b.dispatcher.WaitForCompletion()
	b.serversProtect.RLock()
	for _, srv := range b.servers {
		srv.dispatcher.WaitForCompletion()
	}
	b.serversProtect.RUnlock()
}

// Register adds an event handler to the bot's global dispatcher.
func (b *Bot) Register(event string, handler interface{}) int {
	return b.dispatcher.Register(event, handler)
}

// Register adds an event handler to a server specific dispatcher.
func (b *Bot) RegisterServer(
	server string, event string, handler interface{}) (int, error) {

	b.serversProtect.RLock()
	defer b.serversProtect.RUnlock()

	if s, ok := b.servers[server]; ok {
		s.protect.RLock()
		defer s.protect.RUnlock()
		return s.dispatcher.Register(event, handler), nil
	}
	return 0, errUnknownServerId
}

// Unregister removes an event handler from the bot's global dispatcher
func (b *Bot) Unregister(event string, id int) bool {
	return b.dispatcher.Unregister(event, id)
}

// Unregister removes an event handler from a server specific dispatcher.
func (b *Bot) UnregisterServer(
	server string, event string, id int) (bool, error) {

	b.serversProtect.RLock()
	defer b.serversProtect.RUnlock()

	if s, ok := b.servers[server]; ok {
		s.protect.RLock()
		defer s.protect.RUnlock()
		return s.dispatcher.Unregister(event, id), nil
	}
	return false, errUnknownServerId
}

// dispatchMessages is a constant read-dispatch from the server to the
// dispatcher.
func (b *Bot) dispatchMessages(s *Server) {
	s.protect.RLock()

	read := s.client.ReadChannel()
	stop, disconnect := false, false
	for !stop {
		select {
		case msg, ok := <-read:
			if !ok {
				log.Printf(errFmtReaderClosed, s.name)
				b.dispatchMessage(s, &irc.IrcMessage{Name: irc.DISCONNECT})
				stop, disconnect = true, true
				break
			}
			ircMsg, err := parse.Parse(string(msg))
			if err != nil {
				log.Printf(errFmtParsingIrcMessage, err)
			} else {
				b.dispatchMessage(s, ircMsg)
			}
		case <-s.killdispatch:
			log.Printf(errFmtReaderClosed, s.name)
			stop = true
			break
		}
	}
	s.protect.RUnlock()

	var reconn bool
	var scale time.Duration
	var timeout uint
	b.ReadConfig(func(c *config.Config) {
		cserver := c.GetServer(s.name)
		reconn = disconnect && !cserver.GetNoReconnect()
		scale = s.reconnScale
		timeout = cserver.GetReconnectTimeout()
	})

	if !reconn {
		b.msgDispatchers.Done()
	}

	log.Printf(fmtDisconnected, s.name)

	if reconn {
		for {
			dur := time.Duration(timeout) * scale
			log.Printf(fmtReconnecting, s.name, dur)
			b.disconnectServer(s)
			s.protect.Lock()
			s.setStarted(false, false)
			s.setReconnecting(true, false)
			s.protect.Unlock()
			select {
			case <-time.After(dur):
				s.setReconnecting(false, true)
				break
			case <-s.killreconn:
				s.setReconnecting(false, true)
				b.msgDispatchers.Done()
				return
			}

			err := b.connectServer(s)
			if err != nil {
				log.Printf(fmtFailedConnecting, s.name, err)
				continue
			} else {
				b.startServer(s, true, true)
				break
			}
		}
		b.msgDispatchers.Done()
	} else if disconnect {
		<-s.killdispatch
	}
}

// Writeln writes a string to the given server's IrcClient.
func (b *Bot) Writeln(server, message string) error {
	b.serversProtect.RLock()
	defer b.serversProtect.RUnlock()

	if srv, ok := b.servers[server]; !ok {
		return errUnknownServerId
	} else {
		return srv.Writeln(message)
	}
}

// dispatch sends a message to both the bot's dispatcher and the given servers
func (b *Bot) dispatchMessage(s *Server, msg *irc.IrcMessage) {
	sender := ServerSender{s.name, s}
	b.dispatcher.Dispatch(msg, sender)
	s.dispatcher.Dispatch(msg, sender)
}

// createBot creates a bot from the given configuration, using the providers
// given to create connections and protocol caps.
func createBot(conf *config.Config, capsProv CapsProvider,
	connProv ConnProvider, attachHandlers bool) (*Bot, error) {

	b := &Bot{
		conf:           conf,
		servers:        make(map[string]*Server, nAssumedServers),
		capsProvider:   capsProv,
		connProvider:   connProv,
		attachHandlers: attachHandlers,
	}

	if capsProv == nil {
		b.caps = irc.CreateProtoCaps()
	} else {
		b.caps = capsProv()
	}

	var err error
	if err = b.createDispatcher(conf.Global.GetChannels()); err != nil {
		return nil, err
	}

	for name, srv := range conf.Servers {
		server, err := b.createServer(srv)
		if err != nil {
			return nil, err
		}

		b.servers[name] = server
	}

	return b, nil
}

// createServer creates a dispatcher, and an irc client to connect to this
// server.
func (b *Bot) createServer(conf *config.Server) (*Server, error) {
	var copyCaps irc.ProtoCaps = *b.caps
	s := &Server{
		bot:          b,
		name:         conf.GetName(),
		caps:         &copyCaps,
		conf:         conf,
		killdispatch: make(chan int),
		killreconn:   make(chan int),
		reconnScale:  defaultReconnScale,
	}

	if err := s.createDispatcher(conf.GetChannels()); err != nil {
		return nil, err
	}

	if b.attachHandlers {
		s.handler = &coreHandler{bot: b}
		s.handlerId =
			s.dispatcher.Register(irc.RAW, s.handler)
	}

	return s, nil
}

// createDispatcher uses the bot's current ProtoCaps to create a dispatcher.
func (b *Bot) createDispatcher(channels []string) error {
	var err error
	b.dispatcher, err = dispatch.CreateRichDispatcher(b.caps, channels)
	if err != nil {
		return err
	}
	return nil
}
