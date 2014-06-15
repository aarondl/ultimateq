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
