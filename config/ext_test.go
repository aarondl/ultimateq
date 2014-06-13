package config

import (
	"reflect"
	"testing"
)

func TestConfig_Ext_GetSet(t *testing.T) {
	t.Parallel()

	c := NewConfig()
	gext := c.ExtGlobal()
	ext := c.NewExt("ext1")

	// Common
	checkExt("UseJson", false, false, true, gext, ext, t)
	checkExt("NoReconnect", false, false, true, gext, ext, t)
	checkExt("ReconnectTimeout", defaultReconnectTimeout, uint(20), uint(40),
		gext, ext, t)

	// Global Only
	checkExt("ExecDir", "", "execdir", "execdir2", gext, nil, t)
	checkExt("Listen", "", "listen", "", gext, nil, t)

	// Ext Only
	checkExt("Exec", "", "exec", "exec2", nil, ext, t)
	checkExt("Server", "", "serv", "serv2", nil, ext, t)
	checkExt("SSL", false, false, true, nil, ext, t)
	checkExt("SSLCert", "", "cert", "cert2", nil, ext, t)
	checkExt("NoVerifyCert", false, false, true, nil, ext, t)
	checkExt("Unix", "", "unix", "unix2", nil, ext, t)
}

func TestConfig_Ext_GetSetActive(t *testing.T) {
	t.Parallel()

	c := NewConfig()
	gext := c.ExtGlobal()
	ext := c.NewExt("ext1")

	if _, ok := gext.Active("net"); ok {
		t.Error("Expected active to be empty.")
	}
	if _, ok := ext.Active("net"); ok {
		t.Error("Expected active to be empty.")
	}

	gext.SetActive("net", []string{"#channel"})

	if act, ok := gext.Active("net"); !ok || act[0] != "#channel" {
		t.Error("Expected act to contain #channel.")
	}
	if act, ok := ext.Active("net"); !ok || act[0] != "#channel" {
		t.Error("Expected act to contain #channel.")
	}

	ext.SetActive("net", []string{"#notchannel"})

	if act, ok := gext.Active("net"); !ok || act[0] != "#channel" {
		t.Error("Expected act to contain #channel.")
	}
	if act, ok := ext.Active("net"); !ok || act[0] != "#notchannel" {
		t.Error("Expected act to contain #channel.")
	}

	// Test coverage, get a blank array and overwrite a previous active value.
	gext.SetActive("net", []string{})

	if _, ok := gext.Active("net"); ok {
		t.Error("Expected active to be empty.")
	}
	if act, ok := ext.Active("net"); !ok || act[0] != "#notchannel" {
		t.Error("Expected act to contain #channel.")
	}

	// Test coverage, get a blank []interface{} array.
	c.values.get("ext").get("active")["net"] = []interface{}{}
	if _, ok := gext.Active("net"); ok {
		t.Error("Expected active to be empty.")
	}
	if act, ok := ext.Active("net"); !ok || act[0] != "#notchannel" {
		t.Error("Expected act to contain #channel.")
	}
}

