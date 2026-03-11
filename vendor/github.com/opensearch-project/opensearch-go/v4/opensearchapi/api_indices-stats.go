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

// IndicesStatsReq represents possible options for the index shrink request
type IndicesStatsReq struct {
	Indices []string
	Metrics []string

	Header http.Header
	Params IndicesStatsParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r IndicesStatsReq) GetRequest() (*http.Request, error) {
	indices := strings.Join(r.Indices, ",")
	metrics := strings.Join(r.Metrics, ",")

	var path strings.Builder
	path.Grow(9 + len(indices) + len(metrics))
	if len(indices) > 0 {
		path.WriteString("/")
		path.WriteString(indices)
	}
	path.WriteString("/_stats")
	if len(metrics) > 0 {
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

// IndicesStatsResp represents the returned struct of the index shrink response
type IndicesStatsResp struct {
	Shards   IndicesStatsShards  `json:"_shards"`
	All      IndicesStatsAll     `json:"_all"`
	Indices  IndicesStatsIndices `json:"indices"`
	response *opensearch.Response
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r IndicesStatsResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// IndicesStatsShards is a sub type of IndicesStatsResp containing information about how many shards got requested
type IndicesStatsShards struct {
	Total      int             `json:"total"`
	Successful int             `json:"successful"`
	Failed     int             `json:"failed"`
	Failures   []FailuresShard `json:"failures"`
}

// IndicesStatsDocs is a sub type of IndicesStatsInfo containing stats about the index documents
type IndicesStatsDocs struct {
	Count   int `json:"count"`
	Deleted int `json:"deleted"`
}

// IndicesStatsStore is a sub type of IndicesStatsInfo containing stats about index storage
type IndicesStatsStore struct {
	SizeInBytes     int64 `json:"size_in_bytes"`
	ReservedInBytes int   `json:"reserved_in_bytes"`
}

// IndicesStatsIndexing is a sub type of IndicesStatsInfo containing stats about document indexing
type IndicesStatsIndexing struct {
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
}

// IndicesStatsGet is a sub type of IndicesStatsInfo containing stats about index get
type IndicesStatsGet struct {
	Total               int    `json:"total"`
	TimeInMillis        int    `json:"time_in_millis"`
	ExistsTotal         int    `json:"exists_total"`
	ExistsTimeInMillis  int    `json:"exists_time_in_millis"`
	MissingTotal        int    `json:"missing_total"`
	MissingTimeInMillis int    `json:"missing_time_in_millis"`
	Current             int    `json:"current"`
	GetTime             string `json:"getTime"`
}

// IndicesStatsSearch is a sub type of IndicesStatsInfo containing stats about index search
type IndicesStatsSearch struct {
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
}

// IndicesStatsMerges is a sub type of IndicesStatsInfo containing stats about index merges
type IndicesStatsMerges struct {
	Current                           int   `json:"current"`
	CurrentDocs                       int   `json:"current_docs"`
	CurrentSizeInBytes                int   `json:"current_size_in_bytes"`
	Total                             int   `json:"total"`
	TotalTimeInMillis                 int   `json:"total_time_in_millis"`
	TotalDocs                         int   `json:"total_docs"`
	TotalSizeInBytes                  int64 `json:"total_size_in_bytes"`
	TotalStoppedTimeInMillis          int   `json:"total_stopped_time_in_millis"`
	TotalThrottledTimeInMillis        int   `json:"total_throttled_time_in_millis"`
	TotalAutoThrottleInBytes          int   `json:"total_auto_throttle_in_bytes"`
	UnreferencedFileCleanupsPerformed int   `json:"unreferenced_file_cleanups_performed"`
}

// IndicesStatsRefresh is a sub type of IndicesStatsInfo containing stats about index refresh
type IndicesStatsRefresh struct {
	Total                     int `json:"total"`
	TotalTimeInMillis         int `json:"total_time_in_millis"`
	ExternalTotal             int `json:"external_total"`
	ExternalTotalTimeInMillis int `json:"external_total_time_in_millis"`
	Listeners                 int `json:"listeners"`
}

// IndicesStatsFlush is a sub type of IndicesStatsInfo containing stats about index flush
type IndicesStatsFlush struct {
	Total             int `json:"total"`
	Periodic          int `json:"periodic"`
	TotalTimeInMillis int `json:"total_time_in_millis"`
}

// IndicesStatsWarmer is a sub type of IndicesStatsInfo containing stats about index warmer
type IndicesStatsWarmer struct {
	Current           int `json:"current"`
	Total             int `json:"total"`
	TotalTimeInMillis int `json:"total_time_in_millis"`
}

// IndicesStatsQueryCache is a sub type of IndicesStatsInfo containing stats about index query cache
type IndicesStatsQueryCache struct {
	MemorySizeInBytes int `json:"memory_size_in_bytes"`
	TotalCount        int `json:"total_count"`
	HitCount          int `json:"hit_count"`
	MissCount         int `json:"miss_count"`
	CacheSize         int `json:"cache_size"`
	CacheCount        int `json:"cache_count"`
	Evictions         int `json:"evictions"`
}

// IndicesStatsFielddata is a sub type of IndicesStatsInfo containing stats about index fielddata
type IndicesStatsFielddata struct {
	MemorySizeInBytes int `json:"memory_size_in_bytes"`
	Evictions         int `json:"evictions"`
}

// IndicesStatsCompletion is a sub type of IndicesStatsInfo containing stats about index completion
type IndicesStatsCompletion struct {
	SizeInBytes int `json:"size_in_bytes"`
}

// IndicesStatsSegments is a sub type of IndicesStatsInfo containing stats about index segments
type IndicesStatsSegments struct {
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
	FixedBitSetMemoryInBytes  int   `json:"fixed_bit_set_memory_in_bytes"`
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
}

// IndicesStatsTranslog is a sub type of IndicesStatsInfo containing stats about index translog
type IndicesStatsTranslog struct {
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
}

// IndicesStatsRequestCache is a sub type of IndicesStatsInfo containing stats about index request cache
type IndicesStatsRequestCache struct {
	MemorySizeInBytes int `json:"memory_size_in_bytes"`
	Evictions         int `json:"evictions"`
	HitCount          int `json:"hit_count"`
	MissCount         int `json:"miss_count"`
}

// IndicesStatsRecovery is a sub type of IndicesStatsInfo containing stats about index recovery
type IndicesStatsRecovery struct {
	CurrentAsSource      int `json:"current_as_source"`
	CurrentAsTarget      int `json:"current_as_target"`
	ThrottleTimeInMillis int `json:"throttle_time_in_millis"`
}

// IndicesStatsInfo is a sub type of IndicesStatsAll, IndicesStatsDetails aggregating all document stats
type IndicesStatsInfo struct {
	Docs         IndicesStatsDocs         `json:"docs"`
	Store        IndicesStatsStore        `json:"store"`
	Indexing     IndicesStatsIndexing     `json:"indexing"`
	Get          IndicesStatsGet          `json:"get"`
	Search       IndicesStatsSearch       `json:"search"`
	Merges       IndicesStatsMerges       `json:"merges"`
	Refresh      IndicesStatsRefresh      `json:"refresh"`
	Flush        IndicesStatsFlush        `json:"flush"`
	Warmer       IndicesStatsWarmer       `json:"warmer"`
	QueryCache   IndicesStatsQueryCache   `json:"query_cache"`
	Fielddata    IndicesStatsFielddata    `json:"fielddata"`
	Completion   IndicesStatsCompletion   `json:"completion"`
	Segments     IndicesStatsSegments     `json:"segments"`
	Translog     IndicesStatsTranslog     `json:"translog"`
	RequestCache IndicesStatsRequestCache `json:"request_cache"`
	Recovery     IndicesStatsRecovery     `json:"recovery"`
}

// IndicesStatsAll is a sub type of IndicesStatsResp containing information about docs stats from all indices
type IndicesStatsAll struct {
	Primaries IndicesStatsInfo `json:"primaries"`
	Total     IndicesStatsInfo `json:"total"`
}

// IndicesStatsDetails is a sub type of IndicesStatsIndices containing the information about the index uuid and index stats
type IndicesStatsDetails struct {
	UUID      string           `json:"uuid"`
	Primaries IndicesStatsInfo `json:"primaries"`
	Total     IndicesStatsInfo `json:"total"`
}

// IndicesStatsIndices is a sub type of IndicesStatsResp containing information about docs stats from specific indices
type IndicesStatsIndices map[string]IndicesStatsDetails
