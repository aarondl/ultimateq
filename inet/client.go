/*
Package inet handles connecting to an irc server and reading and writing to
the connection
*/
package inet

import (
	"bytes"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

const (
	// bufferSize is the size of the buffer to be allocated for writes
	bufferSize = 16348
	// nBufferedWrites is how many writes can succeed before blocking
	nBufferedWrites = 25
	// defaultTimeScale is the default scale of the sleeps and timeouts.
	defaultTimeScale = time.Second
)

var (
	// pong allows replies from pong to write directly without waiting on sleeps
	pong = []byte("PONG")
	// ping is a constant message that is sent to keep the connection alive
	ping = []byte("PING :ping\r\n")
)

// Format strings for errors and logging output
const (
	fmtDiscarded          = "(%v) <- (DISCARDED) %s\n"
	fmtWrite              = "(%v) <- %s\n"
	fmtWriteErr           = "(%v) <- (%v) %s\n"
	fmtRead               = "(%v) -> %s\n"
	fmtErrSiphonReadError = "inet: (%v) read socket error (%s)\n"
	fmtErrPumpReadError   = "inet: (%v) write socket error (%s)\n"
	fmtErrSiphonClosed    = "inet: (%v) siphon closed (%s)\n"
	fmtErrPumpClosed      = "inet: (%v) pump closed (%s)\n"
	errMsgShutdown        = "Shut Down"
)

// IrcClient represents a connection to an irc server. It uses a queueing system
// to throttle writes to the server. And it implements ReadWriteCloser interface
type IrcClient struct {
	isShutdown        bool
	isShutdownProtect sync.RWMutex

	conn        net.Conn
	siphonchan  chan []byte
	pumpchan    chan []byte
	pumpservice chan chan []byte
	killpump    chan int
	killsiphon  chan int
	queue       Queue

	// The name of the connection for logging
	name string

	// write throttling
	lastwrite        time.Time
	penalty          time.Time
	timeout          time.Duration
	basestep         time.Duration
	lenPenaltyFactor float64

	// Time scaling for tests.
	scale time.Duration

	keepalive time.Duration

	// buffering for io.Reader interface
	readbuf []byte
	pos     int
}

// createIrcClient initializes the required fields in the IrcClient
func createIrcClient(conn net.Conn, name string) *IrcClient {
	return &IrcClient{
		name:        name,
		conn:        conn,
		siphonchan:  make(chan []byte),
		pumpchan:    make(chan []byte),
		pumpservice: make(chan chan []byte),
		lastwrite:   time.Time{},
		scale:       defaultTimeScale,
	}
}

// CreateIrcClient creates an irc client with optional flood protection and
// keep alive. scale is used to round the final sleeping values as well as
// scale the penalties incurred by lenPenaltyFactor, if 0 it is time.Second.
func CreateIrcClient(conn net.Conn, name string, lenPenaltyFactor int,
	timeout, basestep, keepalive, scale time.Duration) *IrcClient {

	c := createIrcClient(conn, name)
	if scale != 0 {
		c.scale = scale
	}
	if lenPenaltyFactor > 0 {
		c.lenPenaltyFactor = 1.0 / float64(lenPenaltyFactor)
	}
	c.timeout = timeout
	c.basestep = basestep
	c.keepalive = keepalive
	return c
}

// SpawnWorkers creates two goroutines, one that is constantly reading using
// Siphon, and one that is constantly working on eliminating the write queue by
// writing. Also sets up the instances kill channels.
func (c *IrcClient) SpawnWorkers(pump, siphon bool) {
	c.isShutdownProtect.Lock()
	defer c.isShutdownProtect.Unlock()
	c.isShutdown = false

	if pump {
		c.killpump = make(chan int)
		go c.pump()
	}
	if siphon {
		c.killsiphon = make(chan int)
		go c.siphon()
	}
}

// calcSleepTime calculates the sleep time required by the flood protection
// given a time of write.
func (c *IrcClient) calcSleepTime(t time.Time, msgLen int) time.Duration {
	roundToScale := func(in time.Duration) time.Duration {
		return ((in + (c.scale / 2)) / c.scale) * c.scale
	}

	if c.lastwrite.After(c.penalty) {
		c.penalty = c.lastwrite
	}

	applyPenalty := roundToScale(c.penalty.Sub(t)) >= c.timeout
	c.penalty = c.penalty.Add(c.basestep + roundToScale(
		time.Duration(float64(c.scale)*float64(msgLen)*c.lenPenaltyFactor)))

	if applyPenalty {
		sleep := roundToScale(c.penalty.Sub(t) - c.timeout)
		if sleep > c.timeout {
			sleep = c.timeout
		}
		return sleep
	}

	return 0
}

// pump enqueues the messages given to Write and writes them to the connection.
// It also sleeps a don't-get-glined amount of time between writes.
func (c *IrcClient) pump() {
	var err error
	var sleeper <-chan time.Time
	var pinger <-chan time.Time
	if c.keepalive != 0 {
		pinger = time.After(c.keepalive)
	}
	defer close(c.pumpservice)

	for err == nil {
		select {
		case c.pumpservice <- c.pumpchan:
			message := <-c.pumpchan
			if len(message) == 0 {
				break
			}

			if bytes.HasPrefix(message, pong) {
				if err = c.writeMessage(message); err != nil {
					break
				}
			} else if sleeper == nil {
				sleepTime := c.calcSleepTime(time.Now(), len(message))
				if sleepTime == 0 {
					if err = c.writeMessage(message); err != nil {
						break
					}
				} else {
					c.queue.Enqueue(message)
					sleeper = time.After(sleepTime)
				}
			} else {
				c.queue.Enqueue(message)
			}
		case <-sleeper:
			message := c.queue.Dequeue()
			if err = c.writeMessage(message); err != nil {
				break
			}
			if c.queue.length > 0 {
				sleepTime := c.calcSleepTime(time.Now(), len(message))
				sleeper = time.After(sleepTime)
			} else {
				sleeper = nil
			}
		case <-pinger:
			if sleeper != nil {
				c.queue.Enqueue(ping)
			} else {
				if err = c.writeMessage(ping); err != nil {
					break
				}
			}
			pinger = time.After(c.keepalive)
		case <-c.killpump:
			log.Printf(fmtErrPumpClosed, c.name, errMsgShutdown)
			return
		}
	}

	<-c.killpump
}

// writeMessage writes a byte array out to the socket, sets the last write time.
func (c *IrcClient) writeMessage(msg []byte) error {
	var n int
	var err error
	for written := 0; written < len(msg); written += n {
		n, err = c.conn.Write(msg[written:])
		wrote := msg[written : len(msg)-2]
		if err != nil {
			log.Printf(fmtWriteErr, c.name, err, wrote)
			return err
		}
		log.Printf(fmtWrite, c.name, wrote)
		c.lastwrite = time.Now()
	}
	return nil
}

// Siphon takes messages from the connection given to the IrcClient and then
// uses extractMessages to send them to the readchan.
func (c *IrcClient) siphon() {
	buf := make([]byte, bufferSize)

	var err error
	var shutdown bool
	var position, n int

	for err == nil {
		n, err = c.conn.Read(buf[position:])

		if n > 0 && (err == nil || err == io.EOF) {
			position, shutdown = c.extractMessages(buf[:n+position])
			if shutdown {
				return
			}
		}

		if err != nil {
			log.Printf(fmtErrSiphonReadError, c.name, err)
			break
		}
	}

	close(c.siphonchan)
	<-c.killsiphon
}

// extractMessages takes the information in a buffer and splits on \r\n pairs.
// When it encounters a pair, it creates a copy of the data from start to the
// pair and passes it into the readchan from the IrcClient. If no \r\n is found
// but data is still present in the buffer, it moves this data to the front of
// the buffer and returns an index from which the next read should be started at
//
// Note:
// the reason for the copy is because of the threadedness, the buffer pointed to
// by the slice can be immediately filled with new information once this
// function returns and therefore a copy must be made for thread safety.
func (c *IrcClient) extractMessages(buf []byte) (int, bool) {

	send := func(chunk []byte) bool {
		cpy := make([]byte, len(chunk)-2)
		copy(cpy, chunk[:len(chunk)-2])
		select {
		case c.siphonchan <- cpy:
			log.Printf(fmtRead, c.name, cpy)
			return false
		case <-c.killsiphon:
			return true
		}
	}

	start, remaining, abort := findChunks(buf, send)
	if abort {
		return 0, true
	} else if remaining {
		copy(buf[:len(buf)-start], buf[start:])
		return len(buf) - start, false
	}

	return 0, false
}

// Close closes the socket, sets an all-consuming dequeuer routine to
// eat all the waiting-to-write goroutines, and then waits to acquire a mutex
// that will allow it to safely close the writer channel and set a shutdown var.
func (c *IrcClient) Close() error {
	if c.IsClosed() {
		return nil
	}

	err := c.conn.Close()

	c.isShutdownProtect.Lock()
	c.isShutdown = true

	if c.killpump != nil {
		c.killpump <- 0
	}
	if c.killsiphon != nil {
		c.killsiphon <- 0
	}
	c.isShutdownProtect.Unlock()

	return err
}

// IsClosed returns true if the IrcClient has been closed.
func (c *IrcClient) IsClosed() bool {
	c.isShutdownProtect.RLock()
	defer c.isShutdownProtect.RUnlock()
	return c.isShutdown
}

// ReadMessage gets message from the read channel in it's entirety.
// More efficient than read because read requires you to allocate your own
// buffer, but since we're dealing in routines and splitting the buffer the
// reality is another buffer has been already allocated to copy the bytes
// recieved anyways.
func (c *IrcClient) ReadMessage() ([]byte, bool) {
	ret, ok := <-c.siphonchan
	if !ok {
		return nil, ok
	}
	return ret, ok
}

// ReadChannel retrieves the channel that's used to read.
func (c *IrcClient) ReadChannel() <-chan []byte {
	return c.siphonchan
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
	n := len(buf)
	if n == 0 {
		return 0, nil
	}

	write := func(msg []byte) bool {
		copybuf := make([]byte, len(msg))
		copy(copybuf, msg)
		service, ok := <-c.pumpservice
		if !ok {
			return true
		}
		service <- copybuf
		return false
	}

	start, remaining, abort := findChunks(buf, write)
	if abort {
		return 0, io.EOF
	} else if remaining {
		if write(append(buf[start:], []byte{'\r', '\n'}...)) {
			return start, io.EOF
		}
	}

	return n, nil
}

// findChunks calls a callback for each \r\n encountered.
// if there is still a remaining chunk to be dealt with that did not end with
// \r\n the bool return value will be true.
func findChunks(buf []byte, block func([]byte) bool) (int, bool, bool) {
	var start, i int
	for start, i = 0, 1; i < len(buf); i++ {
		if buf[i-1] == '\r' && buf[i] == '\n' {
			i++
			if block(buf[start:i]) {
				return start, false, true
			}
			if i == len(buf) {
				return start, false, false
			}
			start = i
		}
	}

	return start, true, false
}
