package data

import (
	"encoding/json"
	"reflect"
	"testing"
)

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

func TestStoredChannel_JSONify(t *testing.T) {
	t.Parallel()

	a := &StoredChannel{
		NetID:      "a",
		Name:       "b",
		JSONStorer: JSONStorer{"some": "data"},
	}
	var b StoredChannel

	str, err := json.Marshal(a)
	if err != nil {
		t.Error(err)
	}

	jsonStr := `{"netid":"a","name":"b","data":{"some":"data"}}`

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
