package bot

import (
	"code.google.com/p/gomock/gomock"
	mocks "github.com/aarondl/ultimateq/inet/test"
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

func (s *s) TestBot(c *C) {
	mockCtrl := gomock.NewController(c)
	defer mockCtrl.Finish()

	conn := mocks.NewMockConn(mockCtrl)
	b, err := CreateBot(fakeConfig, func(s string) net.Conn {
		return conn
	})
	c.Assert(err, IsNil)
	c.Assert(b, NotNil)
}

func (s *s) TestBot_createDispatcher(c *C) {
	b := &Bot{caps: nil}
	err := b.createDispatcher()
	c.Assert(err, NotNil)
	b = &Bot{caps: defaultProtoCaps}
	err = b.createDispatcher()
	c.Assert(err, IsNil)
	c.Assert(b.dispatcher, NotNil)
}

func (s *s) TestBot_createIrcClient(c *C) {
	mockCtrl := gomock.NewController(c)
	defer mockCtrl.Finish()

	conn := mocks.NewMockConn(mockCtrl)
	b := &Bot{config: &fakeConfig}
	b.createIrcClient(func(s string) net.Conn {
		return conn
	})
	c.Assert(b.client, NotNil)
}
