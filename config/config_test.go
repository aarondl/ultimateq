package config

import (
	"errors"
	"testing"
)

func TestConfig_New(t *testing.T) {
	t.Parallel()

	c := NewConfig()
	if c == nil {
		t.Error("Expected a configuration to be created.")
	}
}

func TestConfig_Clear(t *testing.T) {
	t.Parallel()

	c := NewConfig()
	c.values = map[string]interface{}{"network": "something"}
	c.errors = errList{errors.New("something")}
	c.filename = "filename"

	c.Clear()
	if len(c.values) != 0 {
		t.Error("Values should be blank, got:", c.values)
	}
	if len(c.errors) != 0 {
		t.Error("Filename should be blank, got:", c.errors)
	}
	if len(c.filename) != 0 {
		t.Error("Filename should be blank, got:", c.filename)
	}
}

func TestConfig_Contexts(t *testing.T) {
	t.Parallel()

	c := NewConfig().FromString(configuration)

	if c.Network("") == nil {
		t.Error("Expected to be able to get global context.")
	}
	if c.Network("ircnet") == nil {
		t.Error("Expected to be able to get network context.")
	}
	if c.ExtGlobal() == nil {
		t.Error("Expected to be able to get ext context.")
	}
	if c.Ext("myext") == nil {
		t.Error("Expected to be able to get ext context.")
	}

	if ctx := c.Network("nonexistent"); ctx != nil {
		t.Error("Should retrieve no context for non-existent things, got:", ctx)
	}
	if ctx := c.Ext("nonexistent"); ctx != nil {
		t.Error("Should retrieve no context for non-existent things, got:", ctx)
	}
}

func TestConfig_NewThings(t *testing.T) {
	t.Parallel()

	c := NewConfig()
	if net := c.NewNetwork("net1"); net == nil {
		t.Error("Should have created a new network.")
	}
	if ext := c.NewExt("ext1"); ext == nil {
		t.Error("Should have created a new extension.")
	}

	if net := c.NewNetwork("net2"); net == nil {
		t.Error("Should have created a new network.")
	}
	if ext := c.NewExt("ext2"); ext == nil {
		t.Error("Should have created a new extension.")
	}

	if net := c.NewNetwork("net1"); net != nil {
		t.Error("Should not have created a new network.")
	}
	if ext := c.NewExt("ext2"); ext != nil {
		t.Error("Should not have created a new extension.")
	}
}

func TestConfig_Config_GetSet(t *testing.T) {
	t.Parallel()

	c := NewConfig()
	if v, ok := c.StoreFile(); ok || v != defaultStoreFile {
		t.Error("Expected store file not to be set, and to get default:", v)
	}
	c.SetStoreFile("a")
	if v, ok := c.StoreFile(); !ok || v != "a" {
		t.Error("Expected store file to be set, and to get a, got:", v)
	}

	if v, ok := c.NoCoreCmds(); ok || v != false {
		t.Error("Expected store file not to be set, and to get default:", v)
	}
	c.SetNoCoreCmds(true)
	if v, ok := c.NoCoreCmds(); !ok || v != true {
		t.Error("Expected store file to be set, and to get a, got:", v)
	}
}
