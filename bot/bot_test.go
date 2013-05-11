package bot

import (
	"code.google.com/p/gomock/gomock"
	mocks "github.com/aarondl/ultimateq/inet/test"
	"github.com/aarondl/ultimateq/irc"
	"io"
	. "launchpad.net/gocheck"
	"log"
	"net"
	"os"
	"testing"
)

func Test(t *testing.T) { TestingT(t) } //Hook into testing package
type s struct{}

var _ = Suite(&s{})

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
	Altnick("nobody_").
	Username("nobody").
	Userhost("bitforge.ca").
	Realname("ultimateq").
	Server(serverId)

func (s *s) TestCreateBot(c *C) {
	bot, err := CreateBot(fakeConfig)
	c.Assert(bot, NotNil)
	c.Assert(err, IsNil)
}

func (s *s) TestBot_StartShutdown(c *C) {
	mockCtrl := gomock.NewController(c)
	defer mockCtrl.Finish()

	conn := mocks.NewMockConn(mockCtrl)
	conn.EXPECT().Close()

	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	b, err := createBot(fakeConfig, nil, connProvider)
	c.Assert(err, IsNil)
	ers := b.Connect()
	c.Assert(len(ers), Equals, 0)
	b.Shutdown()
	b.Start()
	b.WaitForShutdown()
}

func (s *s) TestBot_Register(c *C) {
	mockCtrl := gomock.NewController(c)
	defer mockCtrl.Finish()

	conn := mocks.NewMockConn(mockCtrl)

	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	b, err := createBot(fakeConfig, nil, connProvider)
	gid := b.Register(irc.PRIVMSG, coreHandler{})
	id, err := b.RegisterServer(serverId, irc.PRIVMSG, coreHandler{})
	c.Assert(err, IsNil)

	c.Assert(b.Unregister(irc.PRIVMSG, id), Equals, false)
	c.Assert(b.Unregister(irc.PRIVMSG, gid), Equals, true)

	ok, err := b.UnregisterServer(serverId, irc.PRIVMSG, gid)
	c.Assert(ok, Equals, false)
	ok, err = b.UnregisterServer(serverId, irc.PRIVMSG, id)
	c.Assert(ok, Equals, true)
}

func (s *s) TestcreateBot(c *C) {
	mockCtrl := gomock.NewController(c)
	defer mockCtrl.Finish()

	capsProvider := func() *irc.ProtoCaps {
		return &irc.ProtoCaps{Chantypes: "#"}
	}
	connProvider := func(srv string) (net.Conn, error) {
		return mocks.NewMockConn(mockCtrl), nil
	}

	b, err := createBot(fakeConfig, capsProvider, connProvider)
	c.Assert(b, NotNil)
	c.Assert(err, IsNil)
	c.Assert(len(b.servers), Equals, 1)
	c.Assert(b.caps, NotNil)
	c.Assert(b.capsProvider, NotNil)
	c.Assert(b.connProvider, NotNil)
}

func (s *s) TestBot_Providers(c *C) {
	capsProv := func() *irc.ProtoCaps {
		return &irc.ProtoCaps{Chantypes: "H"}
	}
	connProv := func(s string) (net.Conn, error) {
		return nil, net.ErrWriteToConnected
	}

	b, err := createBot(fakeConfig, capsProv, connProv)
	c.Assert(err, NotNil)
	c.Assert(err, Not(Equals), net.ErrWriteToConnected)
	b, err = createBot(fakeConfig, nil, connProv)
	ers := b.Connect()
	c.Assert(ers[0], Equals, net.ErrWriteToConnected)
}

type testHandler struct {
	callback func(*irc.IrcMessage, irc.Sender)
}

func (h testHandler) HandleRaw(m *irc.IrcMessage, send irc.Sender) {
	if h.callback != nil {
		h.callback(m, send)
	}
}

func (s *s) TestServerSender(c *C) {
	mockCtrl := gomock.NewController(c)
	defer mockCtrl.Finish()

	str := "PRIVMSG user :msg\r\n"

	conn := mocks.NewMockConn(mockCtrl)
	conn.EXPECT().Read(gomock.Any()).Return(0, net.ErrWriteToConnected)
	conn.EXPECT().Write([]byte(str)).Return(len(str), io.EOF)

	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	b, err := createBot(fakeConfig, nil, connProvider)
	c.Assert(err, IsNil)
	srvsender := ServerSender{serverId, b.servers[serverId]}

	ers := b.Connect()
	c.Assert(len(ers), Equals, 0)
	b.Start()
	srvsender.Writeln(str)
	//b.Shutdown()
	b.WaitForShutdown()
}
