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

package apiutil

import (
	"sync"
	"sync/atomic"

	"golang.org/x/time/rate"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

// Adapted from https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/client/apiutil/dynamicrestmapper.go

// dynamicCachable is a Cachable that dynamically discovers resource
// types at runtime.
type dynamicCachable struct {
	mu             sync.RWMutex // protects the following fields
	staticCachable Cachable
	limiter        *rate.Limiter
	newCachable    func() (Cachable, error)

	lazy bool
	// Used for lazy init.
	inited  uint32
	initMtx sync.Mutex
}

// DynamicCachableOption is a functional option on the dynamicCachable
type DynamicCachableOption func(*dynamicCachable) error

// WithLimiter sets the Cachable's underlying limiter to lim.
func WithLimiter(lim *rate.Limiter) DynamicCachableOption {
	return func(drm *dynamicCachable) error {
		drm.limiter = lim
		return nil
	}
}

// WithLazyDiscovery prevents the Cachable from discovering REST mappings
// until an API call is made.
var WithLazyDiscovery DynamicCachableOption = func(drm *dynamicCachable) error {
	drm.lazy = true
	return nil
}

// WithCustomCachable supports setting a custom Cachable refresher instead of
// the default method, which uses a discovery client.
//
// This exists mainly for testing, but can be useful if you need tighter control
// over how discovery is performed, which discovery endpoints are queried, etc.
func WithCustomCachable(newCachable func() (Cachable, error)) DynamicCachableOption {
	return func(drm *dynamicCachable) error {
		drm.newCachable = newCachable
		return nil
	}
}

// NewDynamicCachable returns a dynamic Cachable for cfg. The dynamic
// Cachable dynamically discovers resource types at runtime. opts
// configure the Cachable.
func NewDynamicCachable(cfg *rest.Config, opts ...DynamicCachableOption) (Cachable, error) {
	client, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}
	drm := &dynamicCachable{
		limiter: rate.NewLimiter(rate.Limit(defaultRefillRate), defaultLimitSize),
		newCachable: func() (Cachable, error) {
			return NewCachable(client)
		},
	}
	for _, opt := range opts {
		if err = opt(drm); err != nil {
			return nil, err
		}
	}
	if !drm.lazy {
		if err := drm.setStaticCachable(); err != nil {
			return nil, err
		}
	}
	return drm, nil
}

var (
	// defaultRefilRate is the default rate at which potential calls are
	// added back to the "bucket" of allowed calls.
	defaultRefillRate = 5
	// defaultLimitSize is the default starting/max number of potential calls
	// per second.  Once a call is used, it's added back to the bucket at a rate
	// of defaultRefillRate per second.
	defaultLimitSize = 5
)

// setStaticCachable sets drm's staticCachable by querying its client, regardless
// of reload backoff.
func (drm *dynamicCachable) setStaticCachable() error {
	newCachable, err := drm.newCachable()
	if err != nil {
		return err
	}
	drm.staticCachable = newCachable
	return nil
}

// init initializes drm only once if drm is lazy.
func (drm *dynamicCachable) init() (err error) {
	// skip init if drm is not lazy or has initialized
	if !drm.lazy || atomic.LoadUint32(&drm.inited) != 0 {
		return nil
	}

	drm.initMtx.Lock()
	defer drm.initMtx.Unlock()
	if drm.inited == 0 {
		if err = drm.setStaticCachable(); err == nil {
			atomic.StoreUint32(&drm.inited, 1)
		}
	}
	return err
}

// checkAndReload attempts to call the given callback, which is assumed to be dependent
// on the data in the restmapper.
//
// If the callback returns an error matching meta.IsNoMatchErr, it will attempt to reload
// the Cachable's data and re-call the callback once that's occurred.
// If the callback returns any other error, the function will return immediately regardless.
//
// It will take care of ensuring that reloads are rate-limited and that extraneous calls
// aren't made. If a reload would exceed the limiters rate, it returns the error return by
// the callback.
// It's thread-safe, and worries about thread-safety for the callback (so the callback does
// not need to attempt to lock the restmapper).
func (drm *dynamicCachable) checkAndReload(checkNeedsReload func() error) error {
	// first, check the common path -- data is fresh enough
	// (use an IIFE for the lock's defer)
	err := func() error {
		drm.mu.RLock()
		defer drm.mu.RUnlock()

		return checkNeedsReload()
	}()

	needsReload := meta.IsNoMatchError(err)
	if !needsReload {
		return err
	}

	// if the data wasn't fresh, we'll need to try and update it, so grab the lock...
	drm.mu.Lock()
	defer drm.mu.Unlock()

	// ... and double-check that we didn't reload in the meantime
	err = checkNeedsReload()
	needsReload = meta.IsNoMatchError(err)
	if !needsReload {
		return err
	}

	// we're still stale, so grab a rate-limit token if we can...
	if !drm.limiter.Allow() {
		// return error from static mapper here, we have refreshed often enough (exceeding rate of provided limiter)
		// so that client's can handle this the same way as a "normal" NoResourceMatchError / NoKindMatchError
		return err
	}

	// ...reload...
	if err := drm.setStaticCachable(); err != nil {
		return err
	}

	// ...and return the results of the closure regardless
	return checkNeedsReload()
}

func (drm *dynamicCachable) GVK(gvk schema.GroupVersionKind) (bool, error) {
	if err := drm.init(); err != nil {
		return false, err
	}
	var canCache bool
	err := drm.checkAndReload(func() error {
		var err error
		canCache, err = drm.staticCachable.GVK(gvk)
		return err
	})
	return canCache, err
}

func (drm *dynamicCachable) GVR(gvr schema.GroupVersionResource) (bool, error) {
	if err := drm.init(); err != nil {
		return false, err
	}
	var canCache bool
	err := drm.checkAndReload(func() error {
		var err error
		canCache, err = drm.staticCachable.GVR(gvr)
		return err
	})
	return canCache, err
}
