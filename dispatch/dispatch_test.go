package dispatch

import (
	"github.com/aarondl/ultimateq/irc"
	. "launchpad.net/gocheck"
	"testing"
)

func Test(t *testing.T) { TestingT(t) } //Hook into testing package
type s struct{}

var _ = Suite(&s{})

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
type testCallback func(msg *irc.IrcMessage, ep irc.Endpoint)

type testHandler struct {
	callback testCallback
}

func (handler testHandler) HandleRaw(msg *irc.IrcMessage, ep irc.Endpoint) {
	if handler.callback != nil {
		handler.callback(msg, ep)
	}
}

//===========================================================
// Tests
//===========================================================
func (s *s) TestDispatcher(c *C) {
	d := CreateDispatcher()
	c.Check(d, NotNil)
	c.Check(d.events, NotNil)
	d, err := CreateRichDispatcher(nil, nil)
	c.Check(err, Equals, errProtoCapsMissing)
	p := irc.CreateProtoCaps()
	p.ParseISupport(&irc.IrcMessage{Args: []string{"nick", "CHANTYPES=H"}})
	d, err = CreateRichDispatcher(p, nil)
	c.Check(err, NotNil)
}

func (s *s) TestDispatcher_Registration(c *C) {
	d := CreateDispatcher()
	cb := testHandler{}

	id := d.Register(irc.PRIVMSG, cb)
	c.Check(id, Not(Equals), 0)
	id2 := d.Register(irc.PRIVMSG, cb)
	c.Check(id2, Not(Equals), id)
	ok := d.Unregister("privmsg", id)
	c.Check(ok, Equals, true)
	ok = d.Unregister("privmsg", id)
	c.Check(ok, Equals, false)
}

func (s *s) TestDispatcher_Dispatching(c *C) {
	var msg1, msg2, msg3 *irc.IrcMessage
	var s1, s2, s3 irc.Endpoint
	h1 := testHandler{func(m *irc.IrcMessage, s irc.Endpoint) {
		msg1 = m
		s1 = s
	}}
	h2 := testHandler{func(m *irc.IrcMessage, s irc.Endpoint) {
		msg2 = m
		s2 = s
	}}
	h3 := testHandler{func(m *irc.IrcMessage, s irc.Endpoint) {
		msg3 = m
		s3 = s
	}}

	d := CreateDispatcher()
	send := testPoint{&irc.Helper{}}

	d.Register(irc.PRIVMSG, h1)
	d.Register(irc.PRIVMSG, h2)
	d.Register(irc.QUIT, h3)

	privmsg := &irc.IrcMessage{Name: irc.PRIVMSG}
	quitmsg := &irc.IrcMessage{Name: irc.QUIT}
	d.Dispatch(privmsg, send)
	d.WaitForCompletion()
	c.Check(msg1.Name, Equals, irc.PRIVMSG)
	c.Check(msg1, Equals, msg2)
	c.Check(s1, NotNil)
	c.Check(s1, Equals, s2)
	c.Check(msg3, IsNil)
	d.Dispatch(quitmsg, send)
	d.WaitForCompletion()
	c.Check(msg3.Name, Equals, irc.QUIT)
}

func (s *s) TestDispatcher_RawDispatch(c *C) {
	var msg1, msg2 *irc.IrcMessage
	h1 := testHandler{func(m *irc.IrcMessage, send irc.Endpoint) {
		msg1 = m
	}}
	h2 := testHandler{func(m *irc.IrcMessage, send irc.Endpoint) {
		msg2 = m
	}}

	d := CreateDispatcher()
	send := testPoint{&irc.Helper{}}
	d.Register(irc.PRIVMSG, h1)
	d.Register(irc.RAW, h2)

	privmsg := &irc.IrcMessage{Name: irc.PRIVMSG}
	d.Dispatch(privmsg, send)
	d.WaitForCompletion()
	c.Check(msg1, Equals, privmsg)
	c.Check(msg1, Equals, msg2)
}

