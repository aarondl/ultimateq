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
	"github.com/aarondl/ultimateq/inet"
	"github.com/aarondl/ultimateq/irc"
	"github.com/aarondl/ultimateq/parse"
	"log"
	"net"
	"strconv"
	"sync"
)

// Server States
const (
	STATE_NEW          = 0x0
	STATE_CONNECTED    = 0x1
	STATE_STARTED      = 0x2
	STATE_STOPPED      = ^STATE_STARTED
	STATE_DISCONNECTED = ^STATE_CONNECTED

	MASK_CONNECTION = STATE_CONNECTED
	MASK_DISPATCHER = STATE_STARTED
)

const (
	// defaultChanTypes is used to create a barebones ProtoCaps until
	// the real values can be filled in by the handler.
	defaultChanTypes = "#&~"
	// nAssumedServers is how many servers a bot typically connects to.
	nAssumedServers = 1

	// errFmtParsingIrcMessage is when the bot fails to parse a message
	// during it's dispatch loop.
	errFmtParsingIrcMessage = "bot: Failed to parse irc message (%v)\n"
	// errFmtReaderClosed is when a write fails due to a closed socket or
	// a shutdown on the client.
	errFmtReaderClosed = "bot: %v reader closed\n"
	// errFmtClosingServer is when a IrcClient.Close returns an error.
	errFmtClosingServer = "bot: Error closing server (%v)\n"
	// errServerAlreadyConnected occurs if a server has not been shutdown
	// before another attempt to connect to it is made.
	errFmtAlreadyConnected = "bot: %v already connected.\n"
)

var (
	// errInvalidConfig is when CreateBot was given an invalid configuration.
	errInvalidConfig = errors.New("bot: Invalid Configuration")
	// errInvalidServerId occurs when the user passes in an unknown
	// server id to a method requiring a server id.
	errUnknownServerId = errors.New("bot: Unknown Server id.")
	// temporary error until ssl is fixed.
	errSslNotImplemented = errors.New("bot: Ssl not implemented")
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
	conf    *config.Config
	servers map[string]*Server

	caps       *irc.ProtoCaps
	dispatcher *dispatch.Dispatcher

	capsProvider CapsProvider
	connProvider ConnProvider

	msgDispatchers sync.WaitGroup
	// servers
	serversProtect sync.RWMutex
}

// Server is all the details around a specific server connection. Also contains
// the connection and configuration for the specific server.
type Server struct {
	bot        *Bot
	state      int
	dispatcher *dispatch.Dispatcher
	client     *inet.IrcClient
	conf       *config.Server
	caps       *irc.ProtoCaps

	killdispatch chan int

	handlerId int
	handler   *coreHandler

	// state, conf, client
	protect sync.RWMutex
}

// ServerSender implements the server interface, and wraps the write method
// of a server.
type ServerSender struct {
	id     string
	server *Server
}

// GetKey returns the server id of the current server.
func (s ServerSender) GetKey() string {
	return s.id
}

