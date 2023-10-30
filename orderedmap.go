package orderedmap

import (
	"bytes"
	"encoding/json"
	"sort"
)

type OrderedMap interface {
	SetEscapeHTML(on bool)
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
	Delete(key string)
	Keys() []string
	Values() map[string]interface{}
	SortKeys(sortFunc func(keys []string))
	Sort(lessFunc func(a *Pair, b *Pair) bool)
	UnmarshalJSON(b []byte) error
	MarshalJSON() ([]byte, error)
}

type orderedMapFrontend struct {
	OrderedMapCore
	escapeHTML bool
}

func New() OrderedMap {
	return &orderedMapFrontend{
		OrderedMapCore: NewCore(),
		escapeHTML:     true,
	}
}

func (o *orderedMapFrontend) clone(oms ...map[string]interface{}) *orderedMapFrontend {
	var om map[string]interface{}
	if len(oms) > 0 {
		om = oms[0]
	} else {
		om = make(map[string]interface{})
	}
	return &orderedMapFrontend{
		OrderedMapCore: NewCore(om),
		escapeHTML:     o.escapeHTML,
	}
}

func (o *orderedMapFrontend) SetEscapeHTML(on bool) {
	o.escapeHTML = on
}

func (o *orderedMapFrontend) Get(key string) (interface{}, bool) {
	return o.OrderedMapCore.CoreGetKey(key)
}

func (o *orderedMapFrontend) Set(key string, value interface{}) {
	o.OrderedMapCore.CoreSetKey(key, value)
}

func (o *orderedMapFrontend) Delete(key string) {
	o.OrderedMapCore.CoreDeleteKey(key)
}

func (o *orderedMapFrontend) Keys() []string {
	return o.OrderedMapCore.CoreKeys()
}

func (o *orderedMapFrontend) Values() map[string]interface{} {
	return o.OrderedMapCore.CoreValues()
}

// SortKeys Sort the map keys using your sort func
func (o *orderedMapFrontend) SortKeys(sortFunc func(keys []string)) {
	sortFunc(o.OrderedMapCore.CoreKeys())
}

// Sort Sort the map using your sort func
func (o *orderedMapFrontend) Sort(lessFunc func(a *Pair, b *Pair) bool) {
	keys := o.OrderedMapCore.CoreKeys()
	values := o.OrderedMapCore.CoreValues()
	pairs := make([]*Pair, len(keys))
	for i, key := range keys {
		pairs[i] = &Pair{key, values[key]}
	}
	sort.Sort(ByPair{pairs, lessFunc})
	for i, pair := range pairs {
		keys[i] = pair.key
	}
}

func (o *orderedMapFrontend) UnmarshalJSON(b []byte) error {
	if o.OrderedMapCore == nil {
		// not nice in place of a constructor
		out := New().(*orderedMapFrontend)
		o.OrderedMapCore = out.OrderedMapCore
		o.escapeHTML = out.escapeHTML
	}
	return BoundUnmarshalJSON(o.OrderedMapCore, func(oms ...map[string]interface{}) OrderedMapCore {
		clone := o.clone(oms...)
		return clone
	}, b)
}

func (o orderedMapFrontend) MarshalJSON() ([]byte, error) {
	buf := bytes.Buffer{}
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(o.escapeHTML)
	err := BoundMarshalJSON(o.OrderedMapCore, encoder, &buf)
	// very unlikely side effect: that buf.Bytes must retrieved after
	// BoundMarshalJSON returns
	return buf.Bytes(), err
}
