package bot

import (
	"github.com/aarondl/ultimateq/config"
	"github.com/aarondl/ultimateq/irc"
	"github.com/aarondl/ultimateq/mocks"
	"io"
	. "launchpad.net/gocheck"
	"log"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

func Test(t *testing.T) { TestingT(t) } //Hook into testing package
type s struct{}

var _ = Suite(&s{})

type testHandler struct {
	callback func(*irc.IrcMessage, irc.Sender)
}

func (h testHandler) HandleRaw(m *irc.IrcMessage, send irc.Sender) {
	if h.callback != nil {
		h.callback(m, send)
	}
}

func init() {
	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		log.Println("Could not set logger:", err)
	} else {
		log.SetOutput(f)
	}
}

var serverId = "irc.gamesurge.net"

var fakeConfig = Configure().
	Nick("nobody").
	Altnick("nobody1").
	Username("nobody").
	Userhost("bitforge.ca").
	Realname("ultimateq").
	NoReconnect(true).
	Ssl(true).
	Server(serverId)

//==================================
// Tests begin
//==================================
func (s *s) TestCreateBot(c *C) {
	bot, err := CreateBot(fakeConfig)
	c.Assert(bot, NotNil)
	c.Assert(err, IsNil)
	_, err = CreateBot(Configure())
	c.Assert(err, Equals, errInvalidConfig)
	_, err = CreateBot(ConfigureFunction(
		func(conf *config.Config) *config.Config {
			return fakeConfig
		}),
	)
	c.Assert(err, IsNil)
}

func (s *s) TestBot_StartStop(c *C) {
	conn := mocks.CreateConn()
	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	b, err := createBot(fakeConfig, nil, connProvider, false)
	c.Assert(err, IsNil)
	ers := b.Connect()
	c.Assert(len(ers), Equals, 0)
	b.Start()
	b.Start() // This shouldn't do anything, test cov

	conn.Send([]byte{}, 0, io.EOF)

	b.Stop()
	b.Disconnect()
	b.WaitForHalt()
}

func (s *s) TestBot_StartStopServer(c *C) {
	conn := mocks.CreateConn()
	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	b, err := createBot(fakeConfig, nil, connProvider, false)
	c.Assert(err, IsNil)

	srv := b.servers[serverId]
	c.Assert(srv.IsStarted(), Equals, false)
	c.Assert(srv.IsConnected(), Equals, false)

	_, err = b.ConnectServer(serverId)
	c.Assert(err, IsNil)
	c.Assert(srv.IsConnected(), Equals, true)

	_, err = b.ConnectServer(serverId)
	c.Assert(err, NotNil)

	b.StartServer(serverId)
	c.Assert(srv.IsStarted(), Equals, true)
	c.Assert(srv.IsReading(), Equals, true)
	c.Assert(srv.IsWriting(), Equals, true)

	b.StopServer(serverId)
	c.Assert(srv.IsStarted(), Equals, true)
	c.Assert(srv.IsReading(), Equals, false)
	c.Assert(srv.IsWriting(), Equals, true)

	conn.Send([]byte{}, 0, io.EOF)

	b.DisconnectServer(serverId)
	c.Assert(srv.IsConnected(), Equals, false)
	c.Assert(srv.IsWriting(), Equals, false)

	b.WaitForHalt()

	_, err = b.ConnectServer(serverId)
	conn.ResetDeath()
	c.Assert(err, IsNil)
	b.DisconnectServer(serverId)
}

func (s *s) TestBot_Reconnecting(c *C) {
	conf := Configure().Nick("nobody").Altnick("nobody1").Username("nobody").
		Userhost("bitforge.ca").Realname("ultimateq").NoReconnect(false).
		ReconnectTimeout(1).Ssl(true).Server(serverId)

	cumutex := sync.Mutex{}

	conn := mocks.CreateConn()
	waiter := sync.WaitGroup{}
	ndisc := 0

	var b *Bot
	connProvider := func(srv string) (net.Conn, error) {
		cumutex.Lock()
		defer cumutex.Unlock()
		defer waiter.Done()

		ndisc++

		switch ndisc {
		case 1:
			return conn, nil
		case 2:
			return nil, io.EOF
		case 3:
			go func() {
				b.servers[serverId].killreconn <- 0
			}()
			return conn, nil
		}

		c.Fatal("Unexpected reconnect occured.")
		return nil, nil
	}

	var err error
	b, err = createBot(conf, nil, connProvider, false)
	c.Assert(err, IsNil)
	srv := b.servers[serverId]
	srv.reconnScale = time.Microsecond

	waiter.Add(1)

	b.Connect()
	b.start(false, true)

	waiter.Wait()
	waiter.Add(1)

	conn.Send([]byte{}, 0, io.EOF)
	conn.WaitForDeath()
	conn.ResetDeath()

	waiter.Wait()
	waiter.Add(1)

	conn.Send([]byte{}, 0, io.EOF)
	conn.WaitForDeath()
	conn.ResetDeath()

	waiter.Wait()
	b.WaitForHalt()

	cumutex.Lock()
	c.Assert(ndisc, Equals, 3)
	cumutex.Unlock()
}

