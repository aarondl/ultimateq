package dispatch

import (
	"bytes"
	"log"
	"testing"

	"github.com/aarondl/ultimateq/irc"
)

var core = NewDispatchCore()
var logBuffer = &bytes.Buffer{}

func init() {
	log.SetOutput(logBuffer)
}

//===========================================================
// Set up a type that can be used to mock irc.Writer
//===========================================================
type testPoint struct {
	*irc.Helper
}

func (tsender testPoint) GetKey() string {
	return ""
}

//===========================================================
// Set up a type that can be used to mock a callback for raw.
//===========================================================
type testCallback func(ev *irc.Event, w irc.Writer)

type testHandler struct {
	callback testCallback
}

func (handler testHandler) HandleRaw(ev *irc.Event, w irc.Writer) {
	if handler.callback != nil {
		handler.callback(ev, w)
	}
}

//===========================================================
// Tests
//===========================================================
func TestDispatcher(t *testing.T) {
	t.Parallel()
	d := NewDispatcher(core)
	if d == nil || d.events == nil {
		t.Error("Initialization failed.")
	}
}

func TestDispatcher_Registration(t *testing.T) {
	t.Parallel()
	d := NewDispatcher(core)
	handler := testHandler{}

	id := d.Register(irc.PRIVMSG, handler)
	if id == 0 {
		t.Error("It should have given back an id.")
	}
	id2 := d.Register(irc.PRIVMSG, handler)
	if id == id2 {
		t.Error("It should not produce duplicate ids.")
	}
	if !d.Unregister("privmsg", id) {
		t.Error("It should unregister via it's id regardless of string case")
	}
	if d.Unregister("privmsg", id) {
		t.Error("It should not unregister the same event multiple times.")
	}
}

func TestDispatcher_Dispatching(t *testing.T) {
	t.Parallel()
	var msg1, msg2, msg3 *irc.Event
	var s1, s2 irc.Writer
	h1 := testHandler{func(m *irc.Event, s irc.Writer) {
		msg1 = m
		s1 = s
	}}
	h2 := testHandler{func(m *irc.Event, s irc.Writer) {
		msg2 = m
		s2 = s
	}}
	h3 := testHandler{func(m *irc.Event, s irc.Writer) {
		msg3 = m
	}}

	d := NewDispatcher(core)
	send := testPoint{&irc.Helper{}}

	d.Register(irc.PRIVMSG, h1)
	d.Register(irc.PRIVMSG, h2)
	d.Register(irc.QUIT, h3)

	privmsg := &irc.Event{Name: irc.PRIVMSG}
	quitmsg := &irc.Event{Name: irc.QUIT}
	d.Dispatch(privmsg, send)
	d.WaitForHandlers()

	if msg1 == nil {
		t.Error("Failed to dispatch to h1.")
	}
	if msg1.Name != irc.PRIVMSG {
		t.Error("Got the wrong ev name:", msg1.Name)
	}
	if msg1 != msg2 {
		t.Error("Failed to dispatch to msg2, or the ev data is not shared.")
	}
	if s1 == nil {
		t.Error("The endpoint should never be nil.")
	}
	if s1 != s2 {
		t.Error("The endpoint should be shared.")
	}
	if msg3 != nil {
		t.Error("Erroneously dispatched to h3.")
	}

	d.Dispatch(quitmsg, send)
	d.WaitForHandlers()
	if msg3.Name != irc.QUIT {
		t.Error("Failed to dispatch to h3.")
	}
}

func TestDispatcher_RawDispatch(t *testing.T) {
	t.Parallel()
	var msg1, msg2 *irc.Event
	h1 := testHandler{func(m *irc.Event, send irc.Writer) {
		msg1 = m
	}}
	h2 := testHandler{func(m *irc.Event, send irc.Writer) {
		msg2 = m
	}}

	d := NewDispatcher(core)
	send := testPoint{&irc.Helper{}}
	d.Register(irc.PRIVMSG, h1)
	d.Register(irc.RAW, h2)

	privmsg := &irc.Event{Name: irc.PRIVMSG}
	d.Dispatch(privmsg, send)
	d.WaitForHandlers()
	if msg1 != privmsg {
		t.Error("Failed to dispatch to privmsg handler.")
	}
	if msg1 != msg2 {
		t.Error("Failed to dispatch to raw.")
	}
}

