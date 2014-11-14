package data

import (
	"testing"
)

var testUserKinds, _ = NewUserModeKinds("(ov)@+")
var testChannelKinds = NewChannelModeKinds("b", "c", "d", "axyz")

func TestChannelModeKinds_Create(t *testing.T) {
	t.Parallel()

	m := NewChannelModeKinds("a", "b", "c", "d")
	if exp, got := m.kinds['a'], ARGS_ADDRESS; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.kinds['b'], ARGS_ALWAYS; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.kinds['c'], ARGS_ONSET; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.kinds['d'], ARGS_NONE; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	m = NewChannelModeKinds("a", "b", "c", "d")
	if exp, got := m.kinds['a'], ARGS_ADDRESS; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.kinds['b'], ARGS_ALWAYS; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.kinds['c'], ARGS_ONSET; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.kinds['d'], ARGS_NONE; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	m.Update("d", "c", "b", "a")
	if exp, got := m.kinds['d'], ARGS_ADDRESS; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.kinds['c'], ARGS_ALWAYS; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.kinds['b'], ARGS_ONSET; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.kinds['a'], ARGS_NONE; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}

func TestChannelModeKinds_NewCSV(t *testing.T) {
	t.Parallel()

	m, err := NewChannelModeKindsCSV("")
	if err == nil {
		t.Error("Unexpected nil.")
	}

	m, err = NewChannelModeKindsCSV(",,,")
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	m, err = NewChannelModeKindsCSV(",")
	if err == nil {
		t.Error("Unexpected nil.")
	}

	m, err = NewChannelModeKindsCSV("a,b,c,d")
	if exp, got := m.kinds['a'], ARGS_ADDRESS; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.kinds['b'], ARGS_ALWAYS; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.kinds['c'], ARGS_ONSET; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.kinds['d'], ARGS_NONE; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}

func TestChannelModeKindsUpdate(t *testing.T) {
	t.Parallel()

	m := NewChannelModeKinds("a", "b", "c", "d")
	if exp, got := m.kinds['a'], ARGS_ADDRESS; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.kinds['b'], ARGS_ALWAYS; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.kinds['c'], ARGS_ONSET; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.kinds['d'], ARGS_NONE; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	err := m.UpdateCSV("d,c,b,a")
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	if exp, got := m.kinds['d'], ARGS_ADDRESS; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.kinds['c'], ARGS_ALWAYS; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.kinds['b'], ARGS_ONSET; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.kinds['a'], ARGS_NONE; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	m.Update("a", "b", "c", "d")
	if exp, got := m.kinds['a'], ARGS_ADDRESS; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.kinds['b'], ARGS_ALWAYS; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.kinds['c'], ARGS_ONSET; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.kinds['d'], ARGS_NONE; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	err = m.UpdateCSV("")
	if err == nil {
		t.Error("Unexpected nil.")
	}
}

func TestUserModeKinds_Create(t *testing.T) {
	t.Parallel()

	u, err := NewUserModeKinds("")
	if got := u; got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	if err == nil {
		t.Error("Unexpected nil.")
	}
	u, err = NewUserModeKinds("a")
	if got := u; got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	if err == nil {
		t.Error("Unexpected nil.")
	}
	u, err = NewUserModeKinds("(a")
	if got := u; got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	if err == nil {
		t.Error("Unexpected nil.")
	}

	u, err = NewUserModeKinds("(abcdefghi)!@#$%^&*_")
	if got := u; got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	if err == nil {
		t.Error("Unexpected nil.")
	}

	u, err = NewUserModeKinds("(ov)@+")
	if u == nil {
		t.Error("Unexpected nil.")
	}
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	if exp, got := u.modeInfo[0], [2]rune{'o', '@'}; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := u.modeInfo[1], [2]rune{'v', '+'}; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}

func TestUserModeKinds_GetSymbol(t *testing.T) {
	t.Parallel()

	u, err := NewUserModeKinds("(ov)@+")
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	if exp, got := u.GetSymbol('o'), '@'; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := u.GetSymbol(' '), rune(0); exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}

func TestUserModeKinds_GetMode(t *testing.T) {
	t.Parallel()

	u, err := NewUserModeKinds("(ov)@+")
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	if exp, got := u.GetMode('@'), 'o'; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := u.GetMode(' '), rune(0); exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}

func TestUserModeKinds_Update(t *testing.T) {
	t.Parallel()

	u, err := NewUserModeKinds("(ov)@+")
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	if exp, got := u.GetModeBit('o'), byte(0); exp == got {
		t.Error("Did not want: %v, got: %v", exp, got)
	}
	err = u.UpdateModes("(v)+")
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	if exp, got := u.GetModeBit('o'), byte(0); exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	u, err = NewUserModeKinds("(ov)@+")
	err = u.UpdateModes("")
	if err == nil {
		t.Error("Unexpected nil.")
	}
}
