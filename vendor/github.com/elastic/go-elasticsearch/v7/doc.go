// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

/*
Package elasticsearch provides a Go client for Elasticsearch.

Create the client with the NewDefaultClient function:

		elasticsearch.NewDefaultClient()

The ELASTICSEARCH_URL environment variable is used instead of the default URL, when set.
Use a comma to separate multiple URLs.

To configure the client, pass a Config object to the NewClient function:

		cfg := elasticsearch.Config{
		  Addresses: []string{
		    "http://localhost:9200",
		    "http://localhost:9201",
		  },
		  Username: "foo",
		  Password: "bar",
		  Transport: &http.Transport{
		    MaxIdleConnsPerHost:   10,
		    ResponseHeaderTimeout: time.Second,
		    DialContext:           (&net.Dialer{Timeout: time.Second}).DialContext,
		    TLSClientConfig: &tls.Config{
		      MinVersion:         tls.VersionTLS12,
		    },
		  },
		}

		elasticsearch.NewClient(cfg)

When using the Elastic Service (https://elastic.co/cloud), you can use CloudID instead of Addresses.
When either Addresses or CloudID is set, the ELASTICSEARCH_URL environment variable is ignored.

See the elasticsearch_integration_test.go file and the _examples folder for more information.

Call the Elasticsearch APIs by invoking the corresponding methods on the client:

		res, err := es.Info()
		if err != nil {
		  log.Fatalf("Error getting response: %s", err)
		}

		log.Println(res)

See the github.com/elastic/go-elasticsearch/esapi package for more information about using the API.

See the github.com/elastic/go-elasticsearch/estransport package for more information about configuring the transport.
*/
package elasticsearch
