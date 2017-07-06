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
				sTrimmed = sTrimmed[0 : len(sTrimmed)-1]
			}
			maybeValidJson := sTrimmed + "}"
			testMap := map[string]interface{}{}
			err := json.Unmarshal([]byte(maybeValidJson), &testMap)
			if err == nil {
				// record the position of this key in s
				ki := KeyIndex{
					Key:   k,
					Index: len(sTrimmed),
				}
				orderedKeys = append(orderedKeys, ki)
				// shorten the string to get the next key
				startOfValueIndex := lastIndex + len(kQuoted)
				valueStr := s[startOfValueIndex : len(s)-1]
				valueStr = strings.TrimSpace(valueStr)
				if len(valueStr) > 0 && valueStr[0] == ':' {
					valueStr = valueStr[1:len(valueStr)]
				}
				valueStr = strings.TrimSpace(valueStr)
				if valueStr[0] == '{' {
					// if the value for this key is a map, convert it to an orderedmap.
					// find end of valueStr by removing everything after last }
					// until it forms valid json
					hasValidJson := false
					i := 1
					for i < len(valueStr) && !hasValidJson {
						if valueStr[i] != '}' {
							i = i + 1
							continue
						}
						subTestMap := map[string]interface{}{}
						testValue := valueStr[0 : i+1]
						err = json.Unmarshal([]byte(testValue), &subTestMap)
						if err == nil {
							hasValidJson = true
							valueStr = testValue
							break
						}
						i = i + 1
					}
					// convert to orderedmap
					if hasValidJson {
						mkTyped := m[k].(map[string]interface{})
						oo := OrderedMap{}
						mapToOrderedMap(&oo, valueStr, mkTyped)
						m[k] = oo
					}
				} else if valueStr[0] == '[' {
					// if the value for this key is a []interface, convert any map items to an orderedmap.
					// find end of valueStr by removing everything after last ]
					// until it forms valid json
					hasValidJson := false
					i := 1
					for i < len(valueStr) && !hasValidJson {
						if valueStr[i] != ']' {
							i = i + 1
							continue
						}
						subTestSlice := []interface{}{}
						testValue := valueStr[0 : i+1]
						err = json.Unmarshal([]byte(testValue), &subTestSlice)
						if err == nil {
							hasValidJson = true
							valueStr = testValue
							break
						}
						i = i + 1
					}
					if hasValidJson {
						itemsStr := valueStr[1 : len(valueStr)-1]
						// get next item in the slice
						itemIndex := 0
						startItem := 0
						endItem := 0
						for endItem < len(itemsStr) {
							if itemsStr[endItem] != ',' && endItem < len(itemsStr)-1 {
								endItem = endItem + 1
								continue
							}
							// if this substring compiles to json, it's the next item
							possibleItemStr := strings.TrimSpace(itemsStr[startItem:endItem])
							var possibleItem interface{}
							err = json.Unmarshal([]byte(possibleItemStr), &possibleItem)
							if err != nil {
								endItem = endItem + 1
								continue
							}
							// if item is map, convert to orderedmap
							if possibleItemStr[0] == '{' {
								mkTyped := m[k].([]interface{})
								mkiTyped := mkTyped[itemIndex].(map[string]interface{})
								oo := OrderedMap{}
								mapToOrderedMap(&oo, possibleItemStr, mkiTyped)
								// replace original map with orderedmap
								mkTyped[itemIndex] = oo
								m[k] = mkTyped
							}
							// remove this item from itemsStr
							startItem = endItem + 1
							endItem = endItem + 1
							itemIndex = itemIndex + 1
						}
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
