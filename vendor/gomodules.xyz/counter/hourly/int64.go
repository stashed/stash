package hourly

import "sync"

type Int64 struct {
	c int64
	m sync.RWMutex
}

func (i *Int64) Inc() int64 {
	i.m.Lock()
	defer i.m.Unlock()
	return Inc64(&i.c)
}

func (i *Int64) Get() int64 {
	i.m.RLock()
	defer i.m.RUnlock()
	return Get64(&i.c)
}
