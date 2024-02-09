package hourly

import "sync"

type Int32 struct {
	c int32
	m sync.RWMutex
}

func (i *Int32) Inc() int32 {
	i.m.Lock()
	defer i.m.Unlock()
	return Inc32(&i.c)
}

func (i *Int32) Get() int32 {
	i.m.RLock()
	defer i.m.RUnlock()
	return Get32(&i.c)
}
