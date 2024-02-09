package hourly

import "sync"

type Int struct {
	c int
	m sync.RWMutex
}

func (i *Int) Inc() int {
	i.m.Lock()
	defer i.m.Unlock()
	return Inc(&i.c)
}

func (i *Int) Get() int {
	i.m.RLock()
	defer i.m.RUnlock()
	return Get(&i.c)
}
