/*
Copyright 2017 The Kubernetes Authors.

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

package installer

import (
	"fmt"
	gpath "path"
	"reflect"
	"strings"
	"time"

	"github.com/emicklei/go-restful/v3"

	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apiserver/pkg/endpoints"
	"k8s.io/apiserver/pkg/endpoints/discovery"
	"k8s.io/apiserver/pkg/endpoints/handlers"
	"k8s.io/apiserver/pkg/endpoints/handlers/negotiation"
	"k8s.io/apiserver/pkg/registry/rest"

	cm_handlers "sigs.k8s.io/custom-metrics-apiserver/pkg/apiserver/endpoints/handlers"
	cm_rest "sigs.k8s.io/custom-metrics-apiserver/pkg/apiserver/registry/rest"
)

// NB: the contents of this file should mostly be a subset of the functionality
// in "k8s.io/apiserver/pkg/endpoints".  It would be nice to eventual figure out
// a way to not have to recreate/copy a bunch of the structure from the normal API
// installer, so that this trivially tracks changes to the main installer.

// MetricsAPIGroupVersion is similar to "k8s.io/apiserver/pkg/endpoints".APIGroupVersion,
// except that it installs the metrics REST handlers, which use wildcard resources
// and subresources.
//
// This basically only serves the limitted use case required by the metrics API server --
// the only verb accepted is GET (and perhaps WATCH in the future).
type MetricsAPIGroupVersion struct {
	DynamicStorage rest.Storage

	*endpoints.APIGroupVersion

	ResourceLister discovery.APIResourceLister

	Handlers apiHandlers
}

type apiHandlers interface {
	registerResourceHandlers(a *MetricsAPIInstaller, ws *restful.WebService) error
}

// InstallDynamicREST registers the dynamic REST handlers into a restful Container.
// It is expected that the provided path root prefix will serve all operations.  Root MUST
// NOT end in a slash.  It should mirror InstallREST in the plain APIGroupVersion.
func (g *MetricsAPIGroupVersion) InstallREST(container *restful.Container) error {
	installer := g.newDynamicInstaller()
	ws := installer.NewWebService()

	registrationErrors := installer.Install(ws)
	lister := g.ResourceLister
	if lister == nil {
		return fmt.Errorf("must provide a dynamic lister for dynamic API groups")
	}
	versionDiscoveryHandler := discovery.NewAPIVersionHandler(g.Serializer, g.GroupVersion, lister)
	versionDiscoveryHandler.AddToWebService(ws)
	container.Add(ws)
	return utilerrors.NewAggregate(registrationErrors)
}

// newDynamicInstaller is a helper to create the installer.  It mirrors
// newInstaller in APIGroupVersion.
func (g *MetricsAPIGroupVersion) newDynamicInstaller() *MetricsAPIInstaller {
	prefix := gpath.Join(g.Root, g.GroupVersion.Group, g.GroupVersion.Version)
	installer := &MetricsAPIInstaller{
		group:             g,
		prefix:            prefix,
		minRequestTimeout: g.MinRequestTimeout,
		handlers:          g.Handlers,
	}

	return installer
}

// MetricsAPIInstaller is a specialized API installer for the metrics API.
// It is intended to be fully compliant with the Kubernetes API server conventions,
// but serves wildcard resource/subresource routes instead of hard-coded resources
// and subresources.
type MetricsAPIInstaller struct {
	group             *MetricsAPIGroupVersion
	prefix            string // Path prefix where API resources are to be registered.
	minRequestTimeout time.Duration
	handlers          apiHandlers

	// TODO: do we want to embed a normal API installer here so we can serve normal
	// endpoints side by side with dynamic ones (from the same API group)?
}

// Install installs handlers for External Metrics API resources.
func (a *MetricsAPIInstaller) Install(ws *restful.WebService) (errors []error) {
	errors = make([]error, 0)

	err := a.handlers.registerResourceHandlers(a, ws)
	if err != nil {
		errors = append(errors, fmt.Errorf("error in registering custom metrics resource: %v", err))
	}

	return errors
}

// NewWebService creates a new restful webservice with the api installer's prefix and version.
func (a *MetricsAPIInstaller) NewWebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path(a.prefix)
	// a.prefix contains "prefix/group/version"
	ws.Doc("API at " + a.prefix)
	// Backwards compatibility, we accepted objects with empty content-type at V1.
	// If we stop using go-restful, we can default empty content-type to application/json on an
	// endpoint by endpoint basis
	ws.Consumes("*/*")
	mediaTypes, streamMediaTypes := negotiation.MediaTypesForSerializer(a.group.Serializer)
	ws.Produces(append(mediaTypes, streamMediaTypes...)...)
	ws.ApiVersion(a.group.GroupVersion.String())

	return ws
}

// This magic incantation returns *ptrToObject for an arbitrary pointer
func indirectArbitraryPointer(ptrToObject interface{}) interface{} {
	return reflect.Indirect(reflect.ValueOf(ptrToObject)).Interface()
}

