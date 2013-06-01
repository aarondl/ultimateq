package bot

import (
	"errors"
	"fmt"
	"github.com/aarondl/ultimateq/config"
	"github.com/aarondl/ultimateq/dispatch"
	"github.com/aarondl/ultimateq/inet"
	"github.com/aarondl/ultimateq/irc"
	"net"
	"strconv"
	"sync"
	"time"
)

// Server States
const (
	STATE_NEW          = 0x0
	STATE_CONNECTED    = 0x1
	STATE_READING      = 0x2
	STATE_WRITING      = 0x4
	STATE_RECONNECTING = 0x8

	MASK_STARTED = STATE_READING | STATE_WRITING
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
	bot        *Bot
	name       string
	state      int
	dispatcher *dispatch.Dispatcher
	client     *inet.IrcClient
	conf       *config.Server
	caps       *irc.ProtoCaps

	reconnScale time.Duration

	killdispatch chan int
	killreconn   chan int

	handlerId int
	handler   *coreHandler

	// protects client reading/writing
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
	_, err := s.server.Write([]byte(str))
	return err
}

// Write writes to the ServerSender's IrcClient.
func (s ServerSender) Write(buf []byte) (int, error) {
	return s.server.Write(buf)
}

// Writeln writes to the server's IrcClient.
func (s *Server) Writeln(str string) error {
	_, err := s.Write([]byte(str))
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

	s.client = inet.CreateIrcClient(conn, s.name)
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
	return STATE_CONNECTED == s.state&STATE_CONNECTED
}

// setConnected sets the server's connected flag.
func (s *Server) setConnected(value, lock bool) {
	if lock {
		s.protect.Lock()
		defer s.protect.Unlock()
	}

	if value {
		s.state |= STATE_CONNECTED
	} else {
		s.state &= ^STATE_CONNECTED
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
	return 0 != s.state&MASK_STARTED
}

// setStarted sets the server's reading and writing flags simultaneously.
func (s *Server) setStarted(value, lock bool) {
	if lock {
		s.protect.Lock()
		defer s.protect.Unlock()
	}

	if value {
		s.state |= MASK_STARTED
	} else {
		s.state &= ^MASK_STARTED
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
	return STATE_READING == s.state&STATE_READING
}

// setReading sets the server's reading flag.
func (s *Server) setReading(value, lock bool) {
	if lock {
		s.protect.Lock()
		defer s.protect.Unlock()
	}

	if value {
		s.state |= STATE_READING
	} else {
		s.state &= ^STATE_READING
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
	return STATE_WRITING == s.state&STATE_WRITING
}

// setWriting sets the server's writing flag.
func (s *Server) setWriting(value, lock bool) {
	if lock {
		s.protect.Lock()
		defer s.protect.Unlock()
	}

	if value {
		s.state |= STATE_WRITING
	} else {
		s.state &= ^STATE_WRITING
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
	return STATE_RECONNECTING == s.state&STATE_RECONNECTING
}

// setStarted sets the server's reconnecting flag.
func (s *Server) setReconnecting(value, lock bool) {
	if lock {
		s.protect.Lock()
		defer s.protect.Unlock()
	}

	if value {
		s.state |= STATE_RECONNECTING
	} else {
		s.state &= ^STATE_RECONNECTING
	}
}
