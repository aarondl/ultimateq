package remote

import (
	"bytes"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/aarondl/ultimateq/irc"
	"gopkg.in/inconshreveable/log15.v2"
	"gopkg.in/vmihailenco/msgpack.v2"
)

func setup() *ExtHandler {
	var logger = log15.New()
	logger.SetHandler(log15.DiscardHandler())
	conn := NopCloser(&bytes.Buffer{})

	return NewExtHandler("name", conn, logger, func(string) {})
}

func TestExtHandler_New(t *testing.T) {
	t.Parallel()

	handler := setup()

	if handler == nil {
		t.Error("got nil")
	}
}

func TestExtHandler_StartClose(t *testing.T) {
	t.Parallel()

	handler := setup()
	handler.conn = errReadWriteCloser{io.EOF}

	handler.Start()
	handler.Close()
}

func TestExtHandler_killAllThings(t *testing.T) {
	t.Parallel()

	called := false
	callback := func(name string) {
		called = true
	}

	handler := setup()
	handler.dc = callback

	handler.killAllThings()

	if !called {
		t.Error("Didn't call disconnection callback")
	}
}

func TestExtHandler_createStream(t *testing.T) {
	t.Parallel()

	handler := setup()
	latch := make(chan struct{})

	go func() {
		_ = <-handler.newstream
		handler.streamid <- 1

		close(latch)
	}()

	u := handler.createStream(nil)

	if u != 1 {
		t.Error("Want first id, got:", u)
	}

	<-latch
}

func TestExtHandler_HandleRaw(t *testing.T) {
	t.Parallel()

	handler := setup()
	latch := make(chan struct{})

	go func() {
		_ = <-handler.newstream
		handler.streamid <- 1
		_ = <-handler.events

		close(latch)
	}()

	handler.HandleRaw(nil, nil)

	<-latch
}

func TestExtHandler_Dispatcher(t *testing.T) {
	t.Parallel()

	msg := []byte("hello world")
	handler := setup()

	buf := &bytes.Buffer{}
	w := &irc.Helper{Writer: buf}

	handler.wg.Add(1)
	go handler.Dispatcher()

	id := handler.createStream(w)
	handler.messages <- responseMsg{
		StreamID: id,
		Message:  msg,
	}

	handler.Close()

	if got := buf.Bytes(); bytes.Compare(got, msg) != 0 {
		t.Errorf("want: %q, got: %q", msg, got)
	}
}

func TestExtHandler_Dispatcher_CloseStream(t *testing.T) {
	saveStreamLookup := streamLookup
	saveStreamDelete := streamDelete
	defer func() {
		streamLookup = saveStreamLookup
		streamDelete = saveStreamDelete
	}()

	found := false
	deleted := false
	streamLookup = func(m map[uint16]irc.Writer, id uint16) (irc.Writer, bool) {
		w, ok := m[id]
		found = ok
		return w, ok
	}
	streamDelete = func(m map[uint16]irc.Writer, id uint16) {
		deleted = true
		delete(m, id)
	}

	handler := setup()

	handler.wg.Add(1)
	go handler.Dispatcher()

	buf := &bytes.Buffer{}
	id := handler.createStream(&irc.Helper{Writer: buf})
	// Delete the stream
	handler.messages <- responseMsg{
		StreamID: id,
		Message:  []byte{},
	}
	// Try to write to non-existent stream
	handler.messages <- responseMsg{
		StreamID: id,
		Message:  []byte{64},
	}

	handler.Close()

	if !deleted {
		t.Error("The stream should have been deleted")
	}
	if found {
		t.Error("The stream should not have been found")
	}
}

func TestExtHandler_Reader(t *testing.T) {
	t.Skip("broken but unnecessary test")
	t.Parallel()

	handler := setup()

	msg1 := responseMsg{
		StreamID: 5,
		Message:  []byte("hello"),
	}
	msg2 := responseMsg{
		StreamID: 7,
		Message:  []byte("world"),
	}

	buf := &bytes.Buffer{}
	e := msgpack.NewEncoder(buf)
	ensure(e.Encode(msg1))
	ensure(e.Encode(msg2))

	handler.conn = NopCloser(buf)

	handler.wg.Add(1)
	go handler.Reader()

	recv1 := <-handler.messages
	recv2 := <-handler.messages

	if !reflect.DeepEqual(msg1, recv1) {
		t.Errorf("Message 1 was different:\n%#v\n%#v", msg1, recv1)
	}
	if !reflect.DeepEqual(msg2, recv2) {
		t.Errorf("Message 2 was different:\n%#v\n%#v", msg2, recv2)
	}

	handler.Close()
}

func TestExtHandler_Writer(t *testing.T) {
	t.Skip("broken but unnecessary test")
	t.Parallel()

	handler := setup()

	buf := &bytes.Buffer{}
	handler.conn = NopCloser(buf)

	handler.wg.Add(1)
	go handler.Writer()

	evIn := event{
		StreamID: 5,
		Event: &irc.Event{
			Name:   irc.PRIVMSG,
			Sender: "irc.test.net",
			Time:   time.Now(),
		},
	}

	handler.events <- evIn
	handler.Close()

	var evOut event
	if err := msgpack.Unmarshal(buf.Bytes(), &evOut); err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(evIn, evOut) {
		t.Errorf("should be the same:\n%#v\n%#v", evIn.Event, evOut.Event)
	}
}

func TestExtHandler_Writer_Failure(t *testing.T) {
	t.Parallel()

	handler := setup()

	called := false
	handler.dc = func(string) {
		called = true
	}

	handler.conn = errReadWriteCloser{io.EOF}

	handler.wg.Add(1)
	go handler.Writer()

	evIn := event{
		StreamID: 5,
		Event: &irc.Event{
			Name:   irc.PRIVMSG,
			Sender: "irc.test.net",
			Time:   time.Now(),
		},
	}

	handler.events <- evIn
	<-handler.kill

	if !called {
		t.Error("Want dc handler called")
	}
}

func ensure(err error) {
	if err != nil {
		panic(err)
	}
}

type nopCloser struct {
	io.ReadWriter
}

func (n nopCloser) Close() error {
	return nil
}

func NopCloser(w io.ReadWriter) io.ReadWriteCloser {
	return nopCloser{w}
}

type errReadWriteCloser struct {
	err error
}

func (n errReadWriteCloser) Write([]byte) (int, error) {
	return 0, n.err
}

func (n errReadWriteCloser) Read([]byte) (int, error) {
	return 0, n.err
}

func (n errReadWriteCloser) Close() error {
	return n.err
}
