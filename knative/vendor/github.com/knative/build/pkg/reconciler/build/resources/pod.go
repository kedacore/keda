/*
Copyright 2018 The Knative Authors

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

// Package resources provides methods to convert a Build CRD to a k8s Pod
// resource.
package resources

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"

	v1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	"github.com/knative/build/pkg/credentials"
	"github.com/knative/build/pkg/credentials/dockercreds"
	"github.com/knative/build/pkg/credentials/gitcreds"
	"github.com/knative/pkg/apis"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
)

const workspaceDir = "/workspace"

// These are effectively const, but Go doesn't have such an annotation.
var (
	emptyVolumeSource = corev1.VolumeSource{
		EmptyDir: &corev1.EmptyDirVolumeSource{},
	}
	// These are injected into all of the source/step containers.
	implicitEnvVars = []corev1.EnvVar{{
		Name:  "HOME",
		Value: "/builder/home",
	}}
	implicitVolumeMounts = []corev1.VolumeMount{{
		Name:      "workspace",
		MountPath: workspaceDir,
	}, {
		Name:      "home",
		MountPath: "/builder/home",
	}}
	implicitVolumes = []corev1.Volume{{
		Name:         "workspace",
		VolumeSource: emptyVolumeSource,
	}, {
		Name:         "home",
		VolumeSource: emptyVolumeSource,
	}}

	// Random byte reader used for pod name generation.
	// var for testing.
	randReader = rand.Reader
)

const (
	// Prefixes to add to the name of the init containers.
	// IMPORTANT: Changing these values without changing fluentd collection configuration
	// will break log collection for init containers.
	initContainerPrefix        = "build-step-"
	unnamedInitContainerPrefix = "build-step-unnamed-"
	// A label with the following is added to the pod to identify the pods belonging to a build.
	buildNameLabelKey = "build.knative.dev/buildName"
	// Name of the credential initialization container.
	credsInit = "credential-initializer"
	// Names for source containers.
	gitSource    = "git-source"
	gcsSource    = "gcs-source"
	customSource = "custom-source"
)

var (
	// The container used to initialize credentials before the build runs.
	credsImage = flag.String("creds-image", "override-with-creds:latest",
		"The container image for preparing our Build's credentials.")
	// The container with Git that we use to implement the Git source step.
	gitImage = flag.String("git-image", "override-with-git:latest",
		"The container image containing our Git binary.")
	// The container that just prints build successful.
	nopImage = flag.String("nop-image", "override-with-nop:latest",
		"The container image run at the end of the build to log build success")
	gcsFetcherImage = flag.String("gcs-fetcher-image", "gcr.io/cloud-builders/gcs-fetcher:latest",
		"The container image containing our GCS fetcher binary.")
)

// TODO(mattmoor): Should we move this somewhere common, because of the flag?
func gitToContainer(source v1alpha1.SourceSpec, index int) (*corev1.Container, error) {
	git := source.Git
	if git.Url == "" {
		return nil, apis.ErrMissingField("b.spec.source.git.url")
	}
	if git.Revision == "" {
		return nil, apis.ErrMissingField("b.spec.source.git.revision")
	}

	args := []string{"-url", git.Url,
		"-revision", git.Revision,
	}

	if source.TargetPath != "" {
		args = append(args, []string{"-path", source.TargetPath}...)
	}

	containerName := initContainerPrefix + gitSource + "-"

	// update container name to suffix source name
	if source.Name != "" {
		containerName = containerName + source.Name
	} else {
		containerName = containerName + strconv.Itoa(index)
	}

	return &corev1.Container{
		Name:         containerName,
		Image:        *gitImage,
		Args:         args,
		VolumeMounts: implicitVolumeMounts,
		WorkingDir:   workspaceDir,
		Env:          implicitEnvVars,
	}, nil
}

func gcsToContainer(source v1alpha1.SourceSpec, index int) (*corev1.Container, error) {
	gcs := source.GCS
	if gcs.Location == "" {
		return nil, apis.ErrMissingField("b.spec.source.gcs.location")
	}
	args := []string{"--type", string(gcs.Type), "--location", gcs.Location}
	// dest_dir is the destination directory for GCS files to be copies"
	if source.TargetPath != "" {
		args = append(args, "--dest_dir", filepath.Join(workspaceDir, source.TargetPath))
	}

	// source name is empty then use `build-step-gcs-source` name
	containerName := initContainerPrefix + gcsSource + "-"

	// update container name to include `name` as suffix
	if source.Name != "" {
		containerName = containerName + source.Name
	} else {
		containerName = containerName + strconv.Itoa(index)
	}

	return &corev1.Container{
		Name:         containerName,
		Image:        *gcsFetcherImage,
		Args:         args,
		VolumeMounts: implicitVolumeMounts,
		WorkingDir:   workspaceDir,
		Env:          implicitEnvVars,
	}, nil
}

func customToContainer(source *corev1.Container, name string) (*corev1.Container, error) {
	if source.Name != "" {
		return nil, apis.ErrMissingField("b.spec.source.name")
	}
	custom := source.DeepCopy()

	// source name is empty then use `custom-source` name
	if name == "" {
		name = customSource
	} else {
		name = customSource + "-" + name
	}
	custom.Name = name
	return custom, nil
}

func makeCredentialInitializer(build *v1alpha1.Build, kubeclient kubernetes.Interface) (*corev1.Container, []corev1.Volume, error) {
	serviceAccountName := build.Spec.ServiceAccountName
	if serviceAccountName == "" {
		serviceAccountName = "default"
	}

	sa, err := kubeclient.CoreV1().ServiceAccounts(build.Namespace).Get(serviceAccountName, metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}

	builders := []credentials.Builder{dockercreds.NewBuilder(), gitcreds.NewBuilder()}

	// Collect the volume declarations, there mounts into the cred-init container, and the arguments to it.
	volumes := []corev1.Volume{}
	volumeMounts := implicitVolumeMounts
	args := []string{}
	for _, secretEntry := range sa.Secrets {
		secret, err := kubeclient.CoreV1().Secrets(build.Namespace).Get(secretEntry.Name, metav1.GetOptions{})
		if err != nil {
			return nil, nil, err
		}

		matched := false
		for _, b := range builders {
			if sa := b.MatchingAnnotations(secret); len(sa) > 0 {
				matched = true
				args = append(args, sa...)
			}
		}

		if matched {
			name := fmt.Sprintf("secret-volume-%s", secret.Name)
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      name,
				MountPath: credentials.VolumeName(secret.Name),
			})
			volumes = append(volumes, corev1.Volume{
				Name: name,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: secret.Name,
					},
				},
			})
		}
	}

	return &corev1.Container{
		Name:         initContainerPrefix + credsInit,
		Image:        *credsImage,
		Args:         args,
		VolumeMounts: volumeMounts,
		Env:          implicitEnvVars,
		WorkingDir:   workspaceDir,
	}, volumes, nil
}

// MakePod converts a Build object to a Pod which implements the build specified
// by the supplied CRD.
func MakePod(build *v1alpha1.Build, kubeclient kubernetes.Interface) (*corev1.Pod, error) {
	build = build.DeepCopy()

	// Copy annotations on the build through to the underlying pod to allow users
	// to specify pod annotations.
	annotations := map[string]string{}
	for key, val := range build.Annotations {
		annotations[key] = val
	}
	annotations["sidecar.istio.io/inject"] = "false"

	cred, secrets, err := makeCredentialInitializer(build, kubeclient)
	if err != nil {
		return nil, err
	}

	initContainers := []corev1.Container{*cred}
	var sources []v1alpha1.SourceSpec
	// if source is present convert into sources
	if source := build.Spec.Source; source != nil {
		sources = []v1alpha1.SourceSpec{*source}
	}
	for _, source := range build.Spec.Sources {
		sources = append(sources, source)
	}
	workspaceSubPath := ""

	for i, source := range sources {
		switch {
		case source.Git != nil:
			git, err := gitToContainer(source, i)
			if err != nil {
				return nil, err
			}
			initContainers = append(initContainers, *git)
		case source.GCS != nil:
			gcs, err := gcsToContainer(source, i)
			if err != nil {
				return nil, err
			}
			initContainers = append(initContainers, *gcs)
		case source.Custom != nil:
			cust, err := customToContainer(source.Custom, source.Name)
			if err != nil {
				return nil, err
			}
			// Prepend the custom container to the steps, to be augmented later with env, volume mounts, etc.
			build.Spec.Steps = append([]corev1.Container{*cust}, build.Spec.Steps...)
		}
		// webhook validation checks that only one source has subPath defined
		workspaceSubPath = source.SubPath
	}

	for i, step := range build.Spec.Steps {
		step.Env = append(implicitEnvVars, step.Env...)
		// TODO(mattmoor): Check that volumeMounts match volumes.

		// Add implicit volume mounts, unless the user has requested
		// their own volume mount at that path.
		requestedVolumeMounts := sets.NewString()
		for _, vm := range step.VolumeMounts {
			requestedVolumeMounts.Insert(filepath.Clean(vm.MountPath))
		}
		for _, imp := range implicitVolumeMounts {
			if !requestedVolumeMounts.Has(filepath.Clean(imp.MountPath)) {
				// If the build's source specifies a subpath,
				// use that in the implicit workspace volume
				// mount.
				if workspaceSubPath != "" && imp.Name == "workspace" {
					imp.SubPath = workspaceSubPath
				}
				step.VolumeMounts = append(step.VolumeMounts, imp)
			}
		}

		if step.WorkingDir == "" {
			step.WorkingDir = workspaceDir
		}
		if step.Name == "" {
			step.Name = fmt.Sprintf("%v%d", unnamedInitContainerPrefix, i)
		} else {
			step.Name = fmt.Sprintf("%v%v", initContainerPrefix, step.Name)
		}

		initContainers = append(initContainers, step)
	}
	// Add our implicit volumes and any volumes needed for secrets to the explicitly
	// declared user volumes.
	volumes := append(build.Spec.Volumes, implicitVolumes...)
	volumes = append(volumes, secrets...)
	if err := v1alpha1.ValidateVolumes(volumes); err != nil {
		return nil, err
	}

	var podName string
	if build.Status.Cluster != nil && build.Status.Cluster.PodName != "" {
		podName = build.Status.Cluster.PodName
	} else {
		return nil, fmt.Errorf("Can't create pod for build %q: pod name not set", build.Name)
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			// We execute the build's pod in the same namespace as where the build was
			// created so that it can access colocated resources.
			Namespace: build.Namespace,
			Name:      podName,
			// If our parent Build is deleted, then we should be as well.
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(build, schema.GroupVersionKind{
					Group:   v1alpha1.SchemeGroupVersion.Group,
					Version: v1alpha1.SchemeGroupVersion.Version,
					Kind:    "Build",
				}),
			},
			Annotations: annotations,
			Labels: map[string]string{
				buildNameLabelKey: build.Name,
			},
		},
		Spec: corev1.PodSpec{
			// If the build fails, don't restart it.
			RestartPolicy:  corev1.RestartPolicyNever,
			InitContainers: initContainers,
			Containers: []corev1.Container{{
				Name:  "nop",
				Image: *nopImage,
			}},
			ServiceAccountName: build.Spec.ServiceAccountName,
			Volumes:            volumes,
			NodeSelector:       build.Spec.NodeSelector,
			Affinity:           build.Spec.Affinity,
		},
	}, nil
}

// GetUniquePodName returns a unique name based on the build's name.
func GetUniquePodName(name string) (string, error) {
	// Generate a short random hex string.
	b, err := ioutil.ReadAll(io.LimitReader(randReader, 3))
	if err != nil {
		return "", err
	}

	gibberish := hex.EncodeToString(b)
	return fmt.Sprintf("%s-pod-%s", name, gibberish), nil
}

// BuildStatusFromPod returns a BuildStatus based on the Pod and the original BuildSpec.
func BuildStatusFromPod(p *corev1.Pod, buildSpec v1alpha1.BuildSpec) v1alpha1.BuildStatus {
	status := v1alpha1.BuildStatus{
		Builder: v1alpha1.ClusterBuildProvider,
		Cluster: &v1alpha1.ClusterSpec{
			Namespace: p.Namespace,
			PodName:   p.Name,
		},
		StartTime: &p.CreationTimestamp,
	}

	// Always ignore the first pod status, which is creds-init.
	skip := 1
	if buildSpec.Source != nil {
		// If the build specifies source, skip another container status, which
		// is the source-fetching container.
		skip++
	}
	// Also skip multiple sourcees specified by the build.
	skip += len(buildSpec.Sources)
	if skip <= len(p.Status.InitContainerStatuses) {
		for _, s := range p.Status.InitContainerStatuses[skip:] {
			if s.State.Terminated != nil {
				status.StepsCompleted = append(status.StepsCompleted, s.Name)
			}
			status.StepStates = append(status.StepStates, s.State)
		}
	}

	switch p.Status.Phase {
	case corev1.PodRunning:
		status.SetCondition(&duckv1alpha1.Condition{
			Type:   v1alpha1.BuildSucceeded,
			Status: corev1.ConditionUnknown,
			Reason: "Building",
		})
	case corev1.PodFailed:
		msg := getFailureMessage(p)
		status.SetCondition(&duckv1alpha1.Condition{
			Type:    v1alpha1.BuildSucceeded,
			Status:  corev1.ConditionFalse,
			Message: msg,
		})
	case corev1.PodPending:
		msg := getWaitingMessage(p)
		status.SetCondition(&duckv1alpha1.Condition{
			Type:    v1alpha1.BuildSucceeded,
			Status:  corev1.ConditionUnknown,
			Reason:  "Pending",
			Message: msg,
		})
	case corev1.PodSucceeded:
		status.SetCondition(&duckv1alpha1.Condition{
			Type:   v1alpha1.BuildSucceeded,
			Status: corev1.ConditionTrue,
		})
	}
	return status
}

func getWaitingMessage(pod *corev1.Pod) string {
	// First, try to surface reason for pending/unknown about the actual build step.
	for _, status := range pod.Status.InitContainerStatuses {
		wait := status.State.Waiting
		if wait != nil && wait.Message != "" {
			return fmt.Sprintf("build step %q is pending with reason %q",
				status.Name, wait.Message)
		}
	}
	// Try to surface underlying reason by inspecting pod's recent status if condition is not true
	for i, podStatus := range pod.Status.Conditions {
		if podStatus.Status != corev1.ConditionTrue {
			return fmt.Sprintf("pod status %q:%q; message: %q",
				pod.Status.Conditions[i].Type,
				pod.Status.Conditions[i].Status,
				pod.Status.Conditions[i].Message)
		}
	}
	// Next, return the Pod's status message if it has one.
	if pod.Status.Message != "" {
		return pod.Status.Message
	}

	// Lastly fall back on a generic pending message.
	return "Pending"
}

func getFailureMessage(pod *corev1.Pod) string {
	// First, try to surface an error about the actual build step that failed.
	for _, status := range pod.Status.InitContainerStatuses {
		term := status.State.Terminated
		if term != nil && term.ExitCode != 0 {
			return fmt.Sprintf("build step %q exited with code %d (image: %q); for logs run: kubectl -n %s logs %s -c %s",
				status.Name, term.ExitCode, status.ImageID,
				pod.Namespace, pod.Name, status.Name)
		}
	}
	// Next, return the Pod's status message if it has one.
	if pod.Status.Message != "" {
		return pod.Status.Message
	}
	// Lastly fall back on a generic error message.
	return "build failed for unspecified reasons."
}
