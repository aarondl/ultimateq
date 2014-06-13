package config

import (
	"sync"
	"testing"
)

func TestMapHelpers_Mp(t *testing.T) {
	t.Parallel()

	mp := mp{
		"m": mp{},
		"a": []map[string]interface{}{},
	}

	if mp.get("m") == nil {
		t.Error("Expected to find a map m.")
	}
	if mp.getArr("a") == nil {
		t.Error("Expected to find a array of maps a.")
	}

	mp = nil
	if mp.get("m") != nil {
		t.Error("Expected it to be nil.")
	}
	if mp.getArr("a") != nil {
		t.Error("Expected it to be nil.")
	}
}

func TestMapHelpers_MpEnsure(t *testing.T) {
	t.Parallel()

	var m mp = map[string]interface{}{}
	second := m.ensure("first").ensure("second")
	if second == nil {
		t.Error("Expected it to return the new map.")
	}
	if m["first"] == nil {
		t.Error("Expected first to be created.")
	}
	if m.get("first").get("second") == nil {
		t.Error("Expected second to be created.")
	}
	if m.ensure("first") == nil {
		t.Error("Expected to get an old map of type map[string]interface{}")
	}

	m["first"] = m
	if m.ensure("first") == nil {
		t.Error("Expected to get an old map of type mp.")
	}

	m["first"] = interface{}(5)
	if nil != m.ensure("first").ensure("second") {
		t.Error("Expected a bad type to break it.")
	}
}

func TestMapHelpers_BadTypes(t *testing.T) {
	t.Parallel()

	bad := map[string]interface{}{
		"badstr":   5,
		"badbool":  5,
		"baduint":  "5",
		"badfloat": true,
		"badarr":   false,
	}
	ctx := &NetCTX{&sync.RWMutex{}, nil, bad}

	if _, ok := getStr(ctx, "badstr", false); ok {
		t.Error("Expected the bad type to return nothing.")
	}
	if _, ok := getBool(ctx, "badbool", false); ok {
		t.Error("Expected the bad type to return nothing.")
	}
	if _, ok := getUint(ctx, "baduint", false); ok {
		t.Error("Expected the bad type to return nothing.")
	}
	if _, ok := getFloat64(ctx, "badfloat", false); ok {
		t.Error("Expected the bad type to return nothing.")
	}
	if _, ok := getStrArr(ctx, "badarr", false); ok {
		t.Error("Expected the bad type to return nothing.")
	}
}

func TestMapHelpers_GetStrArrEdgeCases(t *testing.T) {
	t.Parallel()

	parent := map[string]interface{}{
		"arr": []string{},
		"int": []interface{}{},
	}
	child := make(map[string]interface{})

	ctx := &NetCTX{&sync.RWMutex{}, parent, child}
	if _, ok := getStrArr(ctx, "arr", true); ok {
		t.Error("Expected empty array to return nothing.")
	}
	if _, ok := getStrArr(ctx, "int", true); ok {
		t.Error("Expected empty array to return nothing.")
	}
}
