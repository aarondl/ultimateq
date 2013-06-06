package bot

import (
	"bytes"
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

	b, err := createBot(fakeConfig, nil, connProvider, false)
	c.Check(err, IsNil)
	srv := b.servers[serverId]
	srvsender := ServerSender{serverId, srv}
	c.Check(srvsender.GetKey(), Equals, serverId)

	ers := b.Connect()
	c.Check(len(ers), Equals, 0)
	b.start(true, false)
	err = srvsender.Writeln(str)
	c.Check(bytes.Compare(conn.Receive(len(str), nil), []byte(str)), Equals, 0)
	c.Check(err, IsNil)
	_, err = srvsender.Write([]byte(str))
	c.Check(bytes.Compare(conn.Receive(len(str), nil), []byte(str)), Equals, 0)
	c.Check(err, IsNil)
	b.WaitForHalt()
	b.Disconnect()
}

func (s *s) TestServer_Write(c *C) {
	str := "PONG :msg\r\n"

	conn := mocks.CreateConn()
	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	b, err := createBot(fakeConfig, nil, connProvider, false)
	c.Check(err, IsNil)
	srv := b.servers[serverId]

	_, err = srv.Write([]byte{})
	c.Check(err, Equals, errNotConnected)

	ers := b.Connect()
	c.Check(len(ers), Equals, 0)
	b.start(true, false)

	err = srv.Writeln(str)
	c.Check(bytes.Compare(conn.Receive(len(str), nil), []byte(str)), Equals, 0)
	c.Check(err, IsNil)
	_, err = srv.Write([]byte(str))
	c.Check(bytes.Compare(conn.Receive(len(str), nil), []byte(str)), Equals, 0)
	c.Check(err, IsNil)
	err = b.Writeln("notrealserver", str)
	c.Check(err, NotNil)
	b.WaitForHalt()
	b.Disconnect()
}

func (s *s) TestServer_State(c *C) {
	srv := &Server{}

	srv.setStarted(true, false)
	c.Check(srv.IsStarted(), Equals, true)
	srv.setStarted(false, false)
	c.Check(srv.IsStarted(), Equals, false)

	srv.setStarted(true, true)
	c.Check(srv.IsStarted(), Equals, true)
	srv.setStarted(false, true)
	c.Check(srv.IsStarted(), Equals, false)

	srv.setReading(true, false)
	c.Check(srv.IsReading(), Equals, true)
	c.Check(srv.IsStarted(), Equals, true)
	srv.setReading(false, false)
	c.Check(srv.IsReading(), Equals, false)
	c.Check(srv.IsStarted(), Equals, false)

	srv.setReading(true, true)
	c.Check(srv.IsReading(), Equals, true)
	c.Check(srv.IsStarted(), Equals, true)
	srv.setReading(false, true)
	c.Check(srv.IsReading(), Equals, false)
	c.Check(srv.IsStarted(), Equals, false)

	srv.setWriting(true, false)
	c.Check(srv.IsWriting(), Equals, true)
	c.Check(srv.IsStarted(), Equals, true)
	srv.setWriting(false, false)
	c.Check(srv.IsWriting(), Equals, false)
	c.Check(srv.IsStarted(), Equals, false)

	srv.setWriting(true, true)
	c.Check(srv.IsWriting(), Equals, true)
	c.Check(srv.IsStarted(), Equals, true)
	srv.setWriting(false, true)
	c.Check(srv.IsWriting(), Equals, false)
	c.Check(srv.IsStarted(), Equals, false)

	srv.setConnected(true, false)
	c.Check(srv.IsConnected(), Equals, true)
	srv.setConnected(false, false)
	c.Check(srv.IsConnected(), Equals, false)

	srv.setConnected(true, true)
	c.Check(srv.IsConnected(), Equals, true)
	srv.setConnected(false, true)
	c.Check(srv.IsConnected(), Equals, false)

	srv.setReconnecting(true, false)
	c.Check(srv.IsReconnecting(), Equals, true)
	srv.setReconnecting(false, false)
	c.Check(srv.IsReconnecting(), Equals, false)

	srv.setReconnecting(true, true)
	c.Check(srv.IsReconnecting(), Equals, true)
	srv.setReconnecting(false, true)
	c.Check(srv.IsReconnecting(), Equals, false)
}
