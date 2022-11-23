// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package internal // import "go.mongodb.org/mongo-driver/internal"

import (
	"net/http"
	"time"
)

// DefaultHTTPClient is the default HTTP client used across the driver.
var DefaultHTTPClient = &http.Client{
	// TODO(GODRIVER-2623): Use "http.DefaultTransport.Clone" once we change the minimum supported Go version to 1.13.
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}

// CloseIdleHTTPConnections closes any connections which were previously
// connected from previous requests but are now sitting idle in
// a "keep-alive" state. It does not interrupt any connections currently
// in use.
// Borrowed from go standard library.
func CloseIdleHTTPConnections(client *http.Client) {
	type closeIdler interface {
		CloseIdleConnections()
	}
	if tr, ok := client.Transport.(closeIdler); ok {
		tr.CloseIdleConnections()
	}
}
