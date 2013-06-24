package bot

import (
	"errors"
	"fmt"
	"github.com/aarondl/ultimateq/config"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/dispatch"
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
	bot        *Bot
	name       string
	status     byte
	dispatcher *dispatch.Dispatcher
	client     *inet.IrcClient
	conf       *config.Server
	caps       *irc.ProtoCaps
	state      *data.State

	reconnScale time.Duration

	killdispatch chan int
	killreconn   chan int

	handlerId int
	handler   *coreHandler

	// protects client reading/writing
	protect sync.RWMutex

	// protects the state from reading and writing.
	protectState sync.RWMutex
}

// ServerEndpoint implements the Endpoint interface.
type ServerEndpoint struct {
	*irc.Helper
	server *Server
	store  *data.Store
}

// createServerEndpoint creates a ServerEndpoint with a helper.
func createServerEndpoint(srv *Server, store *data.Store) *ServerEndpoint {
	return &ServerEndpoint{&irc.Helper{srv}, srv, store}
}

// GetKey returns the server id of the current server.
func (s *ServerEndpoint) GetKey() string {
	return s.server.name
}

// UsingState calls a callback if this ServerEndpoint can present a data state
// object. The returned boolean is whether or not the function was called.
func (s *ServerEndpoint) UsingState(fn func(*data.State)) (called bool) {
	s.server.protectState.RLock()
	defer s.server.protectState.RUnlock()
	if s.server.state != nil {
		fn(s.server.state)
		called = true
	}
	return
}

// OpenState locks the data state, and returns it. PutState must be called or
// the lock will never be released and the bot will sieze up. The state must
// be checked for nil.
func (s *ServerEndpoint) OpenState() *data.State {
	s.server.protectState.RLock()
	return s.server.state
}

// CloseState unlocks the data state after use by GetState.
func (s *ServerEndpoint) CloseState() {
	s.server.protectState.RUnlock()
}

// UsingStore calls a callback if this ServerEndpoint can present a data store
// object. The returned boolean is whether or not the function was called.
func (s *ServerEndpoint) UsingStore(fn func(*data.Store)) (called bool) {
	if s.store != nil {
		s.server.bot.protectStore.RLock()
		fn(s.store)
		s.server.bot.protectStore.RUnlock()
		called = true
	}
	return
}

// OpenStore locks the data store, and returns it. PutStore must be called or
// the lock will never be released and the bot will sieze up. The store must
// be checked for nil.
func (s *ServerEndpoint) OpenStore() *data.Store {
	s.server.bot.protectStore.RLock()
	return s.server.bot.store
}

// CloseStore unlocks the data store after use by GetStore.
func (s *ServerEndpoint) CloseStore() {
	s.server.bot.protectStore.RUnlock()
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

// createDispatcher uses the server's current ProtoCaps to create a dispatcher.
func (s *Server) createDispatcher(channels []string) (err error) {
	s.dispatcher, err = dispatch.CreateRichDispatcher(s.caps, channels)
	return err
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
