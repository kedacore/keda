/*
Copyright 2017 Google Inc. All Rights Reserved.
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

package build

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/knative/build/pkg/apis/build/v1alpha1"
	"github.com/knative/build/pkg/reconciler/build/resources"
)

func (ac *Reconciler) validateBuild(b *v1alpha1.Build) error {
	if err := ac.validateSecrets(b); err != nil {
		return err
	}

	// If a build specifies a template, all the template's parameters without
	// defaults must be satisfied by the build's parameters.
	var tmpl v1alpha1.BuildTemplateInterface
	var err error
	if b.Spec.Template != nil {
		tmplName := b.Spec.Template.Name
		if b.Spec.Template.Kind == v1alpha1.ClusterBuildTemplateKind && tmplName != "" {
			tmpl, err = ac.buildclientset.BuildV1alpha1().ClusterBuildTemplates().Get(tmplName, metav1.GetOptions{})
			if err != nil {
				return err
			}
		} else if b.Spec.Template.Kind == v1alpha1.BuildTemplateKind || b.Spec.Template.Kind == "" && tmplName != "" {
			tmpl, err = ac.buildclientset.BuildV1alpha1().BuildTemplates(b.Namespace).Get(tmplName, metav1.GetOptions{})
			if err != nil {
				return err
			}
		} else {
			return validationError("Incorrect Template Kind", "the template kind can only be \"BuildTemplate\" or \"ClusterBuildTemplate\" with \"BuildTemplate\" used as the default if nothing is specified.")
		}

		if err := validateArguments(b.Spec.Template.Arguments, tmpl); err != nil {
			return err
		}

		if err := v1alpha1.ValidateVolumes(tmpl.TemplateSpec().Volumes); err != nil {
			return err
		}

		// Validate build template
		if err := validateTemplate(tmpl); err != nil {
			return err
		}
	}

	// Ensure the build can be translated to a Pod.
	_, err = resources.MakePod(b, ac.kubeclientset)
	return err
}

// validateSecrets checks that if the Build specifies a ServiceAccount, that it
// exists, and that any Secrets referenced by it exist, and have valid
// annotations.
func (ac *Reconciler) validateSecrets(b *v1alpha1.Build) error {
	saName := b.Spec.ServiceAccountName
	if saName == "" {
		saName = "default"
	}

	sa, err := ac.kubeclientset.CoreV1().ServiceAccounts(b.Namespace).Get(saName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	for _, se := range sa.Secrets {
		sec, err := ac.kubeclientset.CoreV1().Secrets(b.Namespace).Get(se.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		// Check that the annotation value "index.docker.io" is not
		// present. This annotation value can be misleading, since
		// Dockerhub expects the fully-specified value
		// "https://index.docker.io/v1/", and other registries accept
		// other variants (e.g., "gcr.io" or "https://gcr.io/v1/",
		// etc.). See https://github.com/knative/build/issues/195
		//
		// TODO(jasonhall): Instead of validating a Secret when a Build
		// uses it, set up webhook validation for Secrets, and reject
		// them outright before a Build ever uses them. This would
		// remove latency at Build-time.
		for k, v := range sec.Annotations {
			if strings.HasPrefix(k, "build.knative.dev/docker-") && v == "index.docker.io" {
				return validationError("BadSecretAnnotation", `Secret %q has incorrect annotation %q / %q, value should be "https://index.docker.io/v1/"`, se.Name, k, v)
			}
		}
	}
	return nil
}

func validateArguments(args []v1alpha1.ArgumentSpec, tmpl v1alpha1.BuildTemplateInterface) error {
	// Build must not duplicate argument names.
	seen := sets.NewString()
	for _, a := range args {
		if seen.Has(a.Name) {
			return validationError("DuplicateArgName", "duplicate argument name %q", a.Name)
		}
		seen.Insert(a.Name)
	}
	// If a build specifies a template, all the template's parameters without
	// defaults must be satisfied by the build's parameters.
	if tmpl != nil {
		tmplParams := map[string]string{} // value is the param description.
		for _, p := range tmpl.TemplateSpec().Parameters {
			if p.Default == nil {
				tmplParams[p.Name] = p.Description
			}
		}
		for _, p := range args {
			delete(tmplParams, p.Name)
		}
		if len(tmplParams) > 0 {
			type pair struct{ name, desc string }
			var unused []pair
			for k, v := range tmplParams {
				unused = append(unused, pair{k, v})
			}
			return validationError("UnsatisfiedParameter", "build does not specify these required parameters: %s", unused)
		}
	}
	return nil
}

type verror struct {
	reason, message string
}

func (ve *verror) Error() string { return fmt.Sprintf("%s: %s", ve.reason, ve.message) }

func validationError(reason, format string, fmtArgs ...interface{}) error {
	return &verror{
		reason:  reason,
		message: fmt.Sprintf(format, fmtArgs...),
	}
}
