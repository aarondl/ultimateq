package bot

import (
	"bytes"
	"fmt"
	"net"

	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/irc"
	. "gopkg.in/check.v1"
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
	return netID
}

//==============
// Tests
//==============
func (s *s) TestCoreHandler_Ping(c *C) {
	handler := coreHandler{}
	ev := irc.NewEvent(netID, netInfo, irc.PING, "", "123123123123")
	endpoint := makeTestPoint(nil)
	handler.HandleRaw(ev, endpoint)
	c.Check(endpoint.gets(), Equals, irc.PONG+" :"+ev.Args[0])
}

func (s *s) TestCoreHandler_Connect(c *C) {
	b, err := createBot(fakeConfig, nil, nil, false, false)
	c.Check(err, IsNil)
	cnf := fakeConfig.GetServer(netID)
	handler := coreHandler{bot: b}
	msg1 := fmt.Sprintf("NICK :%v", cnf.GetNick())
	msg2 := fmt.Sprintf("USER %v 0 * :%v",
		cnf.GetUsername(), cnf.GetRealname())

	ev := irc.NewEvent(netID, netInfo, irc.CONNECT, "")
	endpoint := makeTestPoint(b.servers[netID])
	handler.HandleRaw(ev, endpoint)
	c.Check(endpoint.gets(), Equals, msg1+msg2)
}

func (s *s) TestCoreHandler_Nick(c *C) {
	b, err := createBot(fakeConfig, nil, nil, false, false)
	c.Check(err, IsNil)
	cnf := fakeConfig.GetServer(netID)
	handler := coreHandler{bot: b}
	ev := irc.NewEvent(netID, netInfo, irc.ERR_NICKNAMEINUSE, "")

	endpoint := makeTestPoint(b.servers[netID])

	nickstr := "NICK :"
	nick1 := nickstr + cnf.GetAltnick()
	nick2 := nickstr + cnf.GetNick() + "_"
	nick3 := nickstr + cnf.GetNick() + "__"

	handler.HandleRaw(ev, endpoint)
	c.Check(endpoint.gets(), Equals, nick1)
	endpoint.resetTestWritten()
	handler.HandleRaw(ev, endpoint)
	c.Check(endpoint.gets(), Equals, nick2)
	endpoint.resetTestWritten()
	handler.HandleRaw(ev, endpoint)
	c.Check(endpoint.gets(), Equals, nick3)
}

func (s *s) TestCoreHandler_NetInfo(c *C) {
	connProvider := func(srv string) (net.Conn, error) {
		return nil, nil
	}

	b, err := createBot(fakeConfig, connProvider, nil, true, false)
	c.Check(err, IsNil)

	msg1 := irc.NewEvent(netID, netInfo, irc.RPL_MYINFO, "",
		"NICK", "irc.test.net", "testircd-1.2", "acCior", "beiIklmno")
	msg2 := irc.NewEvent(netID, netInfo, irc.RPL_ISUPPORT, "",
		"RFC8213", "CHANTYPES=&$")
	srv := b.servers[netID]
	srv.handler.HandleRaw(msg1, &testPoint{})
	srv.handler.HandleRaw(msg2, &testPoint{})
	c.Check(srv.netInfo.ServerName(), Equals, "irc.test.net")
	c.Check(srv.netInfo.IrcdVersion(), Equals, "testircd-1.2")
	c.Check(srv.netInfo.Usermodes(), Equals, "acCior")
	c.Check(srv.netInfo.LegacyChanmodes(), Equals, "beiIklmno")
	c.Check(srv.netInfo.Chantypes(), Equals, "&$")
}

func (s *s) TestCoreHandler_Join(c *C) {
	connProvider := func(srv string) (net.Conn, error) {
		return nil, nil
	}

	b, err := createBot(fakeConfig, connProvider, nil, true, false)
	srv := b.servers[netID]
	c.Check(err, IsNil)

	srv.state.Self.User = data.NewUser("nick!user@host")
	ev := irc.NewEvent(netID, netInfo, irc.JOIN,
		srv.state.Self.Host(), "#chan")

	endpoint := makeTestPoint(nil)
	srv.handler.HandleRaw(ev, endpoint)
	c.Check(endpoint.gets(), Equals, "WHO :#chanMODE :#chan")
}
