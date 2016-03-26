package config

import (
	"reflect"
	"testing"
)

func TestConfig_Network_GetSet(t *testing.T) {
	t.Parallel()

	c := New()
	glb := c.Network("")
	net := c.NewNetwork("net")

	check("Nick", "", "nick1", "nick2", glb, net, t)

	check("Altnick", "", "altnick1", "altnick2", glb, net, t)

	check("Username", "", "username1", "username2", glb, net, t)

	check("Realname", "", "realname1", "realname2", glb, net, t)

	check("Password", "", "password1", "password2", glb, net, t)

	check("SSL", false, false, true, glb, net, t)

	check("SSLCert", "", "sslcert1", "sslcert2", glb, net, t)

	check("NoVerifyCert", false, false, true, glb, net, t)

	check("NoState", false, false, true, glb, net, t)

	check("NoStore", false, false, true, glb, net, t)

	check("NoAutoJoin", false, false, true, glb, net, t)

	check("JoinDelay", defaultJoinDelay, uint(20), uint(30),
		glb, net, t)

	check("FloodLenPenalty", defaultFloodLenPenalty, uint(20), uint(30),
		glb, net, t)

	check("FloodTimeout", defaultFloodTimeout, 20.0, 30.0, glb, net, t)

	check("FloodStep", defaultFloodStep, 20.0, 30.0, glb, net, t)

	check("KeepAlive", defaultKeepAlive, 20.0, 30.0, glb, net, t)

	check("NoReconnect", false, false, true, glb, net, t)

	check("ReconnectTimeout", defaultReconnectTimeout,
		uint(20), uint(30), glb, net, t)

	check("Prefix", '.', '!', '@', glb, net, t)

	if srvs, ok := net.Servers(); ok || len(srvs) != 0 {
		t.Error("Expected servers to be empty.")
	}

	net.SetServers([]string{"srv"})

	if srvs, ok := net.Servers(); !ok || len(srvs) != 1 {
		t.Error("Expected servers not to be empty.")
	} else if srvs[0] != "srv" {
		t.Error("Expected the first server to be srv, got:", srvs[0])
	}
}

func TestConfig_Network_GetSetChannels(t *testing.T) {
	t.Parallel()

	c := New()
	glb := c.Network("")
	net := c.NewNetwork("net")
	ch1 := Channel{"b", "c"}
	ch2 := Channel{"b", "c"}

	if chans, ok := glb.Channels(); ok || len(chans) != 0 {
		t.Error("Expected servers to be empty.")
	}
	if chans, ok := net.Channels(); ok || len(chans) != 0 {
		t.Error("Expected servers to be empty.")
	}

	glb.SetChannels(map[string]Channel{"a": ch1})

	if chans, ok := glb.Channels(); !ok || len(chans) != 1 {
		t.Error("Expected servers not to be empty.")
	} else if chans["a"] != ch1 {
		t.Errorf("Expected the first channel to be %v, got: %v", ch1, chans["a"])
	}
	if chans, ok := net.Channels(); !ok || len(chans) != 1 {
		t.Error("Expected servers not to be empty.")
	} else if chans["a"] != ch1 {
		t.Errorf("Expected the first channel to be %v, got: %v", ch1, chans["a"])
	}

	net.SetChannels(map[string]Channel{"a": ch2})

	if chans, ok := glb.Channels(); !ok || len(chans) != 1 {
		t.Error("Expected servers not to be empty.")
	} else if chans["a"] != ch1 {
		t.Errorf("Expected the first channel to be %v, got: %v", ch1, chans["a"])
	}
	if chans, ok := net.Channels(); !ok || len(chans) != 1 {
		t.Error("Expected servers not to be empty.")
	} else if chans["a"] != ch2 {
		t.Errorf("Expected the first channel to be %v, got: %v", ch2, chans["a"])
	}

	// Test Coverage, retrieve a value that's not possible.
	c.values["channels"] = 5
	if chans, ok := glb.Channels(); ok || len(chans) != 0 {
		t.Error("Expected servers to be empty.")
	}
}

