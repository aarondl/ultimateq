package bot

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"sync"
	"time"

	"github.com/aarondl/ultimateq/config"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/inet"
	"github.com/aarondl/ultimateq/irc"
	"github.com/pkg/errors"
	"gopkg.in/inconshreveable/log15.v2"
)

// Status is the status of a network connection.
type Status byte

// Server Statuses
const (
	STATUS_STOPPED Status = iota
	STATUS_CONNECTING
	STATUS_STARTED
	STATUS_RECONNECTING
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
	// errServerKilledConn happens when the server is killed mid-connect.
	errServerKilledConn = errors.New("bot: Killed trying to connect")
)

// connResult is used to return results from the channel patterns in
// createIrcClient
type connResult struct {
	conn      net.Conn
	temporary bool
	err       error
}

// certReader is for IoC of the createTLSConfig function.
type certReader func(string) (*x509.CertPool, error)

// Server is all the details around a specific server connection. Also contains
// the connection and configuration for the specific server.
type Server struct {
	bot       *Bot
	networkID string

	log15.Logger

	// Status
	protectStatus   sync.RWMutex
	status          Status
	statusListeners [][]chan Status

	// Configuration
	conf    *config.Config
	netInfo *irc.NetworkInfo

	// Dispatching
	writer irc.Writer

	handlerID uint64
	handler   *coreHandler

	// Network connection
	protectClient sync.RWMutex
	client        *inet.IrcClient

	// State DB and connection state
	state       *data.State
	started     bool
	serverIndex int
	reconnScale time.Duration
	killable    chan int
}

// Write writes to the server's IrcClient.
func (s *Server) Write(buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}
	s.protectStatus.RLock()
	defer s.protectStatus.RUnlock()

	if s.GetStatus() != STATUS_STOPPED {
		return s.client.Write(buf)
	}

	return 0, errNotConnected
}

// createState uses the server's current netInfo to create a state.
func (s *Server) createState() (err error) {
	s.state, err = data.NewState(s.netInfo)
	return err
}

// createIrcClient connects to the configured server, and creates an IrcClient
// for use with that connection.
func (s *Server) createIrcClient() (error, bool) {
	if s.client != nil {
		return fmt.Errorf(errFmtAlreadyConnected, s.networkID), false
	}

	var result *connResult
	resultService := make(chan chan *connResult)
	resultChan := make(chan *connResult)

	go s.createConnection(resultService)

	select {
	case resultService <- resultChan:
		result = <-resultChan
		if result.err != nil {
			return result.err, result.temporary
		}
	case <-s.killable:
		close(resultService)
		return errServerKilledConn, false
	}

	cfg := s.conf.Network(s.networkID)
	floodPenalty, _ := cfg.FloodLenPenalty()
	floodTimeout, _ := cfg.FloodTimeout()
	floodStep, _ := cfg.FloodStep()
	keepAlive, _ := cfg.KeepAlive()

	s.protectClient.Lock()
	s.client = inet.NewIrcClient(
		result.conn,
		s.Logger,
		int(floodPenalty),
		time.Duration(floodTimeout)*time.Second,
		time.Duration(floodStep)*time.Second,
		time.Duration(keepAlive)*time.Second,
		time.Second,
	)
	s.protectClient.Unlock()
	return nil, false
}

// createConnection creates a connection based off the server receiver's
// config variables. It takes a chan of channels to return the result on.
// If the channel is closed before it can send it's result, it will close the
// connection automatically.
func (s *Server) createConnection(resultService chan chan *connResult) {
	r := &connResult{}

	cfg := s.conf.Network(s.networkID)
	srvs, _ := cfg.Servers()
	if s.serverIndex >= len(srvs) {
		s.serverIndex = 0
	}

	server := srvs[s.serverIndex]
	s.Info("Connecting", "host", server)

	if s.bot.connProvider == nil {
		if tlsConfig, err := s.createTLSConfig(); err != nil {
			r.err = err
		} else if tlsConfig != nil {
			r.conn, r.err = tls.Dial("tcp", server, tlsConfig)
		} else {
			r.conn, r.err = net.Dial("tcp", server)
		}
	} else {
		r.conn, r.err = s.bot.connProvider(server)
	}

	if r.err != nil {
		s.Error("Failed to connect", "host", server)
		if e, ok := r.err.(net.Error); ok {
			r.temporary = e.Temporary()
		} else {
			r.temporary = false
		}
		s.serverIndex++
	}

	if resultChan, ok := <-resultService; ok {
		resultChan <- r
	} else {
		if r.conn != nil {
			r.conn.Close()
		}
	}
}

// createTLSConfig creates a tls config appropriate for the
func (s *Server) createTLSConfig() (*tls.Config, error) {
	cfg := s.conf.Network(s.networkID)

	doTLS, _ := cfg.TLS()
	if !doTLS {
		return nil, nil
	}

	ca, caOk := cfg.TLSCACert()
	cert, certOk := cfg.TLSCert()
	key, keyOk := cfg.TLSKey()
	insecure, _ := cfg.TLSInsecureSkipVerify()

	conf := new(tls.Config)

	if caOk {
		caCertBytes, err := ioutil.ReadFile(ca)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read ca cert")
		}

		certPool := new(x509.CertPool)
		certPool.AppendCertsFromPEM(caCertBytes)
		conf.RootCAs = certPool
	}

	if certOk && keyOk {
		certificate, err := tls.LoadX509KeyPair(cert, key)
		if err != nil {
			return nil, err
		}

		conf.Certificates = append(conf.Certificates, certificate)
	}

	if insecure {
		conf.InsecureSkipVerify = true
	}

	return conf, nil
}

// Close shuts down the connection and returns.
func (s *Server) Close() (err error) {
	s.protectClient.Lock()
	defer s.protectClient.Unlock()

	if s.client != nil {
		err = s.client.Close()
	}
	s.client = nil
	return
}

// rehashNetworkInfo delivers updated information to the server's components who
// may need it.
func (s *Server) rehashNetworkInfo() error {
	var err error
	if s.state != nil {
		err = s.state.SetNetworkInfo(s.netInfo)
		if err != nil {
			return err
		}
	}
	return nil
}

// setStatus safely sets the status of the server and notifies any listeners.
func (s *Server) setStatus(newstatus Status) {
	s.protectStatus.Lock()
	defer s.protectStatus.Unlock()

	s.status = newstatus
	if s.statusListeners == nil {
		return
	}
	for _, listener := range s.statusListeners[0] {
		listener <- s.status
	}
	i := byte(newstatus) + 1
	for _, listener := range s.statusListeners[i] {
		listener <- s.status
	}
}

// addStatusListener adds a listener for status changes.
func (s *Server) addStatusListener(listener chan Status, listen ...Status) {
	s.protectStatus.Lock()
	defer s.protectStatus.Unlock()

	if s.statusListeners == nil {
		s.statusListeners = [][]chan Status{
			make([]chan Status, 0),
			make([]chan Status, 0),
			make([]chan Status, 0),
			make([]chan Status, 0),
			make([]chan Status, 0),
		}
	}

	if len(listen) == 0 {
		s.statusListeners[0] = append(s.statusListeners[0], listener)
	} else {
		for _, st := range listen {
			i := byte(st) + 1
			s.statusListeners[i] = append(s.statusListeners[i], listener)
		}
	}
}

// GetStatus safely gets the status of the server.
func (s *Server) GetStatus() Status {
	s.protectStatus.RLock()
	defer s.protectStatus.RUnlock()

	return s.status
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
