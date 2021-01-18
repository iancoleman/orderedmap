package orderedmap

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"
)

func TestOrderedMap(t *testing.T) {
	o := New()
	// number
	o.Set("number", 3)
	v, _ := o.Get("number")
	if v.(int) != 3 {
		t.Error("Set number")
	}
	// string
	o.Set("string", "x")
	v, _ = o.Get("string")
	if v.(string) != "x" {
		t.Error("Set string")
	}
	// string slice
	o.Set("strings", []string{
		"t",
		"u",
	})
	v, _ = o.Get("strings")
	if v.([]string)[0] != "t" {
		t.Error("Set strings first index")
	}
	if v.([]string)[1] != "u" {
		t.Error("Set strings second index")
	}
	// mixed slice
	o.Set("mixed", []interface{}{
		1,
		"1",
	})
	v, _ = o.Get("mixed")
	if v.([]interface{})[0].(int) != 1 {
		t.Error("Set mixed int")
	}
	if v.([]interface{})[1].(string) != "1" {
		t.Error("Set mixed string")
	}
	// overriding existing key
	o.Set("number", 4)
	v, _ = o.Get("number")
	if v.(int) != 4 {
		t.Error("Override existing key")
	}
	// Keys method
	keys := o.Keys()
	expectedKeys := []string{
		"number",
		"string",
		"strings",
		"mixed",
	}
	for i, key := range keys {
		if key != expectedKeys[i] {
			t.Error("Keys method", key, "!=", expectedKeys[i])
		}
	}
	for i, key := range expectedKeys {
		if key != expectedKeys[i] {
			t.Error("Keys method", key, "!=", expectedKeys[i])
		}
	}
	// delete
	o.Delete("strings")
	o.Delete("not a key being used")
	if len(o.Keys()) != 3 {
		t.Error("Delete method")
	}
	_, ok := o.Get("strings")
	if ok {
		t.Error("Delete did not remove 'strings' key")
	}
}

func TestBlankMarshalJSON(t *testing.T) {
	o := New()
	// blank map
	b, err := json.Marshal(o)
	if err != nil {
		t.Error("Marshalling blank map to json", err)
	}
	s := string(b)
	// check json is correctly ordered
	if s != `{}` {
		t.Error("JSON Marshaling blank map value is incorrect", s)
	}
	// convert to indented json
	bi, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		t.Error("Marshalling indented json for blank map", err)
	}
	si := string(bi)
	ei := `{}`
	if si != ei {
		fmt.Println(ei)
		fmt.Println(si)
		t.Error("JSON MarshalIndent blank map value is incorrect", si)
	}
}

func TestMarshalJSON(t *testing.T) {
	o := New()
	// number
	o.Set("number", 3)
	// string
	o.Set("string", "x")
	// string
	o.Set("specialstring", "\\.<>[]{}_-")
	// new value keeps key in old position
	o.Set("number", 4)
	// keys not sorted alphabetically
	o.Set("z", 1)
	o.Set("a", 2)
	o.Set("b", 3)
	// slice
	o.Set("slice", []interface{}{
		"1",
		1,
	})
	// orderedmap
	v := New()
	v.Set("e", 1)
	v.Set("a", 2)
	o.Set("orderedmap", v)
	// escape key
	o.Set("test\n\r\t\\\"ing", 9)
	// convert to json
	b, err := json.Marshal(o)
	if err != nil {
		t.Error("Marshalling json", err)
	}
	s := string(b)
	// check json is correctly ordered
	if s != `{"number":4,"string":"x","specialstring":"\\.\u003c\u003e[]{}_-","z":1,"a":2,"b":3,"slice":["1",1],"orderedmap":{"e":1,"a":2},"test\n\r\t\\\"ing":9}` {
		t.Error("JSON Marshal value is incorrect", s)
	}
	// convert to indented json
	bi, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		t.Error("Marshalling indented json", err)
	}
	si := string(bi)
	ei := `{
  "number": 4,
  "string": "x",
  "specialstring": "\\.\u003c\u003e[]{}_-",
  "z": 1,
  "a": 2,
  "b": 3,
  "slice": [
    "1",
    1
  ],
  "orderedmap": {
    "e": 1,
    "a": 2
  },
  "test\n\r\t\\\"ing": 9
}`
	if si != ei {
		fmt.Println(ei)
		fmt.Println(si)
		t.Error("JSON MarshalIndent value is incorrect", si)
	}
}

