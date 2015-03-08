package data

import (
	"testing"
)

func TestAccess(t *testing.T) {
	t.Parallel()
	a := NewAccess(0)
	if a == nil {
		t.Fatal("Failed to Create")
	}
	if a.Level != 0 || a.Flags != 0 {
		t.Error("Bad init")
	}

	a = NewAccess(100, "aBC", "d")
	if a.Level != 100 {
		t.Error("Level was not set")
	}
	for _, v := range "aBCd" {
		if !a.HasFlag(v) {
			t.Errorf("Flag %c was not found.", v)
		}
	}
}

func TestAccess_HasLevel(t *testing.T) {
	t.Parallel()
	a := NewAccess(50)

	var table = map[uint8]bool{
		50: true,
		49: true,
		51: false,
	}

	for level, result := range table {
		if res := a.HasLevel(level); res != result {
			t.Errorf("HasLevel %v resulted in: %v", level, res)
		}
	}
}

func TestAccess_HasFlag(t *testing.T) {
	t.Parallel()
	a := NewAccess(0, "aBC", "d")
	for _, v := range "aBCd" {
		if !a.HasFlag(v) {
			t.Errorf("Flag %c was not found.", v)
		}
	}
}

func TestAccess_HasFlags(t *testing.T) {
	t.Parallel()
	a := NewAccess(0, "aBC", "d")
	if !a.HasFlags("aBCd") {
		t.Error("Flags were not all found.")
	}
	if !a.HasFlags("aZ") || !a.HasFlags("zB") {
		t.Error("Flags should or together for access.")
	}
}

func TestAccess_SetFlag(t *testing.T) {
	t.Parallel()
	a := NewAccess(0)
	if a.HasFlag('a') {
		t.Error("Really bad init.")
	}
	a.SetFlag('a')
	if !a.HasFlag('a') {
		t.Error("Set flag failed")
	}
	a.SetFlag('A')
	if !a.HasFlag('A') {
		t.Error("Set flag failed")
	}
	a.SetFlag('!')
	if !a.HasFlag('a') || !a.HasFlag('A') || a.HasFlag('!') {
		t.Error("Set flag failed")
	}
}

func TestAccess_MultiEffects(t *testing.T) {
	t.Parallel()
	a := NewAccess(0)
	a.SetFlags("ab", "A")
	if !a.HasFlags("ab", "A") {
		t.Error("Set flags failed")
	}
	a.ClearFlags("ab", "A")
	if a.HasFlags("a") || a.HasFlags("b") || a.HasFlags("A") {
		t.Error("Clear flags failed")
	}
}

func TestAccess_ClearFlag(t *testing.T) {
	t.Parallel()
	a := NewAccess(0, "aBCd")
	for _, v := range "aBCd" {
		if !a.HasFlag(v) {
			t.Errorf("Flag %c was not found.", v)
		}
	}

	a.ClearFlag('a')
	a.ClearFlag('C')

	for _, v := range "Bd" {
		if !a.HasFlag(v) {
			t.Errorf("Flag %c was not found.", v)
		}
	}
	for _, v := range "aC" {
		if a.HasFlag(v) {
			t.Errorf("Flag %c was found.", v)
		}
	}
}

func TestAccess_ClearAllFlags(t *testing.T) {
	t.Parallel()
	a := NewAccess(0, "aBCd")
	a.ClearAllFlags()

	for _, v := range "aBCd" {
		if a.HasFlag(v) {
			t.Errorf("Flag %c was found.", v)
		}
	}
}

func TestAccess_IsZero(t *testing.T) {
	t.Parallel()
	a := Access{}
	if !a.IsZero() {
		t.Error("Should be zero.")
	}
	a.SetAccess(1, "a")
	if a.IsZero() {
		t.Error("Should not be zero.")
	}
}

func TestAccess_String(t *testing.T) {
	t.Parallel()

	var table = []struct {
		Level  uint8
		Flags  string
		Expect string
	}{
		{100, "aBCd", "100 BCad"},
		{0, wholeAlphabet, allFlags},
		{0, "BCad", "BCad"},
		{100, "", "100"},
		{0, "", none},
	}

	for _, test := range table {
		a := NewAccess(test.Level, test.Flags)
		if was := a.String(); was != test.Expect {
			t.Errorf("Expected: %s, was: %s", test.Expect, was)
		}
	}
}

func Test_getFlagBits(t *testing.T) {
	t.Parallel()
	bits := getFlagBits("Aab")
	aFlag, bFlag, AFlag := getFlagBit('a'), getFlagBit('b'), getFlagBit('A')
	if aFlag != aFlag&bits {
		t.Error("The correct bit was not set.")
	}
	if bFlag != bFlag&bits {
		t.Error("The correct bit was not set.")
	}
	if AFlag != AFlag&bits {
		t.Error("The correct bit was not set.")
	}
}

func Test_getFlagBit(t *testing.T) {
	t.Parallel()
	var table = map[rune]uint64{
		'A': 0x1,
		'Z': 0x1 << (nAlphabet - 1),
		'a': 0x1 << nAlphabet,
		'z': 0x1 << (nAlphabet*2 - 1),
		'!': 0x0, '_': 0x0, '|': 0x0,
	}

	for flag, expect := range table {
		if bit := getFlagBit(flag); bit != expect {
			t.Errorf("Flag did not match: %c, %X (%X)",
				flag, expect, bit)
		}
	}
}

func Test_getFlagString(t *testing.T) {
	t.Parallel()
	var table = map[uint64]string{
		0x1: "A",
		0x1 << (nAlphabet - 1):   "Z",
		0x1 << nAlphabet:         "a",
		0x1 << (nAlphabet*2 - 1): "z",
	}

	for bit, expect := range table {
		if flag := getFlagString(bit); flag != expect {
			t.Errorf("Flag did not match: %X, %s (%s)",
				bit, expect, flag)
		}
	}

	bits := getFlagBit('a') | getFlagBit('b') | getFlagBit('A') |
		1<<(nAlphabet*2+1)
	should := "Aab"
	if was := getFlagString(bits); was != should {
		t.Errorf("Flag string should be: (%s) was: (%s)", should, was)
	}
}
