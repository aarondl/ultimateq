package config

type mapGetter interface {
	get(string) (interface{}, bool)
	getParent(string) (interface{}, bool)
	rlock()
	runlock()
}

type mapSetter interface {
	set(string, interface{})
	lock()
	unlock()
}

type mapGetSetter interface {
	mapGetter
	mapSetter
}

func setVal(m mapSetter, key string, value interface{}) {
	m.lock()
	m.set(key, value)
	m.unlock()
}

// getStr gets a string out of a map.
func getStr(m mapGetter, key string, fallback bool) (string, bool) {
	m.rlock()
	defer m.runlock()

	var val interface{}
	var ok bool

	if val, ok = m.get(key); !ok && fallback {
		val, ok = m.getParent(key)
	}

	if !ok {
		return "", false
	}

	if str, ok := val.(string); ok {
		return str, true
	}

	return "", false
}

// getBool gets a bool out of a map.
func getBool(m mapGetter, key string, fallback bool) (bool, bool) {
	m.rlock()
	defer m.runlock()

	var val interface{}
	var ok bool

	if val, ok = m.get(key); !ok && fallback {
		val, ok = m.getParent(key)
	}

	if !ok {
		return false, false
	}

	if boolval, ok := val.(bool); ok {
		return boolval, true
	}

	return false, false
}

// getUint gets a bool out of a map.
func getUint(m mapGetter, key string, fallback bool) (uint, bool) {
	m.rlock()
	defer m.runlock()

	var val interface{}
	var ok bool

	if val, ok = m.get(key); !ok && fallback {
		val, ok = m.getParent(key)
	}

	if !ok {
		return 0, false
	}

	if u, ok := val.(uint); ok {
		return u, true
	}

	return 0, false
}

// getFloat64 gets a bool out of a map.
func getFloat64(m mapGetter, key string, fallback bool) (float64, bool) {
	m.rlock()
	defer m.runlock()

	var val interface{}
	var ok bool

	if val, ok = m.get(key); !ok && fallback {
		val, ok = m.getParent(key)
	}

	if !ok {
		return 0, false
	}

	if float, ok := val.(float64); ok {
		return float, true
	}

	return 0, false
}

// getMap gets a bool out of a map.
func getMap(m mapGetter, key string, fallback bool) (
	map[string]interface{}, bool) {

	m.rlock()
	defer m.runlock()

	var val interface{}
	var ok bool

	if val, ok = m.get(key); !ok && fallback {
		val, ok = m.getParent(key)
	}

	if !ok {
		return nil, false
	}

	if mapvar, ok := val.(map[string]interface{}); ok {
		return mapvar, true
	}

	return nil, false
}

// getStrArr gets a string array out of a map.
func getStrArr(m mapGetter, key string, fallback bool) ([]string, bool) {
	var val interface{}
	var ok bool

	if val, ok = m.get(key); !ok && fallback {
		val, ok = m.getParent(key)
	}

	if !ok {
		return nil, false
	}

	if arr, ok := val.([]string); ok {
		if len(arr) == 0 {
			return nil, true
		}

		cpyArr := make([]string, len(arr))
		copy(cpyArr, arr)

		return cpyArr, true
	}

	return nil, false
}
