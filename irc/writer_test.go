package irc

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestHelper_ImplementsWriter(t *testing.T) {
	var _ Writer = &Helper{nil}
}

func TestHelper_Send(t *testing.T) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	format := "PRIVMSG %v :%v"
	target := "#chan"
	msg := "msg"
	h.Send(format, target, msg)

	expect := fmt.Sprint(format, target, msg)
	if s := buf.String(); s != expect {
		t.Errorf("Expected: %s, got: %s", expect, s)
	}
}

func TestHelper_Sendln(t *testing.T) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	header := "PRIVMSG"
	target := "#chan"
	msg := "msg"
	h.Sendln(header, target, msg)

	expect := fmt.Sprintln(header, target, msg)
	expect = expect[:len(expect)-1]
	if s := buf.String(); s != expect {
		t.Errorf("Expected: %s, got: %s", expect, s)
	}
}

func TestHelper_Sendf(t *testing.T) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	format := "PRIVMSG %v :%v"
	target := "#chan"
	msg := "msg"
	h.Sendf(format, target, msg)

	expect := fmt.Sprintf(format, target, msg)
	if s := buf.String(); s != expect {
		t.Errorf("Expected: %s, got: %s", expect, s)
	}
}

func TestHelper_Privmsg(t *testing.T) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	ch := "#chan"
	s1, s2 := "string1", "string2"
	h.Privmsg(ch, s1, s2)

	expect := fmt.Sprintf("%v %v :%v", PRIVMSG, ch, fmt.Sprint(s1, s2))
	if s := buf.String(); s != expect {
		t.Errorf("Expected: %s, got: %s", expect, s)
	}
}

func TestHelper_Privmsgln(t *testing.T) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	ch := "#chan"
	s1, s2 := "string1", "string2"
	expect := fmt.Sprintln(s1, s2)
	h.Privmsgln(ch, s1, s2)

	expect = fmt.Sprintf("%v %v :%v", PRIVMSG, ch, expect[:len(expect)-1])
	if s := buf.String(); s != expect {
		t.Errorf("Expected: %s, got: %s", expect, s)
	}
}

func TestHelper_Privmsgf(t *testing.T) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	ch := "#chan"
	format := "%v - %v"
	s1, s2 := "string1", "string2"
	h.Privmsgf(ch, format, s1, s2)
	expect := fmt.Sprintf("%v %v :%v", PRIVMSG, ch, fmt.Sprintf(format, s1, s2))
	if s := buf.String(); s != expect {
		t.Errorf("Expected: %s, got: %s", expect, s)
	}
}

func TestHelper_Notice(t *testing.T) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	ch := "#chan"
	s1, s2 := "string1", "string2"
	h.Notice(ch, s1, s2)

	expect := fmt.Sprintf("%v %v :%v", NOTICE, ch, fmt.Sprint(s1, s2))
	if s := buf.String(); s != expect {
		t.Errorf("Expected: %s, got: %s", expect, s)
	}
}

func TestHelper_Noticeln(t *testing.T) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	ch := "#chan"
	s1, s2 := "string1", "string2"
	expect := fmt.Sprintln(s1, s2)
	h.Noticeln(ch, s1, s2)

	expect = fmt.Sprintf("%v %v :%v", NOTICE, ch, expect[:len(expect)-1])
	if s := buf.String(); s != expect {
		t.Errorf("Expected: %s, got: %s", expect, s)
	}
}

func TestHelper_Noticef(t *testing.T) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	ch := "#chan"
	format := "%v - %v"
	s1, s2 := "string1", "string2"
	h.Noticef(ch, format, s1, s2)

	expect := fmt.Sprintf("%v %v :%v", NOTICE, ch, fmt.Sprintf(format, s1, s2))
	if s := buf.String(); s != expect {
		t.Errorf("Expected: %s, got: %s", expect, s)
	}
}

func TestHelper_CTCP(t *testing.T) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	ch := "#chan"
	s1, s2 := "string1", "string2"
	tag := "tag"
	h.CTCP(ch, tag, s1, s2)

	expect := fmt.Sprintf("%v %v :\x01%v %v\x01", PRIVMSG, ch, tag,
		fmt.Sprint(s1, s2))
	if s := buf.String(); s != expect {
		t.Errorf("Expected: %s, got: %s", expect, s)
	}
}

func TestHelper_CTCPln(t *testing.T) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	ch := "#chan"
	s1, s2 := "string1", "string2"
	tag := "tag"
	expect := fmt.Sprintln(s1, s2)
	h.CTCPln(ch, tag, s1, s2)

	expect = fmt.Sprintf("%v %v :\x01%v %v\x01", PRIVMSG, ch, tag,
		expect[:len(expect)-1])
	if s := buf.String(); s != expect {
		t.Errorf("Expected: %s, got: %s", expect, s)
	}
}

func TestHelper_CTCPf(t *testing.T) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	ch := "#chan"
	format := "%v - %v"
	s1, s2 := "string1", "string2"
	tag := "tag"
	h.CTCPf(ch, tag, format, s1, s2)

	expect := fmt.Sprintf("%v %v :\x01%v %v\x01", PRIVMSG, ch, tag,
		fmt.Sprintf(format, s1, s2))
	if s := buf.String(); s != expect {
		t.Errorf("Expected: %s, got: %s", expect, s)
	}
}

