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

package azure

import "time"

// AADToken is the token from Azure AD
type AADToken struct {
	AccessToken         string    `json:"access_token"`
	RefreshToken        string    `json:"refresh_token"`
	ExpiresIn           string    `json:"expires_in"`
	ExpiresOn           string    `json:"expires_on"`
	ExpiresOnTimeObject time.Time `json:"expires_on_object"`
	NotBefore           string    `json:"not_before"`
	Resource            string    `json:"resource"`
	TokenType           string    `json:"token_type"`
	GrantedScopes       []string  `json:"grantedScopes"`
	DeclinedScopes      []string  `json:"DeclinedScopes"`
}
