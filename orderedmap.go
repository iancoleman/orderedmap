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
		escapedK := strings.Replace(k, `"`, `\"`, -1)
		keyStr := `"` + escapedK + `":`
		// Find the least-nested instance of keyStr
		sections := strings.Split(s, keyStr)
		depth := 0
		openBraces := 0
		closeBraces := 0
		i := 0
		for j, section := range sections {
			i = i + len(section)
			openBraces = openBraces + strings.Count(section, "{")
			closeBraces = closeBraces + strings.Count(section, "}")
			depth = depth + openBraces - closeBraces
			if depth <= 1 {
				ki := KeyIndex{
					Key:   k,
					Index: i,
				}
				orderedKeys = append(orderedKeys, ki)
				// check if value is nested map
				if j < len(sections)-1 {
					nextSectionUnclean := sections[j+1]
					nextSection := strings.TrimSpace(nextSectionUnclean)
					if string(nextSection[0]) == "{" {
						// convert to orderedmap
						mkTyped := m[k].(map[string]interface{})
						oo := OrderedMap{}
						mapToOrderedMap(&oo, nextSection, mkTyped)
						m[k] = oo
					}
				}
				break
			}
			i = i + len(k)
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