// ================================
// Testing types
// ================================
type testCallbackMsg func(*irc.Event, irc.Writer)
type testCTCPCallbackMsg func(*irc.Event, string, string, irc.Writer)

type testPrivmsgHandler struct {
	callback testCallbackMsg
}
type testPrivmsgUserHandler struct {
	callback testCallbackMsg
}
type testPrivmsgChannelHandler struct {
	callback testCallbackMsg
}
type testPrivmsgAllHandler struct {
	testCallbackNormal, testCallbackUser, testCallbackChannel testCallbackMsg
}
type testNoticeHandler struct {
	callback testCallbackMsg
}
type testNoticeUserHandler struct {
	callback testCallbackMsg
}
type testNoticeChannelHandler struct {
	callback testCallbackMsg
}
type testNoticeAllHandler struct {
	testCallbackNormal, testCallbackUser, testCallbackChannel testCallbackMsg
}
type testCTCPHandler struct {
	callback testCTCPCallbackMsg
}
type testCTCPChannelHandler struct {
	callback testCTCPCallbackMsg
}
type testCTCPAllHandler struct {
	testCallbackNormal, testCallbackChannel testCTCPCallbackMsg
}
type testCTCPReplyHandler struct {
	callback testCTCPCallbackMsg
}

// ================================
// Testing Callbacks
// ================================
func (t testPrivmsgHandler) Privmsg(ev *irc.Event, w irc.Writer) {
	t.callback(ev, w)
}
func (t testPrivmsgUserHandler) PrivmsgUser(
	ev *irc.Event, w irc.Writer) {

	t.callback(ev, w)
}
func (t testPrivmsgChannelHandler) PrivmsgChannel(
	ev *irc.Event, w irc.Writer) {

	t.callback(ev, w)
}
func (t testPrivmsgAllHandler) Privmsg(
	ev *irc.Event, w irc.Writer) {

	t.testCallbackNormal(ev, w)
}
func (t testPrivmsgAllHandler) PrivmsgUser(
	ev *irc.Event, w irc.Writer) {

	t.testCallbackUser(ev, w)
}
func (t testPrivmsgAllHandler) PrivmsgChannel(
	ev *irc.Event, w irc.Writer) {

	t.testCallbackChannel(ev, w)
}
func (t testNoticeHandler) Notice(ev *irc.Event, w irc.Writer) {
	t.callback(ev, w)
}
func (t testNoticeUserHandler) NoticeUser(
	ev *irc.Event, w irc.Writer) {

	t.callback(ev, w)
}
func (t testNoticeChannelHandler) NoticeChannel(
	ev *irc.Event, w irc.Writer) {

	t.callback(ev, w)
}
func (t testNoticeAllHandler) Notice(
	ev *irc.Event, w irc.Writer) {

	t.testCallbackNormal(ev, w)
}
func (t testNoticeAllHandler) NoticeUser(
	ev *irc.Event, w irc.Writer) {

	t.testCallbackUser(ev, w)
}
func (t testNoticeAllHandler) NoticeChannel(
	ev *irc.Event, w irc.Writer) {

	t.testCallbackChannel(ev, w)
}
func (t testCTCPHandler) CTCP(ev *irc.Event, a, b string, w irc.Writer) {
	t.callback(ev, a, b, w)
}
func (t testCTCPChannelHandler) CTCPChannel(
	ev *irc.Event, a, b string, w irc.Writer) {

	t.callback(ev, a, b, w)
}
func (t testCTCPAllHandler) CTCP(
	ev *irc.Event, a, b string, w irc.Writer) {

	t.testCallbackNormal(ev, a, b, w)
}
func (t testCTCPAllHandler) CTCPChannel(
	ev *irc.Event, a, b string, w irc.Writer) {

	t.testCallbackChannel(ev, a, b, w)
}
func (t testCTCPReplyHandler) CTCPReply(
	ev *irc.Event, a, b string, w irc.Writer) {

	t.callback(ev, a, b, w)
}

var privChanmsg = &irc.Event{
	Name:        irc.PRIVMSG,
	Args:        []string{"#chan", "ev"},
	Sender:      "nick!user@host.com",
	NetworkInfo: netInfo,
}
var privUsermsg = &irc.Event{
	Name:        irc.PRIVMSG,
	Args:        []string{"user", "ev"},
	Sender:      "nick!user@host.com",
	NetworkInfo: netInfo,
}

