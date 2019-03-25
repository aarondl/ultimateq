package dispatch

import (
	"bytes"
	"sync/atomic"
	"testing"

	"github.com/aarondl/ultimateq/irc"
	"gopkg.in/inconshreveable/log15.v2"
)

type testPoint struct {
	irc.Helper
}

type testCallback func(w irc.Writer, ev *irc.Event)

type testHandler struct {
	callback testCallback
}

func (handler testHandler) Handle(w irc.Writer, ev *irc.Event) {
	if handler.callback != nil {
		handler.callback(w, ev)
	}
}

func TestDispatcher(t *testing.T) {
	t.Parallel()
	d := NewDispatcher(NewCore(nil))
	if d == nil || d.trie == nil {
		t.Error("Initialization failed.")
	}
}

func TestDispatcherRegistration(t *testing.T) {
	t.Parallel()
	d := NewDispatcher(NewCore(nil))
	handler := testHandler{}

	id := d.Register("", "", irc.PRIVMSG, handler)
	if id == 0 {
		t.Error("It should have given back an id.")
	}
	id2 := d.Register("", "", irc.PRIVMSG, handler)
	if id == id2 {
		t.Error("It should not produce duplicate ids.")
	}
	if !d.Unregister(id) {
		t.Error("It should unregister via it's id")
	}
	if d.Unregister(id) {
		t.Error("It should not unregister the same event multiple times.")
	}
}

func TestDispatcherDispatch(t *testing.T) {
	t.Parallel()
	d := NewDispatcher(NewCore(nil))

	var count int64
	handler := testHandler{callback: func(irc.Writer, *irc.Event) {
		atomic.AddInt64(&count, 1)
	}}

	id := d.Register("", "", irc.PRIVMSG, handler)
	if id == 0 {
		t.Error("It should have given back an id.")
	}
	id2 := d.Register("", "", irc.PRIVMSG, handler)
	if id == id2 {
		t.Error("It should not produce duplicate ids.")
	}

	ev := irc.NewEvent("network", irc.NewNetworkInfo(), irc.PRIVMSG, "server", "#chan", "hey guys")
	d.Dispatch(nil, ev)
	d.WaitForHandlers()

	if count != 2 {
		t.Error("want 2 calls on the handler, got:", count)
	}
}

func TestDispatcherPanic(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := log15.New()
	logger.SetHandler(log15.StreamHandler(buf, log15.LogfmtFormat()))

	logCore := NewCore(logger)
	d := NewDispatcher(logCore)

	panicMsg := "dispatch panic"
	handler := testHandler{
		func(w irc.Writer, ev *irc.Event) {
			panic(panicMsg)
		},
	}

	d.Register("", "", "", handler)
	ev := irc.NewEvent("network", netInfo, "dispatcher", irc.PRIVMSG, "panic test")
	d.Dispatch(testPoint{irc.Helper{}}, ev)
	d.WaitForHandlers()

	logStr := buf.String()

	if logStr == "" {
		t.Error("Expected not empty log.")
	}

	logBytes := buf.Bytes()
	if !bytes.Contains(logBytes, []byte(panicMsg)) {
		t.Errorf("Log does not contain: %s\n%s", panicMsg, logBytes)
	}

	if !bytes.Contains(logBytes, []byte("dispatcher_test.go")) {
		t.Error("Does not contain a reference to file that panic'd")
	}
}
