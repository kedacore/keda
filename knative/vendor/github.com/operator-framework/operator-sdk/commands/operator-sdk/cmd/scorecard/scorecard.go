// Copyright 2019 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package scorecard

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	k8sInternal "github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/internal/util/yamlutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"

	"github.com/ghodss/yaml"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	olminstall "github.com/operator-framework/operator-lifecycle-manager/pkg/controller/install"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	extscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/kubernetes"
	cgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ConfigOpt             = "config"
	NamespaceOpt          = "namespace"
	KubeconfigOpt         = "kubeconfig"
	InitTimeoutOpt        = "init-timeout"
	OlmDeployedOpt        = "olm-deployed"
	CSVPathOpt            = "csv-path"
	BasicTestsOpt         = "basic-tests"
	OLMTestsOpt           = "olm-tests"
	TenantTestsOpt        = "good-tenant-tests"
	NamespacedManifestOpt = "namespace-manifest"
	GlobalManifestOpt     = "global-manifest"
	CRManifestOpt         = "cr-manifest"
	ProxyImageOpt         = "proxy-image"
	ProxyPullPolicyOpt    = "proxy-pull-policy"
	CRDsDirOpt            = "crds-dir"
	VerboseOpt            = "verbose"
)

const (
	basicOperator  = "Basic Operator"
	olmIntegration = "OLM Integration"
	goodTenant     = "Good Tenant"
)

// TODO: add point weights to tests
type scorecardTest struct {
	testType      string
	name          string
	description   string
	earnedPoints  int
	maximumPoints int
}

type cleanupFn func() error

var (
	kubeconfig     *rest.Config
	scTests        []scorecardTest
	scSuggestions  []string
	dynamicDecoder runtime.Decoder
	runtimeClient  client.Client
	restMapper     *restmapper.DeferredDiscoveryRESTMapper
	deploymentName string
	proxyPod       *v1.Pod
	cleanupFns     []cleanupFn
	ScorecardConf  string
)

const (
	scorecardPodName       = "operator-scorecard-test"
	scorecardContainerName = "scorecard-proxy"
)

