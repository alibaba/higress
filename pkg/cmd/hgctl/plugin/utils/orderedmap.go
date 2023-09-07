package utils

import (
	"sort"

	"gomodules.xyz/orderedmap"
)

type OrderedMap struct {
	m       *orderedmap.OrderedMap
	ordered bool
}

func NewOrderedMap() *OrderedMap {
	return &OrderedMap{orderedmap.New(), false}
}

func (o *OrderedMap) Get(key string) (interface{}, bool) {
	return o.m.Get(key)
}

func (o *OrderedMap) Set(key string, value interface{}) {
	o.m.Set(key, value)
	o.ordered = false
}

func (o *OrderedMap) Keys() []string {
	if !o.ordered {
		o.Sort()
	}
	return o.m.Keys()
}

func (o *OrderedMap) Sort() {
	if o.ordered {
		return
	}
	o.m.SortKeys(sort.Strings)
	o.ordered = true
}

func (o *OrderedMap) MarshalJSON() ([]byte, error) {
	return o.m.MarshalJSON()
}

func (o *OrderedMap) UnmarshalJSON(data []byte) error {
	return o.m.UnmarshalJSON(data)
}

func (o *OrderedMap) MarshalYAML() (interface{}, error) {
	m := make(map[string]interface{}, len(o.Keys()))
	for _, k := range o.Keys() {
		if v, ok := o.Get(k); ok {
			m[k] = v
		}
	}

	return m, nil
}

func (o *OrderedMap) UnmarshalYAML(unmarshal func(interface{}) error) error {
	m := make(map[string]interface{})
	if err := unmarshal(&m); err != nil {
		return err
	}

	for k, v := range m {
		o.Set(k, v)
	}

	return nil
}
