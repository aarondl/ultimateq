package irc

import (
	"bytes"
	"fmt"
	. "gopkg.in/check.v1"
	"strings"
	"testing"
)

func Test(t *testing.T) { TestingT(t) } //Hook into testing package

type s struct{}

var _ = Suite(&s{})

func (s *s) TestIrcMessage_Test(c *C) {
	args := []string{"#chan1", "#chan2"}
	msg := NewMessage("", "nick!user@host", strings.Join(args, ","))
	for i, v := range msg.SplitArgs(0) {
		c.Check(args[i], Equals, v)
	}
	c.Check(msg.Nick(), Equals, "nick")
	c.Check(msg.Username(), Equals, "user")
	c.Check(msg.Hostname(), Equals, "host")
	n, u, h := msg.Split()
	c.Check(n, Equals, "nick")
	c.Check(u, Equals, "user")
	c.Check(h, Equals, "host")
	c.Check(msg.Time.Unix(), Not(Equals), 0)
}

func (s *s) TestMsgTypes_Privmsg(c *C) {
	args := []string{"#chan", "msg arg"}
	pmsg := &Message{
		Name:   PRIVMSG,
		Args:   args,
		Sender: "user@host.com",
	}

	c.Check(pmsg.Target(), Equals, args[0])
	c.Check(pmsg.Message(), Equals, args[1])
}

func (s *s) TestMsgTypes_Notice(c *C) {
	args := []string{"#chan", "msg arg"}
	notice := &Message{
		Name:   NOTICE,
		Args:   args,
		Sender: "user@host.com",
	}

	c.Check(notice.Target(), Equals, args[0])
	c.Check(notice.Message(), Equals, args[1])
}

type fakeHelper struct {
	*Helper
}

func (f *fakeHelper) GetKey() string {
	return ""
}

func (s *s) TestHelper_Send(c *C) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	format := "PRIVMSG %v :%v"
	target := "#chan"
	msg := "msg"
	h.Send(format, target, msg)
	c.Check(string(buf.Bytes()), Equals, fmt.Sprint(format, target, msg))
}

func (s *s) TestHelper_Sendln(c *C) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	header := "PRIVMSG"
	target := "#chan"
	msg := "msg"
	h.Sendln(header, target, msg)
	expect := fmt.Sprintln(header, target, msg)
	c.Check(string(buf.Bytes()), Equals, expect[:len(expect)-1])
}

func (s *s) TestHelper_Sendf(c *C) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	format := "PRIVMSG %v :%v"
	target := "#chan"
	msg := "msg"
	h.Sendf(format, target, msg)
	c.Check(string(buf.Bytes()), Equals, fmt.Sprintf(format, target, msg))
}

func (s *s) TestHelper_Privmsg(c *C) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	ch := "#chan"
	s1, s2 := "string1", "string2"
	h.Privmsg(ch, s1, s2)
	c.Check(string(buf.Bytes()), Equals, fmt.Sprintf("%v %v :%v",
		PRIVMSG, ch, fmt.Sprint(s1, s2)))
}

func (s *s) TestHelper_Privmsgln(c *C) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	ch := "#chan"
	s1, s2 := "string1", "string2"
	expect := fmt.Sprintln(s1, s2)
	h.Privmsgln(ch, s1, s2)
	c.Check(string(buf.Bytes()), Equals, fmt.Sprintf("%v %v :%v",
		PRIVMSG, ch, expect[:len(expect)-1]))
}

func (s *s) TestHelper_Privmsgf(c *C) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	ch := "#chan"
	format := "%v - %v"
	s1, s2 := "string1", "string2"
	h.Privmsgf(ch, format, s1, s2)
	c.Check(string(buf.Bytes()), Equals, fmt.Sprintf("%v %v :%v",
		PRIVMSG, ch, fmt.Sprintf(format, s1, s2)))
}

