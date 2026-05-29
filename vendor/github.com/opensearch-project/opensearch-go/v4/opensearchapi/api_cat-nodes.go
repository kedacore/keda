// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
)

// CatNodesReq represent possible options for the /_cat/nodes request
type CatNodesReq struct {
	Header http.Header
	Params CatNodesParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r CatNodesReq) GetRequest() (*http.Request, error) {
	return opensearch.BuildRequest(
		"GET",
		"/_cat/nodes",
		nil,
		r.Params.get(),
		r.Header,
	)
}

// CatNodesResp represents the returned struct of the /_cat/nodes response
type CatNodesResp struct {
	Nodes    []CatNodesItemResp
	response *opensearch.Response
}

// CatNodesItemResp represents one index of the CatNodesResp
type CatNodesItemResp struct {
	ID                              string  `json:"id"`
	PID                             *string `json:"pid"`
	IP                              string  `json:"ip"`
	Port                            int     `json:"port,string"`
	HTTPAddress                     string  `json:"http_address"`
	Version                         string  `json:"version"`
	Type                            *string `json:"type"`
	Build                           *string `json:"build"`
	JDK                             *string `json:"jdk"`
	DiskTotal                       *string `json:"disk.total"`
	DiskUsed                        *string `json:"disk.used"`
	DiskAvail                       *string `json:"disk.avail"`
	DiskUsedPercent                 *string `json:"disk.used_percent"`
	HeapCurrent                     *string `json:"heap.current"`
	HeapPercent                     *int    `json:"heap.percent,string"`
	HeapMax                         *string `json:"heap.max"`
	RAMCurrent                      *string `json:"ram.current"`
	RAMPercent                      *int    `json:"ram.percent,string"`
	RAMMax                          *string `json:"ram.max"`
	FileDescCurrent                 *int    `json:"file_desc.current,string"`
	FileDescPercent                 *int    `json:"file_desc.percent,string"`
	FileDescMax                     *int    `json:"file_desc.max,string"`
	CPU                             *int    `json:"cpu,string"`
	Load1M                          *string `json:"load_1m"`
	Load5M                          *string `json:"load_5m"`
	Load15M                         *string `json:"load_15m"`
	Uptime                          *string `json:"uptime"`
	Role                            string  `json:"node.role"`
	Roles                           string  `json:"node.roles"`
	Master                          string  `json:"master"`
	ClusterManager                  string  `json:"cluster_manager"`
	Name                            string  `json:"name"`
	CompletionSize                  *string `json:"completion.size"`
	FieldDataMemorySize             *string `json:"fielddata.memory_size"`
	FileldDataEvictions             *int    `json:"fielddata.evictions,string"`
	QueryCacheMemorySize            *string `json:"query_cache.memory_size"`
	QueryCacheEvictions             *int    `json:"query_cache.evictions,string"`
	QueryCacheHitCount              *int    `json:"query_cache.hit_count,string"`
	QueryCacheMissCount             *int    `json:"query_cache.miss_count,string"`
	RequestCacheMemorySize          *string `json:"request_cache.memory_size"`
	RequestCacheEvictions           *int    `json:"request_cache.evictions,string"`
	RequestCacheHitCount            *int    `json:"request_cache.hit_count,string"`
	RequestCacheMissCount           *int    `json:"request_cache.miss_count,string"`
	FlushTotal                      *int    `json:"flush.total,string"`
	FlushTotalTime                  *string `json:"flush.total_time"`
	GetCurrent                      *int    `json:"get.current,string"`
	GetTime                         *string `json:"get.time"`
	GetTotal                        *int    `json:"get.total,string"`
	GetExistsTime                   *string `json:"get.exists_time"`
	GetExistsTotal                  *int    `json:"get.exists_total,string"`
	GetMissingTime                  *string `json:"get.missing_time"`
	GetMissingTotal                 *int    `json:"get.missing_total,string"`
	IndexingDeleteCurrent           *int    `json:"indexing.delete_current,string"`
	IndexingDeleteTime              *string `json:"indexing.delete_time"`
	IndexingDeleteTotal             *int    `json:"indexing.delete_total,string"`
	IndexingIndexCurrent            *int    `json:"indexing.index_current,string"`
	IndexingIndexTime               *string `json:"indexing.index_time"`
	IndexingIndexTotal              *int    `json:"indexing.index_total,string"`
	IndexingIndexFailed             *int    `json:"indexing.index_failed,string"`
	MergesCurrent                   *int    `json:"merges.current,string"`
	MergesCurrentDoc                *int    `json:"merges.current_docs,string"`
	MergesCurrentSize               *string `json:"merges.current_size"`
	MergesTotal                     *int    `json:"merges.total,string"`
	MergesTotalDocs                 *int    `json:"merges.total_docs,string"`
	MergesTotalSize                 *string `json:"merges.total_size"`
	MergesTotalTime                 *string `json:"merges.total_time"`
	RefreshTotal                    *int    `json:"refresh.total,string"`
	RefreshTime                     *string `json:"refresh.time"`
	RefreshExternalTotal            *int    `json:"refresh.external_total,string"`
	RefreshExternalTime             *string `json:"refresh.external_time"`
	RefreshListeners                *int    `json:"refresh.listeners,string"`
	ScriptCompilations              *int    `json:"script.compilations,string"`
	ScriptCacheEvictions            *int    `json:"script.cache_evictions,string"`
	ScriptCompilationLimitTriggered *int    `json:"script.compilation_limit_triggered,string"`
	SearchFetchCurrent              *int    `json:"search.fetch_current,string"`
	SearchFetchTime                 *string `json:"search.fetch_time"`
	SearchFetchTotal                *int    `json:"search.fetch_total,string"`
	SearchOpenContexts              *int    `json:"search.open_contexts,string"`
	SearchQueryCurrent              *int    `json:"search.query_current,string"`
	SearchQueryTime                 *string `json:"search.query_time"`
	SearchQueryTotal                *int    `json:"search.query_total,string"`
	SearchConcurrentQueryCurrent    *int    `json:"search.concurrent_query_current,string"`
	SearchConcurrentQueryTime       *string `json:"search.concurrent_query_time"`
	SearchConcurrentQueryTotal      *int    `json:"search.concurrent_query_total,string"`
	SearchConcurrentAvgSliceCount   *string `json:"search.concurrent_avg_slice_count"`
	SearchScrollCurrent             *int    `json:"search.scroll_current,string"`
	SearchScrollTime                *string `json:"search.scroll_time"`
	SearchScrollTotal               *int    `json:"search.scroll_total,string"`
	SearchPointInTimeCurrent        *int    `json:"search.point_in_time_current,string"`
	SearchPointInTimeTime           *string `json:"search.point_in_time_time"`
	SearchPointInTimeTotal          *int    `json:"search.point_in_time_total,string"`
	SegmentsCount                   *int    `json:"segments.count,string"`
	SegmentsMemory                  *string `json:"segments.memory"`
	SegmentsIndexWriteMemory        *string `json:"segments.index_writer_memory"`
	SegmentsVersionMapMemory        *string `json:"segments.version_map_memory"`
	SegmentsFixedBitsetMemory       *string `json:"segments.fixed_bitset_memory"`
	SuggestCurrent                  *int    `json:"suggest.current,string"`
	SuggestTime                     *string `json:"suggest.time"`
	SuggestTotal                    *int    `json:"suggest.total,string"`
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r CatNodesResp) Inspect() Inspect {
	return Inspect{
		Response: r.response,
	}
}
