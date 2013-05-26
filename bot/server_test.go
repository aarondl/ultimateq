package bot

import (
	"code.google.com/p/gomock/gomock"
	mocks "github.com/aarondl/ultimateq/inet/test"
	"github.com/aarondl/ultimateq/irc"
	. "launchpad.net/gocheck"
	"net"
)

func (s *s) TestServerSender(c *C) {
	mockCtrl := gomock.NewController(c)
	defer mockCtrl.Finish()

	str := "PRIVMSG user :msg\r\n"

	conn := mocks.NewMockConn(mockCtrl)
	conn.Writechan = make(chan []byte)
	gomock.InOrder(
		conn.EXPECT().Write([]byte(str)).Return(len(str), nil),
		conn.EXPECT().Close(),
	)

	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	b, err := createBot(fakeConfig, nil, connProvider)
	c.Assert(err, IsNil)
	srv := b.servers[serverId]
	srv.dispatcher.Unregister(irc.RAW, srv.handlerId)
	srvsender := ServerSender{serverId, srv}
	c.Assert(srvsender.GetKey(), Equals, serverId)

	ers := b.Connect()
	c.Assert(len(ers), Equals, 0)
	b.start(true, false)
	srvsender.Writeln(str)
	<-conn.Writechan
	b.WaitForHalt()
	b.Disconnect()
}

func (s *s) TestServer_State(c *C) {
	srv := &Server{}
	srv.setStarted(false)
	c.Assert(srv.IsStarted(), Equals, true)
	srv.setStopped(false)
	c.Assert(srv.IsStarted(), Equals, false)

	srv.setStarted(true)
	c.Assert(srv.IsStarted(), Equals, true)
	srv.setStopped(true)
	c.Assert(srv.IsStarted(), Equals, false)

	srv.setConnected(true)
	c.Assert(srv.IsConnected(), Equals, true)
	srv.setDisconnected(true)
	c.Assert(srv.IsConnected(), Equals, false)

	srv.setConnected(true)
	c.Assert(srv.IsConnected(), Equals, true)
	srv.setDisconnected(true)
	c.Assert(srv.IsConnected(), Equals, false)

	srv.setReconnecting(true)
	c.Assert(srv.IsReconnecting(), Equals, true)
	srv.setNotReconnecting(true)
	c.Assert(srv.IsReconnecting(), Equals, false)

	srv.setReconnecting(true)
	c.Assert(srv.IsReconnecting(), Equals, true)
	srv.setNotReconnecting(true)
	c.Assert(srv.IsReconnecting(), Equals, false)
}
