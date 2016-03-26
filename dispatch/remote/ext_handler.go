package remote

import (
	"errors"
	"io"
	"sync"

	"github.com/aarondl/ultimateq/dispatch/cmd"
	"github.com/aarondl/ultimateq/irc"
	"gopkg.in/inconshreveable/log15.v2"
	"gopkg.in/vmihailenco/msgpack.v2"
)

// OnDisconnect is a callback that will be executed when the extension
// handlers connection is disconnected. The name of the extension is passed
// in at that time.
type OnDisconnect func(string)

// event wraps an event with a streamID
type event struct {
	StreamID uint16     `msgpack:"s"`
	Event    *irc.Event `msgpack:"e"`
}

type responseMsg struct {
	StreamID uint16 `msgpack:"s"`
	Message  []byte `msgpack:"m"`
}

// ExtHandler provides a way to dispatch irc events and commands.
type ExtHandler struct {
	logger log15.Logger
	name   string
	conn   io.ReadWriteCloser

	dc OnDisconnect

	events chan event
	kill   chan struct{}

	messages  chan responseMsg
	newstream chan irc.Writer
	streamid  chan uint16
	delstream chan int

	once sync.Once
	wg   sync.WaitGroup
}

// NewExtHandler creates
func NewExtHandler(name string, conn io.ReadWriteCloser, logger log15.Logger, dc OnDisconnect) *ExtHandler {
	e := &ExtHandler{
		logger: logger,
		name:   name,
		conn:   conn,

		dc: dc,

		events: make(chan event),
		kill:   make(chan struct{}),

		messages:  make(chan responseMsg),
		newstream: make(chan irc.Writer),
		streamid:  make(chan uint16),
		delstream: make(chan int),
	}

	return e
}

// Start the the extension handlers writer/readers
func (e *ExtHandler) Start() {
	e.wg.Add(3)
	go e.Dispatcher()
	go e.Writer()
	go e.Reader()
}

// Close kills the reader/writer goroutines and runs cleanup.
func (e *ExtHandler) Close() error {
	e.once.Do(e.killAllThings)
	e.wg.Wait()
	return nil
}

// Close kills the reader/writer goroutines and runs cleanup.
func (e *ExtHandler) killAllThings() {
	close(e.kill)
	_ = e.conn.Close()
	e.dc(e.name)
}

// Dispatch a message to an extension handler
func (e *ExtHandler) HandleRaw(w irc.Writer, ev *irc.Event) {
	streamID := e.createStream(w)
	e.events <- event{
		StreamID: streamID,
		Event:    ev,
	}
}

// Cmd handler for the extension.
func (e *ExtHandler) Cmd(name string, _ irc.Writer, ev *cmd.Event) error {
	//e.createStream(w)
	//e.events <- ev.Event

	return nil
}

func (e *ExtHandler) createStream(w irc.Writer) uint16 {
	e.newstream <- w
	return <-e.streamid
}

// Dispatcher has two responsibilities:
// 1. It will register and unregister event handler streams
// 2. It will dispatch to said streams
func (e *ExtHandler) Dispatcher() {
	// TODO(aarondl): Clean up old irc.Writers
	next := uint16(0)
	streams := make(map[uint16]irc.Writer)

	stop := false
	for !stop {
		select {
		case writer := <-e.newstream:
			streams[next] = writer
			e.streamid <- next
			next++
		case msg := <-e.messages:
			if len(msg.Message) == 0 {
				e.logger.Debug("closing stream", "ext", e.name, "id", msg.StreamID)
				streamDelete(streams, msg.StreamID)
				continue
			}

			w, ok := streamLookup(streams, msg.StreamID)
			if !ok {
				e.logger.Error("invalid stream id", "ext", e.name, "id", msg.StreamID)
				continue
			}

			if _, err := w.Write(msg.Message); err != nil {
				e.logger.Error("failed to write to writer", "ext", e.name, "err", err.Error())
				continue
			}
		case <-e.kill:
			stop = true
		}
	}

	e.wg.Done()
}

// streamLookup is a test harness
var streamLookup = func(m map[uint16]irc.Writer, id uint16) (irc.Writer, bool) {
	w, ok := m[id]
	return w, ok
}

// streamDelete is a test harness
var streamDelete = func(m map[uint16]irc.Writer, id uint16) {
	delete(m, id)
}

// Reader processes incoming IRC messages from the read end of the connection.
func (e *ExtHandler) Reader() {
	var resp responseMsg
	decoder := msgpack.NewDecoder(e.conn)

Forloop:
	for {
		if err := decoder.Decode(&resp); err != nil {
			e.logger.Error(err.Error(), "ext", e.name, "err", "msgpack decode failure")
			break Forloop
		}

		select {
		case e.messages <- resp:
			// Keep processing
		case <-e.kill:
			break Forloop
		}
	}

	e.once.Do(e.killAllThings)
	e.wg.Done()
}

// Writer processes outgoing events in msgpack format for the write end
// of the connection.
func (e *ExtHandler) Writer() {
	var ev event

Forloop:
	for {
		select {
		case <-e.kill:
			e.logger.Info("shutting down remote dispatcher", "name", e.name)
			break Forloop

		case ev = <-e.events:
			eventPayload, err := msgpack.Marshal(ev)
			if err != nil {
				e.logger.Error(err.Error(), "ext", e.name, "loc", "marshal")
				continue Forloop
			}

			if _, err = e.conn.Write(eventPayload); err != nil {
				e.logger.Error(err.Error(), "ext", e.name, "loc", "write")
				e.once.Do(e.killAllThings)
				break Forloop
			}
		}
	}

	e.wg.Done()
}

// Write to the socket with logging
func (e *ExtHandler) Write(b []byte) (int, error) {
	n, err := e.conn.Write(b)
	if n != len(b) {
		e.logger.Error("short write", "ext", e.name, "loc", "write")
	} else if err != nil {
		e.logger.Error("write error", "ext", e.name, "loc", "write")
	}

	return n, err
}

// chainer is a simple type to be able to write multiple times
// and only check the error once
type chainer struct {
	E error
}

// Error returns the error of the underlying error.
func (c chainer) Error() string {
	if c.E == nil {
		return ""
	}
	return c.E.Error()
}

// Write to a writer, but sink the errors. If an error has occurred, subsequent
// writes will not be attempted.
func (c *chainer) Write(b []byte, w io.Writer) {
	if c.E != nil {
		return
	}

	n, err := w.Write(b)
	if n != len(b) {
		c.E = errors.New("short write")
	} else if err != nil {
		c.E = err
	}
}
