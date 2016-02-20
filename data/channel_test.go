package data

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestChannel_Create(t *testing.T) {
	t.Parallel()

	ch := NewChannel("", testKinds)
	if got := ch; got != nil {
		t.Error("Expected: %v to be nil.", got)
	}

	name := "#CHAN"
	ch = NewChannel(name, testKinds)
	if ch == nil {
		t.Error("Unexpected nil.")
	}
	if exp, got := ch.Name, name; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := ch.Topic, ""; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}

func TestChannel_Bans(t *testing.T) {
	t.Parallel()

	bans := []string{"ban1", "ban2"}
	ch := NewChannel("name", testKinds)

	ch.SetBans(bans)
	got := ch.Bans()
	for i := 0; i < len(got); i++ {
		if exp, got := got[i], bans[i]; exp != got {
			t.Error("Expected: %v, got: %v", exp, got)
		}
	}
	bans[0] = "ban3"
	if exp, got := got[0], bans[0]; exp == got {
		t.Error("Did not want: %v, got: %v", exp, got)
	}

	if exp, got := ch.HasBan("ban2"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	ch.DeleteBan("ban2")
	if exp, got := ch.HasBan("ban2"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	if exp, got := ch.HasBan("ban2"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	ch.AddBan("ban2")
	if exp, got := ch.HasBan("ban2"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}

func TestChannel_IsBanned(t *testing.T) {
	t.Parallel()

	bans := []string{"*!*@host.com", "nick!*@*"}
	ch := NewChannel("name", testKinds)
	ch.SetBans(bans)
	if exp, got := ch.IsBanned("nick"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := ch.IsBanned("notnick"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := ch.IsBanned("nick!user@host"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := ch.IsBanned("notnick!user@host"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := ch.IsBanned("notnick!user@host.com"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}

func TestChannel_DeleteBanWild(t *testing.T) {
	t.Parallel()

	bans := []string{"*!*@host.com", "nick!*@*", "nick2!*@*"}
	ch := NewChannel("name", testKinds)
	ch.SetBans(bans)
	if exp, got := ch.IsBanned("nick"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := ch.IsBanned("notnick"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := ch.IsBanned("nick!user@host"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := ch.IsBanned("notnick!user@host"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := ch.IsBanned("notnick!user@host.com"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := ch.IsBanned("nick2!user@host"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	ch.DeleteBans("")
	if exp, got := len(ch.Bans()), 3; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	ch.DeleteBans("nick")
	if exp, got := ch.IsBanned("nick"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := ch.IsBanned("notnick"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := ch.IsBanned("nick!user@host"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := ch.IsBanned("nick2!user@host"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := ch.IsBanned("notnick!user@host"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := ch.IsBanned("notnick!user@host.com"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := ch.IsBanned("nick2!user@host"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	if exp, got := len(ch.Bans()), 2; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	ch.DeleteBans("nick2!user@host.com")
	if exp, got := ch.IsBanned("nick2!user@host"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := ch.IsBanned("notnick!user@host.com"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := ch.IsBanned("nick2!user@host"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	if exp, got := len(ch.Bans()), 0; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	ch.DeleteBans("nick2!user@host.com")
	if exp, got := len(ch.Bans()), 0; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}

func TestChannel_String(t *testing.T) {
	t.Parallel()

	ch := NewChannel("name", testKinds)
	if exp, got := ch.String(), "name"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}

func TestChannel_JSONify(t *testing.T) {
	t.Parallel()

	a := &Channel{
		Name:  "a",
		Topic: "b",
	}
	var b Channel

	str, err := json.Marshal(a)
	if err != nil {
		t.Error(err)
	}

	jsonStr := `{"name":"a","topic":"b",` +
		`"channel_modes":{"modes":null,"arg_modes":null,` +
		`"address_modes":null,"addresses":0,"mode_kinds":null}}`

	if string(str) != jsonStr {
		t.Errorf("Wrong JSON: %s", str)
	}

	if err = json.Unmarshal(str, &b); err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(*a, b) {
		t.Error("A and B differ:", a, b)
	}
}