func TestExt_Config(t *testing.T) {
	t.Parallel()

	c := NewConfig()
	glb := c.ExtGlobal()

	if exp, got := 0, glb.Config("", ""); len(got) != exp {
		t.Errorf("Expected: %d values but got: %d", exp, len(got))
	}
	if exp, got := 0, glb.Config("net", ""); len(got) != exp {
		t.Errorf("Expected: %d values but got: %d", exp, len(got))
	}
	if exp, got := 0, glb.Config("", "chan"); len(got) != exp {
		t.Errorf("Expected: %d values but got: %d", exp, len(got))
	}
	if exp, got := 0, glb.Config("net", "chan"); len(got) != exp {
		t.Errorf("Expected: %d values but got: %d", exp, len(got))
	}

	glb.SetConfig("", "", "k1", "v")

	if exp, got := 1, glb.Config("", ""); len(got) != exp {
		t.Errorf("Expected: %d values but got: %d", exp, len(got))
	}
	if exp, got := 1, glb.Config("net", ""); len(got) != exp {
		t.Errorf("Expected: %d values but got: %d", exp, len(got))
	}
	if exp, got := 1, glb.Config("", "chan"); len(got) != exp {
		t.Errorf("Expected: %d values but got: %d", exp, len(got))
	}
	if exp, got := 1, glb.Config("net", "chan"); len(got) != exp {
		t.Errorf("Expected: %d values but got: %d", exp, len(got))
	}

	glb.SetConfig("net", "", "k2", "v")

	if exp, got := 1, glb.Config("", ""); len(got) != exp {
		t.Errorf("Expected: %d values but got: %d", exp, len(got))
	}
	if exp, got := 2, glb.Config("net", ""); len(got) != exp {
		t.Errorf("Expected: %d values but got: %d", exp, len(got))
	}
	if exp, got := 1, glb.Config("", "chan"); len(got) != exp {
		t.Errorf("Expected: %d values but got: %d", exp, len(got))
	}
	if exp, got := 1, glb.Config("net", "chan"); len(got) != exp {
		t.Errorf("Expected: %d values but got: %d", exp, len(got))
	}

	glb.SetConfig("", "chan", "k3", "v")

	if exp, got := 1, glb.Config("", ""); len(got) != exp {
		t.Errorf("Expected: %d values but got: %d", exp, len(got))
	}
	if exp, got := 2, glb.Config("net", ""); len(got) != exp {
		t.Errorf("Expected: %d values but got: %d", exp, len(got))
	}
	if exp, got := 2, glb.Config("", "chan"); len(got) != exp {
		t.Errorf("Expected: %d values but got: %d", exp, len(got))
	}
	if exp, got := 1, glb.Config("net", "chan"); len(got) != exp {
		t.Errorf("Expected: %d values but got: %d", exp, len(got))
	}

	glb.SetConfig("net", "chan", "k4", "v")

	if exp, got := 1, glb.Config("", ""); len(got) != exp {
		t.Errorf("Expected: %d values but got: %d", exp, len(got))
	}
	if exp, got := 2, glb.Config("net", ""); len(got) != exp {
		t.Errorf("Expected: %d values but got: %d", exp, len(got))
	}
	if exp, got := 2, glb.Config("", "chan"); len(got) != exp {
		t.Errorf("Expected: %d values but got: %d", exp, len(got))
	}
	if exp, got := 2, glb.Config("net", "chan"); len(got) != exp {
		t.Errorf("Expected: %d values but got: %d", exp, len(got))
	}

	global := glb.Config("", "")
	net := glb.Config("net", "")
	ch := glb.Config("", "chan")
	netch := glb.Config("net", "chan")

	if _, ok := global["k1"]; !ok {
		t.Error("Expected to get a key of k1 in global.")
	}
	if _, ok := net["k1"]; !ok {
		t.Error("Expected to get a key of k1 in net.")
	}
	if _, ok := net["k2"]; !ok {
		t.Error("Expected to get a key of k2 in net.")
	}
	if _, ok := ch["k1"]; !ok {
		t.Error("Expected to get a key of k1 in ch.")
	}
	if _, ok := ch["k3"]; !ok {
		t.Error("Expected to get a key of k3 in ch.")
	}
	if _, ok := netch["k1"]; !ok {
		t.Error("Expected to get a key of k1 in netch.")
	}
	if _, ok := netch["k4"]; !ok {
		t.Error("Expected to get a key of k4 in netch.")
	}
}

