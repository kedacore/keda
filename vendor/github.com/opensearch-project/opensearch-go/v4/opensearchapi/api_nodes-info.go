// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4"
)

// NodesInfoReq represents possible options for the /_nodes request
type NodesInfoReq struct {
	Metrics []string
	NodeID  []string

	Header http.Header
	Params NodesInfoParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r NodesInfoReq) GetRequest() (*http.Request, error) {
	nodes := strings.Join(r.NodeID, ",")
	metrics := strings.Join(r.Metrics, ",")

	var path strings.Builder

	path.Grow(len("/_nodes//") + len(nodes) + len(metrics))

	path.WriteString("/_nodes")
	if len(r.NodeID) > 0 {
		path.WriteString("/")
		path.WriteString(nodes)
	}
	if len(r.Metrics) > 0 {
		path.WriteString("/")
		path.WriteString(metrics)
	}

	return opensearch.BuildRequest(
		"GET",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// NodesInfoResp represents the returned struct of the /_nodes response
type NodesInfoResp struct {
	NodesInfo struct {
		Total      int             `json:"total"`
		Successful int             `json:"successful"`
		Failed     int             `json:"failed"`
		Failures   []FailuresCause `json:"failures"`
	} `json:"_nodes"`
	ClusterName string               `json:"cluster_name"`
	Nodes       map[string]NodesInfo `json:"nodes"`
	response    *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r NodesInfoResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// NodesInfo is a sub type of NodesInfoResp containing information about nodes and their stats
type NodesInfo struct {
	Name                       string                         `json:"name"`
	TransportAddress           string                         `json:"transport_address"`
	Host                       string                         `json:"host"`
	IP                         string                         `json:"ip"`
	Version                    string                         `json:"version"`
	BuildType                  string                         `json:"build_type"`
	BuildHash                  string                         `json:"build_hash"`
	TotalIndexingBuffer        int64                          `json:"total_indexing_buffer"`
	TotalIndexingBufferInBytes int64                          `json:"total_indexing_buffer_in_bytes"`
	Roles                      []string                       `json:"roles"`
	Attributes                 map[string]string              `json:"attributes"`
	Settings                   json.RawMessage                `json:"settings"` // Won't parse as it may contain fields that we can't know
	OS                         NodesInfoOS                    `json:"os"`
	Process                    NodesInfoProcess               `json:"process"`
	JVM                        NodesInfoJVM                   `json:"jvm"`
	ThreadPool                 map[string]NodesInfoThreadPool `json:"thread_pool"`
	Transport                  NodesInfoTransport             `json:"transport"`
	HTTP                       NodesInfoHTTP                  `json:"http"`
	Plugins                    []NodesInfoPlugin              `json:"plugins"`
	Modules                    []NodesInfoPlugin              `json:"modules"`
	Ingest                     NodesInfoIngest                `json:"ingest"`
	Aggregations               map[string]NodesInfoAgg        `json:"aggregations"`
	SearchPipelines            NodesInfoSearchPipelines       `json:"search_pipelines"`
}

// NodesInfoOS is a sub type of NodesInfo containing information about the Operating System
type NodesInfoOS struct {
	RefreshIntervalInMillis int    `json:"refresh_interval_in_millis"`
	Name                    string `json:"name"`
	PrettyName              string `json:"pretty_name"`
	Arch                    string `json:"arch"`
	Version                 string `json:"version"`
	AvailableProcessors     int    `json:"available_processors"`
	AllocatedProcessors     int    `json:"allocated_processors"`
}

// NodesInfoProcess is a sub type of NodesInfo containing information about the nodes process
type NodesInfoProcess struct {
	RefreshIntervalInMillis int  `json:"refresh_interval_in_millis"`
	ID                      int  `json:"id"`
	Mlockall                bool `json:"mlockall"`
}

// NodesInfoJVM is a sub type of NodesInfo containing information and stats about JVM
type NodesInfoJVM struct {
	PID               int    `json:"pid"`
	Version           string `json:"version"`
	VMName            string `json:"vm_name"`
	VMVersion         string `json:"vm_version"`
	VMVendor          string `json:"vm_vendor"`
	BundledJDK        bool   `json:"bundled_jdk"`
	UsingBundledJDK   bool   `json:"using_bundled_jdk"`
	StartTimeInMillis int64  `json:"start_time_in_millis"`
	Mem               struct {
		HeapInitInBytes    int64 `json:"heap_init_in_bytes"`
		HeapMaxInBytes     int64 `json:"heap_max_in_bytes"`
		NonHeapInitInBytes int   `json:"non_heap_init_in_bytes"`
		NonHeapMaxInBytes  int   `json:"non_heap_max_in_bytes"`
		DirectMaxInBytes   int   `json:"direct_max_in_bytes"`
	} `json:"mem"`
	GcCollectors                          []string `json:"gc_collectors"`
	MemoryPools                           []string `json:"memory_pools"`
	UsingCompressedOrdinaryObjectPointers string   `json:"using_compressed_ordinary_object_pointers"`
	InputArguments                        []string `json:"input_arguments"`
}

// NodesInfoThreadPool is a sub type of NodesInfo containing information about a thread pool
type NodesInfoThreadPool struct {
	Type      string `json:"type"`
	Size      int    `json:"size"`
	QueueSize int    `json:"queue_size"`
	KeepAlive string `json:"keep_alive"`
	Max       int    `json:"max"`
	Core      int    `json:"core"`
}

// NodesInfoTransport is a sub type of NodesInfo containing information about the nodes transport settings
type NodesInfoTransport struct {
	BoundAddress   []string        `json:"bound_address"`
	PublishAddress string          `json:"publish_address"`
	Profiles       json.RawMessage `json:"profiles"` // Unknown content
}

// NodesInfoHTTP is a sub type of NodesInfo containing information about the nodes http settings
type NodesInfoHTTP struct {
	BoundAddress            []string `json:"bound_address"`
	PublishAddress          string   `json:"publish_address"`
	MaxContentLengthInBytes int64    `json:"max_content_length_in_bytes"`
}

// NodesInfoPlugin is a sub type of NodesInfo containing information about a plugin
type NodesInfoPlugin struct {
	Name                    string   `json:"name"`
	Version                 string   `json:"version"`
	OpensearchVersion       string   `json:"opensearch_version"`
	JavaVersion             string   `json:"java_version"`
	Description             string   `json:"description"`
	Classname               string   `json:"classname"`
	CustomFoldername        *string  `json:"custom_foldername"`
	ExtendedPlugins         []string `json:"extended_plugins"`
	HasNativeController     bool     `json:"has_native_controller"`
	OptionalExtendedPlugins []string `json:"optional_extended_plugins"`
}

// NodesInfoIngest is a sub type of NodesInfo containing information about ingest processors
type NodesInfoIngest struct {
	Processors []NodesInfoType `json:"processors"`
}

// NodesInfoAgg is a sub type of NodesInfo containing information about aggregations
type NodesInfoAgg struct {
	Types []string `json:"types"`
}

// NodesInfoSearchPipelines is a sub type of NodesInfo containing information about search pipelines
type NodesInfoSearchPipelines struct {
	RequestProcessors  []NodesInfoType `json:"request_processors"`
	ResponseProcessors []NodesInfoType `json:"response_processors"`
	Processors         []NodesInfoType `json:"processors,omitempty"` // Deprecated field only available in 2.7.0
}

// NodesInfoType is a sub type of NodesInfoIngest, NodesInfoSearchPipelines containing information about a type
type NodesInfoType struct {
	Type string `json:"type"`
}
