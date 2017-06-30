package orderedmap

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

var NoValueError = errors.New("No value for this key")

type KeyIndex struct {
	Key   string
	Index int
}
type ByIndex []KeyIndex

func (a ByIndex) Len() int           { return len(a) }
func (a ByIndex) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByIndex) Less(i, j int) bool { return a[i].Index < a[j].Index }

type OrderedMap struct {
	keys   []string
	values map[string]interface{}
}

func New() *OrderedMap {
	o := OrderedMap{}
	o.keys = []string{}
	o.values = map[string]interface{}{}
	return &o
}

func (o *OrderedMap) Get(key string) (interface{}, bool) {
	val, exists := o.values[key]
	return val, exists
}

func (o *OrderedMap) Set(key string, value interface{}) {
	_, exists := o.values[key]
	if !exists {
		o.keys = append(o.keys, key)
	}
	o.values[key] = value
}

func (o *OrderedMap) Delete(key string) {
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

func (o *OrderedMap) Keys() []string {
	return o.keys
}

func (o *OrderedMap) UnmarshalJSON(b []byte) error {
	m := map[string]interface{}{}
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	s := string(b)
	mapToOrderedMap(o, s, m)
	return nil
}

func mapToOrderedMap(o *OrderedMap, s string, m map[string]interface{}) {
	// Get the order of the keys
	orderedKeys := []KeyIndex{}
	for k, _ := range m {
		kEscaped := strings.Replace(k, `"`, `\"`, -1)
		kQuoted := `"` + kEscaped + `"`
		// Find how much content exists before this key.
		// If all content from this key and after is replaced with a close
		// brace, it should still form a valid json string.
		sTrimmed := s
		for len(sTrimmed) > 0 {
			lastIndex := strings.LastIndex(sTrimmed, kQuoted)
			if lastIndex == -1 {
				break
			}
			sTrimmed = sTrimmed[0:lastIndex]
			sTrimmed = strings.TrimSpace(sTrimmed)
			if len(sTrimmed) > 0 && sTrimmed[len(sTrimmed)-1] == ',' {
				sTrimmed = sTrimmed[0:len(sTrimmed)-1]
			}
			maybeValidJson := sTrimmed + "}"
			testMap := map[string]interface{}{}
			err := json.Unmarshal([]byte(maybeValidJson), &testMap)
			if err == nil {
				// record the position of this key in s
				ki := KeyIndex{
					Key: k,
					Index: len(sTrimmed),
				}
				orderedKeys = append(orderedKeys, ki)
				// if the value for this key is a map, convert it to an orderedmap
				startOfValueIndex := lastIndex + len(kQuoted)
				valueStr := s[startOfValueIndex:len(s)-1]
				valueStr = strings.TrimSpace(valueStr)
				if len(valueStr) > 0 && valueStr[0] == ':' {
					valueStr = valueStr[1:len(valueStr)]
				}
				valueStr = strings.TrimSpace(valueStr)
				if valueStr[0] == '{' {
					// find end of valueStr by removing everything after last }
					// until it forms valid json
					hasValidJson := true
					subTestMap := map[string]interface{}{}
					err = json.Unmarshal([]byte(valueStr), &subTestMap)
					for len(valueStr) > 0 && err != nil {
						lastCloseBrace := strings.LastIndex(valueStr, "}")
						if lastCloseBrace == -1 {
							hasValidJson = false
							break
						}
						valueStr = valueStr[0:lastCloseBrace+1]
						err = json.Unmarshal([]byte(valueStr), &subTestMap)
					}
					// convert to orderedmap
					if hasValidJson {
						mkTyped := m[k].(map[string]interface{})
						oo := OrderedMap{}
						mapToOrderedMap(&oo, valueStr, mkTyped)
						m[k] = oo
					}
				}
				break
			}
		}
	}
	// Sort the keys
	sort.Sort(ByIndex(orderedKeys))
	// Convert sorted keys to string slice
	k := []string{}
	for _, ki := range orderedKeys {
		k = append(k, ki.Key)
	}
	// Set the OrderedMap values
	o.values = m
	o.keys = k
}

func (o OrderedMap) MarshalJSON() ([]byte, error) {
	s := "{"
	for _, k := range o.keys {
		// add key
		kEscaped := strings.Replace(k, `"`, `\"`, -1)
		s = s + `"` + kEscaped + `":`
		// add value
		v := o.values[k]
		vBytes, err := json.Marshal(v)
		if err != nil {
			return []byte{}, err
		}
		s = s + string(vBytes) + ","
	}
	if len(o.keys) > 0 {
		s = s[0 : len(s)-1]
	}
	s = s + "}"
	return []byte(s), nil
}
