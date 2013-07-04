package bot

import (
	"bytes"
	"fmt"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/irc"
	. "launchpad.net/gocheck"
	"net"
)

//===================================================================
// Fixtures for basic responses as well as full bot required messages
//===================================================================
type testPoint struct {
	*irc.Helper
	buf *bytes.Buffer
	srv *Server
}

func makeTestPoint(srv *Server) *testPoint {
	buf := &bytes.Buffer{}
	t := &testPoint{&irc.Helper{buf}, buf, srv}
	return t
}

func (t *testPoint) gets() string {
	return string(t.buf.Bytes())
}

func (t *testPoint) resetTestWritten() {
	t.buf.Reset()
}

func (t *testPoint) GetKey() string {
	return serverId
}

//==============
// Tests
//==============
func (s *s) TestCoreHandler_Ping(c *C) {
	handler := coreHandler{}
	msg := &irc.IrcMessage{
		Name: irc.PING,
		Args: []string{"123123123123"},
	}
	endpoint := makeTestPoint(nil)
	handler.HandleRaw(msg, endpoint)
	c.Check(endpoint.gets(), Equals, irc.PONG+" :"+msg.Args[0])
}

func (s *s) TestCoreHandler_Connect(c *C) {
	b, err := createBot(fakeConfig, nil, nil, nil, false, false)
	c.Check(err, IsNil)
	cnf := fakeConfig.GetServer(serverId)
	handler := coreHandler{bot: b}
	msg1 := fmt.Sprintf("NICK :%v", cnf.GetNick())
	msg2 := fmt.Sprintf("USER %v 0 * :%v",
		cnf.GetUsername(), cnf.GetRealname())

	msg := &irc.IrcMessage{Name: irc.CONNECT}
	endpoint := makeTestPoint(b.servers[serverId])
	handler.HandleRaw(msg, endpoint)
	c.Check(endpoint.gets(), Equals, msg1+msg2)
}

func (s *s) TestCoreHandler_Nick(c *C) {
	b, err := createBot(fakeConfig, nil, nil, nil, false, false)
	c.Check(err, IsNil)
	cnf := fakeConfig.GetServer(serverId)
	handler := coreHandler{bot: b}
	msg := &irc.IrcMessage{Name: irc.ERR_NICKNAMEINUSE}

	endpoint := makeTestPoint(b.servers[serverId])

	nickstr := "NICK :"
	nick1 := nickstr + cnf.GetAltnick()
	nick2 := nickstr + cnf.GetNick() + "_"
	nick3 := nickstr + cnf.GetNick() + "__"

	handler.HandleRaw(msg, endpoint)
	c.Check(endpoint.gets(), Equals, nick1)
	endpoint.resetTestWritten()
	handler.HandleRaw(msg, endpoint)
	c.Check(endpoint.gets(), Equals, nick2)
	endpoint.resetTestWritten()
	handler.HandleRaw(msg, endpoint)
	c.Check(endpoint.gets(), Equals, nick3)
}

func (s *s) TestCoreHandler_Caps(c *C) {
	connProvider := func(srv string) (net.Conn, error) {
		return nil, nil
	}

	b, err := createBot(fakeConfig, nil, connProvider, nil, true, false)
	c.Check(err, IsNil)

	msg1 := &irc.IrcMessage{
		Name: irc.RPL_MYINFO,
		Args: []string{
			"NICK", "irc.test.net", "testircd-1.2", "acCior", "beiIklmno",
		},
	}
	msg2 := &irc.IrcMessage{
		Name: irc.RPL_ISUPPORT,
		Args: []string{"RFC8213", "CHANTYPES=&$"},
	}
	srv := b.servers[serverId]
	srv.handler.HandleRaw(msg1, &testPoint{})
	srv.handler.HandleRaw(msg2, &testPoint{})
	c.Check(srv.caps.ServerName(), Equals, "irc.test.net")
	c.Check(srv.caps.IrcdVersion(), Equals, "testircd-1.2")
	c.Check(srv.caps.Usermodes(), Equals, "acCior")
	c.Check(srv.caps.LegacyChanmodes(), Equals, "beiIklmno")
	c.Check(srv.caps.Chantypes(), Equals, "&$")
}

func (s *s) TestCoreHandler_Join(c *C) {
	connProvider := func(srv string) (net.Conn, error) {
		return nil, nil
	}

	b, err := createBot(fakeConfig, nil, connProvider, nil, true, false)
	srv := b.servers[serverId]
	c.Check(err, IsNil)

	srv.state.Self.User = data.CreateUser("nick!user@host")
	msg := &irc.IrcMessage{
		Name:   irc.JOIN,
		Sender: srv.state.Self.GetFullhost(),
		Args:   []string{"#chan"},
	}

	endpoint := makeTestPoint(nil)
	srv.handler.HandleRaw(msg, endpoint)
	c.Check(endpoint.gets(), Equals, "WHO :#chanMODE :#chan")
}