func TestMarshalJSONNoEscapeHTML(t *testing.T) {
	o := New()
	o.SetEscapeHTML(false)
	// string special characters
	o.Set("specialstring", "\\.<>[]{}_-")
	// convert to json
	b, err := o.MarshalJSON()
	if err != nil {
		t.Error("Marshalling json", err)
	}
	s := strings.Replace(string(b), "\n", "", -1)
	// check json is correctly ordered
	if s != `{"specialstring":"\\.<>[]{}_-"}` {
		t.Error("JSON Marshal value is incorrect", s)
	}
}

func TestMarshalJSONNoEscapeHTMLRecursive(t *testing.T) {
	src := `{"x":"<>","y":[{"z":["<>"]}]}`
	o := New()
	o.SetEscapeHTML(false)
	err := json.Unmarshal([]byte(src), &o)
	if err != nil {
		t.Error("JSON Unmarshal error with special chars", err)
	}
	b, err := o.MarshalJSON()
	if err != nil {
		t.Error("Marshalling json", err)
	}
	s := strings.Replace(string(b), "\n", "", -1)
	if s != src {
		t.Error("JSON Marshal value is incorrect", s)
	}
}

func TestUnmarshalJSON(t *testing.T) {
	s := `{
  "number": 4,
  "string": "x",
  "z": 1,
  "a": "should not break with unclosed { character in value",
  "b": 3,
  "slice": [
    "1",
    1
  ],
  "orderedmap": {
    "e": 1,
    "a { nested key with brace": "with a }}}} }} {{{ brace value",
	"after": {
		"link": "test {{{ with even deeper nested braces }"
	}
  },
  "test\"ing": 9,
  "after": 1,
  "multitype_array": [
    "test",
	1,
	{ "map": "obj", "it" : 5, ":colon in key": "colon: in value" },
	[{"inner": "map"}]
  ],
  "should not break with { character in key": 1
}`
	o := New()
	err := json.Unmarshal([]byte(s), &o)
	if err != nil {
		t.Error("JSON Unmarshal error", err)
	}
	// Check the root keys
	expectedKeys := []string{
		"number",
		"string",
		"z",
		"a",
		"b",
		"slice",
		"orderedmap",
		"test\"ing",
		"after",
		"multitype_array",
		"should not break with { character in key",
	}
	k := o.Keys()
	for i := range k {
		if k[i] != expectedKeys[i] {
			t.Error("Unmarshal root key order", i, k[i], "!=", expectedKeys[i])
		}
	}
	// Check nested maps are converted to orderedmaps
	// nested 1 level deep
	expectedKeys = []string{
		"e",
		"a { nested key with brace",
		"after",
	}
	vi, ok := o.Get("orderedmap")
	if !ok {
		t.Error("Missing key for nested map 1 deep")
	}
	v := vi.(OrderedMap)
	k = v.Keys()
	for i := range k {
		if k[i] != expectedKeys[i] {
			t.Error("Key order for nested map 1 deep ", i, k[i], "!=", expectedKeys[i])
		}
	}
	// nested 2 levels deep
	expectedKeys = []string{
		"link",
	}
	vi, ok = v.Get("after")
	if !ok {
		t.Error("Missing key for nested map 2 deep")
	}
	v = vi.(OrderedMap)
	k = v.Keys()
	for i := range k {
		if k[i] != expectedKeys[i] {
			t.Error("Key order for nested map 2 deep", i, k[i], "!=", expectedKeys[i])
		}
	}
	// multitype array
	expectedKeys = []string{
		"map",
		"it",
		":colon in key",
	}
	vislice, ok := o.Get("multitype_array")
	if !ok {
		t.Error("Missing key for multitype array")
	}
	vslice := vislice.([]interface{})
	vmap := vslice[2].(OrderedMap)
	k = vmap.Keys()
	for i := range k {
		if k[i] != expectedKeys[i] {
			t.Error("Key order for nested map 2 deep", i, k[i], "!=", expectedKeys[i])
		}
	}
	// nested map 3 deep
	vislice, _ = o.Get("multitype_array")
	vslice = vislice.([]interface{})
	expectedKeys = []string{"inner"}
	vinnerslice := vslice[3].([]interface{})
	vinnermap := vinnerslice[0].(OrderedMap)
	k = vinnermap.Keys()
	for i := range k {
		if k[i] != expectedKeys[i] {
			t.Error("Key order for nested map 3 deep", i, k[i], "!=", expectedKeys[i])
		}
	}
}