func ScorecardTests(cmd *cobra.Command, args []string) error {
	if err := initConfig(); err != nil {
		return err
	}
	if err := validateScorecardFlags(); err != nil {
		return err
	}
	cmd.SilenceUsage = true
	if viper.GetBool(VerboseOpt) {
		log.SetLevel(log.DebugLevel)
	}
	defer func() {
		if err := cleanupScorecard(); err != nil {
			log.Errorf("Failed to clenup resources: (%v)", err)
		}
	}()

	var (
		tmpNamespaceVar string
		err             error
	)
	kubeconfig, tmpNamespaceVar, err = k8sInternal.GetKubeconfigAndNamespace(viper.GetString(KubeconfigOpt))
	if err != nil {
		return fmt.Errorf("failed to build the kubeconfig: %v", err)
	}
	if viper.GetString(NamespaceOpt) == "" {
		viper.Set(NamespaceOpt, tmpNamespaceVar)
	}
	scheme := runtime.NewScheme()
	// scheme for client go
	if err := cgoscheme.AddToScheme(scheme); err != nil {
		return fmt.Errorf("failed to add client-go scheme to client: (%v)", err)
	}
	// api extensions scheme (CRDs)
	if err := extscheme.AddToScheme(scheme); err != nil {
		return fmt.Errorf("failed to add failed to add extensions api scheme to client: (%v)", err)
	}
	// olm api (CS
	if err := olmapiv1alpha1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("failed to add failed to add oml api scheme (CSVs) to client: (%v)", err)
	}
	dynamicDecoder = serializer.NewCodecFactory(scheme).UniversalDeserializer()
	// if a user creates a new CRD, we need to be able to reset the rest mapper
	// temporary kubeclient to get a cached discovery
	kubeclient, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to get a kubeclient: %v", err)
	}
	cachedDiscoveryClient := cached.NewMemCacheClient(kubeclient.Discovery())
	restMapper = restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient)
	restMapper.Reset()
	runtimeClient, _ = client.New(kubeconfig, client.Options{Scheme: scheme, Mapper: restMapper})

	csv := &olmapiv1alpha1.ClusterServiceVersion{}
	if viper.GetBool(OLMTestsOpt) {
		yamlSpec, err := ioutil.ReadFile(viper.GetString(CSVPathOpt))
		if err != nil {
			return fmt.Errorf("failed to read csv: %v", err)
		}
		if err = yaml.Unmarshal(yamlSpec, csv); err != nil {
			return fmt.Errorf("error getting ClusterServiceVersion: %v", err)
		}
	}

	// Extract operator manifests from the CSV if olm-deployed is set.
	if viper.GetBool(OlmDeployedOpt) {
		// Get deploymentName from the deployment manifest within the CSV.
		strat, err := (&olminstall.StrategyResolver{}).UnmarshalStrategy(csv.Spec.InstallStrategy)
		if err != nil {
			return err
		}
		stratDep, ok := strat.(*olminstall.StrategyDetailsDeployment)
		if !ok {
			return fmt.Errorf("expected StrategyDetailsDeployment, got strategy of type %T", strat)
		}
		deploymentName = stratDep.DeploymentSpecs[0].Name
		// Get the proxy pod, which should have been created with the CSV.
		proxyPod, err = getPodFromDeployment(deploymentName, viper.GetString(NamespaceOpt))
		if err != nil {
			return err
		}

		// Create a temporary CR manifest from metadata if one is not provided.
		crJSONStr, ok := csv.ObjectMeta.Annotations["alm-examples"]
		if ok && viper.GetString(CRManifestOpt) == "" {
			var crs []interface{}
			if err = json.Unmarshal([]byte(crJSONStr), &crs); err != nil {
				return err
			}
			// TODO: run scorecard against all CR's in CSV.
			cr := crs[0]
			crJSONBytes, err := json.Marshal(cr)
			if err != nil {
				return err
			}
			crYAMLBytes, err := yaml.JSONToYAML(crJSONBytes)
			if err != nil {
				return err
			}
			crFile, err := ioutil.TempFile("", "cr.yaml")
			if err != nil {
				return err
			}
			if _, err := crFile.Write(crYAMLBytes); err != nil {
				return err
			}
			viper.Set(CRManifestOpt, crFile.Name())
			defer func() {
				err := os.Remove(viper.GetString(CRManifestOpt))
				if err != nil {
					log.Errorf("Could not delete temporary CR manifest file: (%v)", err)
				}
			}()
		}

	} else {
		// If no namespaced manifest path is given, combine
		// deploy/{service_account,role.yaml,role_binding,operator}.yaml.
		if viper.GetString(NamespacedManifestOpt) == "" {
			file, err := yamlutil.GenerateCombinedNamespacedManifest(scaffold.DeployDir)
			if err != nil {
				return err
			}
			viper.Set(NamespacedManifestOpt, file.Name())
			defer func() {
				err := os.Remove(viper.GetString(NamespacedManifestOpt))
				if err != nil {
					log.Errorf("Could not delete temporary namespace manifest file: (%v)", err)
				}
			}()
		}
		// If no global manifest is given, combine all CRD's in the given CRD's dir.
		if viper.GetString(GlobalManifestOpt) == "" {
			gMan, err := yamlutil.GenerateCombinedGlobalManifest(viper.GetString(CRDsDirOpt))
			if err != nil {
				return err
			}
			viper.Set(GlobalManifestOpt, gMan.Name())
			defer func() {
				err := os.Remove(viper.GetString(GlobalManifestOpt))
				if err != nil {
					log.Errorf("Could not delete global manifest file: (%v)", err)
				}
			}()
		}
		if err := createFromYAMLFile(viper.GetString(GlobalManifestOpt)); err != nil {
			return fmt.Errorf("failed to create global resources: %v", err)
		}
		if err := createFromYAMLFile(viper.GetString(NamespacedManifestOpt)); err != nil {
			return fmt.Errorf("failed to create namespaced resources: %v", err)
		}
	}

	if err := createFromYAMLFile(viper.GetString(CRManifestOpt)); err != nil {
		return fmt.Errorf("failed to create cr resource: %v", err)
	}
	obj, err := yamlToUnstructured(viper.GetString(CRManifestOpt))
	if err != nil {
		return fmt.Errorf("failed to decode custom resource manifest into object: %s", err)
	}

	// Run tests.
	if viper.GetBool(BasicTestsOpt) {
		fmt.Println("Checking for existence of spec and status blocks in CR")
		err = checkSpecAndStat(runtimeClient, obj, false)
		if err != nil {
			return err
		}
		// This test is far too inconsistent and unreliable to be meaningful,
		// so it has been disabled
		/*
			fmt.Println("Checking that operator actions are reflected in status")
			err = checkStatusUpdate(runtimeClient, obj)
			if err != nil {
				return err
			}
		*/
		fmt.Println("Checking that writing into CRs has an effect")
		logs, err := writingIntoCRsHasEffect(obj)
		if err != nil {
			return err
		}
		log.Debugf("Scorecard Proxy Logs: %v\n", logs)
	} else {
		// checkSpecAndStat is used to make sure the operator is ready in this case
		// the boolean argument set at the end tells the function not to add the result to scTests
		err = checkSpecAndStat(runtimeClient, obj, true)
		if err != nil {
			return err
		}
	}
	if viper.GetBool(OLMTestsOpt) {
		fmt.Println("Checking if all CRDs have validation")
		if err := crdsHaveValidation(viper.GetString(CRDsDirOpt), runtimeClient, obj); err != nil {
			return err
		}
		fmt.Println("Checking for CRD resources")
		crdsHaveResources(obj, csv)
		fmt.Println("Checking for existence of example CRs")
		annotationsContainExamples(csv)
		fmt.Println("Checking spec descriptors")
		err = specDescriptors(csv, runtimeClient, obj)
		if err != nil {
			return err
		}
		fmt.Println("Checking status descriptors")
		err = statusDescriptors(csv, runtimeClient, obj)
		if err != nil {
			return err
		}
	}

	var totalEarned, totalMax int
	var enabledTestTypes []string
	if viper.GetBool(BasicTestsOpt) {
		enabledTestTypes = append(enabledTestTypes, basicOperator)
	}
	if viper.GetBool(OLMTestsOpt) {
		enabledTestTypes = append(enabledTestTypes, olmIntegration)
	}
	if viper.GetBool(TenantTestsOpt) {
		enabledTestTypes = append(enabledTestTypes, goodTenant)
	}
	for _, testType := range enabledTestTypes {
		fmt.Printf("%s:\n", testType)
		for _, test := range scTests {
			if test.testType == testType {
				if !(test.earnedPoints == 0 && test.maximumPoints == 0) {
					fmt.Printf("\t%s: %d/%d points\n", test.name, test.earnedPoints, test.maximumPoints)
				} else {
					fmt.Printf("\t%s: N/A (depends on an earlier test that failed)\n", test.name)
				}
				totalEarned += test.earnedPoints
				totalMax += test.maximumPoints
			}
		}
	}
	fmt.Printf("\nTotal Score: %d/%d points\n", totalEarned, totalMax)
	for _, suggestion := range scSuggestions {
		// 33 is yellow (specifically, the same shade of yellow that logrus uses for warnings)
		fmt.Printf("\x1b[%dmSUGGESTION:\x1b[0m %s\n", 33, suggestion)
	}
	return nil
}