func TestDispatcher_Privmsg(t *testing.T) {
	t.Parallel()
	var p, pu, pc *irc.Event
	ph := testPrivmsgHandler{func(m *irc.Event, _ irc.Writer) {
		p = m
	}}
	puh := testPrivmsgUserHandler{func(m *irc.Event, _ irc.Writer) {
		pu = m
	}}
	pch := testPrivmsgChannelHandler{func(m *irc.Event, _ irc.Writer) {
		pc = m
	}}

	d := NewDispatcher(core)
	d.Register(irc.PRIVMSG, ph)
	d.Register(irc.PRIVMSG, puh)
	d.Register(irc.PRIVMSG, pch)

	d.Dispatch(privUsermsg, nil)
	d.WaitForHandlers()
	if p == nil {
		t.Error("Failed to dispatch to handler.")
	}
	if pu != p {
		t.Error("Failed to dispatch to user handler.")
	}
	if pc != nil {
		t.Error("Erroneously to dispatched to channel handler.")
	}

	p, pu, pc = nil, nil, nil
	d.Dispatch(privChanmsg, nil)
	d.WaitForHandlers()
	if p == nil {
		t.Error("Failed to dispatch to handler.")
	}
	if pu != nil {
		t.Error("Erroneously dispatched to user handler.")
	}
	if pc != p {
		t.Error("Failed to dispatch to channel handler.")
	}
}

func TestDispatcher_PrivmsgMultiple(t *testing.T) {
	t.Parallel()
	var p, pu, pc *irc.Event
	pall := testPrivmsgAllHandler{
		func(m *irc.Event, _ irc.Writer) {
			p = m
		},
		func(m *irc.Event, _ irc.Writer) {
			pu = m
		},
		func(m *irc.Event, _ irc.Writer) {
			pc = m
		},
	}

	d := NewDispatcher(core)
	d.Register(irc.PRIVMSG, pall)

	p, pu, pc = nil, nil, nil
	d.Dispatch(privChanmsg, nil)
	d.WaitForHandlers()
	if p != nil {
		t.Error("Erroneously dispatched to handler.")
	}
	if pu != nil {
		t.Error("Erroneously dispatched to user handler.")
	}
	if pc == nil {
		t.Error("Failed to dispatch to channel handler.")
	}

	p, pu, pc = nil, nil, nil
	d.Dispatch(privUsermsg, nil)
	d.WaitForHandlers()
	if p != nil {
		t.Error("Erroneously dispatched to handler.")
	}
	if pu == nil {
		t.Error("Failed to dispatch to user handler.")
	}
	if pc != nil {
		t.Error("Erroneously dispatched to user handler.")
	}
}

var noticeChanmsg = &irc.Event{
	Name:        irc.NOTICE,
	Args:        []string{"#chan", "ev"},
	Sender:      "nick!user@host.com",
	NetworkInfo: netInfo,
}
var noticeUsermsg = &irc.Event{
	Name:        irc.NOTICE,
	Args:        []string{"user", "ev"},
	Sender:      "nick!user@host.com",
	NetworkInfo: netInfo,
}

func TestDispatcher_Notice(t *testing.T) {
	t.Parallel()
	var n, nu, nc *irc.Event
	nh := testNoticeHandler{func(m *irc.Event, _ irc.Writer) {
		n = m
	}}
	nuh := testNoticeUserHandler{func(m *irc.Event, _ irc.Writer) {
		nu = m
	}}
	nch := testNoticeChannelHandler{func(m *irc.Event, _ irc.Writer) {
		nc = m
	}}

	d := NewDispatcher(core)
	d.Register(irc.NOTICE, nh)
	d.Register(irc.NOTICE, nuh)
	d.Register(irc.NOTICE, nch)

	d.Dispatch(noticeUsermsg, nil)
	d.WaitForHandlers()
	if n == nil {
		t.Error("Failed to dispatch to handler.")
	}
	if nu != n {
		t.Error("Failed to dispatch to user handler.")
	}
	if nc != nil {
		t.Error("Erroneously dispatched to channel handler.")
	}

	n, nu, nc = nil, nil, nil
	d.Dispatch(noticeChanmsg, nil)
	d.WaitForHandlers()
	if n == nil {
		t.Error("Failed to dispatch to handler.")
	}
	if nu != nil {
		t.Error("Erroneously dispatched to user handler.")
	}
	if nc != n {
		t.Error("Failed to dispatch to channel handler.")
	}
}