func TestExt_ConfigOverride(t *testing.T) {
	t.Parallel()

	c := NewConfig()
	glb := c.ExtGlobal()

	glb.SetConfig("", "", "k", "v1")
	glb.SetConfig("net", "", "k", "v2")
	glb.SetConfig("", "chan", "k", "v3")
	glb.SetConfig("net", "chan", "k", "v4")

	global := glb.Config("", "")
	net := glb.Config("net", "")
	ch := glb.Config("", "chan")
	netch := glb.Config("net", "chan")

	if v, ok := global["k"]; !ok || v != "v1" {
		t.Error("Expected to get a value of v1 in global.")
	}
	if v, ok := net["k"]; !ok || v != "v2" {
		t.Error("Expected to get a value of v2 in net.")
	}
	if v, ok := ch["k"]; !ok || v != "v3" {
		t.Error("Expected to get a value of v3 in ch.")
	}
	if v, ok := netch["k"]; !ok || v != "v4" {
		t.Error("Expected to get a value of v4 in netch.")
	}
}

func checkExt(
	name string, defaultVal, afterGlobal, afterNormal interface{},
	global *extGlobalCtx, normal *extNormalCtx, t *testing.T) {

	globalCtxType := reflect.TypeOf(global)
	normalCtxType := reflect.TypeOf(normal)
	var ok bool

	def := reflect.ValueOf(defaultVal)
	aGlobal := reflect.ValueOf(afterGlobal)
	aNormal := reflect.ValueOf(afterNormal)
	glb := reflect.ValueOf(global)
	nrm := reflect.ValueOf(normal)

	getGlobal, _ := globalCtxType.MethodByName(name)
	setGlobal, _ := globalCtxType.MethodByName("Set" + name)

	getNormal, _ := normalCtxType.MethodByName(name)
	setNormal, _ := normalCtxType.MethodByName("Set" + name)

	var exp, got interface{}
	var ret []reflect.Value
	getargs := make([]reflect.Value, 1)
	setargs := make([]reflect.Value, 2)

	if normal != nil {
		getargs[0] = nrm
		ret = getNormal.Func.Call(getargs)
		exp, got, ok = def.Interface(), ret[0].Interface(), ret[1].Bool()
		if !reflect.DeepEqual(exp, got) || ok {
			t.Errorf("Expected %s to be: %#v, got: %#v", name, exp, got)
		}
	}

	if global != nil {
		getargs[0] = glb
		ret = getGlobal.Func.Call(getargs)
		exp, got, ok = def.Interface(), ret[0].Interface(), ret[1].Bool()
		if !reflect.DeepEqual(exp, got) || ok {
			t.Errorf("Expected %s to be: %#v, got: %#v", name, exp, got)
		}

		setargs[0], setargs[1] = glb, aGlobal
		setGlobal.Func.Call(setargs)

		getargs[0] = glb
		ret = getGlobal.Func.Call(getargs)
		exp, got, ok = aGlobal.Interface(), ret[0].Interface(), ret[1].Bool()
		if !reflect.DeepEqual(exp, got) || !ok {
			t.Errorf("Expected %s to be: %#v, got: %#v", name, exp, got)
		}

		if normal != nil {
			getargs[0] = nrm
			ret = getNormal.Func.Call(getargs)
			exp, got, ok = aGlobal.Interface(), ret[0].Interface(),
				ret[1].Bool()
			if !reflect.DeepEqual(exp, got) || !ok {
				t.Errorf("Expected %s to be: %#v, got: %#v", name, exp, got)
			}
		}
	}

	if normal != nil {
		setargs[0], setargs[1] = nrm, aNormal
		setNormal.Func.Call(setargs)

		if global != nil {
			if global != nil {
				getargs[0] = glb
				ret = getGlobal.Func.Call(getargs)
				exp, got, ok = aGlobal.Interface(), ret[0].Interface(),
					ret[1].Bool()
				if !reflect.DeepEqual(exp, got) || !ok {
					t.Errorf("Expected %s to be: %#v, got: %#v", name, exp, got)
				}
			}
		}

		getargs[0] = nrm
		ret = getNormal.Func.Call(getargs)
		exp, got, ok = aNormal.Interface(), ret[0].Interface(), ret[1].Bool()
		if !reflect.DeepEqual(exp, got) || !ok {
			t.Errorf("Expected %s to be: %#v, got: %#v", name, exp, got)
		}
	}
}
