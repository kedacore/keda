/*
Copyright 2024 The KEDA Authors

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

package aws

type AuthorizationMetadata struct {
	AwsRoleArn        string
	AwsRoleExternalID string

	AwsAccessKeyID     string
	AwsSecretAccessKey string
	AwsSessionToken    string

	AwsRegion string

	// Deprecated
	PodIdentityOwner bool
	// Pod identity owner is confusing and it'll be removed when we get
	// rid of the old aws podIdentities (aws-eks and aws-kiam) as UsingPodIdentity
	// replaces it. For more context:
	// https://github.com/kedacore/keda/pull/5061/#discussion_r1441016441
	UsingPodIdentity bool

	TriggerUniqueKey string
}
