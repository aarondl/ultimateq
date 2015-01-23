package data

import (
	"regexp"
	"testing"
)

func TestModeDiff_Create(t *testing.T) {
	t.Parallel()
	m := NewModeDiff(testKinds)
	var _ moder = &m
}

func TestModeDiff_Clone(t *testing.T) {
	t.Parallel()

	diff := NewModeDiff(testKinds)
	diff.Apply("a-m")
	clone := diff.Clone()
	if !clone.IsSet("a") || !clone.IsUnset("m") {
		t.Error("Expected a to be set and m to be unset.")
	}
}

func TestModeDiff_Apply(t *testing.T) {
	t.Parallel()

	d := NewModeDiff(testKinds)
	pos, neg := d.Apply("+ab-c 10 ")
	if got, exp := len(pos), 0; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := len(neg), 0; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := d.IsSet("ab 10"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := d.IsSet("c"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := d.IsUnset("c"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	d = NewModeDiff(testKinds)
	pos, neg = d.Apply("+b-b 10 10")
	if got, exp := len(pos), 0; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := len(neg), 0; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := d.IsSet("b 10"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := d.IsUnset("b 10"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	d = NewModeDiff(testKinds)
	pos, neg = d.Apply("-b+b 10 10")
	if got, exp := len(pos), 0; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := len(neg), 0; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := d.IsSet("b 10"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := d.IsUnset("b 10"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	pos, neg = d.Apply("+x-y+z")
	if got, exp := len(pos), 0; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := len(neg), 0; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := d.IsSet("x"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := d.IsUnset("y"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := d.IsSet("z"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := d.IsUnset("x"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := d.IsSet("y"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := d.IsUnset("z"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	pos, neg = d.Apply("+vx-yo+vz user1 user2 user3")
	if got, exp := len(pos), 2; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := len(neg), 1; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := pos[0].Mode, 'v'; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := pos[0].Arg, "user1"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := pos[1].Mode, 'v'; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := pos[1].Arg, "user3"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := neg[0].Mode, 'o'; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := neg[0].Arg, "user2"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := d.IsSet("x"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := d.IsUnset("y"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := d.IsSet("z"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := d.IsUnset("x"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := d.IsSet("y"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got, exp := d.IsUnset("z"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}

func TestModeDiff_String(t *testing.T) {
	t.Parallel()

	diff := NewModeDiff(testKinds)
	diff.pos.Set("a", "b host1", "c 1")
	diff.neg.Set("x", "y", "z", "b host2")
	str := diff.String()
	matched, err := regexp.MatchString(
		`^\+[abc]{3}-[xyzb]{4}( 1| host1){2}( host2){1}$`, str)
	if err != nil {
		t.Error("Regexp failed to compile:", err)
	}
	if !matched {
		t.Errorf("Expected: %q to match the pattern.", str)
	}

	diff = NewModeDiff(testKinds)
	diff.pos.Set("x", "y", "z")
	diff.neg.Set("x", "y", "z")
	str = diff.String()
	matched, err = regexp.MatchString(`^\+[xyz]{3}-[xyz]{3}$`, str)
	if err != nil {
		t.Error("Regexp failed to compile:", err)
	}
	if !matched {
		t.Errorf("Expected: %q to match the pattern.", str)
	}
}
