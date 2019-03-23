package schedluer

import (
	"sync"
	"sync/atomic"
)

type CurrentMap struct {
	maps   sync.Map
	length uint64
}

func (c *CurrentMap) Put(key interface{}, value interface{}) {
	c.maps.Store(key, value)
	atomic.AddUint64(&c.length, 1)
}

func (c *CurrentMap) Get(key interface{}) (interface{}, bool) {
	return c.maps.Load(key)
}

func (c *CurrentMap) Length() uint64 {
	return atomic.LoadUint64(&c.length)
}

func NewCurrentMap() CurrentMap{
	cu := CurrentMap{}
	atomic.StoreUint64(&cu.length,0)
	return cu
}