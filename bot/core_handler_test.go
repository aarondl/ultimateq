package bot

import (
	"bytes"
	"fmt"
	"github.com/aarondl/ultimateq/config"
	"github.com/aarondl/ultimateq/irc"
	"github.com/aarondl/ultimateq/mocks"
	"io"
	. "launchpad.net/gocheck"
	"net"
)

//===================================================================
// Fixtures for basic responses as well as full bot required messages
//===================================================================
var testWritten []string = make([]string, 0, 10)

type testSender struct {
}

func resetTestWritten() {
	testWritten = testWritten[:0]
}

func (t testSender) GetKey() string {
	return serverId
}

func (t testSender) Writef(format string, args ...interface{}) error {
	testWritten = append(testWritten, fmt.Sprintf(format, args...))
	return nil
}

func (t testSender) Writeln(args ...interface{}) error {
	testWritten = append(testWritten, fmt.Sprint(args...))
	return nil
}

func (t testSender) Write(buf []byte) (int, error) {
	testWritten = append(testWritten, string(buf))
	return len(buf), nil
}

// callBack gets the Bot, the mock Connection, the server's config, and the
// write channel if during the call to testHandlerResponse startWriter was true
type callBack func(*Bot, *mocks.Conn, *config.Server)

func testHandlerResponse(c *C, startWriter, startReader bool,
	before callBack, after callBack) {

	conn := mocks.CreateConn()
	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	b, err := createBot(fakeConfig, nil, connProvider, true)
	c.Check(err, IsNil)

	server := b.servers[serverId]

	if before != nil {
		before(b, conn, server.conf)
	}

	b.Connect()
	b.start(startWriter, startReader)

	if after != nil {
		after(b, conn, server.conf)
	}

	b.WaitForHalt()
	if startReader {
		b.Stop()
	}
	b.Disconnect()
}

//==============
// Tests
//==============
func (s *s) TestCoreHandler_Ping(c *C) {
	handler := coreHandler{}
	msg := &irc.IrcMessage{
		Name: "PING",
		Args: []string{"123123123123"},
	}
	handler.HandleRaw(msg, testSender{})
	c.Check(testWritten[0], Equals, "PONG :"+msg.Args[0])
}

func (s *s) TestCoreHandler_Connect(c *C) {
	testHandlerResponse(c, true, false, nil,
		func(b *Bot, conn *mocks.Conn, conf *config.Server) {
			msg1 := []byte(fmt.Sprintf("NICK :%v\r\n", conf.GetNick()))
			msg2 := []byte(fmt.Sprintf("USER %v 0 * :%v\r\n",
				conf.GetUsername(), conf.GetRealname()))

			c.Check(bytes.Compare(conn.Receive(len(msg1), nil), msg1),
				Equals, 0)
			c.Check(bytes.Compare(conn.Receive(len(msg2), nil), msg2),
				Equals, 0)
		},
	)
}

func (s *s) TestCoreHandler_Nick(c *C) {
	testHandlerResponse(c, true, true, nil,
		func(_ *Bot, conn *mocks.Conn, conf *config.Server) {
			nickstr := "NICK :%v\r\n"
			nick1 := []byte(fmt.Sprintf(nickstr, conf.GetNick()))
			nick2 := []byte(fmt.Sprintf(nickstr, conf.GetAltnick()))
			nick3 := []byte(fmt.Sprintf(nickstr, conf.GetNick()+"_"))
			nick4 := []byte(fmt.Sprintf(nickstr, conf.GetNick()+"__"))
			user := []byte(fmt.Sprintf("USER %v 0 * :%v\r\n",
				conf.GetUsername(), conf.GetRealname()))
			errmsg := []byte(fmt.Sprintf("433 :Nick is in use\r\n"))

			c.Check(bytes.Compare(conn.Receive(len(nick1), nil), nick1),
				Equals, 0)
			c.Check(bytes.Compare(conn.Receive(len(user), nil), user),
				Equals, 0)
			conn.Send(errmsg, len(errmsg), nil)
			c.Check(bytes.Compare(conn.Receive(len(nick2), nil), nick2),
				Equals, 0)
			conn.Send(errmsg, len(errmsg), nil)
			c.Check(bytes.Compare(conn.Receive(len(nick3), nil), nick3),
				Equals, 0)
			conn.Send(errmsg, len(errmsg), nil)
			c.Check(bytes.Compare(conn.Receive(len(nick4), nil), nick4),
				Equals, 0)
			conn.Send(errmsg, 0, io.EOF)
		},
	)
}

func (s *s) TestCoreHandler_Caps(c *C) {
	connProvider := func(srv string) (net.Conn, error) {
		return nil, nil
	}

	b, err := createBot(fakeConfig, nil, connProvider, true)
	c.Check(err, IsNil)

	msg1 := &irc.IrcMessage{
		Name: irc.RPL_MYINFO,
		Args: []string{"irc.test.net", "testircd-1.2", "acCior", "beiIklmno"},
	}
	msg2 := &irc.IrcMessage{
		Name: irc.RPL_ISUPPORT,
		Args: []string{"RFC8213", "CHANTYPES=&$"},
	}
	srv := b.servers[serverId]
	srv.handler.HandleRaw(msg1, testSender{})
	srv.handler.HandleRaw(msg2, testSender{})
	c.Check(srv.caps.ServerName(), Equals, "irc.test.net")
	c.Check(srv.caps.IrcdVersion(), Equals, "testircd-1.2")
	c.Check(srv.caps.Usermodes(), Equals, "acCior")
	c.Check(srv.caps.LegacyChanmodes(), Equals, "beiIklmno")
	c.Check(srv.caps.Chantypes(), Equals, "&$")
}