// ================================
// Testing types
// ================================
type testCallbackMsg func(*irc.Message, irc.Endpoint)

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

var privChanmsg = &irc.IrcMessage{
	Name:   irc.PRIVMSG,
	Args:   []string{"#chan", "msg"},
	Sender: "nick!user@host.com",
}
var privUsermsg = &irc.IrcMessage{
	Name:   irc.PRIVMSG,
	Args:   []string{"user", "msg"},
	Sender: "nick!user@host.com",
}

func (s *s) TestDispatcher_Privmsg(c *C) {
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

	d, err := CreateRichDispatcher(irc.CreateProtoCaps(), nil)
	c.Check(err, IsNil)
	d.Register(irc.PRIVMSG, ph)
	d.Register(irc.PRIVMSG, puh)
	d.Register(irc.PRIVMSG, pch)

	d.Dispatch(privUsermsg, nil)
	d.WaitForCompletion()
	c.Check(p, NotNil)
	c.Check(pu.IrcMessage, Equals, p.IrcMessage)

	p, pu, pc = nil, nil, nil
	d.Dispatch(privChanmsg, nil)
	d.WaitForCompletion()
	c.Check(p, NotNil)
	c.Check(pc.IrcMessage, Equals, p.IrcMessage)
}

func (s *s) TestDispatcher_PrivmsgMultiple(c *C) {
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

	d, err := CreateRichDispatcher(irc.CreateProtoCaps(), nil)
	c.Check(err, IsNil)
	d.Register(irc.PRIVMSG, pall)

	p, pu, pc = nil, nil, nil
	d.Dispatch(privChanmsg, nil)
	d.WaitForCompletion()
	c.Check(p, IsNil)
	c.Check(pu, IsNil)
	c.Check(pc, NotNil)

	p, pu, pc = nil, nil, nil
	d.Dispatch(privUsermsg, nil)
	d.WaitForCompletion()
	c.Check(p, IsNil)
	c.Check(pu, NotNil)
	c.Check(pc, IsNil)

	d = CreateDispatcher()
	d.Dispatch(privChanmsg, nil)
	d.Register(irc.PRIVMSG, pall)

	p, pu, pc = nil, nil, nil
	d.Dispatch(privUsermsg, nil)
	d.WaitForCompletion()
	c.Check(p, NotNil)
	c.Check(pu, IsNil)
	c.Check(pc, IsNil)
}

var noticeChanmsg = &irc.IrcMessage{
	Name:   irc.NOTICE,
	Args:   []string{"#chan", "msg"},
	Sender: "nick!user@host.com",
}
var noticeUsermsg = &irc.IrcMessage{
	Name:   irc.NOTICE,
	Args:   []string{"user", "msg"},
	Sender: "nick!user@host.com",
}

func (s *s) TestDispatcher_Notice(c *C) {
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

	d, err := CreateRichDispatcher(irc.CreateProtoCaps(), nil)
	c.Check(err, IsNil)
	d.Register(irc.NOTICE, nh)
	d.Register(irc.NOTICE, nuh)
	d.Register(irc.NOTICE, nch)

	d.Dispatch(noticeUsermsg, nil)
	d.WaitForCompletion()
	c.Check(n, NotNil)
	c.Check(nu.IrcMessage, Equals, n.IrcMessage)

	n, nu, nc = nil, nil, nil
	d.Dispatch(noticeChanmsg, nil)
	d.WaitForCompletion()
	c.Check(n, NotNil)
	c.Check(nc.IrcMessage, Equals, n.IrcMessage)
}

