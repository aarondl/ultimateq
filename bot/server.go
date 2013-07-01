package bot

import (
	"errors"
	"fmt"
	"github.com/aarondl/ultimateq/config"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/dispatch"
	"github.com/aarondl/ultimateq/dispatch/commander"
	"github.com/aarondl/ultimateq/inet"
	"github.com/aarondl/ultimateq/irc"
	"net"
	"strconv"
	"sync"
	"time"
)

// Server Statuses
const (
	STATUS_NEW          = byte(0x0)
	STATUS_CONNECTED    = byte(0x1)
	STATUS_READING      = byte(0x2)
	STATUS_WRITING      = byte(0x4)
	STATUS_RECONNECTING = byte(0x8)

	MASK_STARTED = STATUS_READING | STATUS_WRITING
)

const (
	// errServerAlreadyConnected occurs if a server has not been shutdown
	// before another attempt to connect to it is made.
	errFmtAlreadyConnected = "bot: %v already connected.\n"
)

var (
	// errNotConnected happens when a write occurs to a disconnected server.
	errNotConnected = errors.New("bot: Server not connected")
	// temporary error until ssl is fixed.
	errSslNotImplemented = errors.New("bot: Ssl not implemented")
)

// Server is all the details around a specific server connection. Also contains
// the connection and configuration for the specific server.
type Server struct {
	bot    *Bot
	name   string
	status byte

	// Configuration
	conf *config.Server
	caps *irc.ProtoCaps

	// Dispatching
	dispatchCore *dispatch.DispatchCore
	dispatcher   *dispatch.Dispatcher
	commander    *commander.Commander
	endpoint     *ServerEndpoint

	handlerId int
	handler   *coreHandler

	// State and Connection
	client       *inet.IrcClient
	state        *data.State
	reconnScale  time.Duration
	killdispatch chan int
	killreconn   chan int

	// protects client reading/writing
	protect sync.RWMutex

	// protects the state from reading and writing.
	protectState sync.RWMutex
}

// ServerEndpoint implements the Endpoint interface.
type ServerEndpoint struct {
	*data.DataEndpoint
	server *Server
}

// GetKey returns the server id of the current server.
func (s *ServerEndpoint) GetKey() string {
	return s.server.name
}

// Writeln writes to the server's IrcClient.
func (s *Server) Writeln(args ...interface{}) error {
	_, err := s.Write([]byte(fmt.Sprint(args...)))
	return err
}

// Write writes to the server's IrcClient.
func (s *Server) Write(buf []byte) (int, error) {
	s.protect.RLock()
	defer s.protect.RUnlock()

	if s.isConnected() {
		return s.client.Write(buf)
	}

	return 0, errNotConnected
}

// createServerEndpoint creates a ServerEndpoint with an embedded DataEndpoint.
func (s *Server) createServerEndpoint(store *data.Store, mutex *sync.RWMutex) {
	s.endpoint = &ServerEndpoint{
		DataEndpoint: data.CreateDataEndpoint(
			s.name,
			s,
			s.state,
			store,
			&s.protectState,
			mutex,
		),
		server: s,
	}
}

// createDispatcher uses the server's current ProtoCaps to create a dispatcher.
func (s *Server) createDispatching(prefix rune, channels []string) error {
	var err error
	s.dispatchCore, err = dispatch.CreateDispatchCore(s.caps, channels...)
	if err != nil {
		return err
	}
	s.dispatcher = dispatch.CreateDispatcher(s.dispatchCore)
	s.commander = commander.CreateCommander(prefix, s.dispatchCore)
	return nil
}

// createState uses the server's current ProtoCaps to create a state.
func (s *Server) createState() (err error) {
	s.state, err = data.CreateState(s.caps)
	return err
}

