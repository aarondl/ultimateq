package bot

import (
	"code.google.com/p/gomock/gomock"
	mocks "github.com/aarondl/ultimateq/inet/test"
	"github.com/aarondl/ultimateq/irc"
	. "launchpad.net/gocheck"
	"net"
	"testing"
)

func Test(t *testing.T) { TestingT(t) } //Hook into testing package
type s struct{}

var _ = Suite(&s{})

var fakeConfig = fakeBotConfig{
	server:   "irc.gamesurge.net",
	port:     6667,
	nick:     "nobody_",
	username: "username",
	host:     "bitforge.ca",
	fullname: "nobody",
}

func (s *s) TestCreateBot(c *C) {
	c.SucceedNow()
	CreateBot(fakeConfig) // This function cannot be tested due to the socket
}

func (s *s) TestCreateBotFull(c *C) {
	mockCtrl := gomock.NewController(c)
	defer mockCtrl.Finish()

	conn := mocks.NewMockConn(mockCtrl)
	b, err := createBotFull(fakeConfig, nil, func(s string) (net.Conn, error) {
		return conn, nil
	})
	c.Assert(b, NotNil)
	c.Assert(err, IsNil)

	capsProv := func() *irc.ProtoCaps {
		return &irc.ProtoCaps{Chantypes: "H"}
	}
	connProv := func(s string) (net.Conn, error) {
		return nil, net.ErrWriteToConnected
	}
	_, err = createBotFull(fakeConfig, nil, connProv)
	c.Assert(err, Equals, net.ErrWriteToConnected)
	_, err = createBotFull(fakeConfig, capsProv, connProv)
	c.Assert(err, NotNil)
	c.Assert(err, Not(Equals), net.ErrWriteToConnected)
}

func (s *s) TestBot_createDispatcher(c *C) {
	b := &Bot{caps: nil}
	err := b.createDispatcher()
	c.Assert(err, NotNil)
	b = &Bot{caps: &irc.ProtoCaps{Chantypes: defaultChanTypes}}
	err = b.createDispatcher()
	c.Assert(err, IsNil)
	c.Assert(b.dispatcher, NotNil)
}

func (s *s) TestBot_createIrcClient(c *C) {
	mockCtrl := gomock.NewController(c)
	defer mockCtrl.Finish()

	conn := mocks.NewMockConn(mockCtrl)
	b := &Bot{config: &fakeConfig}
	b.createIrcClient(func(s string) (net.Conn, error) {
		return conn, nil
	})
	c.Assert(b.client, NotNil)

	err := b.createIrcClient(func(s string) (net.Conn, error) {
		return nil, net.ErrWriteToConnected
	})
	c.Assert(err, Equals, net.ErrWriteToConnected)
}
