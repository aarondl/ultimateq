package config

import (
	"errors"
	"reflect"
	"testing"
)

func TestConfig_New(t *testing.T) {
	t.Parallel()

	c := New()
	if c == nil {
		t.Error("Expected a configuration to be created.")
	}
}

func TestConfig_Clear(t *testing.T) {
	t.Parallel()

	c := New()
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

func TestConfig_Replace(t *testing.T) {
	t.Parallel()

	c1 := New().FromString(`nick = "hello"`)
	c2 := New().FromString(`nick = "there"`)

	if val, ok := c1.Network("").Nick(); !ok || val != "hello" {
		t.Error(`Expected nick to be "hello", got:`, val)
	}
	if val, ok := c1.Replace(c2).Network("").Nick(); !ok || val != "there" {
		t.Error(`Expected nick to be "there", got:`, val)
	}
}

func TestConfig_Clone(t *testing.T) {
	t.Parallel()

	c := New().FromString(`
	string = "str"
	[[channels]]
		uint = 5
	[networks.ircnet]
		servers = ["str"]
	`)

	c.NewNetwork("othernet").
		SetServers([]string{"str"}).
		SetChannels(map[string]Channel{"a": {"b", "c"}})

	nc := c.Clone()

	checkMap(nc.values, c.values, t)
}

// checkMap is essentially useless, but hopefully it's doing something.
func checkMap(dest, src mp, t *testing.T) {
	for key, value := range src {
		switch v := value.(type) {
		case map[string]interface{}:
			checkMap(dest.get(key), v, t)
		case mp:
			checkMap(dest.get(key), v, t)
		case []Channel:
			destChans := dest[key].([]Channel)
			for i, c := range v {
				if &c == &destChans[i] {
					t.Error("Expected channels to be deep copied.")
				}
			}
		// The Following cases are immutable so we don't care so much.
		case string:
		case int:
		case uint:
		case float64:
		default:
			orig := reflect.ValueOf(v)
			clone := reflect.ValueOf(dest[key])
			if reflect.DeepEqual(orig, clone) {
				t.Errorf("Expected %s to be deep copied.", key)
			}
		}
	}
}

func TestConfig_Networks(t *testing.T) {
	t.Parallel()

	if New().Networks() != nil {
		t.Error("Expected networks to be empty.")
	}

	c := New().FromString(configuration)

	nets := c.Networks()
	exps := []string{"ircnet", "noirc"}

	for _, exp := range exps {
		found := false
		for _, net := range nets {
			if net == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Did not find: %s in network list.", exp)
		}
	}
}

func TestConfig_Exts(t *testing.T) {
	t.Parallel()

	if New().Exts() != nil {
		t.Error("Expected exts to be empty.")
	}

	c := New().FromString(configuration)
	exts := c.Exts()
	exps := []string{"myext"}

	for _, exp := range exps {
		found := false
		for _, ext := range exts {
			if ext == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Did not find: %s in ext list.", exp)
		}
	}
}

func TestConfig_Contexts(t *testing.T) {
	t.Parallel()

	c := New().FromString(configuration)

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

	c := New()
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

	c := New()
	if v, ok := c.StoreFile(); ok || v != defaultStoreFile {
		t.Error("Expected store file not to be set, and to get default:", v)
	}
	c.SetStoreFile("a")
	if v, ok := c.StoreFile(); !ok || v != "a" {
		t.Error("Expected store file to be set, and to get a, got:", v)
	}

	if v, ok := c.LogFile(); ok || v != "" {
		t.Error("Expected log file not to be set, and to get default:", v)
	}
	c.SetLogFile("a")
	if v, ok := c.LogFile(); !ok || v != "a" {
		t.Error("Expected log file to be set, and to get a, got:", v)
	}

	if v, ok := c.LogLevel(); ok || v != defaultLogLevel {
		t.Error("Expected log level not to be set, and to get default:", v)
	}
	c.SetLogLevel("a")
	if v, ok := c.LogLevel(); !ok || v != "a" {
		t.Error("Expected log level to be set, and to get a, got:", v)
	}

	if v, ok := c.NoCoreCmds(); ok || v != false {
		t.Error("Expected no core cmds not to be set, and to get default:", v)
	}
	c.SetNoCoreCmds(true)
	if v, ok := c.NoCoreCmds(); !ok || v != true {
		t.Error("Expected no core cmds to be set, and to get a, got:", v)
	}

	if v, ok := c.SecretKey(); ok || v != "" {
		t.Error("Expected secret key not to be set, and to get default:", v)
	}
	c.SetSecretKey("a")
	if v, ok := c.SecretKey(); !ok || v != "a" {
		t.Error("Expected secret key to be set, and to get a, got:", v)
	}
}
