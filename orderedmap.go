package orderedmap

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"fmt"
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

type Pair struct {
	key string
	value interface{}
}

func (kv *Pair) Key() string {
	return kv.key
}

func (kv *Pair) Value() interface{} {
	return kv.value
}

type ByPair struct {
	Pairs []*Pair
	LessFunc func(a *Pair, j *Pair) bool
}

func (a ByPair) Len() int           { return len(a.Pairs) }
func (a ByPair) Swap(i, j int)      { a.Pairs[i], a.Pairs[j] = a.Pairs[j], a.Pairs[i] }
func (a ByPair) Less(i, j int) bool { return a.LessFunc(a.Pairs[i], a.Pairs[j]) }

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

// SortKeys Sort the map keys using your sort func
func (o *OrderedMap) SortKeys(sortFunc func(keys []string)) {
	sortFunc(o.keys)
}

// Sort Sort the map using your sort func
func (o *OrderedMap) Sort(lessFunc func(a *Pair, b *Pair) bool) {
	pairs := make([]*Pair, len(o.keys))
	for i, key := range o.keys {
		pairs[i] = &Pair{key, o.values[key]}
	}

	sort.Sort(ByPair{pairs, lessFunc})

	for i, pair := range pairs {
		o.keys[i] = pair.key
	}
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

func (o *OrderedMap) UnmarshalYAML(b []byte) error {
	m := map[string]interface{}{}
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	s := string(b)
	mapToOrderedMap(o, s, m)
	return nil
}

func findClosingBraces(str string, left byte, right byte) int {
	mark := 1
	isLiteral := false
	i := 1

	for ; i < len(str); i++ {
		if str[i] == '\\' {
			// consume the next symbol
			i++
		} else if str[i] == '"' {
			isLiteral = !isLiteral
		} else if !isLiteral {
			if str[i] == left {
				mark++
			} else if str[i] == right {
				mark--
			}
		}
		if mark == 0 {
			break
		}
	}
	return i
}

func parseSliceInMap(o *OrderedMap, str string, content []interface{}) {
	for i, item := range content {
		switch itemTyped := item.(type) {
		case map[string]interface{}: // map
			oo := OrderedMap{}
			str = str[strings.IndexByte(str, '{'):]
			idx := findClosingBraces(str, '{', '}') + 1
			mapToOrderedMap(&oo, str[:idx], itemTyped)
			content[i] = oo
			str = str[idx:]
		case []interface{}: // slice
			str = str[strings.IndexByte(str, '['):]
			idx := findClosingBraces(str, '[', ']') + 1
			parseSliceInMap(o, str[:idx], itemTyped)
			str = str[idx:]
		default: // scalar
			itemStr := fmt.Sprint(itemTyped)
			itemIdx := strings.Index(str, itemStr)
			str = str[itemIdx+len(itemStr)+1:]
		}
	}
}

func mapToOrderedMap(o *OrderedMap, s string, m map[string]interface{}) {
	orderedKeys := []KeyIndex{}
	genericMap := map[string]interface{}{}

	// get all the keys sorted out first
	for k, _ := range m {
		kEscaped := strings.Replace(k, `"`, `\"`, -1)
		kQuoted := `"` + kEscaped + `"`
		sTrimmed := s
		for len(sTrimmed) > 0 {
			lastIndex := strings.LastIndex(sTrimmed, kQuoted)
			if lastIndex == -1 {
				break
			}
			sTrimmed = sTrimmed[0:lastIndex]
			sTrimmed = strings.TrimRight(sTrimmed, ", \n\r\t")
			maybeValidJson := sTrimmed + "}"

			// If we can successfully unmarshal the previous part, it means the match is a top-level key
			err := json.Unmarshal([]byte(maybeValidJson), &genericMap)
			if err == nil {
				// record the position of this key in s
				ki := KeyIndex{
					Key:   k,
					Index: lastIndex,
				}
				orderedKeys = append(orderedKeys, ki)
				break
			}
		}
	}
	orderedKeys = append(orderedKeys, KeyIndex{Key: "", Index: len(s) - 1})
	sort.Sort(ByIndex(orderedKeys))

	for i := 0; i < len(orderedKeys)-1; i++ {
		contentKey := orderedKeys[i].Key
		contentKeyEscaped := `"` + strings.Replace(contentKey, `"`, `\"`, -1) + `"`
		contentEnd := orderedKeys[i+1].Index
		contentStart := orderedKeys[i].Index + len(contentKeyEscaped)
		contentStr := strings.Trim(s[contentStart:contentEnd], " \n\r:,")

		switch contentTyped := m[contentKey].(type) {
		case map[string]interface{}:
			oo := OrderedMap{}
			mapToOrderedMap(&oo, contentStr, contentTyped)
			m[contentKey] = oo
		case []interface{}:
			parseSliceInMap(o, contentStr, contentTyped)
		}

	}

	k := []string{}
	for _, ki := range orderedKeys {
		if ki.Key != "" {
			k = append(k, ki.Key)
		}
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
