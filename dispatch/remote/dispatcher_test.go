package remote

import (
	"bytes"
	"testing"
)

func TestNewDispatcher(t *testing.T) {
	t.Parallel()

	d := NewDispatcher(nil, nil)

	if d == nil {
		t.Error("Should not be nil")
	}
}

func TestDispatcher_New(t *testing.T) {
	t.Parallel()

	d := NewDispatcher(nil, nil)

	var b *bytes.Buffer
	e := d.New("a", NopCloser(b), func(string) {})

	if e == nil {
		t.Error("Should not be nil")
	}

	if e != d.exts["a"] {
		t.Error("Want a to be stored")
	}
}

func TestDispatcher_NewReplace(t *testing.T) {
	t.Parallel()

	d := NewDispatcher(nil, nil)

	var b *bytes.Buffer
	e1 := d.New("a", NopCloser(b), func(string) {})
	e2 := d.New("a", NopCloser(b), func(string) {})

	if e1 == nil || e2 == nil {
		t.Error("Handlers should not be nil")
	}

	if e1 == d.exts["a"] {
		t.Error("Want e1 not to be stored")
	}
	if e2 != d.exts["a"] {
		t.Error("Want e2 to be stored")
	}
}

func TestDispatcher_Get(t *testing.T) {
	t.Parallel()

	d := NewDispatcher(nil, nil)

	var b *bytes.Buffer
	e := d.New("a", NopCloser(b), func(string) {})

	if e != d.Get("a") || e != d.exts["a"] {
		t.Error("Should be stored and be retrieved")
	}
}

func TestDispatcher_Remove(t *testing.T) {
	t.Parallel()

	d := NewDispatcher(nil, nil)

	var b *bytes.Buffer
	e := d.New("a", NopCloser(b), func(string) {})

	if e != d.exts["a"] {
		t.Error("e should be stored")
	}

	d.Remove("a")

	if nil != d.exts["a"] {
		t.Error("e should be deleted")
	}
}
