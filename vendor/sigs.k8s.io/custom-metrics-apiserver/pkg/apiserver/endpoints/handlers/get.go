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

package handlers

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metainternalversionscheme "k8s.io/apimachinery/pkg/apis/meta/internalversion/scheme"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/handlers"
	"k8s.io/apiserver/pkg/endpoints/handlers/negotiation"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/apiserver/pkg/endpoints/request"
	utiltrace "k8s.io/utils/trace"

	cm_rest "sigs.k8s.io/custom-metrics-apiserver/pkg/apiserver/registry/rest"
)

func ListResourceWithOptions(r cm_rest.ListerWithOptions, scope handlers.RequestScope) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// For performance tracking purposes.
		trace := utiltrace.New("List " + req.URL.Path)

		requestInfo, ok := request.RequestInfoFrom(req.Context())
		if !ok {
			err := errors.NewBadRequest("missing requestInfo")
			writeError(&scope, err, w, req)
			return
		}

		// handle metrics describing namespaces
		if requestInfo.Namespace != "" && requestInfo.Resource == "metrics" {
			requestInfo.Subresource = requestInfo.Name
			requestInfo.Name = requestInfo.Namespace
			requestInfo.Resource = "namespaces"
			requestInfo.Namespace = ""
			requestInfo.Parts = append([]string{"namespaces", requestInfo.Name}, requestInfo.Parts[1:]...)
		}

		// handle invalid requests, e.g. /namespaces/name/foo
		if len(requestInfo.Parts) < 3 {
			err := errors.NewBadRequest("invalid request path")
			writeError(&scope, err, w, req)
			return
		}

		namespace, err := scope.Namer.Namespace(req)
		if err != nil {
			writeError(&scope, err, w, req)
			return
		}

		// Watches for single objects are routed to this function.
		// Treat a name parameter the same as a field selector entry.
		hasName := true
		_, name, err := scope.Namer.Name(req)
		if err != nil {
			hasName = false
		}

		ctx := req.Context()
		ctx = request.WithNamespace(ctx, namespace)

		opts := metainternalversion.ListOptions{}
		if err := metainternalversionscheme.ParameterCodec.DecodeParameters(req.URL.Query(), scope.MetaGroupVersion, &opts); err != nil {
			err = errors.NewBadRequest(err.Error())
			writeError(&scope, err, w, req)
			return
		}

		// transform fields
		// TODO: DecodeParametersInto should do this.
		if opts.FieldSelector != nil {
			fn := func(label, value string) (newLabel, newValue string, err error) {
				return scope.Convertor.ConvertFieldLabel(scope.Kind, label, value)
			}
			if opts.FieldSelector, err = opts.FieldSelector.Transform(fn); err != nil {
				// TODO: allow bad request to set field causes based on query parameters
				err = errors.NewBadRequest(err.Error())
				writeError(&scope, err, w, req)
				return
			}
		}

		if hasName {
			// metadata.name is the canonical internal name.
			// SelectionPredicate will notice that this is a request for
			// a single object and optimize the storage query accordingly.
			nameSelector := fields.OneTermEqualSelector("metadata.name", name)

			// Note that fieldSelector setting explicitly the "metadata.name"
			// will result in reaching this branch (as the value of that field
			// is propagated to requestInfo as the name parameter.
			// That said, the allowed field selectors in this branch are:
			// nil, fields.Everything and field selector matching metadata.name
			// for our name.
			if opts.FieldSelector != nil && !opts.FieldSelector.Empty() {
				selectedName, ok := opts.FieldSelector.RequiresExactMatch("metadata.name")
				if !ok || name != selectedName {
					writeError(&scope, errors.NewBadRequest("fieldSelector metadata.name doesn't match requested name"), w, req)
					return
				}
			} else {
				opts.FieldSelector = nameSelector
			}
		}

		// Log only long List requests (ignore Watch).
		defer trace.LogIfLong(500 * time.Millisecond)
		trace.Step("About to List from storage")
		extraOpts, hasSubpath, subpathKey := r.NewListOptions()
		if err := getRequestOptions(req, scope, extraOpts, hasSubpath, subpathKey, false); err != nil {
			err = errors.NewBadRequest(err.Error())
			writeError(&scope, err, w, req)
			return
		}
		result, err := r.List(ctx, &opts, extraOpts)
		if err != nil {
			writeError(&scope, err, w, req)
			return
		}
		trace.Step("Listing from storage done")

		responsewriters.WriteObjectNegotiated(scope.Serializer, negotiation.DefaultEndpointRestrictions, scope.Kind.GroupVersion(), w, req, http.StatusOK, result, false)
		trace.Step("Writing http response done")
	}
}

// getRequestOptions parses out options and can include path information.  The path information shouldn't include the subresource.
func getRequestOptions(req *http.Request, scope handlers.RequestScope, into runtime.Object, hasSubpath bool, subpathKey string, isSubresource bool) error {
	if into == nil {
		return nil
	}

	query := req.URL.Query()
	if hasSubpath {
		newQuery := make(url.Values)
		for k, v := range query {
			newQuery[k] = v
		}

		ctx := req.Context()
		requestInfo, _ := request.RequestInfoFrom(ctx)
		startingIndex := 2
		if isSubresource {
			startingIndex = 3
		}

		p := strings.Join(requestInfo.Parts[startingIndex:], "/")

		// ensure non-empty subpaths correctly reflect a leading slash
		if len(p) > 0 && !strings.HasPrefix(p, "/") {
			p = "/" + p
		}

		// ensure subpaths correctly reflect the presence of a trailing slash on the original request
		if strings.HasSuffix(requestInfo.Path, "/") && !strings.HasSuffix(p, "/") {
			p += "/"
		}

		newQuery[subpathKey] = []string{p}
		query = newQuery
	}
	return scope.ParameterCodec.DecodeParameters(query, scope.Kind.GroupVersion(), into)
}
