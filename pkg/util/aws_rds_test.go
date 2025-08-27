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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeRDSSecret(t *testing.T) {
	tests := []struct {
		name        string
		secretJSON  string
		wantErr     bool
		errContains string
		validate    func(t *testing.T, secret *RDSSecret)
	}{
		{
			name:       "valid secret with all fields",
			secretJSON: `{"username":"dbuser","password":"secret123","engine":"mysql","host":"my-db.123456789012.us-east-1.rds.amazonaws.com","port":3306,"dbname":"mydb","dbInstanceIdentifier":"my-db-instance"}`,
			wantErr:    false,
			validate: func(t *testing.T, secret *RDSSecret) {
				assert.Equal(t, "dbuser", secret.Username)
				assert.Equal(t, "secret123", secret.Password)
				assert.Equal(t, "mysql", secret.Engine)
				assert.Equal(t, "my-db.123456789012.us-east-1.rds.amazonaws.com", secret.Host)
				assert.Equal(t, 3306, secret.Port)
				assert.Equal(t, "mydb", secret.DBName)
				assert.Equal(t, "my-db-instance", secret.DBInstanceIdentifier)
			},
		},
		{
			name:       "valid secret with minimal fields",
			secretJSON: `{"username":"dbuser","host":"my-db.123456789012.us-east-1.rds.amazonaws.com","port":3306,"dbname":"mydb"}`,
			wantErr:    false,
			validate: func(t *testing.T, secret *RDSSecret) {
				assert.Equal(t, "dbuser", secret.Username)
				assert.Equal(t, "", secret.Password)
				assert.Equal(t, "", secret.Engine)
				assert.Equal(t, "my-db.123456789012.us-east-1.rds.amazonaws.com", secret.Host)
				assert.Equal(t, 3306, secret.Port)
				assert.Equal(t, "mydb", secret.DBName)
				assert.Equal(t, "", secret.DBInstanceIdentifier)
			},
		},
		{
			name:       "valid secret with empty password",
			secretJSON: `{"username":"dbuser","password":"","host":"my-db.123456789012.us-east-1.rds.amazonaws.com","port":3306,"dbname":"mydb"}`,
			wantErr:    false,
			validate: func(t *testing.T, secret *RDSSecret) {
				assert.Equal(t, "dbuser", secret.Username)
				assert.Equal(t, "", secret.Password)
				assert.Equal(t, "", secret.Engine)
				assert.Equal(t, "my-db.123456789012.us-east-1.rds.amazonaws.com", secret.Host)
				assert.Equal(t, 3306, secret.Port)
				assert.Equal(t, "mydb", secret.DBName)
			},
		},
		{
			name:        "missing required field - username",
			secretJSON:  `{"password":"secret123","host":"my-db.123456789012.us-east-1.rds.amazonaws.com","port":3306,"dbname":"mydb"}`,
			wantErr:     true,
			errContains: "missing required field: username",
		},
		{
			name:        "missing required field - host",
			secretJSON:  `{"username":"dbuser","password":"secret123","port":3306,"dbname":"mydb"}`,
			wantErr:     true,
			errContains: "missing required field: host",
		},
		{
			name:        "missing required field - port",
			secretJSON:  `{"username":"dbuser","password":"secret123","host":"my-db.123456789012.us-east-1.rds.amazonaws.com","dbname":"mydb"}`,
			wantErr:     true,
			errContains: "missing required field: port",
		},
		{
			name:        "missing required field - dbname",
			secretJSON:  `{"username":"dbuser","password":"secret123","host":"my-db.123456789012.us-east-1.rds.amazonaws.com","port":3306}`,
			wantErr:     true,
			errContains: "missing required field: dbname",
		},
		{
			name:        "invalid JSON",
			secretJSON:  `{"username":"dbuser","password":"secret123","host":"my-db.123456789012.us-east-1.rds.amazonaws.com","port":3306,"dbname":"mydb"`,
			wantErr:     true,
			errContains: "failed to decode RDS secret JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret, err := DecodeRDSSecret(tt.secretJSON)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, secret)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, secret)
				if tt.validate != nil {
					tt.validate(t, secret)
				}
			}
		})
	}
}
