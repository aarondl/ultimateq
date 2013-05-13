/*
inet package handles connecting to an irc server and reading and writing to
the connection
*/
package inet

import (
	"io"
	"log"
	"math"
	"net"
	"sync"
	"time"
)

const (
	// bufferSize is the size of the buffer to be allocated for writes
	bufferSize = 16348
	// resetDuration in the time between messages required to bypass sleep
	resetDuration = 3 * time.Second
)

// Format strings for errors and logging output
const (
	fmtDiscarded          = "(%v) <- (DISCARDED) %s\n"
	fmtWrite              = "(%v) <- %s\n"
	fmtWriteErr           = "(%v) <- (%v) %s\n"
	fmtRead               = "(%v) -> %s\n"
	fmtErrSiphonReadError = "inet: (%v) socket closed (%s)\n"
)

// IrcClient represents a connection to an irc server. It uses a queueing system
// to throttle writes to the server. And it implements ReadWriteCloser interface
type IrcClient struct {
	shutdown  bool
	conn      net.Conn
	readchan  chan []byte
	writechan chan int
	queue     Queue
	waiter    sync.WaitGroup

	// The name of the connection for logging
	name string

	// the write channel may be closed at any time by any thread
	// so we have to protect it until all writing threads have finished sending
	// to the channel
	writeProtect sync.RWMutex

	// write throttling
	lastwrite  time.Time
	nThrottled int

	// buffering for io.Reader interface
	readbuf []byte
	pos     int
}

// CreateIrcClient initializes the required fields in the IrcClient
func CreateIrcClient(conn net.Conn, name string) *IrcClient {
	return &IrcClient{
		conn:      conn,
		readchan:  make(chan []byte),
		writechan: make(chan int),
		lastwrite: time.Now().Truncate(resetDuration),
		name:      name,
	}
}

// SpawnWorkers creates two goroutines, one that is constantly reading using
// Siphon, and one that is constantly working on eliminating the write queue by
// writing. Also sets up the instances waiter, and a subsequent call to Wait()
// will block until the workers have returned.
func (c *IrcClient) SpawnWorkers(pump, siphon bool) {
	if pump {
		c.waiter.Add(1)
		go c.Pump()
	}
	if siphon {
		c.waiter.Add(1)
		go c.Siphon()
	}
}

// Wait blocks until the workers from SpawnWorkers have returned.
func (c *IrcClient) Wait() {
	c.waiter.Wait()
}

// calcSleepTime checks to ensure that if we've been writing in quick succession
// we get some sleep time in between writes.
func (c *IrcClient) calcSleepTime(t time.Time) time.Duration {
	dur := t.Sub(c.lastwrite)
	if dur < 0 {
		dur = 0
	}

	if dur > resetDuration {
		c.nThrottled = 0
		return time.Duration(0)
	} else {
		sleep := time.Second * time.Duration(
			0.5+math.Max(0, math.Log2(float64(c.nThrottled)-3.0)),
		)
		c.nThrottled += 1
		return sleep
	}
}

// Pump is meant to be run off the main thread and dequeues the messages and
// writes them to the connection. This function blocks often (reading from
// channels, or sleeping) and therefore checks for shutdown just as often.
func (c *IrcClient) Pump() {
	var err error
	for !c.shutdown && err == nil {
		nMessages, ok := <-c.writechan
		if !ok {
			break
		}
		toWrite := c.queue.Dequeue(nMessages)
		if c.shutdown {
			c.discardMessages(toWrite)
			break
		}

		err = c.writeMessages(toWrite)
		if err != nil {
			break
		}
	}

	c.waiter.Done()
}

// writeMessages writes each byte array in messages out to the socket, sleeping
// for an appropriate amount of time in between. May discard messages if
// shutdown is set.
func (c *IrcClient) writeMessages(messages [][]byte) error {
	var n int
	var err error
	for i, msg := range messages {
		sleepTime := c.calcSleepTime(time.Now())
		if sleepTime > 0 {
			time.Sleep(sleepTime)
		}

		if c.shutdown {
			c.discardMessages(messages[i:])
			return nil
		}

		for written := 0; written < len(msg); written += n {
			n, err = c.conn.Write(msg[written:])
			wrote := msg[written : len(msg)-2]
			if err != nil {
				log.Printf(fmtWriteErr, c.name, err, wrote)
				break
			}
			log.Printf(fmtWrite, c.name, wrote)
			c.lastwrite = time.Now()
		}

		if err != nil {
			return err
		}
	}
	return nil
}

// discardMessages is used to log when messages are not able to be handled by
// a the write queue due to shutdown.
func (c *IrcClient) discardMessages(messages [][]byte) {
	for _, msg := range messages {
		log.Printf(fmtDiscarded, c.name, msg)
	}
}

