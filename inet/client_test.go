package inet

import (
	"bytes"
	"code.google.com/p/gomock/gomock"
	mocks "github.com/aarondl/ultimateq/inet/test"
	"io"
	. "launchpad.net/gocheck"
	"log"
	"os"
	"sync"
	"testing"
	"time"
)

func Test(t *testing.T) { TestingT(t) } //Hook into testing package

type s struct{}

var _ = Suite(&s{})

func init() {
	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		log.Println("Could not set logger:", err)
	} else {
		log.SetOutput(f)
	}
}

func (s *s) TestCreateIrcClient(c *C) {
	mockCtrl := gomock.NewController(c)
	defer mockCtrl.Finish()

	conn := mocks.NewMockConn(mockCtrl)
	client := CreateIrcClient(conn)
	c.Assert(client.shutdown, Equals, false)
	c.Assert(client.conn, Equals, conn)
	c.Assert(client.readchan, NotNil)
	c.Assert(client.writechan, NotNil)
	c.Assert(client.queue, NotNil)
	c.Assert(client.waiter, NotNil)
	c.Assert(client.lastwrite.Before(time.Now()), Equals, true)
}

func (s *s) TestIrcClient_ImplementsReadWriteCloser(c *C) {
	client := CreateIrcClient(nil)
	c.Assert(client, FitsTypeOf, io.ReadWriteCloser(client))
}

func (s *s) TestIrcClient_SpawnWorkers(c *C) {
	mockCtrl := gomock.NewController(c)
	defer mockCtrl.Finish()

	conn := mocks.NewMockConn(mockCtrl)
	conn.EXPECT().Close()

	client := CreateIrcClient(conn)
	client.Close()
	client.SpawnWorkers()
	client.Wait()
}

func (s *s) TestIrcClient_Pump(c *C) {
	mockCtrl := gomock.NewController(c)
	defer mockCtrl.Finish()

	test := []byte("PRIVMSG :arg1 arg2\r\n")
	split := 2

	conn := mocks.NewMockConn(mockCtrl)
	conn.EXPECT().Write(test).Return(split, nil)
	conn.EXPECT().Write(test[split:]).Return(len(test[split:]), nil)
	conn.EXPECT().Close().Return(nil)

	client := CreateIrcClient(conn)

	waiter := sync.WaitGroup{}
	waiter.Add(1)

	go func() {
		client.Write(test)
		err := client.Close()
		c.Assert(err, IsNil)
	}()

	client.waiter.Add(1)
	fakelast := time.Now().Truncate(5 * time.Hour)
	client.Pump()
	c.Assert(client.lastwrite.Equal(fakelast), Equals, false)
	client.Wait()
}

/* WARNING:
 This test requires the mock to perform work on the buffer passed in. gomock
 tells us not to modify for obvious reasons, but there's no workaround here.

 The following code should be put inside the Read routine for testing.

var ByteFiller []byte
func (_m *MockConn) Read(_param0 []byte) (int, error) {
	ret := _m.ctrl.Call(_m, "Read", _param0)
	for i := 0; i < len(_param0) && i < len(ByteFiller); i++ {
		_param0[i] = ByteFiller[i]
	}
*/
func (s *s) TestIrcClient_Siphon(c *C) {
	mockCtrl := gomock.NewController(c)
	defer mockCtrl.Finish()

	test1 := []byte("PRIVMSG :msg\r\n")
	test2 := []byte("NOTICE :msg\r\n")
	test3 := []byte("PRIV")

	mocks.ByteFiller =
		append(append(append([]byte{}, test1...), test2...), test3...)

	conn := mocks.NewMockConn(mockCtrl)
	conn.EXPECT().Read(gomock.Any()).Return(len(mocks.ByteFiller), nil)
	conn.EXPECT().Read(gomock.Any()).Return(0, io.EOF)

	client := CreateIrcClient(conn)
	client.waiter.Add(1)
	go func() {
		client.Siphon()
	}()

	msg := <-client.readchan
	c.Assert(bytes.Compare(test1[:len(test1)-2], msg), Equals, 0)
	msg = <-client.readchan
	c.Assert(bytes.Compare(test2[:len(test2)-2], msg), Equals, 0)
	client.Wait() // This should be pointless
	_, ok := <-client.readchan
	c.Assert(ok, Equals, false)
}

