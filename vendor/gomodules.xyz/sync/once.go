// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sync

import (
	gsync "sync"
	"sync/atomic"
)

// Once is an object that will perform exactly one action.
//
// A Once must not be copied after first use.
type Once struct {
	// done indicates whether the action has been performed.
	// It is first in the struct because it is used in the hot path.
	// The hot path is inlined at every call site.
	// Placing done first allows more compact instructions on some architectures (amd64/x86),
	// and fewer instructions (to calculate offset) on other architectures.
	done int32
	m    gsync.Mutex
}

// Do calls the function f if and only if Do is being called for the
// first time successfully for this instance of Once. In other words, given
// 	var once Once
// if once.Do(f) is called multiple times, f will be invoked until first successful execution.
// A new instance of Once is required for each function to execute.
//
// Do is intended for initialization that must be run exactly once successfully.
//
// Because no call to Do returns until the one call to f returns, if f causes
// Do to be called, it will deadlock.
//
// If f panics, Do considers it to have returned successfully; future calls of Do return
// without calling f.
//
// This is an adaptation from https://golang.org/pkg/sync/#Once
//
func (o *Once) Do(f func() error) {
	// Note: Here is an incorrect implementation of Do:
	//
	//	if atomic.CompareAndSwapInt32(&o.done, 0, 1) {
	//		f()
	//	}
	//
	// Do guarantees that when it returns, f has finished.
	// This implementation would not implement that guarantee:
	// given two simultaneous calls, the winner of the cas would
	// call f, and the second would return immediately, without
	// waiting for the first's call to f to complete.
	// This is why the slow path falls back to a mutex, and why
	// the atomic.StoreInt32 must be delayed until after f returns.

	if atomic.LoadInt32(&o.done) == 0 {
		// Outlined slow-path to allow inlining of the fast-path.
		o.doSlow(f)
	}
}

func (o *Once) doSlow(f func() error) {
	o.m.Lock()
	defer o.m.Unlock()
	if o.done == 0 {
		var err error
		defer func() {
			if err == nil {
				atomic.StoreInt32(&o.done, 1)
			}
		}()
		err = f()
	}
}
