package bot

import (
	"code.google.com/p/gomock/gomock"
	"fmt"
	"github.com/aarondl/ultimateq/config"
	mocks "github.com/aarondl/ultimateq/inet/test"
	"github.com/aarondl/ultimateq/irc"
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
	return ""
}

func (t testSender) Writeln(str string) error {
	testWritten = append(testWritten, str)
	return nil
}

type callBack func(*Bot, *mocks.MockConn, *config.Server)

func testHandlerResponse(c *C, startWriter, startReader bool,
	before callBack, after callBack) {

	mockCtrl := gomock.NewController(c)
	defer mockCtrl.Finish()

	conn := mocks.NewMockConn(mockCtrl)

	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	b, err := createBot(fakeConfig, nil, connProvider)
	c.Assert(err, IsNil)

	server := b.servers[serverId]
	handler := coreHandler{b}
	b.Register(irc.RAW, handler)

	if before != nil {
		before(b, conn, server.conf)
	}

	b.Connect()
	b.start(startWriter, startReader)

	if after != nil {
		after(b, conn, server.conf)
	}

	b.WaitForHalt()
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
	c.Assert(testWritten[0], Equals, "PONG :"+msg.Args[0])
}

func (s *s) TestCoreHandler_Connect(c *C) {
	testHandlerResponse(c, true, false,
		func(_ *Bot, conn *mocks.MockConn, conf *config.Server) {
			msg1 := fmt.Sprintf("NICK :%v\r\n", conf.GetNick())
			msg2 := fmt.Sprintf("USER %v 0 * :%v\r\n",
				conf.GetUsername(), conf.GetRealname())

			gomock.InOrder(
				conn.EXPECT().Write([]byte(msg1)).Return(len(msg1), nil),
				conn.EXPECT().Write([]byte(msg2)).Return(len(msg2), io.EOF),
			)
		},
		nil,
	)
}

func (s *s) TestCoreHandler_Disconnect(c *C) {
	testHandlerResponse(c, false, false, nil,
		func(b *Bot, conn *mocks.MockConn, conf *config.Server) {
			conn.EXPECT().Close()
			b.handler.HandleRaw(
				&irc.IrcMessage{Name: irc.DISCONNECT},
				ServerSender{
					conf.GetHost(),
					b.servers[serverId],
				},
			)
			c.Assert(b.servers[serverId].client.IsClosed(), Equals, true)
		},
	)
}
