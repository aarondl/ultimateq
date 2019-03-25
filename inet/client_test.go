package inet

import (
	"bytes"
	"io"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/aarondl/ultimateq/mocks"
)

func init() {
	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		log.Println("Could not set logger:", err)
	} else {
		log.SetOutput(f)
	}
}

func TestcreateIrcClient(t *testing.T) {
	conn := mocks.NewConn()
	client := createIrcClient(conn, nil)
	if exp, got := client.isShutdown, false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := client.conn, conn; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if client.siphonchan == nil {
		t.Error("Unexpected nil.")
	}
	if client.pumpchan == nil {
		t.Error("Unexpected nil.")
	}
	if client.pumpservice == nil {
		t.Error("Unexpected nil.")
	}
	if exp, got := client.lastwrite.Before(time.Now()), true; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestIrcClient_ImplementsReadWriteCloser(t *testing.T) {
	client := createIrcClient(nil, nil)
	var _ io.ReadWriteCloser = client
}

func TestIrcClient_SpawnWorkers(t *testing.T) {
	conn := mocks.NewConn()

	client := createIrcClient(conn, nil)
	if exp, got := client.IsClosed(), false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	client.SpawnWorkers(true, true)
	if exp, got := client.IsClosed(), false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	conn.Send([]byte{}, 0, io.EOF)
	client.Close()
	conn.WaitForDeath()
}

func TestIrcClient_Pump(t *testing.T) {
	test1 := []byte("PONG :arg1 arg2\r\n")
	test2 := []byte("NOTICE :arg1\r\n")
	split := 2

	conn := mocks.NewConn()
	client := createIrcClient(conn, nil)

	fakelast := time.Now().Truncate(5 * time.Hour)
	client.SpawnWorkers(true, false)
	ch := <-client.pumpservice
	ch <- []byte{} //Inconsequential, testcov error handling

	go func() {
		client.Write(test1)
		client.Write(test2)
		client.Close()
	}()

	if exp, got :=
		bytes.Compare(conn.Receive(split, nil), test1), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	got := bytes.Compare(conn.Receive(len(test1[split:]), nil), test1[split:])
	if got != 0 {
		t.Error("Expected 0, got:", got)
	}

	if exp, got :=
		bytes.Compare(conn.Receive(len(test2), nil), test2), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	conn.WaitForDeath()
	conn.ResetDeath()

	//Shameful test coverage
	client = createIrcClient(conn, nil)
	client.SpawnWorkers(true, false)
	client.Write(test1)
	if exp, got :=
		bytes.Compare(conn.Receive(0, io.EOF), test1), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	client.Close()
	conn.WaitForDeath()
	if exp, got := client.lastwrite.Equal(fakelast), false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestIrcClient_PumpFloodProtect(t *testing.T) {
	test1 := []byte("PRIVMSG :arg1 arg2\r\n")

	conn := mocks.NewConn()
	client := NewIrcClient(conn, nil, 10, 2, 120, 0, time.Millisecond)
	client.SpawnWorkers(true, false)

	go func() {
		for i := 0; i < 10; i++ {
			_, err := client.Write(test1)
			if got := err; got != nil {
				t.Errorf("Expected: %v to be nil.", got)
			}
		}
	}()

	for i := 0; i < 9; i++ {
		if exp, got :=
			bytes.Compare(conn.Receive(len(test1), nil), test1), 0; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
	}
	if exp, got :=
		bytes.Compare(conn.Receive(len(test1), io.EOF), test1), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	client.Close()
	conn.WaitForDeath()
}

func TestIrcClient_Siphon(t *testing.T) {
	test1 := []byte("PRIVMSG :msg\r\n")
	test2 := []byte("NOTICE :msg\r\n")
	test3 := []byte("PRIV")

	buf := append(append(append([]byte{}, test1...), test2...), test3...)

	conn := mocks.NewConn()
	client := createIrcClient(conn, nil)
	ch := client.ReadChannel()
	client.SpawnWorkers(false, true)

	go func() {
		conn.Send(buf, len(buf), io.EOF)
	}()

	msg := <-ch
	if exp, got := bytes.Compare(test1[:len(test1)-2], msg), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	msg = <-ch
	if exp, got := bytes.Compare(test2[:len(test2)-2], msg), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	_, ok := <-ch
	if exp, got := ok, false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	client.Close()
	conn.WaitForDeath()
	conn.ResetDeath()

	client = createIrcClient(conn, nil)
	client.SpawnWorkers(false, true)
	go func() { conn.Send(buf, len(buf), nil) }()
	client.Close()
	conn.WaitForDeath()
}

func TestIrcClient_ExtractMessages(t *testing.T) {
	test1 := []byte("irc message 1\r\n")
	test2 := []byte("irc message 2\r\n")
	test3 := []byte("irc mess")
	buf := append(append(append([]byte{}, test1...), test2...), test3...)

	waiter := sync.WaitGroup{}
	waiter.Add(1)

	client := createIrcClient(nil, nil)
	ret := 0

	go func() {
		var abort bool
		ret, abort = client.extractMessages(buf)
		if exp, got := ret, len(test3); exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
		if exp, got := abort, false; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
		if exp, got := bytes.Compare(buf[:ret], test3), 0; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
		waiter.Done()
	}()
	msg1 := <-client.siphonchan
	if exp, got := bytes.Compare(msg1, test1[:len(test1)-2]), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	msg2 := <-client.siphonchan
	if exp, got := bytes.Compare(msg2, test2[:len(test2)-2]), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	waiter.Wait()

	buf = append(buf[:ret], []byte{'\r', '\n'}...)
	waiter.Add(1)
	go func() {
		var abort bool
		ret, abort := client.extractMessages(buf)
		if exp, got := ret, 0; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
		if exp, got := abort, false; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
		waiter.Done()
	}()
	msg3 := <-client.siphonchan
	if exp, got := bytes.Compare(msg3, test3), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	waiter.Wait()

	waiter.Add(1)
	client.killsiphon = make(chan error)
	go func() {
		_, abort := client.extractMessages(test1)
		if exp, got := abort, true; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
		waiter.Done()
	}()
	<-client.killsiphon
	waiter.Wait()
}

func TestIrcClient_Close(t *testing.T) {
	conn := mocks.NewConn()

	client := createIrcClient(conn, nil)

	go func() {
		err := client.Close()
		if got := err; got != nil {
			t.Errorf("Expected: %v to be nil.", got)
		}
	}()
	conn.WaitForDeath()
	if exp, got := client.IsClosed(), true; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	err := client.Close() // Double closing should do nothing
	if got := err; got != nil {
		t.Errorf("Expected: %v to be nil.", got)
	}
	if exp, got := client.IsClosed(), true; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestIrcClient_ReadMessage(t *testing.T) {
	client := createIrcClient(nil, nil)
	read := []byte("PRIVMSG #chan :msg")
	go func() {
		client.siphonchan <- read
		close(client.siphonchan)
	}()
	msg, ok := client.ReadMessage()
	if exp, got := ok, true; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := bytes.Compare(msg, read), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	msg, ok = client.ReadMessage()
	if exp, got := ok, false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestIrcClient_Read(t *testing.T) {
	client := createIrcClient(nil, nil)
	read := []byte("PRIVMSG #chan :msg")
	go func() {
		client.siphonchan <- read
		close(client.siphonchan)
	}()
	buf := make([]byte, len(read))
	breakat := 2

	n, err := client.Read(buf[:breakat])
	if got := err; got != nil {
		t.Errorf("Expected: %v to be nil.", got)
	}
	if exp, got := n, breakat; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := bytes.Compare(buf[:breakat], read[:breakat]), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	n, err = client.Read(buf[breakat:])
	if got := err; got != nil {
		t.Errorf("Expected: %v to be nil.", got)
	}
	if exp, got := n, len(read)-breakat; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := bytes.Compare(buf, read), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	n, err = client.Read(buf)
	if exp, got := n, 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := err, io.EOF; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestIrcClient_Write(t *testing.T) {
	test1 := []byte("PRIVMSG #chan :msg\r\n")
	test2 := []byte("PRIVMSG #chan :msg2")

	client := createIrcClient(nil, nil)
	ch := make(chan []byte)
	go func() {
		arg := append(test1, test2...)
		client.Write(nil) //Should be Consequenceless test cov
		n, err := client.Write(arg)
		if got := err; got != nil {
			t.Errorf("Expected: %v to be nil.", got)
		}
		if exp, got := n, len(arg); exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
	}()
	client.pumpservice <- ch
	expectedMsg := append(test1[:len(test1)-2], test2...)
	expectedMsg = append(expectedMsg, []byte("\r\n")...)
	if exp, got := bytes.Compare(<-ch, expectedMsg), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	close(client.pumpservice)
}

func TestIrcClient_Keepalive(t *testing.T) {
	// Check not throttled
	conn := mocks.NewConn()
	client := NewIrcClient(conn, nil, 0, 0, 0, time.Millisecond,
		time.Millisecond)
	client.SpawnWorkers(true, false)
	msg := conn.Receive(len(ping), io.EOF)
	if exp, got := bytes.Compare(msg, ping), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	client.Close()

	// Check throttled
	conn = mocks.NewConn()
	client = NewIrcClient(conn, nil, 1000,
		10*time.Millisecond,
		10*time.Millisecond,
		40*time.Millisecond, time.Millisecond)

	test := []byte("test")
	client.queue.Enqueue(test)
	client.lastwrite = time.Now()
	client.penalty = client.lastwrite.Add(time.Hour)

	go func() {
		<-client.pumpservice <- test
	}()

	client.SpawnWorkers(true, false)

	msg = conn.Receive(len(test), nil)
	if exp, got := bytes.Compare(msg, test), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	msg = conn.Receive(len(test), nil)
	if exp, got := bytes.Compare(msg, test), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	msg = conn.Receive(len(ping), io.EOF)
	if exp, got := bytes.Compare(msg, ping), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	client.Close()
}

func TestIrcClient_calcSleepTime(t *testing.T) {
	var penFact = 120
	var scale = time.Millisecond
	var timeout, step, keepalive time.Duration = 10 * time.Millisecond,
		2 * time.Millisecond, 0

	var sleep time.Duration
	client := createIrcClient(nil, nil)

	sleep = client.calcSleepTime(time.Now(), 0)
	if exp, got := sleep, time.Duration(0); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	client = NewIrcClient(nil, nil, penFact, timeout, step, keepalive, scale)
	client.lastwrite = time.Now()
	for i := 1; i <= 5; i++ {
		sleep = client.calcSleepTime(time.Now(), 0)
		if exp, got := sleep, time.Duration(0); exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
	}

	sleep = client.calcSleepTime(time.Now(), 0)
	if exp, got := sleep, time.Duration(0); exp == got {
		t.Errorf("Did not want: %v, got: %v", exp, got)
	}

	// Check no-sleep and negative cases
	client = NewIrcClient(nil, nil, penFact, timeout, step, keepalive, scale)
	client.lastwrite = time.Now()
	sleep = client.calcSleepTime(time.Time{}, 0)
	if exp, got := sleep, time.Duration(0); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	sleep = client.calcSleepTime(time.Now().Add(5*time.Hour), 0)
	if exp, got := sleep, time.Duration(0); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestfindChunks(t *testing.T) {
	test1 := []byte("PRIVMSG #chan :msg\r\n")
	test2 := []byte("NOTICE #chan :msg2\r\n")
	test3 := []byte("PRIV")

	args := append(append(test1, test2...), test3...)
	expected := [][]byte{test1, test2, test3}
	start, remaining, abort := findChunks(args, func(result []byte) bool {
		if exp, got := bytes.Compare(result, expected[0]), 0; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
		expected = expected[1:]
		return false
	})
	if exp, got := abort, false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := bytes.Compare(args[start:], test3), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	start, remaining, abort = findChunks(test1, func(result []byte) bool {
		if exp, got := bytes.Compare(test1, result), 0; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
		return false
	})
	if exp, got := start, 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := abort, false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := remaining, false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	_, _, abort = findChunks(args, func(result []byte) bool {
		return true
	})
	if exp, got := abort, true; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestIrcClient_ClientError(t *testing.T) {
	var _ error = ClientError{} // Should compile
	e := ClientError{}
	if exp, got := len(e.Error()), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got := e.CheckNeeded(); got != nil {
		t.Errorf("Expected: %v to be nil.", got)
	}
	e.Siphon = io.EOF
	if exp, got := e.Error(), "Siphon: "+io.EOF.Error(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	e.Pump = io.EOF

	exp := "Pump: " + io.EOF.Error() + " || Siphon: " + io.EOF.Error()
	if got := e.Error(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	e.Socket = io.EOF
	exp = "Socket: " + io.EOF.Error() + " || Pump: " + io.EOF.Error() +
		" || Siphon: " + io.EOF.Error()
	if got := e.Error(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if e.CheckNeeded() == nil {
		t.Error("Unexpected nil.")
	}
}