func (s *s) TestHelper_Notice(c *C) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	ch := "#chan"
	s1, s2 := "string1", "string2"
	h.Notice(ch, s1, s2)
	c.Check(string(buf.Bytes()), Equals, fmt.Sprintf("%v %v :%v",
		NOTICE, ch, fmt.Sprint(s1, s2)))
}

func (s *s) TestHelper_Noticeln(c *C) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	ch := "#chan"
	s1, s2 := "string1", "string2"
	expect := fmt.Sprintln(s1, s2)
	h.Noticeln(ch, s1, s2)
	c.Check(string(buf.Bytes()), Equals, fmt.Sprintf("%v %v :%v",
		NOTICE, ch, expect[:len(expect)-1]))
}

func (s *s) TestHelper_Noticef(c *C) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	ch := "#chan"
	format := "%v - %v"
	s1, s2 := "string1", "string2"
	h.Noticef(ch, format, s1, s2)
	c.Check(string(buf.Bytes()), Equals, fmt.Sprintf("%v %v :%v",
		NOTICE, ch, fmt.Sprintf(format, s1, s2)))
}

func (s *s) TestHelper_Join(c *C) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	ch := "#chan"
	h.Join()
	c.Check(buf.Len(), Equals, 0)
	h.Join(ch)
	c.Check(string(buf.Bytes()), Equals, fmt.Sprintf("%v :%v", JOIN, ch))

	buf = bytes.Buffer{}
	h.Writer = &buf
	h.Join(ch, ch)
	c.Check(string(buf.Bytes()), Equals, fmt.Sprintf("%v :%v,%v", JOIN, ch, ch))
}

func (s *s) TestHelper_Part(c *C) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	ch := "#chan"
	h.Part()
	c.Check(buf.Len(), Equals, 0)
	h.Part(ch)
	c.Check(string(buf.Bytes()), Equals, fmt.Sprintf("%v :%v", PART, ch))

	buf = bytes.Buffer{}
	h.Writer = &buf
	h.Part(ch, ch)
	c.Check(string(buf.Bytes()), Equals, fmt.Sprintf("%v :%v,%v", PART, ch, ch))
}

func (s *s) TestHelper_Quit(c *C) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	msg := "quitting"
	h.Quit(msg)
	c.Check(string(buf.Bytes()), Equals, fmt.Sprintf("%v :%v", QUIT, msg))
}

func (s *s) TestHelper_splitSend(c *C) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	header := "PRIVMSG #chan :"
	s0 := "message"
	h.splitSend([]byte(header), []byte(s0))
	c.Check(buf.Len(), Equals, len(header)+len(s0))

	buf.Reset()
	header = "PRIVMSG #chan :"
	s1 := strings.Repeat("a", IRC_MAX_LENGTH)
	s2 := strings.Repeat("b", IRC_MAX_LENGTH)
	s3 := strings.Repeat("c", 300)
	err := h.splitSend([]byte(header), []byte(s1+s2+s3))
	c.Check(err, IsNil)
	c.Check(buf.Len(), Equals, len(header)*3+len(s1)+len(s2)+len(s3))

	buf.Reset()
	header = "PRIVMSG #chan :"
	s4 := strings.Repeat("a", IRC_MAX_LENGTH-len(header))
	s5 := strings.Repeat("b", IRC_MAX_LENGTH-len(header))
	err = h.splitSend([]byte(header), []byte(s4+s5))
	c.Check(err, IsNil)
	c.Check(buf.Len(), Equals, len(header)*2+len(s4)+len(s5))

	buf.Reset()
	header = "PRIVMSG #chan :"
	s6 := strings.Repeat("a", IRC_MAX_LENGTH-len(header)-SPLIT_BACKWARD+1) + " "
	s7 := strings.Repeat("b", IRC_MAX_LENGTH-len(header)-1)
	err = h.splitSend([]byte(header), []byte(s6+s7))
	c.Check(err, IsNil)
	c.Check(buf.Len(), Equals, len(header)*2+len(s6)+len(s7)-1) //-1 space char
	c.Check(buf.Bytes()[len(header)+len(s6)-1], Equals, header[0])
}
