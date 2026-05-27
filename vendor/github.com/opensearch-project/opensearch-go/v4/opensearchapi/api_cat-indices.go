// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"net/http"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4"
)

// CatIndicesReq represent possible options for the /_cat/indices request
type CatIndicesReq struct {
	Indices []string
	Header  http.Header
	Params  CatIndicesParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r CatIndicesReq) GetRequest() (*http.Request, error) {
	indices := strings.Join(r.Indices, ",")
	var path strings.Builder
	path.Grow(len("/_cat/indices/") + len(indices))
	path.WriteString("/_cat/indices")
	if len(r.Indices) > 0 {
		path.WriteString("/")
		path.WriteString(indices)
	}
	return opensearch.BuildRequest(
		"GET",
		path.String(),
		nil,
		r.Params.get(),
		r.Header,
	)
}

// CatIndicesResp represents the returned struct of the /_cat/indices response
type CatIndicesResp struct {
	Indices  []CatIndexResp
	response *opensearch.Response
}

// CatIndexResp represents one index of the CatIndicesResp
type CatIndexResp struct {
	Health             string `json:"health"`
	Status             string `json:"status"`
	Index              string `json:"index"`
	UUID               string `json:"uuid"`
	Primary            *int   `json:"pri,string"`
	Replica            *int   `json:"rep,string"`
	DocsCount          *int   `json:"docs.count,string"`
	DocDeleted         *int   `json:"docs.deleted,string"`
	CreationDate       int    `json:"creation.date,string"`
	CreationDateString string `json:"creation.date.string"`
	// Pointer as newly created indices can return null
	StoreSize                            *string `json:"store.size"`
	PrimaryStoreSize                     *string `json:"pri.store.size"`
	CompletionSize                       *string `json:"completion.size"`
	PrimaryCompletionSize                *string `json:"pri.completion.size"`
	FieldDataMemorySize                  *string `json:"fielddata.memory_size"`
	PrimaryFieldDataMemorySize           *string `json:"pri.fielddata.memory_size"`
	FieldDataEvictions                   *int    `json:"fielddata.evictions,string"`
	PrimaryFieldDataEvictions            *int    `json:"pri.fielddata.evictions,string"`
	QueryCacheMemorySize                 *string `json:"query_cache.memory_size"`
	PrimaryQueryCacheMemorySize          *string `json:"pri.query_cache.memory_size"`
	QueryCacheEvictions                  *int    `json:"query_cache.evictions,string"`
	PrimaryQueryCacheEvictions           *int    `json:"pri.query_cache.evictions,string"`
	RequestCacheMemorySize               *string `json:"request_cache.memory_size"`
	PrimaryRequestCacheMemorySize        *string `json:"pri.request_cache.memory_size"`
	RequestCacheEvictions                *int    `json:"request_cache.evictions,string"`
	PrimaryRequestCacheEvictions         *int    `json:"pri.request_cache.evictions,string"`
	RequestCacheHitCount                 *int    `json:"request_cache.hit_count,string"`
	PrimaryRequestCacheHitCount          *int    `json:"pri.request_cache.hit_count,string"`
	RequestCacheMissCount                *int    `json:"request_cache.miss_count,string"`
	PrimaryRequestCacheMissCount         *int    `json:"pri.request_cache.miss_count,string"`
	FlushTotal                           *int    `json:"flush.total,string"`
	PrimaryFlushTotal                    *int    `json:"pri.flush.total,string"`
	FlushTime                            *string `json:"flush.total_time"`
	PrimaryFlushTime                     *string `json:"pri.flush.total_time"`
	GetCurrent                           *int    `json:"get.current,string"`
	PrimaryGetCurrent                    *int    `json:"pri.get.current,string"`
	GetTime                              *string `json:"get.time"`
	PrimaryGetTime                       *string `json:"pri.get.time"`
	GetTotal                             *int    `json:"get.total,string"`
	PrimaryGetTotal                      *int    `json:"pri.get.total,string"`
	GetExistsTime                        *string `json:"get.exists_time"`
	PrimaryGetExistsTime                 *string `json:"pri.get.exists_time"`
	GetExistsTotal                       *int    `json:"get.exists_total,string"`
	PrimaryGetExistsTotal                *int    `json:"pri.get.exists_total,string"`
	GetMissingTime                       *string `json:"get.missing_time"`
	PrimaryGetMissingTime                *string `json:"pri.get.missing_time"`
	GetMissingTotal                      *int    `json:"get.missing_total,string"`
	PrimaryGetMissingTotal               *int    `json:"pri.get.missing_total,string"`
	IndexingDeleteCurrent                *int    `json:"indexing.delete_current,string"`
	PrimaryIndexingDeleteCurrent         *int    `json:"pri.indexing.delete_current,string"`
	IndexingDeleteTime                   *string `json:"indexing.delete_time"`
	PrimaryIndexingDeleteTime            *string `json:"pri.indexing.delete_time"`
	IndexingDeleteTotal                  *int    `json:"indexing.delete_total,string"`
	PrimaryIndexingDeleteTotal           *int    `json:"pri.indexing.delete_total,string"`
	IndexingIndexCurrent                 *int    `json:"indexing.index_current,string"`
	PrimaryIndexingIndexCurrent          *int    `json:"pri.indexing.index_current,string"`
	IndexingIndexTime                    *string `json:"indexing.index_time"`
	PrimaryIndexingIndexTime             *string `json:"pri.indexing.index_time"`
	IndexingIndexTotal                   *int    `json:"indexing.index_total,string"`
	PrimaryIndexingIndexTotal            *int    `json:"pri.indexing.index_total,string"`
	IndexingIndexFailed                  *int    `json:"indexing.index_failed,string"`
	PrimaryIndexingIndexFailed           *int    `json:"pri.indexing.index_failed,string"`
	MergesCurrent                        *int    `json:"merges.current,string"`
	PrimaryMergesCurrent                 *int    `json:"pri.merges.current,string"`
	MergesCurrentDocs                    *int    `json:"merges.current_docs,string"`
	PrimaryMergesCurrentDocs             *int    `json:"pri.merges.current_docs,string"`
	MergesCurrentSize                    *string `json:"merges.current_size"`
	PrimaryMergesCurrentSize             *string `json:"pri.merges.current_size"`
	MergesTotal                          *int    `json:"merges.total,string"`
	PrimaryMergesTotal                   *int    `json:"pri.merges.total,string"`
	MergesTotalDocs                      *int    `json:"merges.total_docs,string"`
	PrimaryMergesTotalDocs               *int    `json:"pri.merges.total_docs,string"`
	MergesTotalSize                      *string `json:"merges.total_size"`
	PrimaryMergesTotalSize               *string `json:"pri.merges.total_size"`
	MergesTotalTime                      *string `json:"merges.total_time"`
	PrimaryMergesTotalTime               *string `json:"pri.merges.total_time"`
	RefreshTotal                         *int    `json:"refresh.total,string"`
	PrimaryRefreshTotal                  *int    `json:"pri.refresh.total,string"`
	RefreshTime                          *string `json:"refresh.time"`
	PrimaryRefreshTime                   *string `json:"pri.refresh.time"`
	RefreshExternalTotal                 *int    `json:"refresh.external_total,string"`
	PrimaryRefreshExternalTotal          *int    `json:"pri.refresh.external_total,string"`
	RefreshExternalTime                  *string `json:"refresh.external_time"`
	PrimaryRefreshExternalTime           *string `json:"pri.refresh.external_time"`
	RefreshListeners                     *int    `json:"refresh.listeners,string"`
	PrimaryRefreshListeners              *int    `json:"pri.refresh.listeners,string"`
	SearchFetchCurrent                   *int    `json:"search.fetch_current,string"`
	PrimarySearchFetchCurrent            *int    `json:"pri.search.fetch_current,string"`
	SearchFetchTime                      *string `json:"search.fetch_time"`
	PrimarySearchFetchTime               *string `json:"pri.search.fetch_time"`
	SearchFetchTotal                     *int    `json:"search.fetch_total,string"`
	PrimarySearchFetchTotal              *int    `json:"pri.search.fetch_total,string"`
	SearchOpenContexts                   *int    `json:"search.open_contexts,string"`
	PrimarySearchOpenContexts            *int    `json:"pri.search.open_contexts,string"`
	SearchQueryCurrent                   *int    `json:"search.query_current,string"`
	PrimarySearchQueryCurrent            *int    `json:"pri.search.query_current,string"`
	SearchQueryTime                      *string `json:"search.query_time"`
	PrimarySearchQueryTime               *string `json:"pri.search.query_time"`
	SearchQueryTotal                     *int    `json:"search.query_total,string"`
	PrimarySearchQueryTotal              *int    `json:"pri.search.query_total,string"`
	SearchConcurrentQueryCurrent         *int    `json:"search.concurrent_query_current,string"`
	PrimarySearchConcurrentQueryCurrent  *int    `json:"pri.search.concurrent_query_current,string"`
	SearchConcurrentQueryTime            *string `json:"search.concurrent_query_time"`
	PrimarySearchConcurrentQueryTime     *string `json:"pri.search.concurrent_query_time"`
	SearchConcurrentQueryTotal           *int    `json:"search.concurrent_query_total,string"`
	PrimarySearchConcurrentQueryTotal    *int    `json:"pri.search.concurrent_query_total,string"`
	SearchConcurrentAvgSliceCount        *string `json:"search.concurrent_avg_slice_count"`
	PrimarySearchConcurrentAvgSliceCount *string `json:"pri.search.concurrent_avg_slice_count"`
	SearchScrollCurrent                  *int    `json:"search.scroll_current,string"`
	PrimarySearchScrollCurrent           *int    `json:"pri.search.scroll_current,string"`
	SearchScrollTime                     *string `json:"search.scroll_time"`
	PrimarySearchScrollTime              *string `json:"pri.search.scroll_time"`
	SearchScrollTotal                    *int    `json:"search.scroll_total,string"`
	PrimarySearchScrollTotal             *int    `json:"pri.search.scroll_total,string"`
	SearchPointInTimeCurrent             *string `json:"search.point_in_time_current"`
	PrimarySearchPointInTimeCurrent      *string `json:"pri.search.point_in_time_current"`
	SearchPointInTimeTime                *string `json:"search.point_in_time_time"`
	PrimarySearchPointInTimeTime         *string `json:"pri.search.point_in_time_time"`
	SearchPointInTimeTotal               *int    `json:"search.point_in_time_total,string"`
	PrimarySearchPointInTimeTotal        *int    `json:"pri.search.point_in_time_total,string"`
	SegmentsCount                        *int    `json:"segments.count,string"`
	PrimarySegmentsCount                 *int    `json:"pri.segments.count,string"`
	SegmentsMemory                       *string `json:"segments.memory"`
	PrimarySegmentsMemory                *string `json:"pri.segments.memory"`
	SegmentsIndexWriteMemory             *string `json:"segments.index_writer_memory"`
	PrimarySegmentsIndexWriteMemory      *string `json:"pri.segments.index_writer_memory"`
	SegmentsVersionMapMemory             *string `json:"segments.version_map_memory"`
	PrimarySegmentsVersionMapMemory      *string `json:"pri.segments.version_map_memory"`
	SegmentsFixedBitsetMemory            *string `json:"segments.fixed_bitset_memory"`
	PrimarySegmentsFixedBitsetMemory     *string `json:"pri.segments.fixed_bitset_memory"`
	WarmerCurrent                        *int    `json:"warmer.current,string"`
	PrimaryWarmerCurrent                 *int    `json:"pri.warmer.current,string"`
	WarmerTotal                          *int    `json:"warmer.total,string"`
	PrimaryWarmerTotal                   *int    `json:"pri.warmer.total,string"`
	WarmerTotalTime                      *string `json:"warmer.total_time"`
	PrimaryWarmerTotalTime               *string `json:"pri.warmer.total_time"`
	SuggestCurrent                       *int    `json:"suggest.current,string"`
	PrimarySuggestCurrent                *int    `json:"pri.suggest.current,string"`
	SuggestTime                          *string `json:"suggest.time"`
	PrimarySuggestTime                   *string `json:"pri.suggest.time"`
	SuggestTotal                         *int    `json:"suggest.total,string"`
	PrimarySuggestTotal                  *int    `json:"pri.suggest.total,string"`
	MemoryTotal                          string  `json:"memory.total"`
	PrimaryMemoryTotal                   string  `json:"pri.memory.total"`
	SearchThrottled                      bool    `json:"search.throttled,string"`
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r CatIndicesResp) Inspect() Inspect {
	return Inspect{
		Response: r.response,
	}
}
