/*
Copyright 2022 The KEDA Authors

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

package k8s

import (
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/scale"
	ctrl "sigs.k8s.io/controller-runtime"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

var log = ctrl.Log.WithName("scaleclient")

// InitScaleClient initializes scale client and returns k8s version
func InitScaleClient(mgr ctrl.Manager) (scale.ScalesGetter, kedautil.K8sVersion, error) {
	kubeVersion := kedautil.K8sVersion{}

	// create Discovery clientset
	// TODO If we need to increase the QPS of scaling API calls, copy and tweak this RESTConfig.
	clientset, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		log.Error(err, "not able to create Discovery clientset")
		return nil, kubeVersion, err
	}

	// Find out Kubernetes version
	version, err := clientset.ServerVersion()
	if err == nil {
		kubeVersion = kedautil.NewK8sVersion(version)
	} else {
		log.Error(err, "not able to get Kubernetes version")
		return nil, kubeVersion, err
	}

	return scale.New(
		clientset.RESTClient(), mgr.GetRESTMapper(),
		dynamic.LegacyAPIPathResolverFunc,
		scale.NewDiscoveryScaleKindResolver(clientset),
	), kubeVersion, nil
}
