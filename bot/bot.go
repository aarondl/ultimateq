/*
Package bot implements the top-level package that any non-extension
will use to start a bot instance.
*/
package bot

import (
	"errors"
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

const (
	// defaultChanTypes is used to create a barebones ProtoCaps until
	// the real values can be filled in by the handler.
	defaultChanTypes = "#&~"
	// nAssumedServers is how many servers a bot typically connects to.
	nAssumedServers = 1
)

var (
	// errInvalidConfig is when CreateBot was given an invalid configuration.
	errInvalidConfig = errors.New("bot: Invalid Configuration")
	// errInvalidServerId occurs when the user passes in an unknown
	// server id to a method requiring a server id.
	errUnknownServerId = errors.New("bot: Unknown Server id.")
	// temporary error until ssl is fixed.
	errSslNotImplemented = errors.New("bot: Ssl not implemented")

	// errMsgParsingIrcMessage is when the bot fails to parse a message
	// during it's dispatch loop.
	errMsgParsingIrcMessage = "bot: Failed to parse irc message"
	// errMsgReaderClosed is when a write fails due to a closed socket or
	// a shutdown on the client.
	errMsgReaderClosed = "bot: %v Reader closed"
)

type (
	// CapsProvider returns a usable ProtoCaps to start the bot with
	CapsProvider func() *irc.ProtoCaps
	// ConnProvider transforms a "server:port" string into a net.Conn
	ConnProvider func(string) (net.Conn, error)
)

// Bot is the main type that will proxy various requests to different.
type Bot struct {
	caps           *irc.ProtoCaps
	dispatcher     *dispatch.Dispatcher
	conf           *config.Config
	servers        map[string]*Server
	capsProvider   CapsProvider
	connProvider   ConnProvider
	handler        coreHandler
	msgDispatchers sync.WaitGroup
}

// Server is all the details around a specific server connection. Also contains
// the connection and configuration for the specific server.
type Server struct {
	bot        *Bot
	dispatcher *dispatch.Dispatcher
	client     *inet.IrcClient
	conf       *config.Server
	caps       *irc.ProtoCaps
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
	_, err := s.server.client.Write([]byte(str))
	return err
}

// Configure starts a configuration by calling CreateConfig. Alias for
// config.CreateConfig
func Configure() *config.Config {
	return config.CreateConfig()
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
	for _, srv := range b.servers {
		err := srv.createIrcClient()
		if err != nil {
			ers = append(ers, err)
		}
	}

	if len(ers) > 0 {
		return ers
	}
	return nil
}

// Start begins message pumps on all defined and connected servers.
func (b *Bot) Start() {
	b.msgDispatchers = sync.WaitGroup{}
	for _, srv := range b.servers {
		if srv.client != nil {
			srv.client.SpawnWorkers()

			b.dispatchMessage(srv, &irc.IrcMessage{Name: irc.CONNECT})

			b.msgDispatchers.Add(1)
			go b.dispatchMessages(srv)
		}
	}
}

// Shutdown closes all connections to the servers
func (b *Bot) Shutdown() {
	for _, srv := range b.servers {
		srv.client.Close()
	}
}

// WaitForShutdown waits on someone else to call shutdown.
func (b *Bot) WaitForShutdown() {
	for _, srv := range b.servers {
		srv.client.Wait()
	}
	b.msgDispatchers.Wait()
}

// Register adds an event handler to the bot's global dispatcher.
func (b *Bot) Register(event string, handler dispatch.EventHandler) int {
	return b.dispatcher.Register(event, handler)
}

// Register adds an event handler to a server specific dispatcher.
func (b *Bot) RegisterServer(
	server string, event string, handler dispatch.EventHandler) (int, error) {

	if s, ok := b.servers[server]; ok {
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

	if s, ok := b.servers[server]; ok {
		return s.dispatcher.Unregister(event, id), nil
	}
	return false, errUnknownServerId
}

// dispatchMessages is a constant read-dispatch from the server to the
// dispatcher.
func (b *Bot) dispatchMessages(s *Server) {
	for {
		msg, ok := s.client.ReadMessage()
		if !ok {
			log.Printf(errMsgReaderClosed, s.conf.GetHost())
			break
		}
		ircMsg, err := parse.Parse(string(msg))
		if err != nil {
			log.Println(errMsgParsingIrcMessage, err)
		} else {
			b.dispatchMessage(s, ircMsg)
		}
	}
	b.msgDispatchers.Done()
}

// dispatch sends a message to both the bot's dispatcher and the given servers
func (b *Bot) dispatchMessage(s *Server, msg *irc.IrcMessage) {
	sender := ServerSender{s.conf.GetHost(), s}
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
	if err = b.createDispatcher(); err != nil {
		return nil, err
	}

	b.handler = coreHandler{b}
	b.dispatcher.Register(irc.RAW, b.handler)

	for host, srv := range conf.Servers {
		server, err := b.createServer(srv)
		if err != nil {
			return nil, err
		}
		b.servers[host] = server
	}

	return b, nil
}

// createServer creates a dispatcher, and an irc client to connect to this
// server.
func (b *Bot) createServer(conf *config.Server) (*Server, error) {
	var copyCaps irc.ProtoCaps = *b.caps
	s := &Server{
		bot:  b,
		caps: &copyCaps,
		conf: conf,
	}

	if err := s.createDispatcher(conf.GetChannels()); err != nil {
		return nil, err
	}

	return s, nil
}

// createDispatcher uses the bot's current ProtoCaps to create a dispatcher.
func (b *Bot) createDispatcher() error {
	var err error
	b.dispatcher, err = dispatch.CreateRichDispatcher(b.caps, nil)
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

	s.client = inet.CreateIrcClient(conn)
	return nil
}
