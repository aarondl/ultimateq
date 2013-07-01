package bot

import (
	"github.com/aarondl/ultimateq/config"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/dispatch/commander"
	"github.com/aarondl/ultimateq/irc"
	"github.com/aarondl/ultimateq/mocks"
	"io"
	. "launchpad.net/gocheck"
	"log"
	"net"
	"os"
	"regexp/syntax"
	"sync"
	"testing"
	"time"
)

func Test(t *testing.T) { TestingT(t) } //Hook into testing package
type s struct{}

var _ = Suite(&s{})

type testHandler struct {
	callback func(*irc.IrcMessage, irc.Endpoint)
}

func (h testHandler) HandleRaw(m *irc.IrcMessage, send irc.Endpoint) {
	if h.callback != nil {
		h.callback(m, send)
	}
}

type testCommand struct {
	callback func(string, *irc.Message,
		*data.DataEndpoint, *commander.CommandData) error
}

func (h testCommand) Command(cmd string, msg *irc.Message,
	ep *data.DataEndpoint, cdata *commander.CommandData) error {

	if h.callback != nil {
		return h.callback(cmd, msg, ep, cdata)
	}
	return nil
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
	NoStore(true).
	Ssl(true).
	Server(serverId)

//==================================
// Tests begin
//==================================
func (s *s) TestCreateBot(c *C) {
	bot, err := CreateBot(fakeConfig)
	c.Check(bot, NotNil)
	c.Check(err, IsNil)
	_, err = CreateBot(Configure())
	c.Check(err, Equals, errInvalidConfig)
	_, err = CreateBot(ConfigureFunction(
		func(conf *config.Config) *config.Config {
			return fakeConfig
		}),
	)
	c.Check(err, IsNil)
}

func (s *s) TestBot_StartStop(c *C) {
	conn := mocks.CreateConn()
	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	b, err := createBot(fakeConfig, nil, connProvider, nil, false)
	c.Check(err, IsNil)
	ers := b.Connect()
	c.Check(len(ers), Equals, 0)
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

	b, err := createBot(fakeConfig, nil, connProvider, nil, false)
	c.Check(err, IsNil)

	srv := b.servers[serverId]
	c.Check(srv.IsStarted(), Equals, false)
	c.Check(srv.IsConnected(), Equals, false)

	_, err = b.ConnectServer(serverId)
	c.Check(err, IsNil)
	c.Check(srv.IsConnected(), Equals, true)

	_, err = b.ConnectServer(serverId)
	c.Check(err, NotNil)

	b.StartServer(serverId)
	c.Check(srv.IsStarted(), Equals, true)
	c.Check(srv.IsReading(), Equals, true)
	c.Check(srv.IsWriting(), Equals, true)

	b.StopServer(serverId)
	c.Check(srv.IsStarted(), Equals, true)
	c.Check(srv.IsReading(), Equals, false)
	c.Check(srv.IsWriting(), Equals, true)

	conn.Send([]byte{}, 0, io.EOF)

	b.DisconnectServer(serverId)
	c.Check(srv.IsConnected(), Equals, false)
	c.Check(srv.IsWriting(), Equals, false)

	b.WaitForHalt()

	_, err = b.ConnectServer(serverId)
	conn.ResetDeath()
	c.Check(err, IsNil)
	b.DisconnectServer(serverId)
}

func (s *s) TestBot_Reconnecting(c *C) {
	conf := Configure().Nick("nobody").Altnick("nobody1").Username("nobody").
		Userhost("bitforge.ca").Realname("ultimateq").NoReconnect(false).
		ReconnectTimeout(1).Ssl(true).NoStore(true).Server(serverId)

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
	b, err = createBot(conf, nil, connProvider, nil, false)
	c.Check(err, IsNil)
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
	c.Check(ndisc, Equals, 3)
	cumutex.Unlock()
}

func (s *s) TestBot_InterruptReconnect(c *C) {
	conf := Configure().Nick("nobody").Altnick("nobody1").Username("nobody").
		Userhost("bitforge.ca").Realname("ultimateq").NoReconnect(false).
		ReconnectTimeout(1).Ssl(true).NoStore(true).Server(serverId)

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
	b, err = createBot(conf, nil, connProvider, nil, false)
	c.Check(err, IsNil)
	srv := b.servers[serverId]

	b.connectServer(srv)
	b.startServer(srv, false, true)

	conn.Send([]byte{}, 0, io.EOF)
	conn.WaitForDeath()

	c.Check(b.InterruptReconnect(serverId), Equals, true)
	cumutex.Lock()
	c.Check(ndisc, Equals, 1)
	cumutex.Unlock()
}

func (s *s) TestBot_Dispatching(c *C) {
	str := []byte(":user!host@chan PRIVMSG chan :msg\r\n#\r\n")

	conn := mocks.CreateConn()
	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	waiter := sync.WaitGroup{}
	waiter.Add(2)
	b, err := createBot(fakeConfig, nil, connProvider, nil, false)

	b.Register(irc.PRIVMSG, &testHandler{
		func(_ *irc.IrcMessage, _ irc.Endpoint) {
			waiter.Done()
		},
	})
	err = b.RegisterCommand("msg", &testCommand{
		func(_ string, _ *irc.Message, _ *data.DataEndpoint,
			_ *commander.CommandData) error {
			waiter.Done()
			return nil
		},
	}, commander.PRIVMSG, commander.PRIVATE)

	c.Check(err, IsNil)
	ers := b.Connect()
	c.Check(len(ers), Equals, 0)
	b.start(false, true)

	conn.Send(str, len(str), io.EOF)

	waiter.Wait()
	b.Stop()
	b.WaitForHalt()
	b.Disconnect()

	b.UnregisterCommand("msg")
}

func (s *s) TestBot_Register(c *C) {
	b, err := createBot(fakeConfig, nil, nil, nil, false)
	gid := b.Register(irc.PRIVMSG, &coreHandler{})
	id, err := b.RegisterServer(serverId, irc.PRIVMSG, &coreHandler{})
	c.Check(err, IsNil)

	c.Check(b.Unregister(irc.PRIVMSG, id), Equals, false)
	c.Check(b.Unregister(irc.PRIVMSG, gid), Equals, true)

	ok, err := b.UnregisterServer(serverId, irc.PRIVMSG, gid)
	c.Check(ok, Equals, false)
	ok, err = b.UnregisterServer(serverId, irc.PRIVMSG, id)
	c.Check(ok, Equals, true)

	_, err = b.RegisterServer("", "", &coreHandler{})
	c.Check(err, Equals, errUnknownServerId)
	_, err = b.UnregisterServer("", "", 0)
	c.Check(err, Equals, errUnknownServerId)
}

func (s *s) TestBot_RegisterCommand(c *C) {
	var err error
	var success bool
	b, err := createBot(fakeConfig, nil, nil, nil, false)
	cmd := "cmd"
	err = b.RegisterCommand(cmd, &testCommand{}, commander.ALL,
		commander.ALL)

	err = b.RegisterCommand(cmd, &testCommand{}, commander.ALL,
		commander.ALL)
	c.Check(err, NotNil) // Duplicate
	success = b.UnregisterCommand(cmd)
	c.Check(success, Equals, true)

	err = b.RegisterAuthedCommand(cmd, &testCommand{}, commander.ALL,
		commander.ALL, 100, "a")
	c.Check(err, IsNil)
	success = b.UnregisterCommand(cmd)
	c.Check(success, Equals, true)

	err = b.RegisterServerCommand(serverId, cmd, &testCommand{},
		commander.ALL, commander.ALL)
	c.Check(err, IsNil)
	success = b.UnregisterServerCommand(serverId, cmd)
	c.Check(success, Equals, true)

	err = b.RegisterAuthedServerCommand(serverId, cmd, &testCommand{},
		commander.ALL, commander.ALL, 100, "a")
	c.Check(err, IsNil)
	success = b.UnregisterServerCommand(serverId, cmd)
	c.Check(success, Equals, true)

	err = b.RegisterServerCommand("badServer", cmd, &testCommand{},
		commander.ALL, commander.ALL)
	c.Check(err, Equals, errUnknownServerId)
	err = b.RegisterAuthedServerCommand("badServer", cmd, &testCommand{},
		commander.ALL, commander.ALL, 100, "a")
	c.Check(err, Equals, errUnknownServerId)

	success = b.UnregisterServerCommand("badServer", cmd)
}

func (s *s) TestBot_createBot(c *C) {
	capsProvider := func() *irc.ProtoCaps {
		return irc.CreateProtoCaps()
	}
	conn := mocks.CreateConn()
	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	b, err := createBot(fakeConfig, capsProvider, connProvider, nil, false)
	c.Check(b, NotNil)
	c.Check(err, IsNil)
	c.Check(len(b.servers), Equals, 1)
	c.Check(b.caps, NotNil)
	c.Check(b.capsProvider, NotNil)
	c.Check(b.connProvider, NotNil)
}

func (s *s) TestBot_createServer(c *C) {
	b, err := createBot(fakeConfig, nil, nil, nil, true)
	c.Check(err, IsNil)
	srv := b.servers[serverId]
	c.Check(srv.dispatcher, NotNil)
	c.Check(srv.commander, NotNil)
	c.Check(srv.dispatchCore, NotNil)
	c.Check(srv.state, NotNil)
	c.Check(srv.handler, NotNil)

	cnf := fakeConfig.Clone()
	cnf.GlobalContext().NoState(true)
	b, err = createBot(cnf, nil, nil, nil, false)
	c.Check(err, IsNil)
	srv = b.servers[serverId]
	c.Check(srv.dispatcher, NotNil)
	c.Check(srv.state, IsNil)
	c.Check(srv.handler, IsNil)
}

func (s *s) TestBot_Providers(c *C) {
	storeConf1 := fakeConfig.Clone().GlobalContext().NoStore(false)
	storeConf2 := storeConf1.Clone().ServerContext(serverId).NoStore(false)
	storeConf3 := storeConf1.Clone().ServerContext(serverId).NoStore(true)

	capsProv := func() *irc.ProtoCaps {
		p := irc.CreateProtoCaps()
		p.ParseISupport(&irc.IrcMessage{Args: []string{"nick", "CHANTYPES=H"}})
		return p
	}
	badConnProv := func(s string) (net.Conn, error) {
		return nil, net.ErrWriteToConnected
	}
	goodConnProv := func(s string) (net.Conn, error) {
		return mocks.CreateConn(), nil
	}
	badStoreProv := func(s string) (*data.Store, error) {
		return nil, io.EOF
	}

	b, err := createBot(
		fakeConfig, capsProv, goodConnProv, badStoreProv, false)
	c.Check(err, FitsTypeOf, &syntax.Error{})

	b, err = createBot(fakeConfig, nil, badConnProv, badStoreProv, false)
	ers := b.Connect()
	c.Check(ers[0], Equals, net.ErrWriteToConnected)

	b, err = createBot(fakeConfig, nil, nil, badStoreProv, false)
	c.Check(err, IsNil)
	b, err = createBot(storeConf1, nil, nil, badStoreProv, false)
	c.Check(err, Equals, io.EOF)
	b, err = createBot(storeConf2, nil, nil, badStoreProv, false)
	c.Check(err, Equals, io.EOF)
	b, err = createBot(storeConf3, nil, nil, badStoreProv, false)
	c.Check(err, IsNil)
}

func (s *s) TestBot_createIrcClient(c *C) {
	b, err := createBot(fakeConfig, nil, nil, nil, false)
	c.Check(err, IsNil)
	ers := b.Connect()
	c.Check(ers[0], Equals, errSslNotImplemented)
}

func (s *s) TestBot_createDispatcher(c *C) {
	_, err := createBot(fakeConfig, func() *irc.ProtoCaps {
		return nil
	}, nil, nil, false)
	c.Check(err, NotNil)
}
