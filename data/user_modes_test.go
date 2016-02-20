package data

import (
	"encoding/json"
	"reflect"
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

func TestUserModes_JSONify(t *testing.T) {
	t.Parallel()

	a := NewUserModes(testKinds)
	a.SetMode('o')
	var b UserModes

	str, err := json.Marshal(a)
	if err != nil {
		t.Error(err)
	}

	jsonStr := `{"modes":1,"mode_kinds":` +
		`{"user_prefixes":[["o","@"],["v","+"]],` +
		`"channel_modes":{"a":1,"b":4,"c":2,"d":3,"x":1,"y":1,"z":1}}}`

	if string(str) != jsonStr {
		t.Errorf("Wrong JSON: %s", str)
	}

	if err = json.Unmarshal(str, &b); err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(a, b) {
		t.Error("A and B differ:", a, b)
	}
}
