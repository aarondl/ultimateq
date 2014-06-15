package data

import "testing"

func TestStoredChannel(t *testing.T) {
	t.Parallel()

	sc := NewStoredChannel("netID", "name")
	if sc == nil {
		t.Error("Failed creating new stored channel.")
	}

	if sc.NetID != "netID" || sc.Name != "name" {
		t.Error("Values not set correctly.")
	}

	if sc.JSONStorer == nil {
		t.Error("Did not initialize JSONStorer.")
	}
}

func TestStoredChannel_SerializeDeserialize(t *testing.T) {
	t.Parallel()

	netID, channel := "netID", "#bots"
	a := NewStoredChannel(netID, channel)

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
		t.Error("Name not deserlialize correctly.")
	}
	if a.NetID != b.NetID {
		t.Error("NetID not deserlialize correctly.")
	}
}