func (s *s) TestDispatcher_NoticeMultiple(c *C) {
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

	d, err := CreateRichDispatcher(irc.CreateProtoCaps(), nil)
	c.Check(err, IsNil)
	d.Register(irc.NOTICE, nall)

	n, nu, nc = nil, nil, nil
	d.Dispatch(noticeChanmsg, nil)
	d.WaitForCompletion()
	c.Check(n, IsNil)
	c.Check(nu, IsNil)
	c.Check(nc, NotNil)

	n, nu, nc = nil, nil, nil
	d.Dispatch(noticeUsermsg, nil)
	d.WaitForCompletion()
	c.Check(n, IsNil)
	c.Check(nu, NotNil)
	c.Check(nc, IsNil)

	d = CreateDispatcher()
	d.Dispatch(noticeChanmsg, nil)
	d.Register(irc.NOTICE, nall)

	n, nu, nc = nil, nil, nil
	d.Dispatch(noticeUsermsg, nil)
	d.WaitForCompletion()
	c.Check(n, NotNil)
	c.Check(nu, IsNil)
	c.Check(nc, IsNil)
}

func (s *s) TestDispatcher_Sender(c *C) {
	d := CreateDispatcher()
	send := testPoint{&irc.Helper{}}

	msg := &irc.IrcMessage{
		Name:   irc.PRIVMSG,
		Args:   []string{"#chan", "msg"},
		Sender: "nick!user@host.com",
	}

	d.Register(irc.PRIVMSG, func(msg *irc.IrcMessage, point irc.Endpoint) {
		c.Check(point.GetKey(), NotNil)
		c.Check(point.Sendln(""), IsNil)
	})
	d.Dispatch(msg, send)
}

