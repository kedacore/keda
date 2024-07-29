//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package sas

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/internal/exported"
)

// QueueSignatureValues is used to generate a Shared Access Signature (SAS) for an Azure Storage Queue.
// For more information on creating service sas, see https://docs.microsoft.com/rest/api/storageservices/constructing-a-service-sas
// Delegation SAS not supported for queues service
type QueueSignatureValues struct {
	Version     string    `param:"sv"`  // If not specified, this defaults to Version
	Protocol    Protocol  `param:"spr"` // See the Protocol* constants
	StartTime   time.Time `param:"st"`  // Not specified if IsZero
	ExpiryTime  time.Time `param:"se"`  // Not specified if IsZero
	Permissions string    `param:"sp"`  // Create by initializing a QueuePermissions and then call String()
	IPRange     IPRange   `param:"sip"`
	Identifier  string    `param:"si"`
	QueueName   string
}

// SignWithSharedKey uses an account's SharedKeyCredential to sign this signature values to produce the proper SAS query parameters.
func (v QueueSignatureValues) SignWithSharedKey(sharedKeyCredential *SharedKeyCredential) (QueryParameters, error) {
	if v.ExpiryTime.IsZero() || v.Permissions == "" {
		return QueryParameters{}, errors.New("service SAS is missing at least one of these: ExpiryTime or Permissions")
	}

	//Make sure the permission characters are in the correct order
	perms, err := parseQueuePermissions(v.Permissions)
	if err != nil {
		return QueryParameters{}, err
	}
	v.Permissions = perms.String()
	if v.Version == "" {
		v.Version = Version
	}
	startTime, expiryTime := formatTimesForSigning(v.StartTime, v.ExpiryTime)

	signedIdentifier := v.Identifier

	// String to sign: http://msdn.microsoft.com/en-us/library/azure/dn140255.aspx
	stringToSign := strings.Join([]string{
		v.Permissions,
		startTime,
		expiryTime,
		getCanonicalName(sharedKeyCredential.AccountName(), v.QueueName),
		signedIdentifier,
		v.IPRange.String(),
		string(v.Protocol),
		v.Version},
		"\n")

	signature, err := exported.ComputeHMACSHA256(sharedKeyCredential, stringToSign)
	if err != nil {
		return QueryParameters{}, err
	}

	p := QueryParameters{
		// Common SAS parameters
		version:     v.Version,
		protocol:    v.Protocol,
		startTime:   v.StartTime,
		expiryTime:  v.ExpiryTime,
		permissions: v.Permissions,
		ipRange:     v.IPRange,
		// Calculated SAS signature
		signature: signature,
	}

	return p, nil
}

// getCanonicalName computes the canonical name for a queue resource for SAS signing.
func getCanonicalName(account string, queueName string) string {
	elements := []string{"/queue/", account, "/", queueName}
	return strings.Join(elements, "")
}

// QueuePermissions type simplifies creating the permissions string for an Azure Storage Queue SAS.
// Initialize an instance of this type and then call its String method to set QueueSignatureValues' Permissions field.
type QueuePermissions struct {
	Read, Add, Update, Process bool
}

// String produces the SAS permissions string for an Azure Storage Queue.
// Call this method to set QueueSignatureValues' Permissions field.
func (p *QueuePermissions) String() string {
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

// Parse initializes the QueuePermissions' fields from a string.
func parseQueuePermissions(s string) (QueuePermissions, error) {
	p := QueuePermissions{}
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
			return QueuePermissions{}, fmt.Errorf("invalid permission: '%v'", r)
		}
	}
	return p, nil
}