func TestUnmarshalJSONDuplicateKeys(t *testing.T) {
	s := `{
		"a": [{}, []],
		"b": {"x":[1]},
		"c": "x",
		"d": {"x":1},
		"b": [{"x":[]}],
		"c": 1,
		"d": {"y": 2},
		"e": [{"x":1}],
		"e": [[]],
		"e": [{"z":2}],
		"a": {},
		"b": [[1]]
	}`
	o := New()
	err := json.Unmarshal([]byte(s), &o)
	if err != nil {
		t.Error("JSON Unmarshal error with special chars", err)
	}
	expectedKeys := []string{
		"c",
		"d",
		"e",
		"a",
		"b",
	}
	keys := o.Keys()
	if len(keys) != len(expectedKeys) {
		t.Error("Unmarshal key count", len(keys), "!=", len(expectedKeys))
	}
	for i, key := range keys {
		if key != expectedKeys[i] {
			t.Errorf("Unmarshal root key order: %d, %q != %q", i, key, expectedKeys[i])
		}
	}
	vimap, _ := o.Get("a")
	_ = vimap.(OrderedMap)
	vislice, _ := o.Get("b")
	_ = vislice.([]interface{})
	vival, _ := o.Get("c")
	_ = vival.(float64)

	vimap, _ = o.Get("d")
	m := vimap.(OrderedMap)
	expectedKeys = []string{"y"}
	keys = m.Keys()
	if len(keys) != len(expectedKeys) {
		t.Error("Unmarshal key count", len(keys), "!=", len(expectedKeys))
	}
	for i, key := range keys {
		if key != expectedKeys[i] {
			t.Errorf("Unmarshal key order: %d, %q != %q", i, key, expectedKeys[i])
		}
	}

	vislice, _ = o.Get("e")
	m = vislice.([]interface{})[0].(OrderedMap)
	expectedKeys = []string{"z"}
	keys = m.Keys()
	if len(keys) != len(expectedKeys) {
		t.Error("Unmarshal key count", len(keys), "!=", len(expectedKeys))
	}
	for i, key := range keys {
		if key != expectedKeys[i] {
			t.Errorf("Unmarshal key order: %d, %q != %q", i, key, expectedKeys[i])
		}
	}
}

func TestUnmarshalJSONSpecialChars(t *testing.T) {
	s := `{ " \u0041\n\r\t\\\\\\\\\\\\ "  : { "\\\\\\" : "\\\\\"\\" }, "\\":  " \\\\ test ", "\n": "\r" }`
	o := New()
	err := json.Unmarshal([]byte(s), &o)
	if err != nil {
		t.Error("JSON Unmarshal error with special chars", err)
	}
	expectedKeys := []string{
		" \u0041\n\r\t\\\\\\\\\\\\ ",
		"\\",
		"\n",
	}
	keys := o.Keys()
	if len(keys) != len(expectedKeys) {
		t.Error("Unmarshal key count", len(keys), "!=", len(expectedKeys))
	}
	for i, key := range keys {
		if key != expectedKeys[i] {
			t.Errorf("Unmarshal root key order: %d, %q != %q", i, key, expectedKeys[i])
		}
	}
}

