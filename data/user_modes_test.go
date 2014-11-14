package data

import (
	"testing"
)

func TestUserModes(t *testing.T) {
	t.Parallel()

	m := NewUserModes(testUserKinds)
	if exp, got := m.HasMode('o'), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.HasMode('v'), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	m.SetMode('o')
	if exp, got := m.HasMode('o'), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.HasMode('v'), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	m.SetMode('v')
	if exp, got := m.HasMode('o'), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.HasMode('v'), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	m.UnsetMode('o')
	if exp, got := m.HasMode('o'), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.HasMode('v'), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	m.UnsetMode('v')
	if exp, got := m.HasMode('o'), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.HasMode('v'), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}

func TestUserModes_String(t *testing.T) {
	t.Parallel()

	m := NewUserModes(testUserKinds)
	if exp, got := m.String(), ""; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.StringSymbols(), ""; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	m.SetMode('o')
	if exp, got := m.String(), "o"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.StringSymbols(), "@"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	m.SetMode('v')
	if exp, got := m.String(), "ov"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.StringSymbols(), "@+"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	m.UnsetMode('o')
	if exp, got := m.String(), "v"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.StringSymbols(), "+"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}
