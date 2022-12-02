/*
Copyright 2018 The Kubernetes Authors.

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
	"net/http"

	"github.com/emicklei/go-restful/v3"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/endpoints/handlers"
	"k8s.io/apiserver/pkg/endpoints/handlers/negotiation"
	"k8s.io/apiserver/pkg/endpoints/metrics"

	"sigs.k8s.io/custom-metrics-apiserver/pkg/apiserver/registry/rest"
)

type CMHandlers struct{}

// registerResourceHandlers registers the resource handlers for custom metrics.
// Compared to the normal installer, this plays fast and loose a bit, but should still
// follow the API conventions.
func (ch *CMHandlers) registerResourceHandlers(a *MetricsAPIInstaller, ws *restful.WebService) error {
	optionsExternalVersion := a.group.GroupVersion
	if a.group.OptionsExternalVersion != nil {
		optionsExternalVersion = *a.group.OptionsExternalVersion
	}

	fqKindToRegister, err := a.getResourceKind(a.group.DynamicStorage)
	if err != nil {
		return err
	}

	kind := fqKindToRegister.Kind

	lister := a.group.DynamicStorage.(rest.ListerWithOptions)
	list := lister.NewList()
	listGVKs, _, err := a.group.Typer.ObjectKinds(list)
	if err != nil {
		return err
	}
	versionedListPtr, err := a.group.Creater.New(a.group.GroupVersion.WithKind(listGVKs[0].Kind))
	if err != nil {
		return err
	}
	versionedList := indirectArbitraryPointer(versionedListPtr)

	versionedListOptions, err := a.group.Creater.New(optionsExternalVersion.WithKind("ListOptions"))
	if err != nil {
		return err
	}

	listOptions, _, _ := lister.NewListOptions()
	listOptionsInternalKinds, _, err := a.group.Typer.ObjectKinds(listOptions)
	if err != nil {
		return err
	}
	listOptionsInternalKind := listOptionsInternalKinds[0]
	versionedListExtraOptions, err := a.group.Creater.New(a.group.GroupVersion.WithKind(listOptionsInternalKind.Kind))
	if err != nil {
		versionedListExtraOptions, err = a.group.Creater.New(optionsExternalVersion.WithKind(listOptionsInternalKind.Kind))
		if err != nil {
			return err
		}
	}

	nameParam := ws.PathParameter("name", "name of the described resource").DataType("string")
	resourceParam := ws.PathParameter("resource", "the name of the resource").DataType("string")
	subresourceParam := ws.PathParameter("subresource", "the name of the subresource").DataType("string")

	// metrics describing non-namespaced objects (e.g. nodes)
	rootScopedParams := []*restful.Parameter{
		resourceParam,
		nameParam,
		subresourceParam,
	}
	rootScopedPath := "{resource}/{name}/{subresource}"

	// metrics describing namespaced objects (e.g. pods)
	namespaceParam := ws.PathParameter("namespace", "object name and auth scope, such as for teams and projects").DataType("string")
	namespacedParams := []*restful.Parameter{
		namespaceParam,
		resourceParam,
		nameParam,
		subresourceParam,
	}
	namespacedPath := "namespaces/{namespace}/{resource}/{name}/{subresource}"

	namespaceSpecificPath := "namespaces/{namespace}/metrics/{name}"
	namespaceSpecificParams := []*restful.Parameter{
		namespaceParam,
		nameParam,
	}

	mediaTypes, streamMediaTypes := negotiation.MediaTypesForSerializer(a.group.Serializer)
	allMediaTypes := append(mediaTypes, streamMediaTypes...) //nolint: gocritic
	ws.Produces(allMediaTypes...)

	reqScope := handlers.RequestScope{
		Serializer:      a.group.Serializer,
		ParameterCodec:  a.group.ParameterCodec,
		Creater:         a.group.Creater,
		Convertor:       a.group.Convertor,
		Typer:           a.group.Typer,
		UnsafeConvertor: a.group.UnsafeConvertor,

		// TODO: support TableConvertor?

		// TODO: This seems wrong for cross-group subresources. It makes an assumption that a subresource and its parent are in the same group version. Revisit this.
		Resource:    a.group.GroupVersion.WithResource("*"),
		Subresource: "*",
		Kind:        fqKindToRegister,

		MetaGroupVersion: metav1.SchemeGroupVersion,
	}
	if a.group.MetaGroupVersion != nil {
		reqScope.MetaGroupVersion = *a.group.MetaGroupVersion
	}

	// we need one path for namespaced resources, one for non-namespaced resources
	doc := "list custom metrics describing an object or objects"
	reqScope.Namer = MetricsNaming{
		handlers.ContextBasedNaming{
			Namer:         a.group.Namer,
			ClusterScoped: true,
		},
	}

	rootScopedHandler := metrics.InstrumentRouteFunc(
		"LIST",
		a.group.GroupVersion.Group,
		a.group.GroupVersion.Version,
		reqScope.Resource.Resource,
		reqScope.Subresource,
		"cluster",
		"custom-metrics",
		false,
		"",
		restfulListResourceWithOptions(lister, reqScope),
	)

	// install the root-scoped route
	rootScopedRoute := ws.GET(rootScopedPath).To(rootScopedHandler).
		Doc(doc).
		Param(ws.QueryParameter("pretty", "If 'true', then the output is pretty printed.")).
		Operation("list"+kind).
		Produces(allMediaTypes...).
		Returns(http.StatusOK, "OK", versionedList).
		Writes(versionedList)
	if err := addObjectParams(ws, rootScopedRoute, versionedListOptions); err != nil {
		return err
	}
	if err := addObjectParams(ws, rootScopedRoute, versionedListExtraOptions); err != nil {
		return err
	}
	addParams(rootScopedRoute, rootScopedParams)
	ws.Route(rootScopedRoute)

	// install the namespace-scoped route
	reqScope.Namer = MetricsNaming{
		handlers.ContextBasedNaming{
			Namer:         a.group.Namer,
			ClusterScoped: false,
		},
	}
	namespacedHandler := metrics.InstrumentRouteFunc(
		"LIST",
		a.group.GroupVersion.Group,
		a.group.GroupVersion.Version,
		reqScope.Resource.Resource,
		reqScope.Subresource,
		"resource",
		"custom-metrics",
		false,
		"",
		restfulListResourceWithOptions(lister, reqScope),
	)

	namespacedRoute := ws.GET(namespacedPath).To(namespacedHandler).
		Doc(doc).
		Param(ws.QueryParameter("pretty", "If 'true', then the output is pretty printed.")).
		Operation("listNamespaced"+kind).
		Produces(allMediaTypes...).
		Returns(http.StatusOK, "OK", versionedList).
		Writes(versionedList)
	if err := addObjectParams(ws, namespacedRoute, versionedListOptions); err != nil {
		return err
	}
	if err := addObjectParams(ws, namespacedRoute, versionedListExtraOptions); err != nil {
		return err
	}
	addParams(namespacedRoute, namespacedParams)
	ws.Route(namespacedRoute)

	// install the special route for metrics describing namespaces (last b/c we modify the context func)
	reqScope.Namer = MetricsNaming{
		handlers.ContextBasedNaming{
			Namer:         a.group.Namer,
			ClusterScoped: false,
		},
	}

	namespaceSpecificHandler := metrics.InstrumentRouteFunc(
		"LIST",
		a.group.GroupVersion.Group,
		a.group.GroupVersion.Version,
		reqScope.Resource.Resource,
		reqScope.Subresource,
		"resource",
		"custom-metrics",
		false,
		"",
		restfulListResourceWithOptions(lister, reqScope),
	)

	namespaceSpecificRoute := ws.GET(namespaceSpecificPath).To(namespaceSpecificHandler).
		Doc(doc).
		Param(ws.QueryParameter("pretty", "If 'true', then the output is pretty printed.")).
		Operation("read"+kind+"ForNamespace").
		Produces(allMediaTypes...).
		Returns(http.StatusOK, "OK", versionedList).
		Writes(versionedList)
	if err := addObjectParams(ws, namespaceSpecificRoute, versionedListOptions); err != nil {
		return err
	}
	if err := addObjectParams(ws, namespaceSpecificRoute, versionedListExtraOptions); err != nil {
		return err
	}
	addParams(namespaceSpecificRoute, namespaceSpecificParams)
	ws.Route(namespaceSpecificRoute)

	return nil
}
