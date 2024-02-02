//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package sas

import (
	"net/url"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/internal/shared"
)

// IPEndpointStyleInfo is used for IP endpoint style URL when working with Azure storage emulator.
// Ex: "https://10.132.141.33/accountname/queuename"
type IPEndpointStyleInfo struct {
	AccountName string // "" if not using IP endpoint style
}

// URLParts object represents the components that make up an Azure Storage Queue URL.
// NOTE: Changing any SAS-related field requires computing a new SAS signature.
type URLParts struct {
	Scheme              string // Ex: "https://"
	Host                string // Ex: "account.queue.core.windows.net", "10.132.141.33", "10.132.141.33:80"
	IPEndpointStyleInfo IPEndpointStyleInfo
	QueueName           string // "" if no queue
	SAS                 QueryParameters
	UnparsedParams      string
}

// ParseURL parses a URL initializing URLParts' fields including any SAS-related query parameters.
// Any other query parameters remain in the UnparsedParams field.
func ParseURL(u string) (URLParts, error) {
	uri, err := url.Parse(u)
	if err != nil {
		return URLParts{}, err
	}

	up := URLParts{
		Scheme: uri.Scheme,
		Host:   uri.Host,
	}

	// Find the queue name (if any)
	if uri.Path != "" {
		path := uri.Path
		if path[0] == '/' {
			path = path[1:] // If path starts with a slash, remove it
		}
		if shared.IsIPEndpointStyle(up.Host) {
			if accountEndIndex := strings.Index(path, "/"); accountEndIndex == -1 { // Slash not found; path has account name & no queue name
				up.IPEndpointStyleInfo.AccountName = path
				path = "" // No queue name present in the URL so path should be empty
			} else {
				up.IPEndpointStyleInfo.AccountName = path[:accountEndIndex] // The account name is the part between the slashes
				path = path[accountEndIndex+1:]                             // path refers to portion after the account name now (queue name)
			}
		}
		up.QueueName = path
	}

	// Convert the query parameters to a case-sensitive map & trim whitespace
	paramsMap := uri.Query()
	up.SAS = newQueryParameters(paramsMap, true)
	up.UnparsedParams = paramsMap.Encode()
	return up, nil
}

// String returns a URL object whose fields are initialized from the URLParts fields. The URL's RawQuery
// field contains the SAS and unparsed query parameters.
func (up URLParts) String() string {
	path := ""
	if shared.IsIPEndpointStyle(up.Host) && up.IPEndpointStyleInfo.AccountName != "" {
		path += "/" + up.IPEndpointStyleInfo.AccountName
	}
	// Concatenate queue name
	if up.QueueName != "" {
		path += "/" + up.QueueName
	}

	rawQuery := up.UnparsedParams
	sas := up.SAS.Encode()
	if sas != "" {
		if len(rawQuery) > 0 {
			rawQuery += "&"
		}
		rawQuery += sas
	}
	u := url.URL{
		Scheme:   up.Scheme,
		Host:     up.Host,
		Path:     path,
		RawQuery: rawQuery,
	}
	return u.String()
}
