//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package shared

import (
	"errors"
	"fmt"
	"hash/crc64"
	"net"
	"strings"
)

const (
	TokenScope = "https://storage.azure.com/.default"
)

const (
	HeaderAuthorization     = "Authorization"
	HeaderXmsDate           = "x-ms-date"
	HeaderContentLength     = "Content-Length"
	HeaderContentEncoding   = "Content-Encoding"
	HeaderContentLanguage   = "Content-Language"
	HeaderContentType       = "Content-Type"
	HeaderContentMD5        = "Content-MD5"
	HeaderIfModifiedSince   = "If-Modified-Since"
	HeaderIfMatch           = "If-Match"
	HeaderIfNoneMatch       = "If-None-Match"
	HeaderIfUnmodifiedSince = "If-Unmodified-Since"
	HeaderRange             = "Range"
)

const crc64Polynomial uint64 = 0x9A6C9329AC4BC9B5

var CRC64Table = crc64.MakeTable(crc64Polynomial)

// CopyOptions returns a zero-value T if opts is nil.
// If opts is not nil, a copy is made and its address returned.
func CopyOptions[T any](opts *T) *T {
	if opts == nil {
		return new(T)
	}
	cp := *opts
	return &cp
}

var errConnectionString = errors.New("connection string is either blank or malformed. The expected connection string " +
	"should contain key value pairs separated by semicolons. For example 'DefaultEndpointsProtocol=https;AccountName=<accountName>;" +
	"AccountKey=<accountKey>;EndpointSuffix=core.windows.net'")

type ParsedConnectionString struct {
	ServiceURL  string
	AccountName string
	AccountKey  string
}

func ParseConnectionString(connectionString string) (ParsedConnectionString, error) {
	const (
		defaultScheme = "https"
		defaultSuffix = "core.windows.net"
	)

	connStrMap := make(map[string]string)
	connectionString = strings.TrimRight(connectionString, ";")

	splitString := strings.Split(connectionString, ";")
	if len(splitString) == 0 {
		return ParsedConnectionString{}, errConnectionString
	}
	for _, stringPart := range splitString {
		parts := strings.SplitN(stringPart, "=", 2)
		if len(parts) != 2 {
			return ParsedConnectionString{}, errConnectionString
		}
		connStrMap[parts[0]] = parts[1]
	}

	protocol, ok := connStrMap["DefaultEndpointsProtocol"]
	if !ok {
		protocol = defaultScheme
	}

	suffix, ok := connStrMap["EndpointSuffix"]
	if !ok {
		suffix = defaultSuffix
	}

	queueEndpoint, has_queueEndpoint := connStrMap["QueueEndpoint"]
	accountName, has_accountName := connStrMap["AccountName"]

	var serviceURL string
	if has_queueEndpoint {
		serviceURL = queueEndpoint
	} else if has_accountName {
		serviceURL = fmt.Sprintf("%v://%v.queue.%v", protocol, accountName, suffix)
	} else {
		return ParsedConnectionString{}, errors.New("connection string needs either AccountName or QueueEndpoint")
	}

	if !strings.HasSuffix(serviceURL, "/") {
		// add a trailing slash to be consistent with the portal
		serviceURL += "/"
	}

	accountKey, has_accountKey := connStrMap["AccountKey"]
	sharedAccessSignature, has_sharedAccessSignature := connStrMap["SharedAccessSignature"]

	if has_accountName && has_accountKey {
		return ParsedConnectionString{
			ServiceURL:  serviceURL,
			AccountName: accountName,
			AccountKey:  accountKey,
		}, nil
	} else if has_sharedAccessSignature {
		return ParsedConnectionString{
			ServiceURL: fmt.Sprintf("%v?%v", serviceURL, sharedAccessSignature),
		}, nil
	} else {
		return ParsedConnectionString{}, errors.New("connection string needs either AccountKey or SharedAccessSignature")
	}
}

func GetClientOptions[T any](o *T) *T {
	if o == nil {
		return new(T)
	}
	return o
}

// IsIPEndpointStyle checkes if URL's host is IP, in this case the storage account endpoint will be composed as:
// http(s)://IP(:port)/storageaccount/queue/...
// As url's Host property, host could be both host or host:port
func IsIPEndpointStyle(host string) bool {
	if host == "" {
		return false
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	// For IPv6, there could be case where SplitHostPort fails for cannot finding port.
	// In this case, eliminate the '[' and ']' in the URL.
	// For details about IPv6 URL, please refer to https://tools.ietf.org/html/rfc2732
	if host[0] == '[' && host[len(host)-1] == ']' {
		host = host[1 : len(host)-1]
	}
	return net.ParseIP(host) != nil
}
