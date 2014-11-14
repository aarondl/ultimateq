package data

import "testing"

func TestJSONStorer(t *testing.T) {
	t.Parallel()

	js := make(JSONStorer)
	js.Put("Hello", "world")
	if got, ok := js.Get("Hello"); !ok || got != "world" {
		t.Error("Expected world, got", got)
	}

	st := []struct {
		Name string
		Age  int
	}{
		{"zamn", 21},
		{},
	}

	err := js.PutJSON(st[0].Name, st[0])
	if err != nil {
		t.Error("Unexpected marshalling error:", err)
	}

	if ok, err := js.GetJSON(st[0].Name, &st[1]); !ok {
		t.Error("Key not found:", ok)
	} else if err != nil {
		t.Error("Error unmarshalling value:", err)
	} else if st[0] != st[1] {
		t.Errorf("Structs do not match %#v %#v", st[0], st[1])
	}
}
