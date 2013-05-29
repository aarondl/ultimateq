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

	str := "PONG :msg\r\n"

	conn := mocks.NewMockConn(mockCtrl)
	conn.Writechan = make(chan []byte)
	gomock.InOrder(
		conn.EXPECT().Write([]byte(str)).Return(len(str), nil),
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
	err = srvsender.Writeln(str)
	<-conn.Writechan
	c.Assert(err, IsNil)
	_, err = srvsender.Write([]byte(str))
	<-conn.Writechan
	c.Assert(err, IsNil)
	b.WaitForHalt()
	b.Disconnect()
}

func (s *s) TestServer_Write(c *C) {
	mockCtrl := gomock.NewController(c)
	defer mockCtrl.Finish()

	str := "PONG :msg\r\n"

	conn := mocks.NewMockConn(mockCtrl)
	conn.Writechan = make(chan []byte)
	gomock.InOrder(
		conn.EXPECT().Write([]byte(str)).Return(len(str), nil),
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

	_, err = srv.Write([]byte{})
	c.Assert(err, Equals, errNotConnected)

	ers := b.Connect()
	c.Assert(len(ers), Equals, 0)
	b.start(true, false)

	err = srv.Writeln(str)
	<-conn.Writechan
	c.Assert(err, IsNil)
	err = b.Writeln(serverId, str)
	<-conn.Writechan
	c.Assert(err, IsNil)
	err = b.Writeln("notrealserver", str)
	c.Assert(err, NotNil)
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
