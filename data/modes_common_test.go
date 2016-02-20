package data

import (
	"encoding/json"
	"reflect"
	"testing"
)

var testUserKindStr = `(ov)@+`
var testChannelKindStr = `b,c,d,axyz`
var testKinds, _ = newModeKinds(testUserKindStr, testChannelKindStr)

func TestModeKinds_Create(t *testing.T) {
	t.Parallel()

	m, err := newModeKinds(testUserKindStr, "a,b,c,d")
	if err != nil {
		t.Errorf("Unexpected error:", err)
	}
	if got, exp := m.channelModes['a'], ARGS_ADDRESS; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := m.channelModes['b'], ARGS_ALWAYS; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := m.channelModes['c'], ARGS_ONSET; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := m.channelModes['d'], ARGS_NONE; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	m, err = newModeKinds("(o)@", "a, b, c, d")
	if err != nil {
		t.Errorf("Unexpected error:", err)
	}
	if got, exp := m.channelModes['a'], ARGS_ADDRESS; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := m.channelModes['b'], ARGS_ALWAYS; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := m.channelModes['c'], ARGS_ONSET; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := m.channelModes['d'], ARGS_NONE; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	err = m.update("(o)@", "d, c, b, a")
	if err != nil {
		t.Errorf("Unexpected error:", err)
	}
	if got, exp := m.channelModes['d'], ARGS_ADDRESS; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := m.channelModes['c'], ARGS_ALWAYS; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := m.channelModes['b'], ARGS_ONSET; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := m.channelModes['a'], ARGS_NONE; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestModeKindsUpdate(t *testing.T) {
	t.Parallel()

	m, err := newModeKinds(testUserKindStr, "a,b,c,d")
	if got, exp := m.channelModes['a'], ARGS_ADDRESS; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := m.channelModes['b'], ARGS_ALWAYS; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := m.channelModes['c'], ARGS_ONSET; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := m.channelModes['d'], ARGS_NONE; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	err = m.update(testUserKindStr, "d,c,b,a")
	if err != nil {
		t.Errorf("Unexpected Errorf:", err)
	}
	if got, exp := m.channelModes['d'], ARGS_ADDRESS; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := m.channelModes['c'], ARGS_ALWAYS; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := m.channelModes['b'], ARGS_ONSET; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := m.channelModes['a'], ARGS_NONE; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestUserModeKinds_Create(t *testing.T) {
	t.Parallel()

	u, err := newModeKinds("", testChannelKindStr)
	if got := u; got != nil {
		t.Errorf("Expected: %v to be nil.", got)
	}
	if err == nil {
		t.Errorf("Unexpected nil.")
	}
	u, err = newModeKinds("a", testChannelKindStr)
	if got := u; got != nil {
		t.Errorf("Expected: %v to be nil.", got)
	}
	if err == nil {
		t.Errorf("Unexpected nil.")
	}
	u, err = newModeKinds("(a", testChannelKindStr)
	if got := u; got != nil {
		t.Errorf("Expected: %v to be nil.", got)
	}
	if err == nil {
		t.Errorf("Unexpected nil.")
	}

	u, err = newModeKinds("(abcdefghi)!@#$%^&*_", testChannelKindStr)
	if got := u; got != nil {
		t.Errorf("Expected: %v to be nil.", got)
	}
	if err == nil {
		t.Errorf("Unexpected nil.")
	}

	u, err = newModeKinds("(ov)@+", testChannelKindStr)
	if u == nil {
		t.Errorf("Unexpected nil.")
	}
	if err != nil {
		t.Errorf("Unexpected Errorf:", err)
	}
	if got, exp := u.userPrefixes[0], [2]rune{'o', '@'}; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := u.userPrefixes[1], [2]rune{'v', '+'}; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestUserModeKinds_Symbol(t *testing.T) {
	t.Parallel()

	u, err := newModeKinds("(ov)@+", testChannelKindStr)
	if err != nil {
		t.Errorf("Unexpected Errorf:", err)
	}
	if got, exp := u.Symbol('o'), '@'; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := u.Symbol(' '), rune(0); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestUserModeKinds_Mode(t *testing.T) {
	t.Parallel()

	u, err := newModeKinds("(ov)@+", testChannelKindStr)
	if err != nil {
		t.Errorf("Unexpected Errorf:", err)
	}
	if got, exp := u.Mode('@'), 'o'; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := u.Mode(' '), rune(0); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestUserModeKinds_Update(t *testing.T) {
	t.Parallel()

	u, err := newModeKinds("(ov)@+", testChannelKindStr)
	if err != nil {
		t.Errorf("Unexpected Errorf:", err)
	}
	if got, exp := u.modeBit('o'), byte(0); exp == got {
		t.Errorf("Did not want: %v, got: %v", exp, got)
	}
	err = u.update("(v)+", "")
	if err != nil {
		t.Errorf("Unexpected Errorf:", err)
	}
	if got, exp := u.modeBit('o'), byte(0); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestUserModeKinds_JSONify(t *testing.T) {
	t.Parallel()

	a := testKinds
	var b modeKinds

	str, err := json.Marshal(a)
	if err != nil {
		t.Error(err)
	}

	jsonStr := `{"user_prefixes":[["o","@"],["v","+"]],` +
		`"channel_modes":{"a":1,"b":4,"c":2,"d":3,"x":1,"y":1,"z":1}}`

	if string(str) != jsonStr {
		t.Errorf("Wrong JSON: %s", str)
	}

	if err = json.Unmarshal(str, &b); err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(a.userPrefixes, b.userPrefixes) {
		t.Error("A and B differ:", a.userPrefixes, b.userPrefixes)
	}
	if !reflect.DeepEqual(a.channelModes, b.channelModes) {
		t.Error("A and B differ:", a.channelModes, b.channelModes)
	}
}
