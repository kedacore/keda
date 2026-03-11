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

// ClusterStatsReq represents possible options for the /_cluster/stats request
type ClusterStatsReq struct {
	NodeFilters []string

	Header http.Header
	Params ClusterStatsParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ClusterStatsReq) GetRequest() (*http.Request, error) {
	filters := strings.Join(r.NodeFilters, ",")

	var path strings.Builder
	path.Grow(22 + len(filters))
	path.WriteString("/_cluster/stats")
	if len(filters) > 0 {
		path.WriteString("/nodes/")
		path.WriteString(filters)
	}

	return opensearch.BuildRequest(
		"GET",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// ClusterStatsResp represents the returned struct of the ClusterStatsReq response
type ClusterStatsResp struct {
	NodesInfo struct {
		Total      int             `json:"total"`
		Successful int             `json:"successful"`
		Failed     int             `json:"failed"`
		Failures   []FailuresCause `json:"failures"`
	} `json:"_nodes"`
	ClusterName string              `json:"cluster_name"`
	ClusterUUID string              `json:"cluster_uuid"`
	Timestamp   int64               `json:"timestamp"`
	Status      string              `json:"status"`
	Indices     ClusterStatsIndices `json:"indices"`
	Nodes       ClusterStatsNodes   `json:"nodes"`
	response    *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r ClusterStatsResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// ClusterStatsIndices is a sub type of ClusterStatsResp containing cluster information about indices
type ClusterStatsIndices struct {
	Count  int `json:"count"`
	Shards struct {
		Total       int     `json:"total"`
		Primaries   int     `json:"primaries"`
		Replication float64 `json:"replication"`
		Index       struct {
			Shards struct {
				Min float64 `json:"min"`
				Max float64 `json:"max"`
				Avg float64 `json:"avg"`
			} `json:"shards"`
			Primaries struct {
				Min float64 `json:"min"`
				Max float64 `json:"max"`
				Avg float64 `json:"avg"`
			} `json:"primaries"`
			Replication struct {
				Min float64 `json:"min"`
				Max float64 `json:"max"`
				Avg float64 `json:"avg"`
			} `json:"replication"`
		} `json:"index"`
	} `json:"shards"`
	Docs struct {
		Count   int64 `json:"count"`
		Deleted int   `json:"deleted"`
	} `json:"docs"`
	Store struct {
		SizeInBytes     int64 `json:"size_in_bytes"`
		ReservedInBytes int   `json:"reserved_in_bytes"`
	} `json:"store"`
	Fielddata struct {
		MemorySizeInBytes int `json:"memory_size_in_bytes"`
		Evictions         int `json:"evictions"`
	} `json:"fielddata"`
	QueryCache struct {
		MemorySizeInBytes int `json:"memory_size_in_bytes"`
		TotalCount        int `json:"total_count"`
		HitCount          int `json:"hit_count"`
		MissCount         int `json:"miss_count"`
		CacheSize         int `json:"cache_size"`
		CacheCount        int `json:"cache_count"`
		Evictions         int `json:"evictions"`
	} `json:"query_cache"`
	Completion struct {
		SizeInBytes int `json:"size_in_bytes"`
	} `json:"completion"`
	Segments struct {
		Count                     int   `json:"count"`
		MemoryInBytes             int   `json:"memory_in_bytes"`
		TermsMemoryInBytes        int   `json:"terms_memory_in_bytes"`
		StoredFieldsMemoryInBytes int   `json:"stored_fields_memory_in_bytes"`
		TermVectorsMemoryInBytes  int   `json:"term_vectors_memory_in_bytes"`
		NormsMemoryInBytes        int   `json:"norms_memory_in_bytes"`
		PointsMemoryInBytes       int   `json:"points_memory_in_bytes"`
		DocValuesMemoryInBytes    int   `json:"doc_values_memory_in_bytes"`
		IndexWriterMemoryInBytes  int   `json:"index_writer_memory_in_bytes"`
		VersionMapMemoryInBytes   int   `json:"version_map_memory_in_bytes"`
		FixedBitSetMemoryInBytes  int64 `json:"fixed_bit_set_memory_in_bytes"`
		MaxUnsafeAutoIDTimestamp  int64 `json:"max_unsafe_auto_id_timestamp"`
		RemoteStore               struct {
			Upload struct {
				TotalUploadSize struct {
					StartedBytes   int `json:"started_bytes"`
					SucceededBytes int `json:"succeeded_bytes"`
					FailedBytes    int `json:"failed_bytes"`
				} `json:"total_upload_size"`
				RefreshSizeLag struct {
					TotalBytes int `json:"total_bytes"`
					MaxBytes   int `json:"max_bytes"`
				} `json:"refresh_size_lag"`
				MaxRefreshTimeLagInMillis int `json:"max_refresh_time_lag_in_millis"`
				TotalTimeSpentInMillis    int `json:"total_time_spent_in_millis"`
				Pressure                  struct {
					TotalRejections int `json:"total_rejections"`
				} `json:"pressure"`
			} `json:"upload"`
			Download struct {
				TotalDownloadSize struct {
					StartedBytes   int `json:"started_bytes"`
					SucceededBytes int `json:"succeeded_bytes"`
					FailedBytes    int `json:"failed_bytes"`
				} `json:"total_download_size"`
				TotalTimeSpentInMillis int `json:"total_time_spent_in_millis"`
			} `json:"download"`
		} `json:"remote_store"`
		SegmentReplication struct {
			// Type is json.RawMessage due to difference in opensearch versions from string to int
			MaxBytesBehind    json.RawMessage `json:"max_bytes_behind"`
			TotalBytesBehind  json.RawMessage `json:"total_bytes_behind"`
			MaxReplicationLag json.RawMessage `json:"max_replication_lag"`
		} `json:"segment_replication"`
		FileSizes json.RawMessage `json:"file_sizes"`
	} `json:"segments"`
	Mappings struct {
		FieldTypes []struct {
			Name       string `json:"name"`
			Count      int    `json:"count"`
			IndexCount int    `json:"index_count"`
		} `json:"field_types"`
	} `json:"mappings"`
	Analysis struct {
		CharFilterTypes    []json.RawMessage `json:"char_filter_types"`
		TokenizerTypes     []json.RawMessage `json:"tokenizer_types"`
		FilterTypes        []json.RawMessage `json:"filter_types"`
		AnalyzerTypes      []json.RawMessage `json:"analyzer_types"`
		BuiltInCharFilters []json.RawMessage `json:"built_in_char_filters"`
		BuiltInTokenizers  []json.RawMessage `json:"built_in_tokenizers"`
		BuiltInFilters     []json.RawMessage `json:"built_in_filters"`
		BuiltInAnalyzers   []json.RawMessage `json:"built_in_analyzers"`
	} `json:"analysis"`
	RepositoryCleanup struct {
		RepositoryCleanup []json.RawMessage `json:"repository_cleanup"`
	} `json:"repository_cleanup"`
}

// ClusterStatsNodes is a sub type of ClusterStatsResp containing information about node stats
type ClusterStatsNodes struct {
	Count struct {
		Total               int `json:"total"`
		ClusterManager      int `json:"cluster_manager"`
		CoordinatingOnly    int `json:"coordinating_only"`
		Data                int `json:"data"`
		Ingest              int `json:"ingest"`
		Master              int `json:"master"`
		RemoteClusterClient int `json:"remote_cluster_client"`
		Search              int `json:"search"`
		Warm                int `json:"warm"`
	} `json:"count"`
	Versions []string `json:"versions"`
	Os       struct {
		AvailableProcessors int `json:"available_processors"`
		AllocatedProcessors int `json:"allocated_processors"`
		Names               []struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		} `json:"names"`
		PrettyNames []struct {
			PrettyName string `json:"pretty_name"`
			Count      int    `json:"count"`
		} `json:"pretty_names"`
		Mem struct {
			TotalInBytes int64 `json:"total_in_bytes"`
			FreeInBytes  int64 `json:"free_in_bytes"`
			UsedInBytes  int64 `json:"used_in_bytes"`
			FreePercent  int   `json:"free_percent"`
			UsedPercent  int   `json:"used_percent"`
		} `json:"mem"`
	} `json:"os"`
	Process struct {
		CPU struct {
			Percent int `json:"percent"`
		} `json:"cpu"`
		OpenFileDescriptors struct {
			Min int `json:"min"`
			Max int `json:"max"`
			Avg int `json:"avg"`
		} `json:"open_file_descriptors"`
	} `json:"process"`
	Jvm struct {
		MaxUptimeInMillis int64 `json:"max_uptime_in_millis"`
		Versions          []struct {
			Version         string `json:"version"`
			VMName          string `json:"vm_name"`
			VMVersion       string `json:"vm_version"`
			VMVendor        string `json:"vm_vendor"`
			BundledJdk      bool   `json:"bundled_jdk"`
			UsingBundledJdk bool   `json:"using_bundled_jdk"`
			Count           int    `json:"count"`
		} `json:"versions"`
		Mem struct {
			HeapUsedInBytes int64 `json:"heap_used_in_bytes"`
			HeapMaxInBytes  int64 `json:"heap_max_in_bytes"`
		} `json:"mem"`
		Threads int `json:"threads"`
	} `json:"jvm"`
	Fs struct {
		TotalInBytes         int64 `json:"total_in_bytes"`
		FreeInBytes          int64 `json:"free_in_bytes"`
		AvailableInBytes     int64 `json:"available_in_bytes"`
		CacheReservedInBytes int   `json:"cache_reserved_in_bytes"`
	} `json:"fs"`
	Plugins []struct {
		Name                    string   `json:"name"`
		Version                 string   `json:"version"`
		OpensearchVersion       string   `json:"opensearch_version"`
		JavaVersion             string   `json:"java_version"`
		Description             string   `json:"description"`
		Classname               string   `json:"classname"`
		CustomFoldername        *string  `json:"custom_foldername"`
		ExtendedPlugins         []string `json:"extended_plugins"`
		OptionalExtendedPlugins []string `json:"optional_extended_plugins"`
		HasNativeController     bool     `json:"has_native_controller"`
	} `json:"plugins"`
	NetworkTypes struct {
		TransportTypes map[string]int `json:"transport_types"`
		HTTPTypes      map[string]int `json:"http_types"`
	} `json:"network_types"`
	DiscoveryTypes map[string]int `json:"discovery_types"`
	PackagingTypes []struct {
		Type  string `json:"type"`
		Count int    `json:"count"`
	} `json:"packaging_types"`
	Ingest struct {
		NumberOfPipelines int             `json:"number_of_pipelines"`
		ProcessorStats    json.RawMessage `json:"processor_stats"`
	} `json:"ingest"`
}
