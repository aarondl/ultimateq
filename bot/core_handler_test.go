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
	testWritten = testWritten[0:0]
}

func (t testSender) GetKey() string {
	return ""
}

func (t testSender) Writeln(str string) error {
	testWritten = append(testWritten, str)
	return nil
}

func testHandlerResponse(c *C, function func(*mocks.MockConn, *config.Server)) {
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

	function(conn, server.conf)

	b.Connect()
	b.Start()
	//b.Shutdown()
	b.WaitForShutdown()
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
	testHandlerResponse(c, func(conn *mocks.MockConn, conf *config.Server) {
		msg1 := fmt.Sprintf("NICK :%v\r\n", conf.GetNick())
		msg2 := fmt.Sprintf("USER %v 0 * :%v\r\n",
			conf.GetUsername(), conf.GetRealname())

		conn.EXPECT().Write([]byte(msg1)).Return(len(msg1), nil)
		conn.EXPECT().Write([]byte(msg2)).Return(len(msg2), io.EOF)
		conn.EXPECT().Read(gomock.Any()).Return(0, net.ErrWriteToConnected)
	})
}
