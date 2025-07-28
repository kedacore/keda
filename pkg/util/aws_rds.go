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

package util

import (
	"encoding/json"
	"fmt"
)

// RDSSecret represents the structure of an AWS RDS secret
type RDSSecret struct {
	Username             string `json:"username"`
	Password             string `json:"password"`
	Engine               string `json:"engine"`
	Host                 string `json:"host"`
	Port                 int    `json:"port"`
	DBName               string `json:"dbname"`
	DBInstanceIdentifier string `json:"dbInstanceIdentifier,omitempty"`
}

// DecodeRDSSecret takes a JSON string and validates that it matches the AWS RDS secret structure.
// Returns the decoded secret and any validation errors.
func DecodeRDSSecret(secretString string) (*RDSSecret, error) {
	var secret RDSSecret
	if err := json.Unmarshal([]byte(secretString), &secret); err != nil {
		return nil, fmt.Errorf("failed to decode RDS secret JSON: %w", err)
	}

	// Validate required fields
	if secret.Username == "" {
		return nil, fmt.Errorf("RDS secret missing required field: username")
	}
	if secret.Host == "" {
		return nil, fmt.Errorf("RDS secret missing required field: host")
	}
	if secret.Port == 0 {
		return nil, fmt.Errorf("RDS secret missing required field: port")
	}
	if secret.DBName == "" {
		return nil, fmt.Errorf("RDS secret missing required field: dbname")
	}

	return &secret, nil
}
