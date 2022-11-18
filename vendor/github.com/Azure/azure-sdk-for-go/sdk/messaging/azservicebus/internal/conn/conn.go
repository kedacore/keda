// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.
package conn

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

type (
	// ParsedConn is the structure of a parsed Service Bus or Event Hub connection string.
	ParsedConn struct {
		Namespace string
		HubName   string

		KeyName string
		Key     string

		SAS string
	}
)

// ParsedConnectionFromStr takes a string connection string from the Azure portal and returns the parsed representation.
// The method will return an error if the Endpoint, SharedAccessKeyName or SharedAccessKey is empty.
func ParsedConnectionFromStr(connStr string) (*ParsedConn, error) {
	const (
		endpointKey              = "Endpoint"
		sharedAccessKeyNameKey   = "SharedAccessKeyName"
		sharedAccessKeyKey       = "SharedAccessKey"
		entityPathKey            = "EntityPath"
		sharedAccessSignatureKey = "SharedAccessSignature"
	)

	// We can parse two types of connection strings.
	// 1. Connection strings generated from the portal (or elsewhere) that contain an embedded key and keyname.
	// 2. A specially formatted connection string with an embedded SharedAccessSignature:
	//   Endpoint=sb://<sb>.servicebus.windows.net;SharedAccessSignature=SharedAccessSignature sr=<sb>.servicebus.windows.net&sig=<base64-sig>&se=<expiry>&skn=<keyname>"
	var namespace, hubName, keyName, secret, sas string
	splits := strings.Split(connStr, ";")

	for _, split := range splits {
		keyAndValue := strings.SplitN(split, "=", 2)
		if len(keyAndValue) < 2 {
			return nil, errors.New("failed parsing connection string due to unmatched key value separated by '='")
		}

		// if a key value pair has `=` in the value, recombine them
		key := keyAndValue[0]
		value := strings.Join(keyAndValue[1:], "=")
		switch {
		case strings.EqualFold(endpointKey, key):
			u, err := url.Parse(value)
			if err != nil {
				return nil, errors.New("failed parsing connection string due to an incorrectly formatted Endpoint value")
			}
			namespace = u.Host
		case strings.EqualFold(sharedAccessKeyNameKey, key):
			keyName = value
		case strings.EqualFold(sharedAccessKeyKey, key):
			secret = value
		case strings.EqualFold(entityPathKey, key):
			hubName = value
		case strings.EqualFold(sharedAccessSignatureKey, key):
			sas = value
		}
	}

	parsed := &ParsedConn{
		Namespace: namespace,
		KeyName:   keyName,
		Key:       secret,
		HubName:   hubName,
		SAS:       sas,
	}

	if namespace == "" {
		return parsed, fmt.Errorf("key %q must not be empty", endpointKey)
	}

	if sas == "" && keyName == "" {
		return parsed, fmt.Errorf("key %q must not be empty", sharedAccessKeyNameKey)
	}

	if secret == "" && sas == "" {
		return parsed, fmt.Errorf("key %q or %q cannot both be empty", sharedAccessKeyKey, sharedAccessSignatureKey)
	}

	return parsed, nil
}
