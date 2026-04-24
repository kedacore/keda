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

// NodesStatsReq represents possible options for the /_nodes request
type NodesStatsReq struct {
	IndexMetric []string
	Metric      []string
	NodeID      []string
	Header      http.Header
	Params      NodesStatsParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r NodesStatsReq) GetRequest() (*http.Request, error) {
	var path strings.Builder

	path.Grow(13 + len(strings.Join(r.NodeID, ",")) + 1 + len(strings.Join(r.Metric, ",")) + 1 + len(strings.Join(r.IndexMetric, ",")))

	path.WriteString("/")
	path.WriteString("_nodes")

	if len(r.NodeID) > 0 {
		path.WriteString("/")
		path.WriteString(strings.Join(r.NodeID, ","))
	}

	path.WriteString("/")
	path.WriteString("stats")

	if len(r.Metric) > 0 {
		path.WriteString("/")
		path.WriteString(strings.Join(r.Metric, ","))
	}

	if len(r.IndexMetric) > 0 {
		path.WriteString("/")
		path.WriteString(strings.Join(r.IndexMetric, ","))
	}

	return opensearch.BuildRequest(
		"GET",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// NodesStatsResp represents the returned struct of the /_nodes response
type NodesStatsResp struct {
	NodesInfo struct {
		Total      int             `json:"total"`
		Successful int             `json:"successful"`
		Failed     int             `json:"failed"`
		Failures   []FailuresCause `json:"failures"`
	} `json:"_nodes"`
	ClusterName string                `json:"cluster_name"`
	Nodes       map[string]NodesStats `json:"nodes"`
	response    *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r NodesStatsResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// NodesStats is a map item of NodesStatsResp representing all values of a node
type NodesStats struct {
	Timestamp                      int                                      `json:"timestamp"`
	Name                           string                                   `json:"name"`
	TransportAddress               string                                   `json:"transport_address"`
	Host                           string                                   `json:"host"`
	IP                             string                                   `json:"ip"`
	Roles                          []string                                 `json:"roles"`
	Attributes                     map[string]string                        `json:"attributes"`
	Indices                        NodesStatsIndices                        `json:"indices"`
	OS                             NodesStatsOS                             `json:"os"`
	Process                        NodesStatsProcess                        `json:"process"`
	JVM                            NodesStatsJVM                            `json:"jvm"`
	ThreadPool                     NodesStatsThreadPool                     `json:"thread_pool"`
	FS                             NodesStatsFS                             `json:"fs"`
	Transport                      NodesStatsTransport                      `json:"transport"`
	HTTP                           NodesStatsHTTP                           `json:"http"`
	Breakers                       NodesStatsBreakers                       `json:"breakers"`
	Scripts                        NodesStatsScript                         `json:"script"`
	Discovery                      NodesStatsDiscovery                      `json:"discovery"`
	Ingest                         NodesStatsIngest                         `json:"ingest"`
	AdaptiveSelection              NodesStatsAdaptiveSelection              `json:"adaptive_selection"`
	ScriptCache                    NodesStatsScriptCache                    `json:"script_cache"`
	IndexingPressure               NodesStatsIndexingPressure               `json:"indexing_pressure"`
	ShardIndexingPressure          NodesStatsShardIndexingPressure          `json:"shard_indexing_pressure"`
	SearchBackpressure             NodesStatsSearchBackpressure             `json:"search_backpressure"`
	ClusterManagerThrottling       NodesStatsClusterManagerThrottling       `json:"cluster_manager_throttling"`
	WeightedRouting                NodesStatsWeightedRouting                `json:"weighted_routing"`
	SearchPipeline                 NodesStatsSearchPipeline                 `json:"search_pipeline"`
	TaskCancellation               NodesStatsTaskCancellation               `json:"task_cancellation"`
	ResourceUsageStats             map[string]NodesStatsResourceUsageStats  `json:"resource_usage_stats"`
	SegmentReplicationBackpressure NodesStatsSegmentReplicationBackpressure `json:"segment_replication_backpressure"`
	Repositories                   []json.RawMessage                        `json:"repositories"`
	AdmissionControl               NodesStatsAdmissionControl               `json:"admission_control"`
	Caches                         NodesStatsCaches                         `json:"caches"`
	RemoteStore                    NodeStatsRemoteStore                     `json:"remote_store"`
}

// NodesStatsIndices is a sub type of NodesStats representing Indices information of the node
type NodesStatsIndices struct {
	Docs struct {
		Count   int `json:"count"`
		Deleted int `json:"deleted"`
	} `json:"docs"`
	Store struct {
		SizeInBytes     int `json:"size_in_bytes"`
		ReservedInBytes int `json:"reserved_in_bytes"`
	} `json:"store"`
	Indexing struct {
		IndexTotal           int            `json:"index_total"`
		IndexTimeInMillis    int            `json:"index_time_in_millis"`
		IndexCurrent         int            `json:"index_current"`
		IndexFailed          int            `json:"index_failed"`
		DeleteTotal          int            `json:"delete_total"`
		DeleteTimeInMillis   int            `json:"delete_time_in_millis"`
		DeleteCurrent        int            `json:"delete_current"`
		NoopUpdateTotal      int            `json:"noop_update_total"`
		IsThrottled          bool           `json:"is_throttled"`
		ThrottleTimeInMillis int            `json:"throttle_time_in_millis"`
		DocStatus            map[string]int `json:"doc_status"`
	} `json:"indexing"`
	Get struct {
		Total               int    `json:"total"`
		TimeInMillis        int    `json:"time_in_millis"`
		ExistsTotal         int    `json:"exists_total"`
		ExistsTimeInMillis  int    `json:"exists_time_in_millis"`
		MissingTotal        int    `json:"missing_total"`
		MissingTimeInMillis int    `json:"missing_time_in_millis"`
		Current             int    `json:"current"`
		GetTime             string `json:"getTime"`
	} `json:"get"`
	Search struct {
		OpenContexts                int     `json:"open_contexts"`
		QueryTotal                  int     `json:"query_total"`
		QueryTimeInMillis           int     `json:"query_time_in_millis"`
		QueryCurrent                int     `json:"query_current"`
		ConcurrentQueryTotal        int     `json:"concurrent_query_total"`
		ConcurrentQueryTimeInMillis int     `json:"concurrent_query_time_in_millis"`
		ConcurrentQueryCurrent      int     `json:"concurrent_query_current"`
		ConcurrentAVGSliceCount     float32 `json:"concurrent_avg_slice_count"`
		FetchTotal                  int     `json:"fetch_total"`
		FetchTimeInMillis           int     `json:"fetch_time_in_millis"`
		FetchCurrent                int     `json:"fetch_current"`
		ScrollTotal                 int     `json:"scroll_total"`
		ScrollTimeInMillis          int     `json:"scroll_time_in_millis"`
		ScrollCurrent               int     `json:"scroll_current"`
		PointInTimeTotal            int     `json:"point_in_time_total"`
		PointInTimeTimeInMillis     int     `json:"point_in_time_time_in_millis"`
		PointInTimeCurrent          int     `json:"point_in_time_current"`
		SuggestTotal                int     `json:"suggest_total"`
		SuggestTimeInMillis         int     `json:"suggest_time_in_millis"`
		SuggestCurrent              int     `json:"suggest_current"`
		IdleReactivateCountTotal    int     `json:"search_idle_reactivate_count_total"`
		Request                     struct {
			DfsPreQuery NodesStatsIndicesSearchRequest `json:"dfs_pre_query"`
			Query       NodesStatsIndicesSearchRequest `json:"query"`
			Fetch       NodesStatsIndicesSearchRequest `json:"fetch"`
			DfsQuery    NodesStatsIndicesSearchRequest `json:"dfs_query"`
			Expand      NodesStatsIndicesSearchRequest `json:"expand"`
			CanMatch    NodesStatsIndicesSearchRequest `json:"can_match"`
			Took        NodesStatsIndicesSearchRequest `json:"took"`
		} `json:"request"`
	} `json:"search"`
	Merges struct {
		Current                           int `json:"current"`
		CurrentDocs                       int `json:"current_docs"`
		CurrentSizeInBytes                int `json:"current_size_in_bytes"`
		Total                             int `json:"total"`
		TotalTimeInMillis                 int `json:"total_time_in_millis"`
		TotalDocs                         int `json:"total_docs"`
		TotalSizeInBytes                  int `json:"total_size_in_bytes"`
		TotalStoppedTimeInMillis          int `json:"total_stopped_time_in_millis"`
		TotalThrottledTimeInMillis        int `json:"total_throttled_time_in_millis"`
		TotalAutoThrottleInBytes          int `json:"total_auto_throttle_in_bytes"`
		UnreferencedFileCleanupsPerformed int `json:"unreferenced_file_cleanups_performed"`
	} `json:"merges"`
	Refresh struct {
		Total                     int `json:"total"`
		TotalTimeInMillis         int `json:"total_time_in_millis"`
		ExternalTotal             int `json:"external_total"`
		ExternalTotalTimeInMillis int `json:"external_total_time_in_millis"`
		Listeners                 int `json:"listeners"`
	} `json:"refresh"`
	Flush struct {
		Total             int `json:"total"`
		Periodic          int `json:"periodic"`
		TotalTimeInMillis int `json:"total_time_in_millis"`
	} `json:"flush"`
	Warmer struct {
		Current           int `json:"current"`
		Total             int `json:"total"`
		TotalTimeInMillis int `json:"total_time_in_millis"`
	} `json:"warmer"`
	QueryCache struct {
		MemorySizeInBytes int `json:"memory_size_in_bytes"`
		TotalCount        int `json:"total_count"`
		HitCount          int `json:"hit_count"`
		MissCount         int `json:"miss_count"`
		CacheSize         int `json:"cache_size"`
		CacheCount        int `json:"cache_count"`
		Evictions         int `json:"evictions"`
	} `json:"query_cache"`
	Fielddata struct {
		MemorySizeInBytes int `json:"memory_size_in_bytes"`
		Evictions         int `json:"evictions"`
	} `json:"fielddata"`
	Completion struct {
		SizeInBytes int `json:"size_in_bytes"`
	} `json:"completion"`
	Segments struct {
		Count                     int `json:"count"`
		MemoryInBytes             int `json:"memory_in_bytes"`
		TermsMemoryInBytes        int `json:"terms_memory_in_bytes"`
		StoredFieldsMemoryInBytes int `json:"stored_fields_memory_in_bytes"`
		TermVectorsMemoryInBytes  int `json:"term_vectors_memory_in_bytes"`
		NormsMemoryInBytes        int `json:"norms_memory_in_bytes"`
		PointsMemoryInBytes       int `json:"points_memory_in_bytes"`
		DocValuesMemoryInBytes    int `json:"doc_values_memory_in_bytes"`
		IndexWriterMemoryInBytes  int `json:"index_writer_memory_in_bytes"`
		VersionMapMemoryInBytes   int `json:"version_map_memory_in_bytes"`
		FixedBitSetMemoryInBytes  int `json:"fixed_bit_set_memory_in_bytes"`
		MaxUnsafeAutoIDTimestamp  int `json:"max_unsafe_auto_id_timestamp"`
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
	Translog struct {
		Operations              int `json:"operations"`
		SizeInBytes             int `json:"size_in_bytes"`
		UncommittedOperations   int `json:"uncommitted_operations"`
		UncommittedSizeInBytes  int `json:"uncommitted_size_in_bytes"`
		EarliestLastModifiedAge int `json:"earliest_last_modified_age"`
		RemoteStore             struct {
			Upload struct {
				TotalUploads struct {
					Started   int `json:"started"`
					Failed    int `json:"failed"`
					Succeeded int `json:"succeeded"`
				} `json:"total_uploads"`
				TotalUploadSize struct {
					StartedBytes   int `json:"started_bytes"`
					FailedBytes    int `json:"failed_bytes"`
					SucceededBytes int `json:"succeeded_bytes"`
				} `json:"total_upload_size"`
			} `json:"upload"`
		} `json:"remote_store"`
	} `json:"translog"`
	RequestCache struct {
		MemorySizeInBytes int `json:"memory_size_in_bytes"`
		Evictions         int `json:"evictions"`
		HitCount          int `json:"hit_count"`
		MissCount         int `json:"miss_count"`
	} `json:"request_cache"`
	Recovery struct {
		CurrentAsSource      int `json:"current_as_source"`
		CurrentAsTarget      int `json:"current_as_target"`
		ThrottleTimeInMillis int `json:"throttle_time_in_millis"`
	} `json:"recovery"`
}

// NodesStatsOS is a sub type of NodesStats representing operating system information of the node
type NodesStatsOS struct {
	Timestamp int `json:"timestamp"`
	CPU       struct {
		Percent     int `json:"percent"`
		LoadAverage struct {
			OneM  float64 `json:"1m"`
			FiveM float64 `json:"5m"`
			One5M float64 `json:"15m"`
		} `json:"load_average"`
	} `json:"cpu"`
	Mem struct {
		TotalInBytes int `json:"total_in_bytes"`
		FreeInBytes  int `json:"free_in_bytes"`
		UsedInBytes  int `json:"used_in_bytes"`
		FreePercent  int `json:"free_percent"`
		UsedPercent  int `json:"used_percent"`
	} `json:"mem"`
	Swap struct {
		TotalInBytes int `json:"total_in_bytes"`
		FreeInBytes  int `json:"free_in_bytes"`
		UsedInBytes  int `json:"used_in_bytes"`
	} `json:"swap"`
}

// NodesStatsProcess is a sub type of NodesStats representing processor information of the node
type NodesStatsProcess struct {
	Timestamp           int `json:"timestamp"`
	OpenFileDescriptors int `json:"open_file_descriptors"`
	MaxFileDescriptors  int `json:"max_file_descriptors"`
	CPU                 struct {
		Percent       int `json:"percent"`
		TotalInMillis int `json:"total_in_millis"`
	} `json:"cpu"`
	Mem struct {
		TotalVirtualInBytes int `json:"total_virtual_in_bytes"`
	} `json:"mem"`
}

// NodesStatsJVMPool is a sub type of NodesStatsJVM represeting all information a pool can have
type NodesStatsJVMPool struct {
	UsedInBytes     int `json:"used_in_bytes"`
	MaxInBytes      int `json:"max_in_bytes"`
	PeakUsedInBytes int `json:"peak_used_in_bytes"`
	PeakMaxInBytes  int `json:"peak_max_in_bytes"`
	LastGcStats     struct {
		UsedInBytes  int `json:"used_in_bytes"`
		MaxInBytes   int `json:"max_in_bytes"`
		UsagePercent int `json:"usage_percent"`
	} `json:"last_gc_stats"`
}

// NodesStatsJVMBufferPool is a sub map type represeting all information a buffer pool can have
type NodesStatsJVMBufferPool struct {
	Count                int `json:"count"`
	UsedInBytes          int `json:"used_in_bytes"`
	TotalCapacityInBytes int `json:"total_capacity_in_bytes"`
}

// NodesStatsJVM is a sub type of NodesStats representing java virtual maschine information of the node
type NodesStatsJVM struct {
	Timestamp      int `json:"timestamp"`
	UptimeInMillis int `json:"uptime_in_millis"`
	Mem            struct {
		HeapUsedInBytes         int `json:"heap_used_in_bytes"`
		HeapUsedPercent         int `json:"heap_used_percent"`
		HeapCommittedInBytes    int `json:"heap_committed_in_bytes"`
		HeapMaxInBytes          int `json:"heap_max_in_bytes"`
		NonHeapUsedInBytes      int `json:"non_heap_used_in_bytes"`
		NonHeapCommittedInBytes int `json:"non_heap_committed_in_bytes"`
		Pools                   struct {
			Young    NodesStatsJVMPool `json:"young"`
			Old      NodesStatsJVMPool `json:"old"`
			Survivor NodesStatsJVMPool `json:"survivor"`
		} `json:"pools"`
	} `json:"mem"`
	Threads struct {
		Count     int `json:"count"`
		PeakCount int `json:"peak_count"`
	} `json:"threads"`
	Gc struct {
		Collectors map[string]NodesStatsJVMGCCollectors `json:"collectors"`
	} `json:"gc"`
	// Not parsing each field directly as one of them contains singe quotes which are not allowed as tag in golang json
	// https://github.com/golang/go/issues/22518
	BufferPools map[string]NodesStatsJVMBufferPool `json:"buffer_pools"`
	Classes     struct {
		CurrentLoadedCount int `json:"current_loaded_count"`
		TotalLoadedCount   int `json:"total_loaded_count"`
		TotalUnloadedCount int `json:"total_unloaded_count"`
	} `json:"classes"`
}

// NodesStatsJVMGCCollectors is a sub type of NodesStatsJVM containing collector information
type NodesStatsJVMGCCollectors struct {
	CollectionCount        int `json:"collection_count"`
	CollectionTimeInMillis int `json:"collection_time_in_millis"`
}

// NodesStatsThreadPoolValues is a sub type of NodesStatsThreadPool representing all information a thread pool can have
type NodesStatsThreadPoolValues struct {
	Threads              int    `json:"threads"`
	Queue                int    `json:"queue"`
	Active               int    `json:"active"`
	Rejected             int    `json:"rejected"`
	Largest              int    `json:"largest"`
	Completed            int    `json:"completed"`
	TotalWaitTimeInNanos *int64 `json:"total_wait_time_in_nanos,omitempty"`
}

// NodesStatsThreadPool is a sub type of NodesStats representing thread pool information of the node
type NodesStatsThreadPool map[string]NodesStatsThreadPoolValues

// NodesStatsFS is a sub type of NodesStats representing filesystem information of the node
type NodesStatsFS struct {
	Timestamp int `json:"timestamp"`
	Total     struct {
		TotalInBytes         int `json:"total_in_bytes"`
		FreeInBytes          int `json:"free_in_bytes"`
		AvailableInBytes     int `json:"available_in_bytes"`
		CacheReservedInBytes int `json:"cache_reserved_in_bytes"`
	} `json:"total"`
	Data []struct {
		Path                 string `json:"path"`
		Mount                string `json:"mount"`
		Type                 string `json:"type"`
		TotalInBytes         int    `json:"total_in_bytes"`
		FreeInBytes          int    `json:"free_in_bytes"`
		AvailableInBytes     int    `json:"available_in_bytes"`
		CacheReservedInBytes int    `json:"cache_reserved_in_bytes"`
	} `json:"data"`
	IoStats struct {
		Devices []struct {
			DeviceName      string `json:"device_name"`
			Operations      int    `json:"operations"`
			ReadOperations  int    `json:"read_operations"`
			WriteOperations int    `json:"write_operations"`
			ReadKilobytes   int    `json:"read_kilobytes"`
			WriteKilobytes  int    `json:"write_kilobytes"`
		} `json:"devices"`
		Total struct {
			Operations      int `json:"operations"`
			ReadOperations  int `json:"read_operations"`
			WriteOperations int `json:"write_operations"`
			ReadKilobytes   int `json:"read_kilobytes"`
			WriteKilobytes  int `json:"write_kilobytes"`
		} `json:"total"`
	} `json:"io_stats"`
}

// NodesStatsTransport is a sub type of NodesStats representing network transport information of the node
type NodesStatsTransport struct {
	ServerOpen               int `json:"server_open"`
	TotalOutboundConnections int `json:"total_outbound_connections"`
	RxCount                  int `json:"rx_count"`
	RxSizeInBytes            int `json:"rx_size_in_bytes"`
	TxCount                  int `json:"tx_count"`
	TxSizeInBytes            int `json:"tx_size_in_bytes"`
}

// NodesStatsHTTP is a sub type of NodesStats representing http information of the node
type NodesStatsHTTP struct {
	CurrentOpen int `json:"current_open"`
	TotalOpened int `json:"total_opened"`
}

// NodesStatsBreaker is a sub type of NodesStatsBreakers containing all information a breaker can have
type NodesStatsBreaker struct {
	LimitSizeInBytes     int     `json:"limit_size_in_bytes"`
	LimitSize            string  `json:"limit_size"`
	EstimatedSizeInBytes int     `json:"estimated_size_in_bytes"`
	EstimatedSize        string  `json:"estimated_size"`
	Overhead             float64 `json:"overhead"`
	Tripped              int     `json:"tripped"`
}

// NodesStatsBreakers is a sub type of NodesStats representing breakers information of the node
type NodesStatsBreakers struct {
	Accounting       NodesStatsBreaker `json:"accounting"`
	Request          NodesStatsBreaker `json:"request"`
	Fielddata        NodesStatsBreaker `json:"fielddata"`
	InFlightRequests NodesStatsBreaker `json:"in_flight_requests"`
	Parent           NodesStatsBreaker `json:"parent"`
}

// NodesStatsScript is a sub type of NodesStats representing script information of the node
type NodesStatsScript struct {
	Compilations              int `json:"compilations"`
	CacheEvictions            int `json:"cache_evictions"`
	CompilationLimitTriggered int `json:"compilation_limit_triggered"`
}

// NodesStatsDiscovery is a sub type of NodesStats representing discovery information of the node
type NodesStatsDiscovery struct {
	ClusterStateQueue struct {
		Total     int `json:"total"`
		Pending   int `json:"pending"`
		Committed int `json:"committed"`
	} `json:"cluster_state_queue"`
	PublishedClusterStates struct {
		FullStates        int `json:"full_states"`
		IncompatibleDiffs int `json:"incompatible_diffs"`
		CompatibleDiffs   int `json:"compatible_diffs"`
	} `json:"published_cluster_states"`
	ClusterStateStats struct {
		Overall struct {
			UpdateCount       int `json:"update_count"`
			TotalTimeInMillis int `json:"total_time_in_millis"`
			FailedCount       int `json:"failed_count"`
		} `json:"overall"`
	} `json:"cluster_state_stats"`
}

// NodesStatsIngestDetails is a sub map type of NodsStatsIngest containing all information of ingest pipelines
type NodesStatsIngestDetails struct {
	Count        int               `json:"count"`
	TimeInMillis int               `json:"time_in_millis"`
	Failed       int               `json:"failed"`
	Current      int               `json:"current"`
	Processors   []json.RawMessage `json:"processors"`
}

// NodesStatsIngest is a sub type of NodesStats representing ingest pipelines information of the node
type NodesStatsIngest struct {
	Total struct {
		Count        int `json:"count"`
		TimeInMillis int `json:"time_in_millis"`
		Current      int `json:"current"`
		Failed       int `json:"failed"`
	} `json:"total"`
	Pipelines map[string]NodesStatsIngestDetails `json:"pipelines"`
}

// NodesStatsAdaptiveSelection is a sub type of NodesStats representing adaptive selection information of the node
type NodesStatsAdaptiveSelection map[string]struct {
	OutgoingSearches  int    `json:"outgoing_searches"`
	AvgQueueSize      int    `json:"avg_queue_size"`
	AvgServiceTimeNs  int    `json:"avg_service_time_ns"`
	AvgResponseTimeNs int    `json:"avg_response_time_ns"`
	Rank              string `json:"rank"`
}

// NodesStatsScriptCache is a sub type of NodesStats representing script cache information of the node
type NodesStatsScriptCache struct {
	Sum struct {
		Compilations              int `json:"compilations"`
		CacheEvictions            int `json:"cache_evictions"`
		CompilationLimitTriggered int `json:"compilation_limit_triggered"`
	} `json:"sum"`
	Contexts []struct {
		Context                   string `json:"context"`
		Compilations              int    `json:"compilations"`
		CacheEvictions            int    `json:"cache_evictions"`
		CompilationLimitTriggered int    `json:"compilation_limit_triggered"`
	} `json:"contexts"`
}

// NodesStatsIndexingPressure is a sub type of NodesStats representing indexing pressure information of the node
type NodesStatsIndexingPressure struct {
	Memory struct {
		Current struct {
			CombinedCoordinatingAndPrimaryInBytes int `json:"combined_coordinating_and_primary_in_bytes"`
			CoordinatingInBytes                   int `json:"coordinating_in_bytes"`
			PrimaryInBytes                        int `json:"primary_in_bytes"`
			ReplicaInBytes                        int `json:"replica_in_bytes"`
			AllInBytes                            int `json:"all_in_bytes"`
		} `json:"current"`
		Total struct {
			CombinedCoordinatingAndPrimaryInBytes int `json:"combined_coordinating_and_primary_in_bytes"`
			CoordinatingInBytes                   int `json:"coordinating_in_bytes"`
			PrimaryInBytes                        int `json:"primary_in_bytes"`
			ReplicaInBytes                        int `json:"replica_in_bytes"`
			AllInBytes                            int `json:"all_in_bytes"`
			CoordinatingRejections                int `json:"coordinating_rejections"`
			PrimaryRejections                     int `json:"primary_rejections"`
			ReplicaRejections                     int `json:"replica_rejections"`
		} `json:"total"`
		LimitInBytes int `json:"limit_in_bytes"`
	} `json:"memory"`
}

// NodesStatsShardIndexingPressure is a sub type of NodesStats representing shard indexing pressure information of the node
type NodesStatsShardIndexingPressure struct {
	Stats                            json.RawMessage `json:"stats"` // Unknown, can be added if you have an example
	TotalRejectionsBreakupShadowMode struct {
		NodeLimits                  int `json:"node_limits"`
		NoSuccessfulRequestLimits   int `json:"no_successful_request_limits"`
		ThroughputDegradationLimits int `json:"throughput_degradation_limits"`
	} `json:"total_rejections_breakup_shadow_mode"`
	Enabled  bool `json:"enabled"`
	Enforced bool `json:"enforced"`
}

// NodesStatsSearchBackpressureTracker is a sub type of NodesStatsSearchBrackpressure containing all information of a tracker
type NodesStatsSearchBackpressureTracker struct {
	CancellationCount int `json:"cancellation_count"`
	CurrentMaxMillis  int `json:"current_max_millis"`
	CurrentAvgMillis  int `json:"current_avg_millis"`
}

// NodesStatsSearchBackpressure is a sub type of NodesStats representing search packbressure information of a node
type NodesStatsSearchBackpressure struct {
	SearchTask struct {
		ResourceTrackerStats struct {
			CPUUsageTracker    NodesStatsSearchBackpressureTracker `json:"cpu_usage_tracker"`
			ElapsedTimeTracker NodesStatsSearchBackpressureTracker `json:"elapsed_time_tracker"`
			HeapUsageTracker   struct {
				CancellationCount int `json:"cancellation_count"`
				CurrentMaxBytes   int `json:"current_max_bytes"`
				CurrentAvgBytes   int `json:"current_avg_bytes"`
				RollingAvgBytes   int `json:"rolling_avg_bytes"`
			} `json:"heap_usage_tracker"`
		} `json:"resource_tracker_stats"`
		CancellationStats struct {
			CancellationCount             int `json:"cancellation_count"`
			CancellationLimitReachedCount int `json:"cancellation_limit_reached_count"`
		} `json:"cancellation_stats"`
		CompletionCount int `json:"completion_count"`
	} `json:"search_task"`
	SearchShardTask struct {
		ResourceTrackerStats struct {
			CPUUsageTracker    NodesStatsSearchBackpressureTracker `json:"cpu_usage_tracker"`
			ElapsedTimeTracker NodesStatsSearchBackpressureTracker `json:"elapsed_time_tracker"`
			HeapUsageTracker   struct {
				CancellationCount int `json:"cancellation_count"`
				CurrentMaxBytes   int `json:"current_max_bytes"`
				CurrentAvgBytes   int `json:"current_avg_bytes"`
				RollingAvgBytes   int `json:"rolling_avg_bytes"`
			} `json:"heap_usage_tracker"`
		} `json:"resource_tracker_stats"`
		CancellationStats struct {
			CancellationCount             int `json:"cancellation_count"`
			CancellationLimitReachedCount int `json:"cancellation_limit_reached_count"`
		} `json:"cancellation_stats"`
		CompletionCount int `json:"completion_count"`
	} `json:"search_shard_task"`
	Mode string `json:"mode"`
}

// NodesStatsClusterManagerThrottling is a sub type of NodesStats representing cluster manager throttling information of the node
type NodesStatsClusterManagerThrottling struct {
	Stats struct {
		TotalThrottledTasks       int             `json:"total_throttled_tasks"`
		ThrottledTasksPerTaskType json.RawMessage `json:"throttled_tasks_per_task_type"` // Unknow struct, no example in doc
	} `json:"stats"`
}

// NodesStatsWeightedRouting is a sub type of NodesStats representing weighted routing information of the node
type NodesStatsWeightedRouting struct {
	Stats struct {
		FailOpenCount int `json:"fail_open_count"`
	} `json:"stats"`
}

// NodesStatsSearchPipeline is a sub type of NodesStats containing stats about search pipelines
type NodesStatsSearchPipeline struct {
	TotalRequest struct {
		Count        int `json:"count"`
		TimeInMillis int `json:"time_in_millis"`
		Current      int `json:"current"`
		Failed       int `json:"failed"`
	} `json:"total_request"`
	TotalResponse struct {
		Count        int `json:"count"`
		TimeInMillis int `json:"time_in_millis"`
		Current      int `json:"current"`
		Failed       int `json:"failed"`
	} `json:"total_response"`
	Pipelines json.RawMessage `json:"pipelines"`
}

// NodesStatsTaskCancellation is a sub type of NodesStats containing stats about canceled tasks
type NodesStatsTaskCancellation struct {
	SearchTask      NodesStatsTaskCancellationValues `json:"search_task"`
	SearchShardTask NodesStatsTaskCancellationValues `json:"search_shard_task"`
}

// NodesStatsTaskCancellationValues is a sub type of NodesStatsTaskCancellation
type NodesStatsTaskCancellationValues struct {
	CurrentCountPostCancel int `json:"current_count_post_cancel"`
	TotalCountPostCancel   int `json:"total_count_post_cancel"`
}

// NodesStatsIndicesSearchRequest is a sub type of NodesStatsIndices containing stats about search requests
type NodesStatsIndicesSearchRequest struct {
	TimeInMillis int `json:"time_in_millis"`
	Current      int `json:"current"`
	Total        int `json:"total"`
}

// NodesStatsResourceUsageStats is a sub type of NodesStats containing nodes resource information
type NodesStatsResourceUsageStats struct {
	Timestamp                int64  `json:"timestamp"`
	CPUUtilizationPercent    string `json:"cpu_utilization_percent"`
	MemoryUtilizationPercent string `json:"memory_utilization_percent"`
	IOUsageStats             struct {
		MaxIOUtilizationPercent string `json:"max_io_utilization_percent"`
	} `json:"io_usage_stats"`
}

// NodesStatsSegmentReplicationBackpressure is a sub type of NodesStats containing information about segment replication backpressure
type NodesStatsSegmentReplicationBackpressure struct {
	TotalRejectedRequests int `json:"total_rejected_requests"`
}

// NodesStatsAdmissionControl is a sub type of NodesStats
type NodesStatsAdmissionControl struct {
	GlobalCPUUsage struct {
		Transport struct {
			RejectionCount json.RawMessage `json:"rejection_count"`
		} `json:"transport"`
	} `json:"global_cpu_usage"`
	GlobalIOUsage struct {
		Transport struct {
			RejectionCount json.RawMessage `json:"rejection_count"`
		} `json:"transport"`
	} `json:"global_io_usage"`
}

// NodesStatsCaches is a sub type of NodesStats
type NodesStatsCaches struct {
	RequestCache struct {
		SizeInBytes int    `json:"size_in_bytes"`
		Evictions   int    `json:"evictions"`
		HitCount    int    `json:"hit_count"`
		MissCount   int    `json:"miss_count"`
		ItemCount   int    `json:"item_count"`
		StoreName   string `json:"store_name"`
	} `json:"request_cache"`
}

// NodeStatsRemoteStore is a sub type of NodesStats
type NodeStatsRemoteStore struct {
	LastSuccessfulFetchOfPinnedTimestamps int `json:"last_successful_fetch_of_pinned_timestamps"`
}
