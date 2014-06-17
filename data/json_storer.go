package data

import "encoding/json"

// JSONStorer allows storage of normal strings and json values into a map.
type JSONStorer map[string]string

// Put puts a regular string into the map.
func (js JSONStorer) Put(key, value string) {
	js[key] = value
}

// Get gets a regular string from the map.
func (js JSONStorer) Get(key string) (string, bool) {
	ret, ok := js[key]
	return ret, ok
}

// PutJSON serializes the value and stores it in the map.
func (js JSONStorer) PutJSON(key string, value interface{}) error {
	jsMarsh, err := json.Marshal(value)
	if err != nil {
		return err
	}
	js[key] = string(jsMarsh)
	return nil
}

// GetJSON deserializes the value from the map and stores it in intf.
func (js JSONStorer) GetJSON(key string, intf interface{}) (bool, error) {
	ret, ok := js[key]
	if !ok {
		return false, nil
	}

	err := json.Unmarshal([]byte(ret), intf)
	return true, err
}