// getResourceKind returns the external group version kind registered for the given storage object.
func (a *MetricsAPIInstaller) getResourceKind(storage rest.Storage) (schema.GroupVersionKind, error) {
	object := storage.New()
	fqKinds, _, err := a.group.Typer.ObjectKinds(object)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}

	// a given go type can have multiple potential fully qualified kinds.  Find the one that corresponds with the group
	// we're trying to register here
	fqKindToRegister := schema.GroupVersionKind{}
	for _, fqKind := range fqKinds {
		if fqKind.Group == a.group.GroupVersion.Group {
			fqKindToRegister = a.group.GroupVersion.WithKind(fqKind.Kind)
			break
		}

		// TODO: keep rid of extensions api group dependency here
		// This keeps it doing what it was doing before, but it doesn't feel right.
		if fqKind.Group == "extensions" && fqKind.Kind == "ThirdPartyResourceData" {
			fqKindToRegister = a.group.GroupVersion.WithKind(fqKind.Kind)
		}
	}
	if fqKindToRegister.Empty() {
		return schema.GroupVersionKind{}, fmt.Errorf("unable to locate fully qualified kind for %v: found %v when registering for %v", reflect.TypeOf(object), fqKinds, a.group.GroupVersion)
	}
	return fqKindToRegister, nil
}

func addParams(route *restful.RouteBuilder, params []*restful.Parameter) {
	for _, param := range params {
		route.Param(param)
	}
}

// addObjectParams converts a runtime.Object into a set of go-restful Param() definitions on the route.
// The object must be a pointer to a struct; only fields at the top level of the struct that are not
// themselves interfaces or structs are used; only fields with a json tag that is non empty (the standard
// Go JSON behavior for omitting a field) become query parameters. The name of the query parameter is
// the JSON field name. If a description struct tag is set on the field, that description is used on the
// query parameter. In essence, it converts a standard JSON top level object into a query param schema.
func addObjectParams(ws *restful.WebService, route *restful.RouteBuilder, obj interface{}) error {
	sv, err := conversion.EnforcePtr(obj)
	if err != nil {
		return err
	}
	st := sv.Type()
	if st.Kind() == reflect.Struct {
		for i := 0; i < st.NumField(); i++ {
			name := st.Field(i).Name
			sf, ok := st.FieldByName(name)
			if !ok {
				continue
			}
			switch sf.Type.Kind() {
			case reflect.Interface, reflect.Struct:
			case reflect.Ptr:
				// TODO: This is a hack to let metav1.Time through. This needs to be fixed in a more generic way eventually. bug #36191
				if (sf.Type.Elem().Kind() == reflect.Interface || sf.Type.Elem().Kind() == reflect.Struct) && strings.TrimPrefix(sf.Type.String(), "*") != "metav1.Time" {
					continue
				}
				fallthrough
			default:
				jsonTag := sf.Tag.Get("json")
				if len(jsonTag) == 0 {
					continue
				}
				jsonName := strings.SplitN(jsonTag, ",", 2)[0]
				if len(jsonName) == 0 {
					continue
				}

				var desc string
				if docable, ok := obj.(documentable); ok {
					desc = docable.SwaggerDoc()[jsonName]
				}

				if route.ParameterNamed(jsonName) == nil {
					route.Param(ws.QueryParameter(jsonName, desc).DataType(typeToJSON(sf.Type.String())))
				}
			}
		}
	}
	return nil
}

// TODO: this is incomplete, expand as needed.
// Convert the name of a golang type to the name of a JSON type
func typeToJSON(typeName string) string {
	switch typeName {
	case "bool", "*bool":
		return "boolean"
	case "uint8", "*uint8", "int", "*int", "int32", "*int32", "int64", "*int64", "uint32", "*uint32", "uint64", "*uint64":
		return "integer"
	case "float64", "*float64", "float32", "*float32":
		return "number"
	case "metav1.Time", "*metav1.Time":
		return "string"
	case "byte", "*byte":
		return "string"
	case "v1.DeletionPropagation", "*v1.DeletionPropagation", "v1.ResourceVersionMatch":
		return "string"

	// TODO: Fix these when go-restful supports a way to specify an array query param:
	// https://github.com/emicklei/go-restful/issues/225
	case "[]string", "[]*string":
		return "string"
	case "[]int32", "[]*int32":
		return "integer"

	default:
		return typeName
	}
}

// An interface to see if an object supports swagger documentation as a method
type documentable interface {
	SwaggerDoc() map[string]string
}

// MetricsNaming is similar to handlers.ContextBasedNaming, except that it handles
// polymorphism over subresources.
type MetricsNaming struct {
	handlers.ContextBasedNaming
}

func restfulListResource(r rest.Lister, rw rest.Watcher, scope handlers.RequestScope, forceWatch bool, minRequestTimeout time.Duration) restful.RouteFunction {
	return func(req *restful.Request, res *restful.Response) {
		handlers.ListResource(r, rw, &scope, forceWatch, minRequestTimeout)(res.ResponseWriter, req.Request)
	}
}

func restfulListResourceWithOptions(r cm_rest.ListerWithOptions, scope handlers.RequestScope) restful.RouteFunction {
	return func(req *restful.Request, res *restful.Response) {
		cm_handlers.ListResourceWithOptions(r, scope)(res.ResponseWriter, req.Request)
	}
}