func TestHelper_CTCPReply(t *testing.T) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	ch := "#chan"
	s1, s2 := "string1", "string2"
	tag := "tag"
	h.CTCPReply(ch, tag, s1, s2)

	expect := fmt.Sprintf("%v %v :\x01%v %v\x01", NOTICE, ch, tag,
		fmt.Sprint(s1, s2))
	if s := buf.String(); s != expect {
		t.Errorf("Expected: %s, got: %s", expect, s)
	}
}

func TestHelper_CTCPReplyln(t *testing.T) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	ch := "#chan"
	s1, s2 := "string1", "string2"
	tag := "tag"
	expect := fmt.Sprintln(s1, s2)
	h.CTCPReplyln(ch, tag, s1, s2)

	expect = fmt.Sprintf("%v %v :\x01%v %v\x01", NOTICE, ch, tag,
		expect[:len(expect)-1])
	if s := buf.String(); s != expect {
		t.Errorf("Expected: %s, got: %s", expect, s)
	}
}

func TestHelper_CTCPReplyf(t *testing.T) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	ch := "#chan"
	format := "%v - %v"
	s1, s2 := "string1", "string2"
	tag := "tag"
	h.CTCPReplyf(ch, tag, format, s1, s2)

	expect := fmt.Sprintf("%v %v :\x01%v %v\x01", NOTICE, ch, tag,
		fmt.Sprintf(format, s1, s2))
	if s := buf.String(); s != expect {
		t.Errorf("Expected: %s, got: %s", expect, s)
	}
}

func TestHelper_Join(t *testing.T) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	ch := "#chan"
	h.Join()
	if buf.Len() != 0 {
		t.Error("Expected nothing to be output when no channels are input.")
	}
	h.Join(ch)

	expect := fmt.Sprintf("%v :%v", JOIN, ch)
	if s := buf.String(); s != expect {
		t.Errorf("Expected: %s, got: %s", expect, s)
	}

	buf = bytes.Buffer{}
	h.Writer = &buf
	h.Join(ch, ch)

	expect = fmt.Sprintf("%v :%v,%v", JOIN, ch, ch)
	if s := buf.String(); s != expect {
		t.Errorf("Expected: %s, got: %s", expect, s)
	}
}

func TestHelper_Part(t *testing.T) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	ch := "#chan"
	h.Part()
	if buf.Len() != 0 {
		t.Error("Expected nothing to be output when no channels are input.")
	}
	h.Part(ch)

	expect := fmt.Sprintf("%v :%v", PART, ch)
	if s := buf.String(); s != expect {
		t.Errorf("Expected: %s, got: %s", expect, s)
	}

	buf = bytes.Buffer{}
	h.Writer = &buf
	h.Part(ch, ch)

	expect = fmt.Sprintf("%v :%v,%v", PART, ch, ch)
	if s := buf.String(); s != expect {
		t.Errorf("Expected: %s, got: %s", expect, s)
	}
}

func TestHelper_Quit(t *testing.T) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	msg := "quitting"
	h.Quit(msg)

	expect := fmt.Sprintf("%v :%v", QUIT, msg)
	if s := buf.String(); s != expect {
		t.Errorf("Expected: %s, got: %s", expect, s)
	}
}

func TestHelper_splitSend(t *testing.T) {
	buf := bytes.Buffer{}
	h := &Helper{&buf}
	header := "PRIVMSG #chan :"
	s0 := "message"
	h.splitSend([]byte(header), []byte(s0))
	if l, e := buf.Len(), len(header)+len(s0); l != e {
		t.Errorf("The expected length is: %v but was %v", e, l)
	}

	buf.Reset()
	header = "PRIVMSG #chan :"
	s1 := strings.Repeat("a", IRC_MAX_LENGTH)
	s2 := strings.Repeat("b", IRC_MAX_LENGTH)
	s3 := strings.Repeat("c", 300)
	err := h.splitSend([]byte(header), []byte(s1+s2+s3))
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	if l, e := buf.Len(), len(header)*3+len(s1)+len(s2)+len(s3); l != e {
		t.Errorf("The expected length is: %v but was %v", e, l)
	}

	buf.Reset()
	header = "PRIVMSG #chan :"
	s4 := strings.Repeat("a", IRC_MAX_LENGTH-len(header))
	s5 := strings.Repeat("b", IRC_MAX_LENGTH-len(header))
	err = h.splitSend([]byte(header), []byte(s4+s5))
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	if l, e := buf.Len(), len(header)*2+len(s4)+len(s5); l != e {
		t.Errorf("The expected length is: %v but was %v", e, l)
	}

	buf.Reset()
	header = "PRIVMSG #chan :"
	s6 := strings.Repeat("a", IRC_MAX_LENGTH-len(header)-SPLIT_BACKWARD+1) + " "
	s7 := strings.Repeat("b", IRC_MAX_LENGTH-len(header)-1)
	err = h.splitSend([]byte(header), []byte(s6+s7))
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	if l, e := buf.Len(), len(header)*2+len(s6)+len(s7)-1; l != e {
		t.Errorf("The expected length is: %v but was %v", e, l)
	}
	if got := buf.String()[len(header)+len(s6)-1]; got != header[0] {
		t.Error("Expected header to reoccur at a position, got:", got)
	}
}
