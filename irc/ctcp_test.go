package irc

import (
	. "testing"
	"bytes"
)

func TestIsCTCP(t *T) {
	yes, no := []byte("\x01yes\x01"), []byte("no")
	if !IsCTCP(yes) {
		t.Errorf("Expected (% X) to be a CTCP.", yes)
	}
	if IsCTCP(no) {
		t.Errorf("Expected (% X) to NOT be a CTCP.", no)
	}
}

func TestIsCTCPString(t *T) {
	yes, no := "\x01yes\x01", "no"
	if !IsCTCPString(yes) {
		t.Errorf("Expected (%s) to be a CTCP.", yes)
	}
	if IsCTCPString(no) {
		t.Errorf("Expected (%s) to NOT be a CTCP.", no)
	}
}

func TestCTCPUnpack(t *T) {
	in := []byte("\x01\x10\r\x10\n\x10\x10 \x5Ca\x5C\x5C\x01")
	expect1 := []byte("\r\n\x10")
	expect2 := []byte("\x01\x5C")

	out1, out2 := CTCPunpack(in)
	if 0 != bytes.Compare(out1, expect1) {
		t.Errorf("1: Expected: [% X] Got: [% X]", expect1, out1)
	}
	if 0 != bytes.Compare(out2, expect2) {
		t.Errorf("2: Expected: [% X] Got: [% X]", expect2, out2)
	}
}

func TestCTCPPack(t *T) {
	in1 := []byte("\r\n\x10")
	in2 := []byte("\x01\x5C")
	expect := []byte("\x01\x10\r\x10\n\x10\x10 \x5Ca\x5C\x5C\x01")

	out := CTCPpack(in1, in2)
	if 0 != bytes.Compare(out, expect) {
		t.Errorf("Expected: [% X] Got: [% X]", expect, out)
	}
}

func TestCTCPUnpackString(t *T) {
	in := "\x01DCC SEND moozic.txt 1122250358 37294 130\x01"
	expect1 := "DCC"
	expect2 := "SEND moozic.txt 1122250358 37294 130"

	out1, out2 := CTCPunpackString(in)
	if out1 != expect1 {
		t.Errorf("1: Expected: [%s] Got: [%s]", expect1, out1)
	}
	if out2 != expect2 {
		t.Errorf("2: Expected: [%s] Got: [%s]", expect2, out2)
	}
}

func TestCTCPPackString(t *T) {
	in1 := "DCC"
	in2 := "SEND moozic.txt 1122250358 37294 130"
	expect := "\x01DCC SEND moozic.txt 1122250358 37294 130\x01"

	out := CTCPpackString(in1, in2)
	if out != expect {
		t.Errorf("Expected: [%s] Got: [%s]", expect, out)
	}
}

func TestCTCPunpack(t *T) {
	in := []byte("a b c d")
	expect1 := []byte("a")
	expect2 := []byte("b c d")

	out1, out2 := ctcpUnpack(in)
	if 0 != bytes.Compare(out1, expect1) {
		t.Errorf("1: Expected: [% X] Got: [% X]", expect1, out1)
	}
	if 0 != bytes.Compare(out2, expect2) {
		t.Errorf("2: Expected: [% X] Got: [% X]", expect2, out2)
	}

	in = []byte("abcd")
	expect1 = in
	out1, out2 = ctcpUnpack(in)
	if 0 != bytes.Compare(out1, expect1) {
		t.Errorf("1: Expected: [% X] Got: [% X]", expect1, out1)
	}
	if out2 != nil {
		t.Errorf("2: Expected data to be nil, was: [% X]", out2)
	}
}

func TestCTCPpack(t *T) {
	in1 := []byte("a")
	in2 := []byte("b c d")
	expect := []byte("a b c d")

	out := ctcpPack(in1, in2)
	if 0 != bytes.Compare(out, expect) {
		t.Errorf("1: Expected: [% X] Got: [% X]", expect, out)
	}

	in1 = []byte("abcd")
	in2 = []byte{}
	expect = in1
	out = ctcpPack(in1, in2)
	if 0 != bytes.Compare(out, expect) {
		t.Errorf("1: Expected: [% X] Got: [% X]", expect, out)
	}

	in2 = nil
	out = ctcpPack(in1, in2)
	if 0 != bytes.Compare(out, expect) {
		t.Errorf("1: Expected: [% X] Got: [% X]", expect, out)
	}
}

func TestCTCPHighLevelEscape(t *T) {
	in := []byte("\x01\x5C")
	expect := []byte("\x5Ca\x5C\x5C")

	if out := ctcpHighLevelEscape(in); 0 != bytes.Compare(out, expect) {
		t.Errorf("Expected: [% X] Got: [% X]", expect, out)
	}
}

func TestCTCPHighLevelUnescape(t *T) {
	in := []byte("\x5Ca\x5C\x5C")
	expect := []byte("\x01\x5C")

	if out := ctcpHighLevelUnescape(in); 0 != bytes.Compare(out, expect) {
		t.Errorf("Expected: [% X] Got: [% X]", expect, out)
	}
}

func TestCTCPLowLevelEscape(t *T) {
	in := []byte("\n\r\x00\x10")
	expect := []byte("\x10\n\x10\r\x10\x00\x10\x10")

	if out := ctcpLowLevelEscape(in); 0 != bytes.Compare(out, expect) {
		t.Errorf("Expected: [% X] Got: [% X]", expect, out)
	}
}

func TestCTCPLowLevelUnescape(t *T) {
	in := []byte("\x10\n\x10\r\x10\x00\x10\x10")
	expect := []byte("\n\r\x00\x10")

	if out := ctcpLowLevelUnescape(in); 0 != bytes.Compare(out, expect) {
		t.Errorf("Expected: [% X] Got: [% X]", expect, out)
	}
}
