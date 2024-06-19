//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package exported

import (
	"bytes"
	"fmt"
)

// AccessPolicyPermission type simplifies creating the permissions string for a queue's access policy.
// Initialize an instance of this type and then call its String method to set AccessPolicy's Permission field.
type AccessPolicyPermission struct {
	Read, Add, Update, Process bool
}

// String produces the access policy permission string for an Azure Storage queue.
// Call this method to set AccessPolicy's Permission field.
func (p *AccessPolicyPermission) String() string {
	var b bytes.Buffer
	if p.Read {
		b.WriteRune('r')
	}
	if p.Add {
		b.WriteRune('a')
	}
	if p.Update {
		b.WriteRune('u')
	}
	if p.Process {
		b.WriteRune('p')
	}
	return b.String()
}

// Parse initializes the AccessPolicyPermission's fields from a string.
func (p *AccessPolicyPermission) Parse(s string) error {
	*p = AccessPolicyPermission{} // Clear the flags
	for _, r := range s {
		switch r {
		case 'r':
			p.Read = true
		case 'a':
			p.Add = true
		case 'u':
			p.Update = true
		case 'p':
			p.Process = true
		default:
			return fmt.Errorf("invalid permission: '%v'", r)
		}
	}
	return nil
}
