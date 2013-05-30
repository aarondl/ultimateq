package bot

import (
	"bytes"
	"github.com/aarondl/ultimateq/irc"
	"github.com/aarondl/ultimateq/mocks"
	. "launchpad.net/gocheck"
	"net"
)

func (s *s) TestServerSender(c *C) {
	str := "PONG :msg\r\n"

	conn := mocks.CreateConn()
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
	c.Assert(bytes.Compare(conn.Receive(len(str), nil), []byte(str)), Equals, 0)
	c.Assert(err, IsNil)
	_, err = srvsender.Write([]byte(str))
	c.Assert(bytes.Compare(conn.Receive(len(str), nil), []byte(str)), Equals, 0)
	c.Assert(err, IsNil)
	b.WaitForHalt()
	b.Disconnect()
}

func (s *s) TestServer_Write(c *C) {
	str := "PONG :msg\r\n"

	conn := mocks.CreateConn()
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
	c.Assert(bytes.Compare(conn.Receive(len(str), nil), []byte(str)), Equals, 0)
	c.Assert(err, IsNil)
	_, err = srv.Write([]byte(str))
	c.Assert(bytes.Compare(conn.Receive(len(str), nil), []byte(str)), Equals, 0)
	c.Assert(err, IsNil)
	err = b.Writeln("notrealserver", str)
	c.Assert(err, NotNil)
	b.WaitForHalt()
	b.Disconnect()
}

func (s *s) TestServer_State(c *C) {
	srv := &Server{}

	srv.setStarted(true, false)
	c.Assert(srv.IsStarted(), Equals, true)
	srv.setStarted(false, false)
	c.Assert(srv.IsStarted(), Equals, false)

	srv.setStarted(true, true)
	c.Assert(srv.IsStarted(), Equals, true)
	srv.setStarted(false, true)
	c.Assert(srv.IsStarted(), Equals, false)

	srv.setReading(true, false)
	c.Assert(srv.IsReading(), Equals, true)
	c.Assert(srv.IsStarted(), Equals, true)
	srv.setReading(false, false)
	c.Assert(srv.IsReading(), Equals, false)
	c.Assert(srv.IsStarted(), Equals, false)

	srv.setReading(true, true)
	c.Assert(srv.IsReading(), Equals, true)
	c.Assert(srv.IsStarted(), Equals, true)
	srv.setReading(false, true)
	c.Assert(srv.IsReading(), Equals, false)
	c.Assert(srv.IsStarted(), Equals, false)

	srv.setWriting(true, false)
	c.Assert(srv.IsWriting(), Equals, true)
	c.Assert(srv.IsStarted(), Equals, true)
	srv.setWriting(false, false)
	c.Assert(srv.IsWriting(), Equals, false)
	c.Assert(srv.IsStarted(), Equals, false)

	srv.setWriting(true, true)
	c.Assert(srv.IsWriting(), Equals, true)
	c.Assert(srv.IsStarted(), Equals, true)
	srv.setWriting(false, true)
	c.Assert(srv.IsWriting(), Equals, false)
	c.Assert(srv.IsStarted(), Equals, false)

	srv.setConnected(true, false)
	c.Assert(srv.IsConnected(), Equals, true)
	srv.setConnected(false, false)
	c.Assert(srv.IsConnected(), Equals, false)

	srv.setConnected(true, true)
	c.Assert(srv.IsConnected(), Equals, true)
	srv.setConnected(false, true)
	c.Assert(srv.IsConnected(), Equals, false)

	srv.setReconnecting(true, false)
	c.Assert(srv.IsReconnecting(), Equals, true)
	srv.setReconnecting(false, false)
	c.Assert(srv.IsReconnecting(), Equals, false)

	srv.setReconnecting(true, true)
	c.Assert(srv.IsReconnecting(), Equals, true)
	srv.setReconnecting(false, true)
	c.Assert(srv.IsReconnecting(), Equals, false)
}