func (s *s) TestDispatcher_FilterPrivmsgChannels(c *C) {
	chanmsg2 := &irc.IrcMessage{
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

	d, err := CreateRichDispatcher(
		irc.CreateProtoCaps(), []string{"#CHAN"})
	c.Check(err, IsNil)
	d.Register(irc.PRIVMSG, ph)
	d.Register(irc.PRIVMSG, pch)

	d.Dispatch(privChanmsg, nil)
	d.WaitForCompletion()
	c.Check(p, NotNil)
	c.Check(pc.IrcMessage, Equals, p.IrcMessage)

	p, pc = nil, nil
	d.Dispatch(chanmsg2, nil)
	d.WaitForCompletion()
	c.Check(p, NotNil)
	c.Check(pc, IsNil)
}

func (s *s) TestDispatcher_FilterNoticeChannels(c *C) {
	chanmsg2 := &irc.IrcMessage{
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

	d, err := CreateRichDispatcher(
		irc.CreateProtoCaps(), []string{"#CHAN"})
	c.Check(err, IsNil)
	d.Register(irc.NOTICE, uh)
	d.Register(irc.NOTICE, uch)

	d.Dispatch(noticeChanmsg, nil)
	d.WaitForCompletion()
	c.Check(u, NotNil)
	c.Check(uc.IrcMessage, Equals, u.IrcMessage)

	u, uc = nil, nil
	d.Dispatch(chanmsg2, nil)
	d.WaitForCompletion()
	c.Check(u, NotNil)
	c.Check(uc, IsNil)
}

func (s *s) TestDispatcher_AddRemoveChannels(c *C) {
	chans := []string{"#chan1", "#chan2", "#chan3"}
	d, err := CreateRichDispatcher(irc.CreateProtoCaps(), chans)
	c.Check(err, IsNil)

	c.Check(len(d.chans), Equals, len(chans))
	for i, v := range chans {
		c.Check(d.chans[i], Equals, v)
	}

	d.RemoveChannels(chans...)
	c.Check(d.chans, IsNil)
	d.RemoveChannels(chans...)
	c.Check(d.chans, IsNil)
	d.RemoveChannels()
	c.Check(d.chans, IsNil)

	d.Channels(chans)
	d.RemoveChannels(chans[1:]...)
	c.Check(len(d.chans), Equals, len(chans)-2)
	for i, v := range chans[:1] {
		c.Check(d.chans[i], Equals, v)
	}
	d.AddChannels(chans[1:]...)
	c.Check(len(d.chans), Equals, len(chans))
	for i, v := range chans {
		c.Check(d.chans[i], Equals, v)
	}
	d.AddChannels(chans[0])
	d.AddChannels()
	c.Check(len(d.chans), Equals, len(chans))
	d.RemoveChannels(chans...)
	d.AddChannels(chans...)
	c.Check(len(d.chans), Equals, len(chans))
}

func (s *s) TestDispatcher_GetChannels(c *C) {
	d, err := CreateRichDispatcher(irc.CreateProtoCaps(), nil)
	c.Check(err, IsNil)

	c.Check(d.GetChannels(), IsNil)
	chans := []string{"#chan1", "#chan2"}
	d.Channels(chans)

	for i, ch := range d.GetChannels() {
		c.Check(d.chans[i], Equals, ch)
	}

	first := d.GetChannels()
	first[0] = "#chan3"
	for i, ch := range d.GetChannels() {
		c.Check(d.chans[i], Equals, ch)
	}
}

func (s *s) TestDispatcher_UpdateChannels(c *C) {
	d, err := CreateRichDispatcher(irc.CreateProtoCaps(), nil)
	c.Check(err, IsNil)
	chans := []string{"#chan1", "#chan2"}
	d.Channels(chans)
	c.Check(len(d.chans), Equals, len(chans))
	for i, v := range chans {
		c.Check(d.chans[i], Equals, v)
	}
	d.Channels([]string{})
	c.Check(len(d.chans), Equals, 0)
	d.Channels(chans)
	c.Check(len(d.chans), Equals, len(chans))
	d.Channels(nil)
	c.Check(len(d.chans), Equals, 0)
}

func (s *s) TestDispatcher_UpdateProtoCaps(c *C) {
	p := irc.CreateProtoCaps()
	p.ParseISupport(&irc.IrcMessage{Args: []string{"nick", "CHANTYPES=#"}})
	d, err := CreateRichDispatcher(p, nil)
	c.Check(err, IsNil)
	var should bool
	should = d.shouldDispatch(true, &irc.IrcMessage{Args: []string{"#chan"}})
	c.Check(should, Equals, true)
	should = d.shouldDispatch(true, &irc.IrcMessage{Args: []string{"&chan"}})
	c.Check(should, Equals, false)

	p = irc.CreateProtoCaps()
	p.ParseISupport(&irc.IrcMessage{Args: []string{"nick", "CHANTYPES=&"}})
	err = d.Protocaps(p)
	c.Check(err, IsNil)
	should = d.shouldDispatch(true, &irc.IrcMessage{Args: []string{"#chan"}})
	c.Check(should, Equals, false)
	should = d.shouldDispatch(true, &irc.IrcMessage{Args: []string{"&chan"}})
	c.Check(should, Equals, true)
}

func (s *s) TestDispatcher_shouldDispatch(c *C) {
	d, err := CreateRichDispatcher(irc.CreateProtoCaps(), nil)
	c.Check(err, IsNil)

	var should bool
	should = d.shouldDispatch(true, &irc.IrcMessage{Args: []string{"#chan"}})
	c.Check(should, Equals, true)
	should = d.shouldDispatch(false, &irc.IrcMessage{Args: []string{"#chan"}})
	c.Check(should, Equals, false)

	should = d.shouldDispatch(true, &irc.IrcMessage{Args: []string{"chan"}})
	c.Check(should, Equals, false)
	should = d.shouldDispatch(false, &irc.IrcMessage{Args: []string{"chan"}})
	c.Check(should, Equals, true)
}

func (s *s) TestDispatcher_filterChannelDispatch(c *C) {
	d, err := CreateRichDispatcher(
		irc.CreateProtoCaps(), []string{"#CHAN"})
	c.Check(err, IsNil)
	c.Check(d.chans, NotNil)

	var should bool
	should = d.checkChannels(&irc.IrcMessage{Args: []string{"#chan"}})
	c.Check(should, Equals, true)
	should = d.checkChannels(&irc.IrcMessage{Args: []string{"#chan2"}})
	c.Check(should, Equals, false)
}
