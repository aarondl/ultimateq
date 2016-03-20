package registrar

import "testing"

func TestProxy_New(t *testing.T) {
	t.Parallel()

	m := &mockReg{}
	p := NewProxy(m)

	if p.holders == nil {
		t.Error("holders not initialized")
	}
}

func TestProxy_Get(t *testing.T) {
	t.Parallel()

	m := &mockReg{}
	p := NewProxy(m)

	if ln := len(p.holders); ln != 0 {
		t.Error("should be empty:", ln)
	}

	i := p.Get("test")
	if _, ok := i.(*holder); !ok {
		t.Errorf("wrong type: %T", i)
	}

	if ln := len(p.holders); ln != 1 {
		t.Error("should have one:", ln)
	}

	again := p.Get("test")
	if i != again {
		t.Error("should re-use existing holders")
	}

	if ln := len(p.holders); ln != 1 {
		t.Error("should have one:", ln)
	}
}

func TestProxy_Unregister(t *testing.T) {
	t.Parallel()

	m := &mockReg{}
	p := NewProxy(m)

	p.Get("hello").Register("n", "c", "e", nil)
	p.Get("hello").Register("n", "c", "e", nil)

	p.Unregister("hello")

	m.verifyMock(t, 2, 2, 0, 0)
}
