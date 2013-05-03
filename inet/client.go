// inet package handles connecting to an irc server and reading and writing to
// the connection
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

// IrcClient represents a connection to an irc server. It uses a queueing system
// to throttle writes to the server. And it implements ReadWriteCloser interface
type IrcClient struct {
	shutdown  bool
	conn      net.Conn
	readchan  chan []byte
	writechan chan int
	queue     Queue
	waiter    sync.WaitGroup

	// write throttling
	lastwrite  time.Time
	nThrottled int

	// buffering for io.Reader interface
	readbuf []byte
	pos     int
}

// CreateIrcClient initializes the required fields in the IrcClient
func CreateIrcClient(conn net.Conn) *IrcClient {
	return &IrcClient{
		conn:      conn,
		readchan:  make(chan []byte),
		writechan: make(chan int),
		lastwrite: time.Now().Truncate(resetDuration),
	}
}

// SpawnWorkers creates two goroutines, one that is constantly reading using
// Siphon, and one that is constantly working on eliminating the write queue by
// writing. Also sets up the instances waiter, and a subsequent call to Wait()
// will block until the workers have returned.
func (c *IrcClient) SpawnWorkers() {
	c.waiter.Add(2)
	go c.Pump()
	go c.Siphon()
}

// Wait blocks until the workers in SpawnWorker have returned.
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

	if dur > (3 * time.Second) {
		c.nThrottled = 0
		return time.Duration(0)
	} else {
		sleep := time.Second * time.Duration(
			0.5+math.Max(0, math.Log2(float64(c.nThrottled))),
		)
		c.nThrottled += 1
		return sleep
	}
}

// Pump is meant to be run off the main thread and dequeues the messages and
// writes them to the connection.
func (c *IrcClient) Pump() {
	var n int
	var err error
	for !c.shutdown && err == nil {
		nMessages, ok := <-c.writechan
		if !ok {
			break
		}
		toWrite := c.queue.Dequeue(nMessages)

		for _, msg := range toWrite {
			sleepTime := c.calcSleepTime(time.Now())
			if sleepTime > 0 {
				time.Sleep(sleepTime)
			}

			for written := 0; written < len(msg); written += n {
				log.Println("<-", string(msg[written:len(msg)-2]))
				n, err = c.conn.Write(msg[written:])
				c.lastwrite = time.Now()
				if err != nil {
					break
				}
			}

			if err != nil {
				break
			}
		}
	}

	c.waiter.Done()
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
			// Do something useful.
			// Log/Reconnect etc.
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
	var i int
	start := 0
	for i = 1; i < len(buf); i++ {
		if buf[i-1] == '\r' && buf[i] == '\n' {

			cpy := make([]byte, i-start-2+1)
			copy(cpy, buf[start:i-2+1])
			log.Println("->", string(cpy))
			c.readchan <- cpy

			i++ // skip over checking [\n] + [first byte]
			start = i
		}
	}

	if start < len(buf) {
		copy(buf[:len(buf)-start], buf[start:])
		return len(buf) - start
	}

	return 0
}

// Close sets the shutdown variable, closes the writer channel, and closes
// the underlying socket. All of these measures should ensure the worker
// routines shutdown quickly.
func (c *IrcClient) Close() error {
	c.shutdown = true
	close(c.writechan)
	return c.conn.Close()
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
// directly to the socket. Returns EOF if the client has been closed.
func (c *IrcClient) Write(buf []byte) (int, error) {
	if c.shutdown {
		return 0, io.EOF
	}
	c.queue.Enqueue(buf)
	c.writechan <- 1
	return len(buf), nil
}