func TestDispatcher_NoticeMultiple(t *testing.T) {
	t.Parallel()
	var n, nu, nc *irc.Event
	nall := testNoticeAllHandler{
		func(m *irc.Event, _ irc.Writer) {
			n = m
		},
		func(m *irc.Event, _ irc.Writer) {
			nu = m
		},
		func(m *irc.Event, _ irc.Writer) {
			nc = m
		},
	}

	d := NewDispatcher(core)
	d.Register(irc.NOTICE, nall)

	n, nu, nc = nil, nil, nil
	d.Dispatch(noticeChanmsg, nil)
	d.WaitForHandlers()
	if n != nil {
		t.Error("Erroneously dispatched to handler.")
	}
	if nu != nil {
		t.Error("Erroneously dispatched to user handler.")
	}
	if nc == nil {
		t.Error("Failed to dispatch to channel handler.")
	}

	n, nu, nc = nil, nil, nil
	d.Dispatch(noticeUsermsg, nil)
	d.WaitForHandlers()
	if n != nil {
		t.Error("Erroneously dispatched to handler.")
	}
	if nu == nil {
		t.Error("Failed to dispatch to user handler.")
	}
	if nc != nil {
		t.Error("Erroneously dispatched to user handler.")
	}
}

var ctcpChanmsg = &irc.Event{
	Name:        irc.CTCP,
	Args:        []string{"#chan", "\x01msg args\x01"},
	Sender:      "nick!user@host.com",
	NetworkInfo: netInfo,
}
var ctcpMsg = &irc.Event{
	Name:        irc.CTCP,
	Args:        []string{"user", "\x01msg args\x01"},
	Sender:      "nick!user@host.com",
	NetworkInfo: netInfo,
}

func TestDispatcher_CTCP(t *testing.T) {
	t.Parallel()
	var c, cc *irc.Event
	ch := testCTCPHandler{func(m *irc.Event, tag, data string,
		_ irc.Writer) {

		c = m
	}}
	cch := testCTCPChannelHandler{func(m *irc.Event, tag, data string,
		_ irc.Writer) {

		cc = m
	}}

	d := NewDispatcher(core)
	d.Register(irc.CTCP, ch)
	d.Register(irc.CTCP, cch)

	d.Dispatch(ctcpMsg, nil)
	d.WaitForHandlers()
	if c == nil {
		t.Error("Failed to dispatch to handler.")
	}
	if cc != nil {
		t.Error("Erroneously dispatched to channel handler.")
	}

	c, cc = nil, nil
	d.Dispatch(ctcpChanmsg, nil)
	d.WaitForHandlers()
	if c == nil {
		t.Error("Failed to dispatch to handler.")
	}
	if cc == nil {
		t.Error("Failed to dispatch to channel handler.")
	}
}

func TestDispatcher_CTCPMultiple(t *testing.T) {
	t.Parallel()
	var c, cc *irc.Event
	call := testCTCPAllHandler{
		func(m *irc.Event, a, b string, _ irc.Writer) {
			c = m
		},
		func(m *irc.Event, a, b string, _ irc.Writer) {
			cc = m
		},
	}

	d := NewDispatcher(core)
	d.Register(irc.CTCP, call)

	c, cc = nil, nil
	d.Dispatch(ctcpChanmsg, nil)
	d.WaitForHandlers()
	if c != nil {
		t.Error("Erroneously dispatched to handler.")
	}
	if cc == nil {
		t.Error("Failed to dispatch to channel handler.")
	}

	c, cc = nil, nil
	d.Dispatch(ctcpMsg, nil)
	d.WaitForHandlers()
	if c == nil {
		t.Error("Failed to dispatch to handler.")
	}
	if cc != nil {
		t.Error("Erroneously dispatched to user handler.")
	}
}

var ctcpReplyMsg = &irc.Event{
	Name:        irc.CTCPReply,
	Args:        []string{"user", "\x01msg args\x01"},
	Sender:      "nick!user@host.com",
	NetworkInfo: netInfo,
}

