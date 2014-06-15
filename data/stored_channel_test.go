package data

import "testing"

func TestStoredChannel(t *testing.T) {
	sc := NewStoredChannel("Hello")
	if sc == nil {
		t.Error("Failed creating new stored channel.")
	}

	if sc.Name != "Hello" {
		t.Error("Expected Hello, got", sc.Name)
	}

	if sc.JSONStorer == nil {
		t.Error("Did not initialize JSONStorer")
	}
}

func TestStoredChannel_SerializeDeserialize(t *testing.T) {
	channel := "#bots"
	a := NewStoredChannel(channel)

	serialized, err := a.serialize()
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if len(serialized) == 0 {
		t.Error("Serialization did not yield a serialized copy.")
	}

	b, err := deserializeChannel(serialized)
	if err != nil {
		t.Fatal("Deserialization failed.")
	}
	if a.Name != b.Name {
		t.Error("Channelname or Password did not deserializeChannel.")
	}
}