func (s *s) TestBot_InterruptReconnect(c *C) {
	conf := Configure().Nick("nobody").Altnick("nobody1").Username("nobody").
		Userhost("bitforge.ca").Realname("ultimateq").NoReconnect(false).
		ReconnectTimeout(1).Ssl(true).Server(serverId)

	cumutex := sync.Mutex{}

	conn := mocks.CreateConn()
	ndisc := 0
	var b *Bot
	connProvider := func(srv string) (net.Conn, error) {
		cumutex.Lock()
		defer cumutex.Unlock()

		ndisc++
		return conn, nil
	}

	var err error
	b, err = createBot(conf, nil, connProvider, false)
	c.Assert(err, IsNil)
	srv := b.servers[serverId]

	b.connectServer(srv)
	b.startServer(srv, false, true)

	conn.Send([]byte{}, 0, io.EOF)
	conn.WaitForDeath()

	c.Assert(b.InterruptReconnect(serverId), Equals, true)
	cumutex.Lock()
	c.Assert(ndisc, Equals, 1)
	cumutex.Unlock()
}

func (s *s) TestBot_Dispatching(c *C) {
	str := []byte("PRIVMSG #chan :msg\r\n#\r\n")

	conn := mocks.CreateConn()
	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	waiter := sync.WaitGroup{}
	waiter.Add(1)
	b, err := createBot(fakeConfig, nil, connProvider, false)

	b.Register(irc.PRIVMSG, &testHandler{
		func(m *irc.IrcMessage, send irc.Sender) {
			waiter.Done()
		},
	})

	c.Assert(err, IsNil)
	ers := b.Connect()
	c.Assert(len(ers), Equals, 0)
	b.start(false, true)

	conn.Send(str, len(str), io.EOF)

	waiter.Wait()
	b.Stop()
	b.WaitForHalt()
	b.Disconnect()
}

func (s *s) TestBot_Register(c *C) {
	conn := mocks.CreateConn()
	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	b, err := createBot(fakeConfig, nil, connProvider, false)
	gid := b.Register(irc.PRIVMSG, &coreHandler{})
	id, err := b.RegisterServer(serverId, irc.PRIVMSG, &coreHandler{})
	c.Assert(err, IsNil)

	c.Assert(b.Unregister(irc.PRIVMSG, id), Equals, false)
	c.Assert(b.Unregister(irc.PRIVMSG, gid), Equals, true)

	ok, err := b.UnregisterServer(serverId, irc.PRIVMSG, gid)
	c.Assert(ok, Equals, false)
	ok, err = b.UnregisterServer(serverId, irc.PRIVMSG, id)
	c.Assert(ok, Equals, true)

	_, err = b.RegisterServer("", "", &coreHandler{})
	c.Assert(err, Equals, errUnknownServerId)
	_, err = b.UnregisterServer("", "", 0)
	c.Assert(err, Equals, errUnknownServerId)
}

func (s *s) TestBot_createBot(c *C) {
	capsProvider := func() *irc.ProtoCaps {
		return irc.CreateProtoCaps()
	}
	conn := mocks.CreateConn()
	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	b, err := createBot(fakeConfig, capsProvider, connProvider, false)
	c.Assert(b, NotNil)
	c.Assert(err, IsNil)
	c.Assert(len(b.servers), Equals, 1)
	c.Assert(b.caps, NotNil)
	c.Assert(b.capsProvider, NotNil)
	c.Assert(b.connProvider, NotNil)
}

func (s *s) TestBot_Providers(c *C) {
	capsProv := func() *irc.ProtoCaps {
		p := irc.CreateProtoCaps()
		p.ParseProtoCaps(&irc.IrcMessage{Args: []string{"nick", "CHANTYPES=H"}})
		return p
	}
	connProv := func(s string) (net.Conn, error) {
		return nil, net.ErrWriteToConnected
	}

	b, err := createBot(fakeConfig, capsProv, connProv, false)
	c.Assert(err, NotNil)
	c.Assert(err, Not(Equals), net.ErrWriteToConnected)
	b, err = createBot(fakeConfig, nil, connProv, false)
	ers := b.Connect()
	c.Assert(ers[0], Equals, net.ErrWriteToConnected)
}

func (s *s) TestBot_createIrcClient(c *C) {
	b, err := createBot(fakeConfig, nil, nil, false)
	c.Assert(err, IsNil)
	ers := b.Connect()
	c.Assert(ers[0], Equals, errSslNotImplemented)
}

func (s *s) TestBot_createDispatcher(c *C) {
	_, err := createBot(fakeConfig, func() *irc.ProtoCaps {
		return nil
	}, nil, false)
	c.Assert(err, NotNil)
}
