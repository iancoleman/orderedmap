package orderedmap

type PairsIterator struct {
	m      *OrderedMap
	index  int
	length int
}

func (it *PairsIterator) Index() int {
	return it.index
}

func (it *PairsIterator) Length() int {
	return it.length
}

func (it *PairsIterator) Next() (*Pair, error) {
	index := it.index
	it.index++
	key := it.m.keys[index]
	return &Pair{key, it.m.values[key]}, nil
}

func (it *PairsIterator) Done() bool {
	return it.index >= it.length
}

func (it *PairsIterator) Close() error {
	return nil
}

type ValuesIterator struct {
	m      *OrderedMap
	index  int
	length int
}

func (it *ValuesIterator) Index() int {
	return it.index
}

func (it *ValuesIterator) Length() int {
	return it.length
}

func (it *ValuesIterator) Next() (interface{}, error) {
	index := it.index
	it.index++
	return it.m.values[it.m.keys[index]], nil
}

func (it *ValuesIterator) Done() bool {
	return it.index >= it.length
}

func (it *ValuesIterator) Close() error {
	return nil
}
