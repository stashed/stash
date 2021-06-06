/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package hub

import (
	"sync"

	lru "github.com/hashicorp/golang-lru"
)

type Pool struct {
	cache *lru.Cache // pool of registries
	m     sync.Mutex
	f     func() KV
}

const PoolSize = 1024 // This number should match the max number of concurrent clusters handled

func NewPool(kvFactory func() KV) (*Pool, error) {
	cache, err := lru.New(PoolSize)
	if err != nil {
		return nil, err
	}
	return &Pool{cache: cache, f: kvFactory}, nil
}

func MustPool(kvFactory func() KV) *Pool {
	pool, err := NewPool(kvFactory)
	if err != nil {
		panic(err)
	}
	return pool
}

func (p *Pool) GetRegistry(uid string, helm HelmVersion) *Registry {
	p.m.Lock()
	defer p.m.Unlock()

	val, found := p.cache.Get(uid)
	if found {
		return val.(*Registry)
	}
	r := NewRegistry(uid, helm, p.f())
	p.cache.Add(uid, r)
	return r
}
