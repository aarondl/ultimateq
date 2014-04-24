package dispatch

import (
	"bytes"
	"log"
	. "testing"

	"github.com/aarondl/ultimateq/irc"
)

var core = CreateDispatchCore(irc.CreateProtoCaps())
var logBuffer = &bytes.Buffer{}

func init() {
	log.SetOutput(logBuffer)
}

//===========================================================
// Set up a type that can be used to mock irc.Endpoint
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
type testCallback func(msg *irc.Message, ep irc.Endpoint)

type testHandler struct {
	callback testCallback
}

func (handler testHandler) HandleRaw(msg *irc.Message, ep irc.Endpoint) {
	if handler.callback != nil {
		handler.callback(msg, ep)
	}
}

//===========================================================
// Tests
//===========================================================
func TestDispatcher(t *T) {
	t.Parallel()
	d := CreateDispatcher(core)
	if d == nil || d.events == nil {
		t.Error("Initialization failed.")
	}
}

func TestDispatcher_Registration(t *T) {
	t.Parallel()
	d := CreateDispatcher(core)
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

func TestDispatcher_Dispatching(t *T) {
	t.Parallel()
	var msg1, msg2, msg3 *irc.Message
	var s1, s2 irc.Endpoint
	h1 := testHandler{func(m *irc.Message, s irc.Endpoint) {
		msg1 = m
		s1 = s
	}}
	h2 := testHandler{func(m *irc.Message, s irc.Endpoint) {
		msg2 = m
		s2 = s
	}}
	h3 := testHandler{func(m *irc.Message, s irc.Endpoint) {
		msg3 = m
	}}

	d := CreateDispatcher(core)
	send := testPoint{&irc.Helper{}}

	d.Register(irc.PRIVMSG, h1)
	d.Register(irc.PRIVMSG, h2)
	d.Register(irc.QUIT, h3)

	privmsg := &irc.Message{Name: irc.PRIVMSG}
	quitmsg := &irc.Message{Name: irc.QUIT}
	d.Dispatch(privmsg, send)
	d.WaitForHandlers()

	if msg1 == nil {
		t.Error("Failed to dispatch to h1.")
	}
	if msg1.Name != irc.PRIVMSG {
		t.Error("Got the wrong msg name:", msg1.Name)
	}
	if msg1 != msg2 {
		t.Error("Failed to dispatch to msg2, or the msg data is not shared.")
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

func TestDispatcher_RawDispatch(t *T) {
	t.Parallel()
	var msg1, msg2 *irc.Message
	h1 := testHandler{func(m *irc.Message, send irc.Endpoint) {
		msg1 = m
	}}
	h2 := testHandler{func(m *irc.Message, send irc.Endpoint) {
		msg2 = m
	}}

	d := CreateDispatcher(core)
	send := testPoint{&irc.Helper{}}
	d.Register(irc.PRIVMSG, h1)
	d.Register(irc.RAW, h2)

	privmsg := &irc.Message{Name: irc.PRIVMSG}
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
type testCallbackMsg func(*irc.Message, irc.Endpoint)
type testCTCPCallbackMsg func(*irc.Message, string, string, irc.Endpoint)

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
func (t testPrivmsgHandler) Privmsg(msg *irc.Message, ep irc.Endpoint) {
	t.callback(msg, ep)
}
func (t testPrivmsgUserHandler) PrivmsgUser(
	msg *irc.Message, ep irc.Endpoint) {

	t.callback(msg, ep)
}
func (t testPrivmsgChannelHandler) PrivmsgChannel(
	msg *irc.Message, ep irc.Endpoint) {

	t.callback(msg, ep)
}
func (t testPrivmsgAllHandler) Privmsg(
	msg *irc.Message, ep irc.Endpoint) {

	t.testCallbackNormal(msg, ep)
}
func (t testPrivmsgAllHandler) PrivmsgUser(
	msg *irc.Message, ep irc.Endpoint) {

	t.testCallbackUser(msg, ep)
}
func (t testPrivmsgAllHandler) PrivmsgChannel(
	msg *irc.Message, ep irc.Endpoint) {

	t.testCallbackChannel(msg, ep)
}
func (t testNoticeHandler) Notice(msg *irc.Message, ep irc.Endpoint) {
	t.callback(msg, ep)
}
func (t testNoticeUserHandler) NoticeUser(
	msg *irc.Message, ep irc.Endpoint) {

	t.callback(msg, ep)
}
func (t testNoticeChannelHandler) NoticeChannel(
	msg *irc.Message, ep irc.Endpoint) {

	t.callback(msg, ep)
}
func (t testNoticeAllHandler) Notice(
	msg *irc.Message, ep irc.Endpoint) {

	t.testCallbackNormal(msg, ep)
}
func (t testNoticeAllHandler) NoticeUser(
	msg *irc.Message, ep irc.Endpoint) {

	t.testCallbackUser(msg, ep)
}
func (t testNoticeAllHandler) NoticeChannel(
	msg *irc.Message, ep irc.Endpoint) {

	t.testCallbackChannel(msg, ep)
}
func (t testCTCPHandler) CTCP(msg *irc.Message, a, b string, ep irc.Endpoint) {
	t.callback(msg, a, b, ep)
}
func (t testCTCPChannelHandler) CTCPChannel(
	msg *irc.Message, a, b string, ep irc.Endpoint) {

	t.callback(msg, a, b, ep)
}
func (t testCTCPAllHandler) CTCP(
	msg *irc.Message, a, b string, ep irc.Endpoint) {

	t.testCallbackNormal(msg, a, b, ep)
}
func (t testCTCPAllHandler) CTCPChannel(
	msg *irc.Message, a, b string, ep irc.Endpoint) {

	t.testCallbackChannel(msg, a, b, ep)
}
func (t testCTCPReplyHandler) CTCPReply(
	msg *irc.Message, a, b string, ep irc.Endpoint) {

	t.callback(msg, a, b, ep)
}

var privChanmsg = &irc.Message{
	Name:   irc.PRIVMSG,
	Args:   []string{"#chan", "msg"},
	Sender: "nick!user@host.com",
}
var privUsermsg = &irc.Message{
	Name:   irc.PRIVMSG,
	Args:   []string{"user", "msg"},
	Sender: "nick!user@host.com",
}

func TestDispatcher_Privmsg(t *T) {
	t.Parallel()
	var p, pu, pc *irc.Message
	ph := testPrivmsgHandler{func(m *irc.Message, _ irc.Endpoint) {
		p = m
	}}
	puh := testPrivmsgUserHandler{func(m *irc.Message, _ irc.Endpoint) {
		pu = m
	}}
	pch := testPrivmsgChannelHandler{func(m *irc.Message, _ irc.Endpoint) {
		pc = m
	}}

	d := CreateDispatcher(core)
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

func TestDispatcher_PrivmsgMultiple(t *T) {
	t.Parallel()
	var p, pu, pc *irc.Message
	pall := testPrivmsgAllHandler{
		func(m *irc.Message, _ irc.Endpoint) {
			p = m
		},
		func(m *irc.Message, _ irc.Endpoint) {
			pu = m
		},
		func(m *irc.Message, _ irc.Endpoint) {
			pc = m
		},
	}

	d := CreateDispatcher(core)
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

var noticeChanmsg = &irc.Message{
	Name:   irc.NOTICE,
	Args:   []string{"#chan", "msg"},
	Sender: "nick!user@host.com",
}
var noticeUsermsg = &irc.Message{
	Name:   irc.NOTICE,
	Args:   []string{"user", "msg"},
	Sender: "nick!user@host.com",
}

func TestDispatcher_Notice(t *T) {
	t.Parallel()
	var n, nu, nc *irc.Message
	nh := testNoticeHandler{func(m *irc.Message, _ irc.Endpoint) {
		n = m
	}}
	nuh := testNoticeUserHandler{func(m *irc.Message, _ irc.Endpoint) {
		nu = m
	}}
	nch := testNoticeChannelHandler{func(m *irc.Message, _ irc.Endpoint) {
		nc = m
	}}

	d := CreateDispatcher(core)
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

func TestDispatcher_NoticeMultiple(t *T) {
	t.Parallel()
	var n, nu, nc *irc.Message
	nall := testNoticeAllHandler{
		func(m *irc.Message, _ irc.Endpoint) {
			n = m
		},
		func(m *irc.Message, _ irc.Endpoint) {
			nu = m
		},
		func(m *irc.Message, _ irc.Endpoint) {
			nc = m
		},
	}

	d := CreateDispatcher(core)
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

var ctcpChanmsg = &irc.Message{
	Name:   irc.CTCP,
	Args:   []string{"#chan", "\x01msg args\x01"},
	Sender: "nick!user@host.com",
}
var ctcpMsg = &irc.Message{
	Name:   irc.CTCP,
	Args:   []string{"user", "\x01msg args\x01"},
	Sender: "nick!user@host.com",
}

func TestDispatcher_CTCP(t *T) {
	t.Parallel()
	var c, cc *irc.Message
	ch := testCTCPHandler{func(m *irc.Message, tag, data string,
		_ irc.Endpoint) {

		c = m
	}}
	cch := testCTCPChannelHandler{func(m *irc.Message, tag, data string,
		_ irc.Endpoint) {

		cc = m
	}}

	d := CreateDispatcher(core)
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

func TestDispatcher_CTCPMultiple(t *T) {
	t.Parallel()
	var c, cc *irc.Message
	call := testCTCPAllHandler{
		func(m *irc.Message, a, b string, _ irc.Endpoint) {
			c = m
		},
		func(m *irc.Message, a, b string, _ irc.Endpoint) {
			cc = m
		},
	}

	d := CreateDispatcher(core)
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

var ctcpReplyMsg = &irc.Message{
	Name:   irc.CTCPReply,
	Args:   []string{"user", "\x01msg args\x01"},
	Sender: "nick!user@host.com",
}

func TestDispatcher_CTCPReply(t *T) {
	t.Parallel()
	var c *irc.Message
	ch := testCTCPReplyHandler{func(m *irc.Message, tag, data string,
		_ irc.Endpoint) {

		c = m
	}}

	d := CreateDispatcher(core)
	d.Register(irc.CTCPReply, ch)

	d.Dispatch(ctcpReplyMsg, nil)
	d.WaitForHandlers()
	if c == nil {
		t.Error("Failed to dispatch to handler.")
	}
}
func TestDispatcher_FilterPrivmsgChannels(t *T) {
	t.Parallel()
	chanmsg2 := &irc.Message{
		Name:   irc.PRIVMSG,
		Args:   []string{"#chan2", "msg"},
		Sender: "nick!user@host.com",
	}

	var p, pc *irc.Message
	ph := testPrivmsgHandler{func(m *irc.Message, _ irc.Endpoint) {
		p = m
	}}
	pch := testPrivmsgChannelHandler{func(m *irc.Message, _ irc.Endpoint) {
		pc = m
	}}

	dcore := CreateDispatchCore(irc.CreateProtoCaps(), "#CHAN")
	d := CreateDispatcher(dcore)
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

func TestDispatcher_FilterNoticeChannels(t *T) {
	t.Parallel()
	chanmsg2 := &irc.Message{
		Name:   irc.NOTICE,
		Args:   []string{"#chan2", "msg"},
		Sender: "nick!user@host.com",
	}

	var u, uc *irc.Message
	uh := testNoticeHandler{func(m *irc.Message, _ irc.Endpoint) {
		u = m
	}}
	uch := testNoticeChannelHandler{func(m *irc.Message, _ irc.Endpoint) {
		uc = m
	}}

	dcore := CreateDispatchCore(irc.CreateProtoCaps(), "#CHAN")
	d := CreateDispatcher(dcore)
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

func TestDispatchCore_ShouldDispatch(t *T) {
	t.Parallel()
	d := CreateDispatcher(core)

	var tests = []struct {
		IsChan bool
		Chan   string
		Expect bool
	}{
		{true, "#chan", true},
		{false, "#chan2", false},
		{true, "user", false},
		{false, "user", true},
	}

	for _, test := range tests {
		should := d.shouldDispatch(test.IsChan, test.Chan)
		if should != test.Expect {
			t.Error("Fail:", test)
			t.Error("Expected:", test.Expect, "got:", should)
		}
	}
}

func TestDispatch_Panic(t *T) {
	logBuffer.Reset()
	d := CreateDispatcher(core)
	panicMsg := "dispatch panic"

	handler := testHandler{
		func(msg *irc.Message, ep irc.Endpoint) {
			panic(panicMsg)
		},
	}

	d.Register(irc.RAW, handler)
	msg := irc.NewMessage("dispatcher", irc.PRIVMSG, "panic test")
	d.Dispatch(msg, testPoint{&irc.Helper{}})
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
