package inet

import (
	"bytes"
	"github.com/aarondl/ultimateq/mocks"
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
	conn := mocks.CreateConn()
	client := CreateIrcClient(conn, "name")
	c.Assert(client.isShutdown, Equals, false)
	c.Assert(client.conn, Equals, conn)
	c.Assert(client.siphonchan, NotNil)
	c.Assert(client.pumpchan, NotNil)
	c.Assert(client.pumpservice, NotNil)
	c.Assert(client.name, Equals, "name")
	c.Assert(client.lastwrite.Before(time.Now()), Equals, true)
}

func (s *s) TestIrcClient_ImplementsReadWriteCloser(c *C) {
	client := CreateIrcClient(nil, "")
	c.Assert(client, FitsTypeOf, io.ReadWriteCloser(client))
}

func (s *s) TestIrcClient_SpawnWorkers(c *C) {
	conn := mocks.CreateConn()

	client := CreateIrcClient(conn, "")
	c.Assert(client.IsClosed(), Equals, false)
	client.SpawnWorkers(true, true)
	c.Assert(client.IsClosed(), Equals, false)
	conn.Send([]byte{}, 0, io.EOF)
	client.Close()
	conn.WaitForDeath()
}

func (s *s) TestIrcClient_Pump(c *C) {
	test1 := []byte("PONG :arg1 arg2\r\n")
	test2 := []byte("NOTICE :arg1\r\n")
	split := 2

	conn := mocks.CreateConn()
	client := CreateIrcClient(conn, "")

	fakelast := time.Now().Truncate(5 * time.Hour)
	client.SpawnWorkers(true, false)
	ch := <-client.pumpservice
	ch <- []byte{} //Inconsequential, testcov error handling

	go func() {
		client.Write(test1)
		client.Write(test2)
		client.Close()
	}()

	c.Assert(bytes.Compare(conn.Receive(split, nil), test1), Equals, 0)
	c.Assert(bytes.Compare(conn.Receive(len(test1[split:]), nil),
		test1[split:]), Equals, 0)

	c.Assert(bytes.Compare(conn.Receive(len(test2), nil), test2), Equals, 0)
	conn.WaitForDeath()
	conn.ResetDeath()

	//Shameful test coverage
	client = CreateIrcClient(conn, "")
	client.SpawnWorkers(true, false)
	client.Write(test1)
	c.Assert(bytes.Compare(conn.Receive(0, io.EOF), test1), Equals, 0)
	client.Close()
	conn.WaitForDeath()
	c.Assert(client.lastwrite.Equal(fakelast), Equals, false)
}

func (s *s) TestIrcClient_PumpTimeouts(c *C) {
	test1 := []byte("PRIVMSG :arg1 arg2\r\n")

	conn := mocks.CreateConn()
	client := CreateIrcClient(conn, "")
	client.timePerTick = time.Millisecond
	client.SpawnWorkers(true, false)

	go func() {
		for i := 0; i < 10; i++ {
			_, err := client.Write(test1)
			c.Assert(err, IsNil)
		}
	}()

	for i := 0; i < 9; i++ {
		c.Assert(bytes.Compare(conn.Receive(len(test1), nil), test1), Equals, 0)
	}
	c.Assert(bytes.Compare(conn.Receive(len(test1), io.EOF), test1), Equals, 0)

	client.Close()
	conn.WaitForDeath()
}

func (s *s) TestIrcClient_Siphon(c *C) {
	test1 := []byte("PRIVMSG :msg\r\n")
	test2 := []byte("NOTICE :msg\r\n")
	test3 := []byte("PRIV")

	buf := append(append(append([]byte{}, test1...), test2...), test3...)

	conn := mocks.CreateConn()
	client := CreateIrcClient(conn, "")
	ch := client.ReadChannel()
	client.SpawnWorkers(false, true)

	go func() {
		conn.Send(buf, len(buf), io.EOF)
	}()

	msg := <-ch
	c.Assert(bytes.Compare(test1[:len(test1)-2], msg), Equals, 0)
	msg = <-ch
	c.Assert(bytes.Compare(test2[:len(test2)-2], msg), Equals, 0)
	_, ok := <-ch
	c.Assert(ok, Equals, false)

	client.Close()
	conn.WaitForDeath()
	conn.ResetDeath()

	client = CreateIrcClient(conn, "")
	client.SpawnWorkers(false, true)
	go func() { conn.Send(buf, len(buf), nil) }()
	client.Close()
	conn.WaitForDeath()
}

