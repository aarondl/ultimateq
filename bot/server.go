package bot

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/aarondl/ultimateq/config"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/dispatch"
	"github.com/aarondl/ultimateq/dispatch/commander"
	"github.com/aarondl/ultimateq/inet"
	"github.com/aarondl/ultimateq/irc"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

// Server Statuses
const (
	STATUS_NEW          = byte(0x0)
	STATUS_CONNECTING   = byte(0x1)
	STATUS_CONNECTED    = byte(0x2)
	STATUS_STARTED      = byte(0x4)
	STATUS_RECONNECTING = byte(0x8)
)

const (
	// errServerAlreadyConnected occurs if a server has not been shutdown
	// before another attempt to connect to it is made.
	errFmtAlreadyConnected = "bot: %v already connected.\n"
)

var (
	// errNotConnected happens when a write occurs to a disconnected server.
	errNotConnected = errors.New("bot: Server not connected")
	// errFailedToLoadCertificate happens when we fail to parse the certificate
	errFailedToLoadCertificate = errors.New("bot: Failed to load certificate")
	// errKilledConn happens when the server is killed mid-connect.
	errKilledConn = errors.New("bot: Killed trying to connect.")
)

// connResult is used to return results from the channel patterns in
// createIrcClient
type connResult struct {
	conn net.Conn
	err error
}

// certReader is for IoC of the createTlsConfig function.
type certReader func(string) (*x509.CertPool, error)

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

	handlerID int
	handler   *coreHandler

	// State and Connection
	client       *inet.IrcClient
	state        *data.State
	reconnScale  time.Duration
	kill chan int
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
	if s.client != nil {
		return fmt.Errorf(errFmtAlreadyConnected, s.name)
	}

	var result *connResult
	resultService := make(chan chan *connResult)
	resultChan := make(chan *connResult)

	go s.createConnection(resultService)

	select {
	case resultService <- resultChan:
		result = <-resultChan
		if result.err != nil {
			return result.err
		}
	case <-s.kill:
		close(resultService)
		return errKilledConn
	}

	s.client = inet.CreateIrcClient(result.conn, s.name,
		int(s.conf.GetFloodLenPenalty()),
		time.Duration(s.conf.GetFloodTimeout()*1000.0)*time.Millisecond,
		time.Duration(s.conf.GetFloodStep()*1000.0)*time.Millisecond,
		time.Duration(s.conf.GetKeepAlive())*time.Second,
		time.Second)
	return nil
}

// createConnection creates a connection based off the server receiver's
// config variables. It takes a chan of channels to return the result on.
// If the channel is closed before it can send it's result, it will close the
// connection automatically.
func (s *Server) createConnection(resultService chan chan *connResult) {
	r := &connResult{}
	port := strconv.Itoa(int(s.conf.GetPort()))
	server := s.conf.GetHost() + ":" + port

	if s.bot.connProvider == nil {
		if s.conf.GetSsl() {
			var conf *tls.Config
			conf, r.err = s.createTlsConfig(readCert)
			if r.err != nil {
				r.conn, r.err = tls.Dial("tcp", server, conf)
			}
		} else {
			r.conn, r.err = net.Dial("tcp", server)
		}
	} else {
		r.conn, r.err = s.bot.connProvider(server)
	}

	if resultChan, ok := <-resultService; ok {
		resultChan <- r
	} else {
		if r.conn != nil {
			r.conn.Close()
		}
	}
}

// createTlsConfig creates a tls config appropriate for the 
func (s *Server) createTlsConfig(cr certReader) (conf *tls.Config, err error) {
	conf = &tls.Config{}
	conf.InsecureSkipVerify = s.conf.GetNoVerifyCert()

	if len(s.conf.GetSslCert()) > 0 {
		conf.RootCAs, err = cr(s.conf.GetSslCert())
	}

	return
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

// IsConnecting checks to see if the server is connecting.
func (s *Server) IsConnecting() bool {
	s.protect.RLock()
	defer s.protect.RUnlock()

	return s.isConnecting()
}

// isConnecting checks to see if the server is connecting without locking
func (s *Server) isConnecting() bool {
	return STATUS_CONNECTED == s.status&STATUS_CONNECTED
}

// setConnecting sets the server's connecting flag.
func (s *Server) setConnecting(value, lock bool) {
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
	return 0 != s.status&STATUS_STARTED
}

// setStarted sets the server's reading and writing flags simultaneously.
func (s *Server) setStarted(value, lock bool) {
	if lock {
		s.protect.Lock()
		defer s.protect.Unlock()
	}

	if value {
		s.status |= STATUS_STARTED
	} else {
		s.status &= ^STATUS_STARTED
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

// readCert returns a CertPool containing the client certificate specified
// in filename.
func readCert(filename string) (certpool *x509.CertPool, err error) {
	var pem []byte
	var file *os.File

	if file, err = os.Open(filename); err != nil {
		return
	}

	defer file.Close()

	pem, err = ioutil.ReadAll(file)
	if err != nil {
		return
	}

	certpool = x509.NewCertPool()
	ok := certpool.AppendCertsFromPEM(pem)
	if !ok {
		err = errFailedToLoadCertificate
	}
	return
}