// Writeln writes to the ServerSender's IrcClient.
func (s ServerSender) Writeln(str string) error {
	s.server.protect.RLock()
	_, err := s.server.client.Write([]byte(str))
	s.server.protect.RUnlock()
	return err
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

// CreateBot simplifies the call to createBotFull by using default
// caps and conn provider functions.
func CreateBot(conf *config.Config) (*Bot, error) {
	if !conf.IsValid() {
		conf.DisplayErrors()
		return nil, errInvalidConfig
	}
	return createBot(conf, nil, nil)
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
	srv.protect.Lock()
	err = srv.createIrcClient()
	if err == nil {
		srv.setConnected(false)
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
	b.msgDispatchers = sync.WaitGroup{}
	b.serversProtect.RLock()
	for _, srv := range b.servers {
		b.startServer(srv, writing, reading)
	}
	b.serversProtect.RUnlock()
}

// startServer begins the called for routines on the specific server
func (b *Bot) startServer(srv *Server, writing, reading bool) {
	srv.protect.Lock()
	defer srv.protect.Unlock()
	if srv.client != nil {
		srv.setStarted(false)
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
	if srv.IsStarted() {
		srv.killdispatch <- 0
		srv.setStopped(true)
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
	srv.protect.Lock()
	defer srv.protect.Unlock()

	if srv.client == nil {
		return
	}
	srv.client.Close()
	srv.dispatcher.WaitForCompletion()
	srv.client = nil
	srv.setDisconnected(false)
}

// WaitForHalt waits for all servers to halt.
func (b *Bot) WaitForHalt() {
	b.msgDispatchers.Wait()
	b.dispatcher.WaitForCompletion()
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
				log.Printf(errFmtReaderClosed, s.conf.GetName())
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
			log.Printf(errFmtReaderClosed, s.conf.GetName())
			stop = true
			break
		}
	}
	s.protect.RUnlock()

	b.msgDispatchers.Done()
	if disconnect {
		<-s.killdispatch
	}
}

// dispatch sends a message to both the bot's dispatcher and the given servers
func (b *Bot) dispatchMessage(s *Server, msg *irc.IrcMessage) {
	sender := ServerSender{s.conf.GetName(), s}
	b.dispatcher.Dispatch(msg, sender)
	s.dispatcher.Dispatch(msg, sender)
}

// createBot creates a bot from the given configuration, using the providers
// given to create connections and protocol caps.
func createBot(conf *config.Config,
	capsProv CapsProvider, connProv ConnProvider) (*Bot, error) {
	b := &Bot{
		conf:         conf,
		servers:      make(map[string]*Server, nAssumedServers),
		capsProvider: capsProv,
		connProvider: connProv,
	}

	if capsProv == nil {
		b.caps = &irc.ProtoCaps{Chantypes: defaultChanTypes}
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
		caps:         &copyCaps,
		conf:         conf,
		killdispatch: make(chan int),
	}

	if err := s.createDispatcher(conf.GetChannels()); err != nil {
		return nil, err
	}

	s.handler = &coreHandler{bot: b}
	s.handlerId = s.dispatcher.Register(irc.RAW, s.handler)

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

// createDispatcher uses the server's current ProtoCaps to create a dispatcher.
func (s *Server) createDispatcher(channels []string) error {
	var err error
	s.dispatcher, err = dispatch.CreateRichDispatcher(s.caps, channels)
	if err != nil {
		return err
	}
	return nil
}

// createIrcClient connects to the configured server, and creates an IrcClient
// for use with that connection.
func (s *Server) createIrcClient() error {
	var conn net.Conn
	var err error

	if s.client != nil {
		return errors.New(fmt.Sprintf(errFmtAlreadyConnected, s.conf.GetName()))
	}

	port := strconv.Itoa(int(s.conf.GetPort()))
	server := s.conf.GetHost() + ":" + port

	if s.bot.connProvider == nil {
		if s.conf.GetSsl() {
			//TODO: Implement SSL
			return errSslNotImplemented
		} else {
			if conn, err = net.Dial("tcp", server); err != nil {
				return err
			}
		}
	} else {
		if conn, err = s.bot.connProvider(server); err != nil {
			return err
		}
	}

	s.client = inet.CreateIrcClient(conn, s.conf.GetName())
	return nil
}

// IsConnected checks to see if the server is connected.
func (s *Server) IsConnected() bool {
	s.protect.RLock()
	defer s.protect.RUnlock()

	return STATE_CONNECTED == s.state&MASK_CONNECTION
}

// setConnected sets the server's connected flag.
func (s *Server) setConnected(lock bool) {
	if lock {
		s.protect.Lock()
		defer s.protect.Unlock()
	}
	s.state |= STATE_CONNECTED
}

// setDisconnected clears the server's connected flag.
func (s *Server) setDisconnected(lock bool) {
	if lock {
		s.protect.Lock()
		defer s.protect.Unlock()
	}
	s.state &= STATE_DISCONNECTED
}

// IsStarted checks to see if the dispatcher is running on the server.
func (s *Server) IsStarted() bool {
	s.protect.RLock()
	defer s.protect.RUnlock()

	return STATE_STARTED == s.state&MASK_DISPATCHER
}

// setStarted clears the server's started flag.
func (s *Server) setStarted(lock bool) {
	if lock {
		s.protect.Lock()
		defer s.protect.Unlock()
	}
	s.state |= STATE_STARTED
}

// setStopped clears the server's started flag.
func (s *Server) setStopped(lock bool) {
	if lock {
		s.protect.Lock()
		defer s.protect.Unlock()
	}
	s.state &= STATE_STOPPED
}
