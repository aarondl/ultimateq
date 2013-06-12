package bot

import (
	"bytes"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/irc"
	"github.com/aarondl/ultimateq/mocks"
	. "launchpad.net/gocheck"
	"net"
)

func (s *s) TestServerSender(c *C) {
	b, err := createBot(fakeConfig, nil, nil, false)
	c.Check(err, IsNil)
	srv := b.servers[serverId]
	srvendpoint := createServerEndpoint(srv)
	c.Check(srvendpoint.GetKey(), Equals, serverId)
}

func (s *s) TestServerSender_OpenStore(c *C) {
	b, err := createBot(fakeConfig, nil, nil, false)
	c.Check(err, IsNil)
	srv := b.servers[serverId]
	srvendpoint := createServerEndpoint(srv)

	c.Check(srvendpoint.GetKey(), Equals, serverId)
	called := false
	reportCalled := false
	reportCalled = srvendpoint.OpenStore(func(*data.Store) {
		called = true
	})
	c.Check(called, Equals, true)
	c.Check(reportCalled, Equals, true)

	srv.store = nil
	called = false
	reportCalled = false
	reportCalled = srvendpoint.OpenStore(func(*data.Store) {
		called = true
	})
	c.Check(called, Equals, false)
	c.Check(reportCalled, Equals, false)
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

func (s *s) TestServer_Protocaps(c *C) {
	caps := irc.CreateProtoCaps()
	srv := &Server{
		caps: caps,
	}
	err := srv.createStore()
	c.Check(err, IsNil)
	err = srv.createDispatcher(nil)
	c.Check(err, IsNil)

	c.Check(srv.caps.Usermodes(), Not(Equals), "q")
	c.Check(srv.caps.Chantypes(), Not(Equals), "!")
	c.Check(srv.caps.Chanmodes(), Not(Equals), ",,,q")
	c.Check(srv.caps.Prefix(), Not(Equals), "(q)@")

	fakeCaps := &irc.ProtoCaps{}
	fakeCaps.ParseISupport(&irc.IrcMessage{Args: []string{
		"NICK", "CHANTYPES=!", "PREFIX=(q)@", "CHANMODES=,,,q",
	}})
	fakeCaps.ParseMyInfo(&irc.IrcMessage{Args: []string{
		"irc.test.net", "test-12", "q", "abc",
	}})

	err = srv.protocaps(fakeCaps)
	c.Check(err, IsNil)

	c.Check(srv.caps.Usermodes(), Equals, "q")
	c.Check(srv.caps.Chantypes(), Equals, "!")
	c.Check(srv.caps.Chanmodes(), Equals, ",,,q")
	c.Check(srv.caps.Prefix(), Equals, "(q)@")

	// Check that there's a copy
	c.Check(srv.caps, Not(Equals), fakeCaps)

	// Check errors
	fakeCaps = &irc.ProtoCaps{}
	fakeCaps.ParseISupport(&irc.IrcMessage{Args: []string{
		"NICK", "CHANTYPES=H",
	}})
	err = srv.protocaps(fakeCaps)
	c.Check(err, NotNil)

	fakeCaps = &irc.ProtoCaps{}
	fakeCaps.ParseISupport(&irc.IrcMessage{Args: []string{
		"NICK", "CHANTYPES=!",
	}})
	err = srv.protocaps(fakeCaps)
	c.Check(err, NotNil)
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
