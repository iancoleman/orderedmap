package orderedmap

import (
	"bytes"
	"encoding/json"
	"io"
)

type OrderedMapCore interface {
	CoreKeys() []string
	CoreSetKeys([]string)
	CoreValues() map[string]interface{}
	CoreSetValues(map[string]interface{})
	CoreGetKey(key string) (interface{}, bool)
	CoreSetKey(key string, value interface{})
	CoreDeleteKey(key string)
}

type CloneFunc func(...map[string]interface{}) OrderedMapCore

type orderedMapCore struct {
	keys   []string
	values map[string]interface{}
}

func NewCore(oms ...map[string]interface{}) OrderedMapCore {
	var om map[string]interface{}
	if len(oms) > 0 {
		om = oms[0]
	} else {
		om = make(map[string]interface{})
	}
	return &orderedMapCore{
		keys:   make([]string, 0, len(om)),
		values: om,
	}
}

func (o *orderedMapCore) CoreGetKey(key string) (interface{}, bool) {
	val, exists := o.values[key]
	return val, exists
}

func (o *orderedMapCore) CoreSetKey(key string, value interface{}) {
	_, exists := o.values[key]
	if !exists {
		o.keys = append(o.keys, key)
	}
	o.values[key] = value
}

func (o *orderedMapCore) CoreDeleteKey(key string) {
	// check key is in use
	_, ok := o.values[key]
	if !ok {
		return
	}
	// remove from keys
	for i, k := range o.keys {
		if k == key {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			break
		}
	}
	// remove from values
	delete(o.values, key)
}

func (o *orderedMapCore) CoreKeys() []string {
	return o.keys
}

func (o *orderedMapCore) CoreSetKeys(keys []string) {
	o.keys = keys
}

func (o *orderedMapCore) CoreValues() map[string]interface{} {
	return o.values
}

func (o *orderedMapCore) CoreSetValues(values map[string]interface{}) {
	o.values = values
}

func BoundUnmarshalJSON(o OrderedMapCore, cloneFn CloneFunc, b []byte) error {
	if o == nil {
		panic("orderedmap: UnmarshalJSON on nil pointer")
	}
	if o.CoreValues() == nil {
		o.CoreSetValues(make(map[string]interface{}))
	}
	val := o.CoreValues()
	err := json.Unmarshal(b, &val)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	if _, err = dec.Token(); err != nil { // skip '{'
		return err
	}
	return decodeOrderedMap(dec, o, cloneFn)
}

func BoundMarshalJSON(o OrderedMapCore, encoder *json.Encoder, buf io.Writer) error {
	buf.Write([]byte{'{'})

	values := o.CoreValues()
	for i, k := range o.CoreKeys() {
		if i > 0 {
			buf.Write([]byte{','})
		}
		// add key
		if err := encoder.Encode(k); err != nil {
			return err
		}
		buf.Write([]byte{':'})
		// add value
		if err := encoder.Encode(values[k]); err != nil {
			return err
		}
	}
	buf.Write([]byte{'}'})
	return nil
}

func decodeOrderedMap(dec *json.Decoder, o OrderedMapCore, cloneFn CloneFunc) error {
	vals := o.CoreValues()
	hasKey := make(map[string]bool, len(vals))
	for {
		token, err := dec.Token()
		if err != nil {
			return err
		}
		if delim, ok := token.(json.Delim); ok && delim == '}' {
			return nil
		}
		key := token.(string)
		if hasKey[key] {
			// duplicate key
			for j, k := range o.CoreKeys() {
				if k == key {
					copy(o.CoreKeys()[j:], o.CoreKeys()[j+1:])
					break
				}
			}
			o.CoreKeys()[len(o.CoreKeys())-1] = key
		} else {
			hasKey[key] = true
			o.CoreSetKeys(append(o.CoreKeys(), key))
		}

		token, err = dec.Token()
		if err != nil {
			return err
		}
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{':
				if values, ok := vals[key].(map[string]interface{}); ok {
					newMap := cloneFn(values)
					if err = decodeOrderedMap(dec, newMap, cloneFn); err != nil {
						return err
					}
					vals[key] = newMap
				} else if oldMap, ok := vals[key].(OrderedMap); ok {
					newMap := cloneFn(oldMap.Values())
					if err = decodeOrderedMap(dec, newMap, cloneFn); err != nil {
						return err
					}
					vals[key] = newMap
				} else if err = decodeOrderedMap(dec, cloneFn(), cloneFn); err != nil {
					return err
				}
			case '[':
				if values, ok := vals[key].([]interface{}); ok {
					if err = decodeSlice(dec, values, o, cloneFn); err != nil {
						return err
					}
				} else if err = decodeSlice(dec, []interface{}{}, o, cloneFn); err != nil {
					return err
				}
			}
		}
	}
}

func decodeSlice(dec *json.Decoder, s []interface{}, o OrderedMapCore, cloneFn CloneFunc) error {
	for index := 0; ; index++ {
		token, err := dec.Token()
		if err != nil {
			return err
		}
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{':
				if index < len(s) {
					if values, ok := s[index].(map[string]interface{}); ok {
						newMap := cloneFn(values)
						if err = decodeOrderedMap(dec, newMap, cloneFn); err != nil {
							return err
						}
						s[index] = newMap
					} else if oldMap, ok := s[index].(OrderedMap); ok {
						newMap := cloneFn(oldMap.Values())
						if err = decodeOrderedMap(dec, newMap, cloneFn); err != nil {
							return err
						}
						s[index] = newMap
					} else if err = decodeOrderedMap(dec, cloneFn(), cloneFn); err != nil {
						return err
					}
				} else if err = decodeOrderedMap(dec, cloneFn(), cloneFn); err != nil {
					return err
				}
			case '[':
				if index < len(s) {
					if values, ok := s[index].([]interface{}); ok {
						if err = decodeSlice(dec, values, o, cloneFn); err != nil {
							return err
						}
					} else if err = decodeSlice(dec, []interface{}{}, o, cloneFn); err != nil {
						return err
					}
				} else if err = decodeSlice(dec, []interface{}{}, o, cloneFn); err != nil {
					return err
				}
			case ']':
				return nil
			}
		}
	}
}
