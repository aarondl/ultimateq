package bot

import (
	"bytes"
	"fmt"
	"net"
	"testing"

	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/irc"
)

//===================================================================
// Fixtures for basic responses as well as full bot required messages
//===================================================================
type testPoint struct {
	irc.Helper
	buf *bytes.Buffer
	srv *Server
}

func makeTestPoint(srv *Server) *testPoint {
	buf := &bytes.Buffer{}
	t := &testPoint{irc.Helper{buf}, buf, srv}
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
func TestCoreHandler_Ping(t *testing.T) {
	handler := coreHandler{}
	ev := irc.NewEvent(netID, netInfo, irc.PING, "", "123123123123")
	endpoint := makeTestPoint(nil)
	handler.HandleRaw(endpoint, ev)
	expect := irc.PONG + " :" + ev.Args[0]
	if got := endpoint.gets(); got != expect {
		t.Errorf("Expected: %s, got: %s", expect, got)
	}
}

func TestCoreHandler_Connect(t *testing.T) {
	cnf := fakeConfig.Clone()
	net := cnf.Network(netID).SetPassword("password")
	b, _ := createBot(cnf, nil, nil, false, false)

	password, _ := net.Password()
	nick, _ := net.Nick()
	username, _ := net.Username()
	realname, _ := net.Realname()

	handler := coreHandler{bot: b}
	msg1 := fmt.Sprintf("PASS :%v", password)
	msg2 := fmt.Sprintf("NICK :%v", nick)
	msg3 := fmt.Sprintf("USER %v 0 * :%v", username, realname)

	ev := irc.NewEvent(netID, netInfo, irc.CONNECT, "")
	endpoint := makeTestPoint(b.servers[netID])
	handler.HandleRaw(endpoint, ev)

	expect := msg1 + msg2 + msg3
	if got := endpoint.gets(); got != expect {
		t.Errorf("Expected: %s, got: %s", expect, got)
	}
}

func TestCoreHandler_Nick(t *testing.T) {
	b, _ := createBot(fakeConfig, nil, nil, false, false)
	cnf := fakeConfig.Network(netID)
	handler := coreHandler{bot: b}
	ev := irc.NewEvent(netID, netInfo, irc.ERR_NICKNAMEINUSE, "")

	endpoint := makeTestPoint(b.servers[netID])

	nick, _ := cnf.Nick()
	altnick, _ := cnf.Altnick()
	nickstr := "NICK :"
	nick1 := nickstr + altnick
	nick2 := nickstr + nick + "_"
	nick3 := nickstr + nick + "__"

	handler.HandleRaw(endpoint, ev)
	if got := endpoint.gets(); got != nick1 {
		t.Errorf("Expected: %s, got: %s", nick1, got)
	}
	endpoint.resetTestWritten()
	handler.HandleRaw(endpoint, ev)
	if got := endpoint.gets(); got != nick2 {
		t.Errorf("Expected: %s, got: %s", nick2, got)
	}
	endpoint.resetTestWritten()
	handler.HandleRaw(endpoint, ev)
	if got := endpoint.gets(); got != nick3 {
		t.Errorf("Expected: %s, got: %s", nick3, got)
	}
}

func TestCoreHandler_NetInfo(t *testing.T) {
	connProvider := func(srv string) (net.Conn, error) {
		return nil, nil
	}

	b, _ := createBot(fakeConfig, connProvider, nil, true, false)

	msg1 := irc.NewEvent(netID, netInfo, irc.RPL_MYINFO, "",
		"NICK", "irc.test.net", "testircd-1.2", "acCior", "beiIklmno")
	msg2 := irc.NewEvent(netID, netInfo, irc.RPL_ISUPPORT, "",
		"RFC8213", "CHANTYPES=&$")
	srv := b.servers[netID]
	srv.handler.HandleRaw(&testPoint{}, msg1)
	srv.handler.HandleRaw(&testPoint{}, msg2)
	if got, exp := srv.netInfo.ServerName(), "irc.test.net"; got != exp {
		t.Errorf("Expected: %s, got: %s", exp, got)
	}
	if got, exp := srv.netInfo.IrcdVersion(), "testircd-1.2"; got != exp {
		t.Errorf("Expected: %s, got: %s", exp, got)
	}
	if got, exp := srv.netInfo.Usermodes(), "acCior"; got != exp {
		t.Errorf("Expected: %s, got: %s", exp, got)
	}
	if got, exp := srv.netInfo.LegacyChanmodes(), "beiIklmno"; got != exp {
		t.Errorf("Expected: %s, got: %s", exp, got)
	}
	if got, exp := srv.netInfo.Chantypes(), "&$"; got != exp {
		t.Errorf("Expected: %s, got: %s", exp, got)
	}
}

func TestCoreHandler_Join(t *testing.T) {
	connProvider := func(srv string) (net.Conn, error) {
		return nil, nil
	}

	b, _ := createBot(fakeConfig, connProvider, nil, true, false)
	srv := b.servers[netID]

	srv.state.Self.User = data.NewUser("nick!user@host")
	ev := irc.NewEvent(netID, netInfo, irc.JOIN,
		srv.state.Self.Host(), "#chan")

	endpoint := makeTestPoint(nil)
	srv.handler.HandleRaw(endpoint, ev)
	if got, exp := endpoint.gets(), "WHO :#chanMODE :#chan"; got != exp {
		t.Errorf("Expected: %s, got: %s", exp, got)
	}
}