func TestDispatcher_CTCPReply(t *testing.T) {
	t.Parallel()
	var c *irc.Event
	ch := testCTCPReplyHandler{func(m *irc.Event, tag, data string,
		_ irc.Writer) {

		c = m
	}}

	d := NewDispatcher(core)
	d.Register(irc.CTCPReply, ch)

	d.Dispatch(ctcpReplyMsg, nil)
	d.WaitForHandlers()
	if c == nil {
		t.Error("Failed to dispatch to handler.")
	}
}
func TestDispatcher_FilterPrivmsgChannels(t *testing.T) {
	t.Parallel()
	chanmsg2 := &irc.Event{
		Name:   irc.PRIVMSG,
		Args:   []string{"#chan2", "ev"},
		Sender: "nick!user@host.com",
	}

	var p, pc *irc.Event
	ph := testPrivmsgHandler{func(m *irc.Event, _ irc.Writer) {
		p = m
	}}
	pch := testPrivmsgChannelHandler{func(m *irc.Event, _ irc.Writer) {
		pc = m
	}}

	dcore := NewDispatchCore("#CHAN")
	d := NewDispatcher(dcore)
	d.Register(irc.PRIVMSG, ph)
	d.Register(irc.PRIVMSG, pch)

	d.Dispatch(privChanmsg, nil)
	d.WaitForHandlers()
	if p == nil {
		t.Error("Failed to dispatch to handler.")
	}
	if pc != p {
		t.Error("Failed to dispatch to channel handler.")
	}

	p, pc = nil, nil
	d.Dispatch(chanmsg2, nil)
	d.WaitForHandlers()
	if p == nil {
		t.Error("Failed to dispatch to handler.")
	}
	if pc != nil {
		t.Error("Erronously dispatched to channel handler.")
	}
}

func TestDispatcher_FilterNoticeChannels(t *testing.T) {
	t.Parallel()
	chanmsg2 := &irc.Event{
		Name:   irc.NOTICE,
		Args:   []string{"#chan2", "ev"},
		Sender: "nick!user@host.com",
	}

	var u, uc *irc.Event
	uh := testNoticeHandler{func(m *irc.Event, _ irc.Writer) {
		u = m
	}}
	uch := testNoticeChannelHandler{func(m *irc.Event, _ irc.Writer) {
		uc = m
	}}

	dcore := NewDispatchCore("#CHAN")
	d := NewDispatcher(dcore)
	d.Register(irc.NOTICE, uh)
	d.Register(irc.NOTICE, uch)

	d.Dispatch(noticeChanmsg, nil)
	d.WaitForHandlers()
	if u == nil {
		t.Error("Failed to dispatch to handler.")
	}
	if uc != u {
		t.Error("Failed to dispatch to channel handler.")
	}

	u, uc = nil, nil
	d.Dispatch(chanmsg2, nil)
	d.WaitForHandlers()
	if u == nil {
		t.Error("Failed to dispatch to handler.")
	}
	if uc != nil {
		t.Error("Erronously dispatched to channel handler.")
	}
}

func TestDispatchCore_ShouldDispatch(t *testing.T) {
	t.Parallel()
	d := NewDispatcher(core)

	var tests = []struct {
		IsChan bool
		Target string
		Expect bool
	}{
		{true, "#chan", true},
		{false, "#chan2", false},
		{true, "user", false},
		{false, "user", true},
	}

	for _, test := range tests {
		ev := irc.NewEvent("", netInfo, irc.PRIVMSG, "", test.Target)
		should := d.shouldDispatch(test.IsChan, ev)
		if should != test.Expect {
			t.Error("Fail:", test)
			t.Error("Expected:", test.Expect, "got:", should)
		}
	}
}

func TestDispatch_Panic(t *testing.T) {
	logBuffer.Reset()
	d := NewDispatcher(core)
	panicMsg := "dispatch panic"

	handler := testHandler{
		func(ev *irc.Event, w irc.Writer) {
			panic(panicMsg)
		},
	}

	d.Register(irc.RAW, handler)
	ev := irc.NewEvent("", netInfo, "dispatcher", irc.PRIVMSG, "panic test")
	d.Dispatch(ev, testPoint{&irc.Helper{}})
	d.WaitForHandlers()
	logStr := logBuffer.String()

	if logStr == "" {
		t.Error("Expected not empty log.")
	}

	logBytes := logBuffer.Bytes()
	if !bytes.Contains(logBytes, []byte(panicMsg)) {
		t.Errorf("Log does not contain: %s\n%s", panicMsg, logBytes)
	}

	if !bytes.Contains(logBytes, []byte("dispatcher_test.go")) {
		t.Error("Does not contain a reference to file that panic'd")
	}
}