func (s *s) TestIrcClient_ExtractMessages(c *C) {
	test1 := []byte("irc message 1\r\n")
	test2 := []byte("irc message 2\r\n")
	test3 := []byte("irc mess")
	buf := append(append(append([]byte{}, test1...), test2...), test3...)

	waiter := sync.WaitGroup{}
	waiter.Add(1)

	client := CreateIrcClient(nil, "")
	ret := 0

	go func() {
		var abort bool
		ret, abort = client.extractMessages(buf)
		c.Assert(ret, Equals, len(test3))
		c.Assert(abort, Equals, false)
		c.Assert(bytes.Compare(buf[:ret], test3), Equals, 0)
		waiter.Done()
	}()
	msg1 := <-client.siphonchan
	c.Assert(bytes.Compare(msg1, test1[:len(test1)-2]), Equals, 0)
	msg2 := <-client.siphonchan
	c.Assert(bytes.Compare(msg2, test2[:len(test2)-2]), Equals, 0)
	waiter.Wait()

	buf = append(buf[:ret], []byte{'\r', '\n'}...)
	waiter.Add(1)
	go func() {
		var abort bool
		ret, abort := client.extractMessages(buf)
		c.Assert(ret, Equals, 0)
		c.Assert(abort, Equals, false)
		waiter.Done()
	}()
	msg3 := <-client.siphonchan
	c.Assert(bytes.Compare(msg3, test3), Equals, 0)
	waiter.Wait()

	waiter.Add(1)
	client.killsiphon = make(chan int)
	go func() {
		_, abort := client.extractMessages(test1)
		c.Assert(abort, Equals, true)
		waiter.Done()
	}()
	client.killsiphon <- 0
	waiter.Wait()
}

func (s *s) TestIrcClient_Close(c *C) {
	conn := mocks.CreateConn()

	client := CreateIrcClient(conn, "")

	go func() {
		err := client.Close()
		c.Assert(err, IsNil)
	}()
	conn.WaitForDeath()
	c.Assert(client.IsClosed(), Equals, true)

	err := client.Close() // Double closing should do nothing
	c.Assert(err, IsNil)
	c.Assert(client.IsClosed(), Equals, true)
}

func (s *s) TestIrcClient_ReadMessage(c *C) {
	client := CreateIrcClient(nil, "")
	read := []byte("PRIVMSG #chan :msg")
	go func() {
		client.siphonchan <- read
		close(client.siphonchan)
	}()
	msg, ok := client.ReadMessage()
	c.Assert(ok, Equals, true)
	c.Assert(bytes.Compare(msg, read), Equals, 0)
	msg, ok = client.ReadMessage()
	c.Assert(ok, Equals, false)
}

func (s *s) TestIrcClient_Read(c *C) {
	client := CreateIrcClient(nil, "")
	read := []byte("PRIVMSG #chan :msg")
	go func() {
		client.siphonchan <- read
		close(client.siphonchan)
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
	test1 := []byte("PRIVMSG #chan :msg\r\n")
	test2 := []byte("PRIVMSG #chan :msg2")

	client := CreateIrcClient(nil, "")
	ch := make(chan []byte)
	go func() {
		arg := append(test1, test2...)
		client.Write(nil) //Should be Consequenceless test cov
		n, err := client.Write(arg)
		c.Assert(err, IsNil)
		c.Assert(n, Equals, len(arg))
	}()
	client.pumpservice <- ch
	c.Assert(bytes.Compare(<-ch, test1), Equals, 0)
	client.pumpservice <- ch
	c.Assert(bytes.Compare(<-ch, append(test2, []byte{13, 10}...)), Equals, 0)

	close(client.pumpservice)
	n, err := client.Write(test1)
	c.Assert(n, Equals, 0)
	c.Assert(err, Equals, io.EOF)
	n, err = client.Write(test2) // Test abortion of no \r\n
	c.Assert(n, Equals, 0)
	c.Assert(err, Equals, io.EOF)
}

func (s *s) TestIrcClient_calcSleepTime(c *C) {
	client := CreateIrcClient(nil, "")

	// Check no-sleep and negative cases
	sleep := client.calcSleepTime(time.Now().Truncate(5 * time.Hour))
	c.Assert(sleep, Equals, time.Duration(0))
	sleep = client.calcSleepTime(time.Now().Add(5 * time.Second))
	c.Assert(sleep, Equals, time.Duration(0))

	// It should take a few messages to get it to delay.
	sleep = client.calcSleepTime(time.Now().Truncate(5 * time.Second))
	c.Assert(sleep, Equals, time.Duration(0))

	for i := 1; i <= 4; i++ {
		sleep = client.calcSleepTime(time.Now())
		c.Assert(sleep, Equals, time.Duration(0))
	}

	sleep = client.calcSleepTime(time.Now())
	c.Assert(sleep, Not(Equals), time.Duration(0))

	sleep2 := client.calcSleepTime(time.Now())
	c.Assert(sleep2 > sleep, Equals, true)
}

func (s *s) TestfindChunks(c *C) {
	test1 := []byte("PRIVMSG #chan :msg\r\n")
	test2 := []byte("NOTICE #chan :msg2\r\n")
	test3 := []byte("PRIV")

	args := append(append(test1, test2...), test3...)
	expected := [][]byte{test1, test2, test3}
	start, remaining, abort := findChunks(args, func(result []byte) bool {
		c.Assert(bytes.Compare(result, expected[0]), Equals, 0)
		expected = expected[1:]
		return false
	})
	c.Assert(abort, Equals, false)
	c.Assert(bytes.Compare(args[start:], test3), Equals, 0)

	start, remaining, abort = findChunks(test1, func(result []byte) bool {
		c.Assert(bytes.Compare(test1, result), Equals, 0)
		return false
	})
	c.Assert(start, Equals, 0)
	c.Assert(abort, Equals, false)
	c.Assert(remaining, Equals, false)

	_, _, abort = findChunks(args, func(result []byte) bool {
		return true
	})
	c.Assert(abort, Equals, true)
}
