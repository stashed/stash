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

package client

import (
	"context"
	"errors"
	"io"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type retryClient struct {
	d        client.Client
	interval time.Duration
	timeout  time.Duration
}

var _ client.Client = &retryClient{}

func NewRetryClient(d client.Client) client.Client {
	return &retryClient{d: d, interval: 500 * time.Millisecond, timeout: 5 * time.Minute}
}

func NewRetryClientWithOptions(d client.Client, interval time.Duration, timeout time.Duration) client.Client {
	return &retryClient{d: d, interval: interval, timeout: timeout}
}

func (r *retryClient) Scheme() *runtime.Scheme {
	return r.d.Scheme()
}

func (r *retryClient) RESTMapper() meta.RESTMapper {
	return r.d.RESTMapper()
}

func (r *retryClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	return r.d.GroupVersionKindFor(obj)
}

func (r *retryClient) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	return r.d.IsObjectNamespaced(obj)
}

func (r *retryClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) (apierror error) {
	_ = wait.PollUntilContextTimeout(ctx, r.interval, r.timeout, true, func(ctx context.Context) (done bool, err error) {
		apierror = r.d.Get(ctx, key, obj, opts...)
		err = apierror
		done = err == nil || !errors.Is(err, io.EOF)
		return
	})
	return
}

func (r *retryClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) (apierror error) {
	_ = wait.PollUntilContextTimeout(ctx, r.interval, r.timeout, true, func(ctx context.Context) (done bool, err error) {
		apierror = r.d.List(ctx, list, opts...)
		err = apierror
		done = err == nil || !errors.Is(err, io.EOF)
		return
	})
	return
}

func (r *retryClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) (apierror error) {
	_ = wait.PollUntilContextTimeout(ctx, r.interval, r.timeout, true, func(ctx context.Context) (done bool, err error) {
		apierror = r.d.Create(ctx, obj, opts...)
		err = apierror
		done = err == nil || !errors.Is(err, io.EOF)
		return
	})
	return
}

func (r *retryClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) (apierror error) {
	_ = wait.PollUntilContextTimeout(ctx, r.interval, r.timeout, true, func(ctx context.Context) (done bool, err error) {
		apierror = r.d.Delete(ctx, obj, opts...)
		err = apierror
		done = err == nil || !errors.Is(err, io.EOF)
		return
	})
	return
}

func (r *retryClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) (apierror error) {
	_ = wait.PollUntilContextTimeout(ctx, r.interval, r.timeout, true, func(ctx context.Context) (done bool, err error) {
		apierror = r.d.Update(ctx, obj, opts...)
		err = apierror
		done = err == nil || !errors.Is(err, io.EOF)
		return
	})
	return
}

func (r *retryClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) (apierror error) {
	_ = wait.PollUntilContextTimeout(ctx, r.interval, r.timeout, true, func(ctx context.Context) (done bool, err error) {
		apierror = r.d.Patch(ctx, obj, patch, opts...)
		err = apierror
		done = err == nil || !errors.Is(err, io.EOF)
		return
	})
	return
}

func (r *retryClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) (apierror error) {
	_ = wait.PollUntilContextTimeout(ctx, r.interval, r.timeout, true, func(ctx context.Context) (done bool, err error) {
		apierror = r.d.DeleteAllOf(ctx, obj, opts...)
		err = apierror
		done = err == nil || !errors.Is(err, io.EOF)
		return
	})
	return
}

func (r *retryClient) Status() client.SubResourceWriter {
	return &retrySubResourceWriter{
		d:        r.d.Status(),
		interval: r.interval,
		timeout:  r.timeout,
	}
}

func (r *retryClient) SubResource(subResource string) client.SubResourceClient {
	return &retrySubResourceClient{
		d:        r.d.SubResource(subResource),
		interval: r.interval,
		timeout:  r.timeout,
	}
}

type retrySubResourceWriter struct {
	d        client.SubResourceWriter
	interval time.Duration
	timeout  time.Duration
}

var _ client.SubResourceWriter = &retrySubResourceWriter{}

func (r *retrySubResourceWriter) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) (apierror error) {
	_ = wait.PollUntilContextTimeout(ctx, r.interval, r.timeout, true, func(ctx context.Context) (done bool, err error) {
		apierror = r.d.Create(ctx, obj, subResource, opts...)
		err = apierror
		done = err == nil || !errors.Is(err, io.EOF)
		return
	})
	return
}

func (r *retrySubResourceWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) (apierror error) {
	_ = wait.PollUntilContextTimeout(ctx, r.interval, r.timeout, true, func(ctx context.Context) (done bool, err error) {
		apierror = r.d.Update(ctx, obj, opts...)
		err = apierror
		done = err == nil || !errors.Is(err, io.EOF)
		return
	})
	return
}

func (r *retrySubResourceWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) (apierror error) {
	_ = wait.PollUntilContextTimeout(ctx, r.interval, r.timeout, true, func(ctx context.Context) (done bool, err error) {
		apierror = r.d.Patch(ctx, obj, patch, opts...)
		err = apierror
		done = err == nil || !errors.Is(err, io.EOF)
		return
	})
	return
}

type retrySubResourceClient struct {
	d        client.SubResourceClient
	interval time.Duration
	timeout  time.Duration
}

var _ client.SubResourceClient = &retrySubResourceClient{}

func (r *retrySubResourceClient) Get(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceGetOption) (apierror error) {
	_ = wait.PollUntilContextTimeout(ctx, r.interval, r.timeout, true, func(ctx context.Context) (done bool, err error) {
		apierror = r.d.Get(ctx, obj, subResource, opts...)
		err = apierror
		done = err == nil || !errors.Is(err, io.EOF)
		return
	})
	return
}

func (r *retrySubResourceClient) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) (apierror error) {
	_ = wait.PollUntilContextTimeout(ctx, r.interval, r.timeout, true, func(ctx context.Context) (done bool, err error) {
		apierror = r.d.Create(ctx, obj, subResource, opts...)
		err = apierror
		done = err == nil || !errors.Is(err, io.EOF)
		return
	})
	return
}

func (r *retrySubResourceClient) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) (apierror error) {
	_ = wait.PollUntilContextTimeout(ctx, r.interval, r.timeout, true, func(ctx context.Context) (done bool, err error) {
		apierror = r.d.Update(ctx, obj, opts...)
		err = apierror
		done = err == nil || !errors.Is(err, io.EOF)
		return
	})
	return
}

func (r *retrySubResourceClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) (apierror error) {
	_ = wait.PollUntilContextTimeout(ctx, r.interval, r.timeout, true, func(ctx context.Context) (done bool, err error) {
		apierror = r.d.Patch(ctx, obj, patch, opts...)
		err = apierror
		done = err == nil || !errors.Is(err, io.EOF)
		return
	})
	return
}