// createIrcClient connects to the configured server, and creates an IrcClient
// for use with that connection.
func (s *Server) createIrcClient() error {
	var conn net.Conn
	var err error

	if s.client != nil {
		return errors.New(fmt.Sprintf(errFmtAlreadyConnected, s.name))
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

	s.client = inet.CreateIrcClientFloodProtect(conn, s.name,
		int(s.conf.GetFloodProtectBurst()),
		int(s.conf.GetFloodProtectTimeout()*1000.0),
		int(s.conf.GetFloodProtectStep()*1000.0),
		time.Millisecond)
	return nil
}

// rehashProtocaps delivers updated protocaps to the server's components who
// may need it.
func (s *Server) rehashProtocaps() error {
	var err error
	if err = s.bot.mergeProtocaps(s.caps); err != nil {
		return err
	}
	if err = s.dispatcher.Protocaps(s.caps); err != nil {
		return err
	}
	s.protectState.Lock()
	if s.state != nil {
		err = s.state.Protocaps(s.caps)
		if err != nil {
			return err
		}
	}
	s.protectState.Unlock()
	return nil
}

// IsConnected checks to see if the server is connected.
func (s *Server) IsConnected() bool {
	s.protect.RLock()
	defer s.protect.RUnlock()

	return s.isConnected()
}

// isConnected checks to see if the server is connected without locking
func (s *Server) isConnected() bool {
	return STATUS_CONNECTED == s.status&STATUS_CONNECTED
}

// setConnected sets the server's connected flag.
func (s *Server) setConnected(value, lock bool) {
	if lock {
		s.protect.Lock()
		defer s.protect.Unlock()
	}

	if value {
		s.status |= STATUS_CONNECTED
	} else {
		s.status &= ^STATUS_CONNECTED
	}
}

// IsStarted checks to see if the the server is currently reading or writing.
func (s *Server) IsStarted() bool {
	s.protect.RLock()
	defer s.protect.RUnlock()

	return s.isStarted()
}

// isStarted checks to see if the the server is currently reading or writing
// without locking.
func (s *Server) isStarted() bool {
	return 0 != s.status&MASK_STARTED
}

// setStarted sets the server's reading and writing flags simultaneously.
func (s *Server) setStarted(value, lock bool) {
	if lock {
		s.protect.Lock()
		defer s.protect.Unlock()
	}

	if value {
		s.status |= MASK_STARTED
	} else {
		s.status &= ^MASK_STARTED
	}
}

// IsReading checks to see if the dispatcher read-loop is running on the server.
func (s *Server) IsReading() bool {
	s.protect.RLock()
	defer s.protect.RUnlock()

	return s.isReading()
}

// isReading checks to see if the dispatcher read-loop is running on the server
// without locking.
func (s *Server) isReading() bool {
	return STATUS_READING == s.status&STATUS_READING
}

// setReading sets the server's reading flag.
func (s *Server) setReading(value, lock bool) {
	if lock {
		s.protect.Lock()
		defer s.protect.Unlock()
	}

	if value {
		s.status |= STATUS_READING
	} else {
		s.status &= ^STATUS_READING
	}
}

// IsWriting checks to see if the server's write loop has been activated.
func (s *Server) IsWriting() bool {
	s.protect.RLock()
	defer s.protect.RUnlock()

	return s.isWriting()
}

// isWriting checks to see if the server's write loop has been activated without
// locking.
func (s *Server) isWriting() bool {
	return STATUS_WRITING == s.status&STATUS_WRITING
}

// setWriting sets the server's writing flag.
func (s *Server) setWriting(value, lock bool) {
	if lock {
		s.protect.Lock()
		defer s.protect.Unlock()
	}

	if value {
		s.status |= STATUS_WRITING
	} else {
		s.status &= ^STATUS_WRITING
	}
}

// IsReconnecting checks to see if the dispatcher is waiting to reconnect the
// server.
func (s *Server) IsReconnecting() bool {
	s.protect.RLock()
	defer s.protect.RUnlock()

	return s.isReconnecting()
}

// isReconnecting checks to see if the dispatcher is waiting to reconnect the
// without locking.
func (s *Server) isReconnecting() bool {
	return STATUS_RECONNECTING == s.status&STATUS_RECONNECTING
}

// setStarted sets the server's reconnecting flag.
func (s *Server) setReconnecting(value, lock bool) {
	if lock {
		s.protect.Lock()
		defer s.protect.Unlock()
	}

	if value {
		s.status |= STATUS_RECONNECTING
	} else {
		s.status &= ^STATUS_RECONNECTING
	}
}