func TestConfig_Network_GetChannelPrefix(t *testing.T) {
	t.Parallel()

	c := New()
	glb := c.Network("")
	net := c.NewNetwork("net")
	ch1 := Channel{"b", "1"}
	ch2 := Channel{"b", "1"}

	if pfx, ok := glb.ChannelPrefix("a"); ok || pfx != defaultPrefix {
		t.Error("Expected the prefix to not be set.")
	}
	if pfx, ok := net.ChannelPrefix("b"); ok || pfx != defaultPrefix {
		t.Error("Expected the prefix to not be set.")
	}

	glb.SetChannels(map[string]Channel{"a": ch1, "b": ch2})
	if pfx, ok := glb.ChannelPrefix("a"); !ok || pfx != '1' {
		t.Error("Expected the prefix be set.")
	}
	if pfx, ok := net.ChannelPrefix("a"); !ok || pfx != '1' {
		t.Error("Expected the prefix be set.")
	}

	ch1.Prefix = "2"
	net.SetChannels(map[string]Channel{"a": ch1, "b": ch2})
	if pfx, ok := glb.ChannelPrefix("a"); !ok || pfx != '1' {
		t.Error("Expected the prefix be set.")
	}
	if pfx, ok := net.ChannelPrefix("a"); !ok || pfx != '2' {
		t.Error("Expected the prefix be set.")
	}

	if pfx, ok := net.ChannelPrefix("nochan"); ok || pfx != defaultPrefix {
		t.Error("Expected the prefix not be set.")
	}
}

func check(
	name string, defaultVal, afterGlobal, afterNetwork interface{},
	global, network *NetCTX, t *testing.T) {

	ctxType := reflect.TypeOf(network)
	def := reflect.ValueOf(defaultVal)
	aGlobal := reflect.ValueOf(afterGlobal)
	aNetwork := reflect.ValueOf(afterNetwork)
	glb := reflect.ValueOf(global)
	net := reflect.ValueOf(network)

	get, ok := ctxType.MethodByName(name)
	set, ok := ctxType.MethodByName("Set" + name)

	var exp, got interface{}
	var ret []reflect.Value
	getargs := make([]reflect.Value, 1)
	setargs := make([]reflect.Value, 2)

	getargs[0] = glb
	ret = get.Func.Call(getargs)
	exp, got, ok = def.Interface(), ret[0].Interface(), ret[1].Bool()
	if !reflect.DeepEqual(exp, got) || ok {
		t.Errorf("Expected %s to be: %#v, got: %#v", name, exp, got)
	}
	getargs[0] = net
	ret = get.Func.Call(getargs)
	exp, got, ok = def.Interface(), ret[0].Interface(), ret[1].Bool()
	if !reflect.DeepEqual(exp, got) || ok {
		t.Errorf("Expected %s to be: %#v, got: %#v", name, exp, got)
	}

	setargs[0], setargs[1] = glb, aGlobal
	set.Func.Call(setargs)

	getargs[0] = glb
	ret = get.Func.Call(getargs)
	exp, got, ok = aGlobal.Interface(), ret[0].Interface(), ret[1].Bool()
	if !reflect.DeepEqual(exp, got) || !ok {
		t.Errorf("Expected %s to be: %#v, got: %#v", name, exp, got)
	}
	getargs[0] = net
	ret = get.Func.Call(getargs)
	exp, got, ok = aGlobal.Interface(), ret[0].Interface(), ret[1].Bool()
	if !reflect.DeepEqual(exp, got) || !ok {
		t.Errorf("Expected %s to be: %#v, got: %#v", name, exp, got)
	}

	setargs[0], setargs[1] = net, aNetwork
	set.Func.Call(setargs)

	getargs[0] = glb
	ret = get.Func.Call(getargs)
	exp, got, ok = aGlobal.Interface(), ret[0].Interface(), ret[1].Bool()
	if !reflect.DeepEqual(exp, got) || !ok {
		t.Errorf("Expected %s to be: %#v, got: %#v", name, exp, got)
	}
	getargs[0] = net
	ret = get.Func.Call(getargs)
	exp, got, ok = aNetwork.Interface(), ret[0].Interface(), ret[1].Bool()
	if !reflect.DeepEqual(exp, got) || !ok {
		t.Errorf("Expected %s to be: %#v, got: %#v", name, exp, got)
	}
}
