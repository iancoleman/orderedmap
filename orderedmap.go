package orderedmap

import (
	"bytes"
	"encoding/json"
	"fmt"
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

type OrderedMap struct {
	keys              []string
	values            map[string]interface{}
	escapeHTML        bool
	jsonLastValueWins bool // Permits duplicates on JSON unmarshal.  Otherwise errors on duplicates on unmarshal.
}

func New() *OrderedMap {
	o := OrderedMap{}
	o.keys = []string{}
	o.values = map[string]interface{}{}
	o.escapeHTML = true
	return &o
}

func (o *OrderedMap) SetEscapeHTML(on bool) {
	o.escapeHTML = on
}

func (o *OrderedMap) SetJSONLastValueWins(on bool) {
	o.jsonLastValueWins = on
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

func (o *OrderedMap) Values() map[string]interface{} {
	return o.values
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
	var err error
	if !o.jsonLastValueWins {
		err = CheckDuplicate(json.NewDecoder(bytes.NewReader(b)))
		if err != nil {
			return err
		}
	}
	if o.values == nil {
		o.values = map[string]interface{}{}
	}
	err = json.Unmarshal(b, &o.values)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	if _, err = dec.Token(); err != nil { // skip '{'
		return err
	}
	o.keys = make([]string, 0, len(o.values))
	return decodeOrderedMap(dec, o)
}

func decodeOrderedMap(dec *json.Decoder, o *OrderedMap) error {
	hasKey := make(map[string]bool, len(o.values))
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
			for j, k := range o.keys {
				if k == key {
					copy(o.keys[j:], o.keys[j+1:])
					break
				}
			}
			o.keys[len(o.keys)-1] = key
		} else {
			hasKey[key] = true
			o.keys = append(o.keys, key)
		}

		token, err = dec.Token()
		if err != nil {
			return err
		}
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{':
				if values, ok := o.values[key].(map[string]interface{}); ok {
					newMap := OrderedMap{
						keys:       make([]string, 0, len(values)),
						values:     values,
						escapeHTML: o.escapeHTML,
					}
					if err = decodeOrderedMap(dec, &newMap); err != nil {
						return err
					}
					o.values[key] = newMap
				} else if oldMap, ok := o.values[key].(OrderedMap); ok {
					newMap := OrderedMap{
						keys:       make([]string, 0, len(oldMap.values)),
						values:     oldMap.values,
						escapeHTML: o.escapeHTML,
					}
					if err = decodeOrderedMap(dec, &newMap); err != nil {
						return err
					}
					o.values[key] = newMap
				} else if err = decodeOrderedMap(dec, &OrderedMap{}); err != nil {
					return err
				}
			case '[':
				if values, ok := o.values[key].([]interface{}); ok {
					if err = decodeSlice(dec, values, o.escapeHTML); err != nil {
						return err
					}
				} else if err = decodeSlice(dec, []interface{}{}, o.escapeHTML); err != nil {
					return err
				}
			}
		}
	}
}

func decodeSlice(dec *json.Decoder, s []interface{}, escapeHTML bool) error {
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
						newMap := OrderedMap{
							keys:       make([]string, 0, len(values)),
							values:     values,
							escapeHTML: escapeHTML,
						}
						if err = decodeOrderedMap(dec, &newMap); err != nil {
							return err
						}
						s[index] = newMap
					} else if oldMap, ok := s[index].(OrderedMap); ok {
						newMap := OrderedMap{
							keys:       make([]string, 0, len(oldMap.values)),
							values:     oldMap.values,
							escapeHTML: escapeHTML,
						}
						if err = decodeOrderedMap(dec, &newMap); err != nil {
							return err
						}
						s[index] = newMap
					} else if err = decodeOrderedMap(dec, &OrderedMap{}); err != nil {
						return err
					}
				} else if err = decodeOrderedMap(dec, &OrderedMap{}); err != nil {
					return err
				}
			case '[':
				if index < len(s) {
					if values, ok := s[index].([]interface{}); ok {
						if err = decodeSlice(dec, values, escapeHTML); err != nil {
							return err
						}
					} else if err = decodeSlice(dec, []interface{}{}, escapeHTML); err != nil {
						return err
					}
				} else if err = decodeSlice(dec, []interface{}{}, escapeHTML); err != nil {
					return err
				}
			case ']':
				return nil
			}
		}
	}
}

func (o OrderedMap) MarshalJSON() ([]byte, error) {
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

// ErrJSONDuplicate allows applications to check for JSON duplicate error.
type ErrJSONDuplicate error

// CheckDuplicate checks for JSON duplicates on ingest (unmarshal).  Note that
// Go maps and structs and Javascript objects (ES6) already require unique
// JSON names.
//
// Duplicate JSON fields are a security issue that wasn't addressed by the
// original spec that results in surprising behavior and is a source of bugs.
// See the article, "[An Exploration of JSON Interoperability
// Vulnerabilities](https://bishopfox.com/blog/json-interoperability-vulnerabilities)"
// and control-f "duplicate".
//
// Until Go releases the planned revision to the JSON package (See
// https://github.com/go-json-experiment/json), or adds support for erroring on
// duplicates to the current package, this function is needed.
//
// After JSON was widely adopted, Douglas Crockford (JSON's inventor), tried to
// fix this by updating JSON to define "must error on duplicates" as the correct
// behavior,  but it was decided it was too late
// (https://esdiscuss.org/topic/json-duplicate-keys).
//
// Although Douglas Crockford couldn't change the JSON spec to force
// implementations to error on duplicate, his Java JSON implementation errors on
// duplicates. Others implementations behaviors are `last-value-wins`, support
// duplicate keys, or other non-standard behavior. The [JSON
// RFC](https://datatracker.ietf.org/doc/html/rfc8259#section-4) states that
// implementations should not allow duplicate keys.  It then notes the varying behavior
// of existing implementations.
//
// Disallowing duplicates conforms to the small I-JSON RFC. The author of
// I-JSON, Tim Bray, is also the author of current JSON specification (RFC
// 8259).  See also https://github.com/json5/json5-spec/issues/38.
func CheckDuplicate(d *json.Decoder) error {
	t, err := d.Token()
	if err != nil {
		return err
	}

	delim, ok := t.(json.Delim) // Is it a delimiter?
	if !ok {
		return nil // scaler type, nothing to do
	}

	switch delim {
	case '{':
		keys := make(map[string]bool)
		for d.More() {
			t, err := d.Token() // Get field key.
			if err != nil {
				return err
			}

			key := t.(string)
			if keys[key] { // Check for duplicates.
				return ErrJSONDuplicate(fmt.Errorf("Coze: JSON duplicate field %q", key))
			}
			keys[key] = true
			err = CheckDuplicate(d) // Recursive, Check value in case value is object.
			if err != nil {
				return err
			}
		}
		if _, err := d.Token(); err != nil { // consume trailing }
			return err
		}

	case '[':
		for d.More() {
			if err := CheckDuplicate(d); err != nil {
				return err
			}
		}
		// consume trailing ]
		if _, err := d.Token(); err != nil {
			return err
		}
	}
	return nil
}