func initConfig() error {
	if ScorecardConf != "" {
		// Use config file from the flag.
		viper.SetConfigFile(ScorecardConf)
	} else {
		viper.AddConfigPath(projutil.MustGetwd())
		// using SetConfigName allows users to use a .yaml, .json, or .toml file
		viper.SetConfigName(".osdk-scorecard")
	}

	if err := viper.ReadInConfig(); err == nil {
		log.Info("Using config file: ", viper.ConfigFileUsed())
	} else {
		log.Warn("Could not load config file; using flags")
	}
	return nil
}

func validateScorecardFlags() error {
	if !viper.GetBool(OlmDeployedOpt) && viper.GetString(CRManifestOpt) == "" {
		return errors.New("cr-manifest config option must be set")
	}
	if !viper.GetBool(BasicTestsOpt) && !viper.GetBool(OLMTestsOpt) {
		return errors.New("at least one test type must be set")
	}
	if viper.GetBool(OLMTestsOpt) && viper.GetString(CSVPathOpt) == "" {
		return fmt.Errorf("csv-path must be set if olm-tests is enabled")
	}
	if viper.GetBool(OlmDeployedOpt) && viper.GetString(CSVPathOpt) == "" {
		return fmt.Errorf("csv-path must be set if olm-deployed is enabled")
	}
	pullPolicy := viper.GetString(ProxyPullPolicyOpt)
	if pullPolicy != "Always" && pullPolicy != "Never" && pullPolicy != "PullIfNotPresent" {
		return fmt.Errorf("invalid proxy pull policy: (%s); valid values: Always, Never, PullIfNotPresent", pullPolicy)
	}
	return nil
}
