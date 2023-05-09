package interceptor

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type (

	// Funcs contains functions that are called instead of the underlying client's methods.
	Funcs struct {
		Get         func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error
		List        func(ctx context.Context, client client.WithWatch, list client.ObjectList, opts ...client.ListOption) error
		Create      func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error
		Delete      func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.DeleteOption) error
		DeleteAllOf func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.DeleteAllOfOption) error
		Update      func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error
		Patch       func(ctx context.Context, client client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error
		Watch       func(ctx context.Context, client client.WithWatch, obj client.ObjectList, opts ...client.ListOption) (watch.Interface, error)
		SubResource func(client client.WithWatch, subResource string) client.SubResourceClient
	}

	// SubResourceFuncs is a set of functions that can be used to intercept calls to a SubResourceClient.
	SubResourceFuncs struct {
		Get    func(ctx context.Context, client client.SubResourceClient, obj client.Object, subResource client.Object, opts ...client.SubResourceGetOption) error
		Create func(ctx context.Context, client client.SubResourceClient, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error
		Update func(ctx context.Context, client client.SubResourceClient, obj client.Object, opts ...client.SubResourceUpdateOption) error
		Patch  func(ctx context.Context, client client.SubResourceClient, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error
	}
)

// NewClient returns a new interceptor client that calls the functions in funcs instead of the underlying client's methods, if they are not nil.
func NewClient(interceptedClient client.WithWatch, funcs Funcs) client.WithWatch {
	return interceptor{client: interceptedClient, funcs: funcs}
}

// NewSubResourceClient returns a SubResourceClient that intercepts calls to the provided client with the provided functions.
func NewSubResourceClient(interceptedClient client.SubResourceClient, funcs SubResourceFuncs) client.SubResourceClient {
	return subResourceInterceptor{client: interceptedClient, funcs: funcs}
}

type interceptor struct {
	client client.WithWatch
	funcs  Funcs
}

var _ client.WithWatch = &interceptor{}

func (c interceptor) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	return c.client.GroupVersionKindFor(obj)
}

func (c interceptor) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	return c.client.IsObjectNamespaced(obj)
}

func (c interceptor) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if c.funcs.Get != nil {
		return c.funcs.Get(ctx, c.client, key, obj, opts...)
	}
	return c.client.Get(ctx, key, obj, opts...)
}

func (c interceptor) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	if c.funcs.List != nil {
		return c.funcs.List(ctx, c.client, list, opts...)
	}
	return c.client.List(ctx, list, opts...)
}

func (c interceptor) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if c.funcs.Create != nil {
		return c.funcs.Create(ctx, c.client, obj, opts...)
	}
	return c.client.Create(ctx, obj, opts...)
}

func (c interceptor) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	if c.funcs.Delete != nil {
		return c.funcs.Delete(ctx, c.client, obj, opts...)
	}
	return c.client.Delete(ctx, obj, opts...)
}

func (c interceptor) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if c.funcs.Update != nil {
		return c.funcs.Update(ctx, c.client, obj, opts...)
	}
	return c.client.Update(ctx, obj, opts...)
}

func (c interceptor) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	if c.funcs.Patch != nil {
		return c.funcs.Patch(ctx, c.client, obj, patch, opts...)
	}
	return c.client.Patch(ctx, obj, patch, opts...)
}

func (c interceptor) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	if c.funcs.DeleteAllOf != nil {
		return c.funcs.DeleteAllOf(ctx, c.client, obj, opts...)
	}
	return c.client.DeleteAllOf(ctx, obj, opts...)
}

func (c interceptor) Status() client.SubResourceWriter {
	return c.SubResource("status")
}

func (c interceptor) SubResource(subResource string) client.SubResourceClient {
	if c.funcs.SubResource != nil {
		return c.funcs.SubResource(c.client, subResource)
	}
	return c.client.SubResource(subResource)
}

func (c interceptor) Scheme() *runtime.Scheme {
	return c.client.Scheme()
}

func (c interceptor) RESTMapper() meta.RESTMapper {
	return c.client.RESTMapper()
}

func (c interceptor) Watch(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) (watch.Interface, error) {
	if c.funcs.Watch != nil {
		return c.funcs.Watch(ctx, c.client, obj, opts...)
	}
	return c.client.Watch(ctx, obj, opts...)
}

type subResourceInterceptor struct {
	client client.SubResourceClient
	funcs  SubResourceFuncs
}

var _ client.SubResourceClient = &subResourceInterceptor{}

func (s subResourceInterceptor) Get(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceGetOption) error {
	if s.funcs.Get != nil {
		return s.funcs.Get(ctx, s.client, obj, subResource, opts...)
	}
	return s.client.Get(ctx, obj, subResource, opts...)
}

func (s subResourceInterceptor) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	if s.funcs.Create != nil {
		return s.funcs.Create(ctx, s.client, obj, subResource, opts...)
	}
	return s.client.Create(ctx, obj, subResource, opts...)
}

func (s subResourceInterceptor) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	if s.funcs.Update != nil {
		return s.funcs.Update(ctx, s.client, obj, opts...)
	}
	return s.client.Update(ctx, obj, opts...)
}

func (s subResourceInterceptor) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	if s.funcs.Patch != nil {
		return s.funcs.Patch(ctx, s.client, obj, patch, opts...)
	}
	return s.client.Patch(ctx, obj, patch, opts...)
}
