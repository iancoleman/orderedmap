package orderedmap

import (
	"bytes"
	"encoding/json"
	"sort"
)

type Pair struct {
	key   string
	value interface{}
}

func (kv *Pair) Key() string {
	return kv.key
}

func (kv *Pair) Value() interface{} {
	return kv.value
}

type ByPair struct {
	Pairs    []*Pair
	LessFunc func(a *Pair, j *Pair) bool
}

func (a ByPair) Len() int           { return len(a.Pairs) }
func (a ByPair) Swap(i, j int)      { a.Pairs[i], a.Pairs[j] = a.Pairs[j], a.Pairs[i] }
func (a ByPair) Less(i, j int) bool { return a.LessFunc(a.Pairs[i], a.Pairs[j]) }

type OrderedMapImpl struct {
	keys       []string
	values     map[string]interface{}
	escapeHTML bool
}

type OrderedMap interface {
	SetEscapeHTML(on bool)
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
	Delete(key string)
	Keys() []string
	SetKeys(keys []string)
	Values() map[string]interface{}
	InitValues()
	SortKeys(sortFunc func(keys []string))
	Sort(lessFunc func(a *Pair, b *Pair) bool)
	UnmarshalJSON(b []byte) error
	MarshalJSON() ([]byte, error)
	Clone(v ...map[string]interface{}) OrderedMap
}

func New() OrderedMap {
	return &OrderedMapImpl{
		keys:       make([]string, 0, 1),
		values:     make(map[string]interface{}, 1),
		escapeHTML: true,
	}
}

func (o *OrderedMapImpl) Clone(oms ...map[string]interface{}) OrderedMap {
	var om map[string]interface{}
	if len(oms) > 0 {
		om = oms[0]
	} else {
		om = make(map[string]interface{})
	}
	return &OrderedMapImpl{
		keys:       make([]string, 0, len(om)),
		values:     om,
		escapeHTML: o.escapeHTML,
	}
}

func (o *OrderedMapImpl) SetEscapeHTML(on bool) {
	o.escapeHTML = on
}

func (o *OrderedMapImpl) Get(key string) (interface{}, bool) {
	val, exists := o.values[key]
	return val, exists
}

func (o *OrderedMapImpl) Set(key string, value interface{}) {
	_, exists := o.values[key]
	if !exists {
		o.keys = append(o.keys, key)
	}
	o.values[key] = value
}

func (o *OrderedMapImpl) Delete(key string) {
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

func (o *OrderedMapImpl) Keys() []string {
	return o.keys
}

func (o *OrderedMapImpl) SetKeys(keys []string) {
	o.keys = keys
}

func (o *OrderedMapImpl) Values() map[string]interface{} {
	return o.values
}

func (o *OrderedMapImpl) InitValues() {
	if o.values == nil {
		o.values = make(map[string]interface{})
	}
}

// SortKeys Sort the map keys using your sort func
func (o *OrderedMapImpl) SortKeys(sortFunc func(keys []string)) {
	sortFunc(o.keys)
}

// Sort Sort the map using your sort func
func (o *OrderedMapImpl) Sort(lessFunc func(a *Pair, b *Pair) bool) {
	pairs := make([]*Pair, len(o.keys))
	for i, key := range o.keys {
		pairs[i] = &Pair{key, o.values[key]}
	}

	sort.Sort(ByPair{pairs, lessFunc})

	for i, pair := range pairs {
		o.keys[i] = pair.key
	}
}

func BoundUnmarshalJSON(o OrderedMap, b []byte) error {
	o.InitValues()
	val := o.Values()
	err := json.Unmarshal(b, &val)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	if _, err = dec.Token(); err != nil { // skip '{'
		return err
	}
	// o.SetKeys(make([]string, 0, len(o.Values())))
	return decodeOrderedMap(dec, o)
}

func (o *OrderedMapImpl) UnmarshalJSON(b []byte) error {
	return BoundUnmarshalJSON(o, b)
}

func decodeOrderedMap(dec *json.Decoder, o OrderedMap) error {
	hasKey := make(map[string]bool, len(o.Values()))
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
			for j, k := range o.Keys() {
				if k == key {
					copy(o.Keys()[j:], o.Keys()[j+1:])
					break
				}
			}
			o.Keys()[len(o.Keys())-1] = key
		} else {
			hasKey[key] = true
			o.SetKeys(append(o.Keys(), key))
		}

		token, err = dec.Token()
		if err != nil {
			return err
		}
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{':
				if values, ok := o.Values()[key].(map[string]interface{}); ok {
					newMap := o.Clone(values)
					if err = decodeOrderedMap(dec, newMap); err != nil {
						return err
					}
					o.Values()[key] = newMap
				} else if oldMap, ok := o.Values()[key].(OrderedMap); ok {
					newMap := o.Clone(oldMap.Values())
					if err = decodeOrderedMap(dec, newMap); err != nil {
						return err
					}
					o.Values()[key] = newMap
				} else if err = decodeOrderedMap(dec, o.Clone()); err != nil {
					return err
				}
			case '[':
				if values, ok := o.Values()[key].([]interface{}); ok {
					if err = decodeSlice(dec, values, o); err != nil {
						return err
					}
				} else if err = decodeSlice(dec, []interface{}{}, o); err != nil {
					return err
				}
			}
		}
	}
}

func decodeSlice(dec *json.Decoder, s []interface{}, o OrderedMap) error {
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
						newMap := o.Clone(values)
						if err = decodeOrderedMap(dec, newMap); err != nil {
							return err
						}
						s[index] = newMap
					} else if oldMap, ok := s[index].(OrderedMap); ok {
						newMap := o.Clone(oldMap.Values())
						if err = decodeOrderedMap(dec, newMap); err != nil {
							return err
						}
						s[index] = newMap
					} else if err = decodeOrderedMap(dec, o.Clone()); err != nil {
						return err
					}
				} else if err = decodeOrderedMap(dec, o.Clone()); err != nil {
					return err
				}
			case '[':
				if index < len(s) {
					if values, ok := s[index].([]interface{}); ok {
						if err = decodeSlice(dec, values, o); err != nil {
							return err
						}
					} else if err = decodeSlice(dec, []interface{}{}, o); err != nil {
						return err
					}
				} else if err = decodeSlice(dec, []interface{}{}, o); err != nil {
					return err
				}
			case ']':
				return nil
			}
		}
	}
}

func (o OrderedMapImpl) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(o.escapeHTML)
	for i, k := range o.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		// add key
		if err := encoder.Encode(k); err != nil {
			return nil, err
		}
		buf.WriteByte(':')
		// add value
		if err := encoder.Encode(o.values[k]); err != nil {
			return nil, err
		}
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}