func (s *s) TestIrcClient_ExtractMessages(c *C) {
	test1 := []byte("irc message 1\r\n")
	test2 := []byte("irc message 2\r\n")
	test3 := []byte("irc mess")
	buf := append(append(append([]byte{}, test1...), test2...), test3...)

	waiter := sync.WaitGroup{}
	waiter.Add(1)

	client := CreateIrcClient(nil)
	ret := 0
	go func() {
		ret = client.extractMessages(buf)
		waiter.Done()
	}()

	msg1 := <-client.readchan
	c.Assert(bytes.Compare(msg1, test1[:len(test1)-2]), Equals, 0)
	msg2 := <-client.readchan
	c.Assert(bytes.Compare(msg2, test2[:len(test2)-2]), Equals, 0)

	waiter.Wait()
	c.Assert(ret, Equals, 8)
	c.Assert(bytes.Compare(buf[:ret], test3), Equals, 0)
}

func (s *s) TestIrcClient_Close(c *C) {
	mockCtrl := gomock.NewController(c)
	defer mockCtrl.Finish()

	conn := mocks.NewMockConn(mockCtrl)
	conn.EXPECT().Close().Return(nil)

	client := CreateIrcClient(conn)

	client.Close()
	c.Assert(client.shutdown, Equals, true)
	_, ok := <-client.writechan
	c.Assert(ok, Equals, false)
}

func (s *s) TestIrcClient_ReadMessage(c *C) {
	client := CreateIrcClient(nil)
	read := []byte("PRIVMSG #chan :msg")
	go func() {
		client.readchan <- read
		close(client.readchan)
	}()
	msg, ok := client.ReadMessage()
	c.Assert(ok, Equals, true)
	c.Assert(bytes.Compare(msg, read), Equals, 0)
	msg, ok = client.ReadMessage()
	c.Assert(ok, Equals, false)
}

func (s *s) TestIrcClient_Read(c *C) {
	client := CreateIrcClient(nil)
	read := []byte("PRIVMSG #chan :msg")
	go func() {
		client.readchan <- read
		close(client.readchan)
	}()
	buf := make([]byte, len(read))
	breakat := 2

	n, err := client.Read(buf[:breakat])
	c.Assert(err, IsNil)
	c.Assert(n, Equals, breakat)
	c.Assert(bytes.Compare(buf[:breakat], read[:breakat]), Equals, 0)

	n, err = client.Read(buf[breakat:])
	c.Assert(err, IsNil)
	c.Assert(n, Equals, len(read)-breakat)
	c.Assert(bytes.Compare(buf, read), Equals, 0)

	n, err = client.Read(buf)
	c.Assert(n, Equals, 0)
	c.Assert(err, Equals, io.EOF)
}

func (s *s) TestIrcClient_Write(c *C) {
	client := CreateIrcClient(nil)
	write := []byte("PRIVMSG #chan :msg")
	go func() {
		n, err := client.Write(write)
		c.Assert(err, IsNil)
		c.Assert(n, Equals, len(write))
	}()
	nMessages := <-client.writechan
	c.Assert(client.queue.length, Equals, 1)
	c.Assert(nMessages, Equals, 1)
	c.Assert(bytes.Compare(*client.queue.dequeue(), write), Equals, 0)
}

func (s *s) TestIrcClient_calcSleepTime(c *C) {
	client := CreateIrcClient(nil)
	sleep := client.calcSleepTime(time.Now().Truncate(5 * time.Second))
	c.Assert(sleep, Equals, time.Duration(0))
	sleep = client.calcSleepTime(time.Now())
	c.Assert(sleep, Equals, time.Duration(0))
	sleep = client.calcSleepTime(time.Now())
	c.Assert(sleep, Not(Equals), time.Duration(0))
	sleep2 := client.calcSleepTime(time.Now())
	c.Assert(sleep2 > sleep, Equals, true)
}
