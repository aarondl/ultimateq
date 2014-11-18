package data

import (
	"testing"
)

func TestUserModes(t *testing.T) {
	t.Parallel()

	m := NewUserModes(testKinds)
	if got, exp := m.HasMode('o'), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := m.HasMode('v'), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	m.SetMode('o')
	if got, exp := m.HasMode('o'), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := m.HasMode('v'), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	m.SetMode('v')
	if got, exp := m.HasMode('o'), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := m.HasMode('v'), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	m.UnsetMode('o')
	if got, exp := m.HasMode('o'), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := m.HasMode('v'), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	m.UnsetMode('v')
	if got, exp := m.HasMode('o'), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := m.HasMode('v'), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}

func TestUserModes_String(t *testing.T) {
	t.Parallel()

	m := NewUserModes(testKinds)
	if got, exp := m.String(), ""; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := m.StringSymbols(), ""; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	m.SetMode('o')
	if got, exp := m.String(), "o"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := m.StringSymbols(), "@"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	m.SetMode('v')
	if got, exp := m.String(), "ov"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := m.StringSymbols(), "@+"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	m.UnsetMode('o')
	if got, exp := m.String(), "v"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := m.StringSymbols(), "+"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}
