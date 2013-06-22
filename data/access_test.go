package data

import (
	. "testing"
)

func TestAccess(t *T) {
	t.Parallel()
	a := CreateAccess(0)
	if a == nil {
		t.Fatal("Failed to Create")
	}
	if a.Level != 0 || a.Flags != 0 {
		t.Error("Bad init")
	}

	a = CreateAccess(100, "aBC", "d")
	if a.Level != 100 {
		t.Error("Level was not set")
	}
	for _, v := range "aBCd" {
		if !a.HasFlag(v) {
			t.Errorf("Flag %c was not found.", v)
		}
	}
}

func TestAccess_HasLevel(t *T) {
	a := CreateAccess(50)

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

func TestAccess_HasFlag(t *T) {
	t.Parallel()
	a := CreateAccess(0, "aBC", "d")
	for _, v := range "aBCd" {
		if !a.HasFlag(v) {
			t.Errorf("Flag %c was not found.", v)
		}
	}
}

func TestAccess_HasFlags(t *T) {
	t.Parallel()
	a := CreateAccess(0, "aBC", "d")
	if !a.HasFlags("aBCd") {
		t.Error("Flags were not all found.")
	}
}

func TestAccess_SetFlag(t *T) {
	t.Parallel()
	a := CreateAccess(0)
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

func TestAccess_MultiEffects(t *T) {
	t.Parallel()
	a := CreateAccess(0)
	a.SetFlags("ab", "A")
	if !a.HasFlags("ab", "A") {
		t.Error("Set flags failed")
	}
	a.ClearFlags("ab", "A")
	if a.HasFlags("a") || a.HasFlags("b") || a.HasFlags("A") {
		t.Error("Clear flags failed")
	}
}

func TestAccess_ClearFlag(t *T) {
	t.Parallel()
	a := CreateAccess(0, "aBCd")
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

func TestAccess_ClearAllFlags(t *T) {
	t.Parallel()
	a := CreateAccess(0, "aBCd")
	a.ClearAllFlags()

	for _, v := range "aBCd" {
		if a.HasFlag(v) {
			t.Errorf("Flag %c was found.", v)
		}
	}
}

func TestAccess_getFlagBit(t *T) {
	t.Parallel()
	nAlphabet := uint(26)
	var table = map[rune]uint64{
		'A': 0x1, 'Z': 0x1 << (nAlphabet - 1),
		'a': 0x1 << nAlphabet, 'z': 0x1 << (nAlphabet*2 - 1),
		'!': 0x0, '_': 0x0, '|': 0x0,
	}

	for flag, expect := range table {
		if bit := getFlagBit(flag); bit != expect {
			t.Errorf("Flag did not match: %c, %X (%X)",
				flag, expect, bit)
		}
	}
}