func TestUnmarshalJSONArrayOfMaps(t *testing.T) {
	s := `
{
  "name": "test",
  "percent": 6,
  "breakdown": [
    {
      "name": "a",
      "percent": 0.9
    },
    {
      "name": "b",
      "percent": 0.9
    },
    {
      "name": "d",
      "percent": 0.4
    },
    {
      "name": "e",
      "percent": 2.7
    }
  ]
}
`
	o := New()
	err := json.Unmarshal([]byte(s), &o)
	if err != nil {
		t.Error("JSON Unmarshal error", err)
	}
	// Check the root keys
	expectedKeys := []string{
		"name",
		"percent",
		"breakdown",
	}
	k := o.Keys()
	for i := range k {
		if k[i] != expectedKeys[i] {
			t.Error("Unmarshal root key order", i, k[i], "!=", expectedKeys[i])
		}
	}
	// Check nested maps are converted to orderedmaps
	// nested 1 level deep
	expectedKeys = []string{
		"name",
		"percent",
	}
	vi, ok := o.Get("breakdown")
	if !ok {
		t.Error("Missing key for nested map 1 deep")
	}
	vs := vi.([]interface{})
	for _, vInterface := range vs {
		v := vInterface.(OrderedMap)
		k = v.Keys()
		for i := range k {
			if k[i] != expectedKeys[i] {
				t.Error("Key order for nested map 1 deep ", i, k[i], "!=", expectedKeys[i])
			}
		}
	}
}

func TestUnmarshalJSONStruct(t *testing.T) {
	var v struct {
		Data *OrderedMap `json:"data"`
	}

	err := json.Unmarshal([]byte(`{ "data": { "x": 1 } }`), &v)
	if err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}

	x, ok := v.Data.Get("x")
	if !ok {
		t.Errorf("missing expected key")
	} else if x != float64(1) {
		t.Errorf("unexpected value: %#v", x)
	}
}

func TestOrderedMap_SortKeys(t *testing.T) {
	s := `
{
  "b": 2,
  "a": 1,
  "c": 3
}
`
	o := New()
	json.Unmarshal([]byte(s), &o)

	o.SortKeys(sort.Strings)

	// Check the root keys
	expectedKeys := []string{
		"a",
		"b",
		"c",
	}
	k := o.Keys()
	for i := range k {
		if k[i] != expectedKeys[i] {
			t.Error("SortKeys root key order", i, k[i], "!=", expectedKeys[i])
		}
	}
}

func TestOrderedMap_Sort(t *testing.T) {
	s := `
{
  "b": 2,
  "a": 1,
  "c": 3
}
`
	o := New()
	json.Unmarshal([]byte(s), &o)
	o.Sort(func(a *Pair, b *Pair) bool {
		return a.value.(float64) > b.value.(float64)
	})

	// Check the root keys
	expectedKeys := []string{
		"c",
		"b",
		"a",
	}
	k := o.Keys()
	for i := range k {
		if k[i] != expectedKeys[i] {
			t.Error("Sort root key order", i, k[i], "!=", expectedKeys[i])
		}
	}
}

// https://github.com/iancoleman/orderedmap/issues/11
func TestOrderedMap_empty_array(t *testing.T) {
	srcStr := `{"x":[]}`
	src := []byte(srcStr)
	om := New()
	json.Unmarshal(src, om)
	bs, _ := json.Marshal(om)
	marshalledStr := string(bs)
	if marshalledStr != srcStr {
		t.Error("Empty array does not serialise to json correctly")
		t.Error("Expect", srcStr)
		t.Error("Got", marshalledStr)
	}
}

// Inspired by
// https://github.com/iancoleman/orderedmap/issues/11
// but using empty maps instead of empty slices
func TestOrderedMap_empty_map(t *testing.T) {
	srcStr := `{"x":{}}`
	src := []byte(srcStr)
	om := New()
	json.Unmarshal(src, om)
	bs, _ := json.Marshal(om)
	marshalledStr := string(bs)
	if marshalledStr != srcStr {
		t.Error("Empty map does not serialise to json correctly")
		t.Error("Expect", srcStr)
		t.Error("Got", marshalledStr)
	}
}