// Siphon takes messages from the connection given to the IrcClient and then
// uses extractMessages to send them to the readchan.
func (c *IrcClient) Siphon() {
	buf := make([]byte, bufferSize)

	var err error = nil
	position, n := 0, 0
	for !c.shutdown && err == nil {
		n, err = c.conn.Read(buf[position:])

		if n > 0 && (err == nil || err == io.EOF) {
			position = c.extractMessages(buf[:n+position])
		}

		if err != nil {
			log.Printf(fmtErrSiphonReadError, c.name, err)
			break
		}
	}

	c.waiter.Done()
	close(c.readchan)
}

// extractMessages takes the information in a buffer and splits on \r\n pairs.
// When it encounters a pair, it creates a copy of the data from start to the
// pair and passes it into the readchan from the IrcClient. If no \r\n is found
// but data is still present in the buffer, it moves this data to the front of
// the buffer and returns an index from which the next read should be started at
//
// Note:
// the reason for the copy is because of the threadedness, the buffer pointed to
// by the slice should be immediately filled with new information once this
// function returns and therefore a copy must be made for thread safety.
func (c *IrcClient) extractMessages(buf []byte) int {

	send := func(chunk []byte) {
		cpy := make([]byte, len(chunk)-2)
		copy(cpy, chunk[:len(chunk)-2])
		log.Printf(fmtRead, c.name, cpy)
		c.readchan <- cpy
	}

	start, remaining := findChunks(buf, send)

	if remaining {
		copy(buf[:len(buf)-start], buf[start:])
		return len(buf) - start
	}

	return 0
}

// Close sets the shutdown variable, sets an all-consuming dequeuer routine to
// eat all the waiting-to-write goroutines, and then waits to acquire a mutex
// that will allow it to safely close the writer channel. It then closes the
// socket and returns any errors from that.
func (c *IrcClient) Close() error {
	if c.shutdown {
		return nil
	}

	c.shutdown = true

	wait := sync.WaitGroup{}
	wait.Add(1)
	go func() {
		for n := range c.writechan {
			msgs := c.queue.Dequeue(n)
			c.discardMessages(msgs)
		}
		wait.Done()
	}()

	c.writeProtect.Lock()
	close(c.writechan)
	c.writeProtect.Unlock()
	wait.Wait()

	return c.conn.Close()
}

// IsClosed returns true if the IrcClient has been closed.
func (c *IrcClient) IsClosed() bool {
	return c.shutdown
}

// Reads a message from the read channel in it's entirety. More efficient than
// read because read requires you to allocate your own buffer, but since we're
// dealing in routines and splitting the buffer the reality is another buffer
// has been already allocated to copy the bytes recieved anyways.
func (c *IrcClient) ReadMessage() ([]byte, bool) {
	ret, ok := <-c.readchan
	if !ok {
		return nil, ok
	}
	return ret, ok
}

// Read implements the io.Reader interface, but this method is just here for
// convenience. It is not efficient and should probably not even be used.
// Instead use ReadMessage as it it has already allocated a buffer and copied
// the contents into it. Using this method requires an extra buffer allocation
// and extra copying.
func (c *IrcClient) Read(buf []byte) (int, error) {
	if c.pos == 0 {
		var ok bool
		c.readbuf, ok = c.ReadMessage()
		if !ok {
			return 0, io.EOF
		}
	}

	n := copy(buf, c.readbuf[c.pos:])
	c.pos += n
	if c.pos == len(c.readbuf) {
		c.readbuf = nil
		c.pos = 0
	}

	return n, nil
}

// Write implements the io.Writer interface and is the preferred way to write
// to the socket. Returns EOF if the client has been closed. The buffer
// is split based on \r\n and each message is queued, then the Pump is signaled
// through the channel with the number of messages queued. A read lock on a
// mutex is required to write to the channel to ensure any other thread
// cannot close the channel while someone is attempting to write to it.
func (c *IrcClient) Write(buf []byte) (int, error) {
	if c.shutdown {
		return 0, io.EOF
	}

	if len(buf) == 0 {
		return 0, nil
	}

	nMessages := 0
	queue := func(msg []byte) {
		c.queue.Enqueue(msg)
		nMessages++
	}

	start, remaining := findChunks(buf, queue)
	if remaining {
		queue(append(buf[start:], []byte{'\r', '\n'}...))
	}

	c.writeProtect.RLock()
	if c.shutdown {
		c.writeProtect.RUnlock()
		return 0, io.EOF
	}
	c.writechan <- nMessages
	c.writeProtect.RUnlock()
	return len(buf), nil
}

// findChunks calls a callback for each \r\n encountered.
// if there is still a remaining chunk to be dealt with that did not end with
// \r\n the bool return value will be true.
func findChunks(buf []byte, block func([]byte)) (int, bool) {
	var start, i int
	for start, i = 0, 1; i < len(buf); i++ {
		if buf[i-1] == '\r' && buf[i] == '\n' {
			i++
			block(buf[start:i])
			if i == len(buf) {
				return start, false
			}
			start = i
		}
	}

	return start, true
}
