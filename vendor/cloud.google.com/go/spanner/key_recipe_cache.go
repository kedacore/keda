/*
Copyright 2026 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package spanner

import (
	"bytes"
	"container/list"
	"encoding/binary"
	"hash"
	"hash/fnv"
	"sort"
	"sync"
	"sync/atomic"

	sppb "cloud.google.com/go/spanner/apiv1/spannerpb"
	"google.golang.org/protobuf/proto"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

const (
	defaultSchemaRecipeCacheSize  = 1000
	defaultPreparedQueryCacheSize = 50_000
	defaultPreparedReadCacheSize  = 50_000
)

type keyRecipeCache struct {
	preparedMu sync.RWMutex
	queryMu    sync.RWMutex

	nextOperationUID atomic.Uint64
	schemaSnapshot   atomic.Pointer[keyRecipeSchemaSnapshot]

	preparedReads   *boundedCache[uint64, *preparedRead]
	preparedQueries *boundedCache[uint64, *preparedQuery]
	queryRecipes    *boundedCache[uint64, *keyRecipe]
}

type keyRecipeSchemaSnapshot struct {
	schemaGeneration []byte
	schemaRecipes    map[string]*keyRecipe
}

type preparedRead struct {
	table        string
	columns      []string
	operationUID uint64
}

type preparedQueryParam struct {
	name    string
	typeRef *sppb.Type
	hasType bool
	kind    int32
}

type preparedQuery struct {
	sql          string
	params       []preparedQueryParam
	queryOptions *sppb.ExecuteSqlRequest_QueryOptions
	operationUID uint64
}

func newKeyRecipeCache() *keyRecipeCache {
	return newKeyRecipeCacheWithSizes(defaultPreparedReadCacheSize, defaultPreparedQueryCacheSize)
}

func newKeyRecipeCacheWithSizes(preparedReadCacheSize, preparedQueryCacheSize int) *keyRecipeCache {
	cache := &keyRecipeCache{
		preparedReads:   newBoundedCache[uint64, *preparedRead](preparedReadCacheSize),
		preparedQueries: newBoundedCache[uint64, *preparedQuery](preparedQueryCacheSize),
		queryRecipes:    newBoundedCache[uint64, *keyRecipe](preparedQueryCacheSize),
	}
	cache.nextOperationUID.Store(1)
	cache.schemaSnapshot.Store(&keyRecipeSchemaSnapshot{
		schemaRecipes: make(map[string]*keyRecipe, defaultSchemaRecipeCacheSize),
	})
	return cache
}

func fingerprintReadRequest(req *sppb.ReadRequest) uint64 {
	h := fnv.New64a()
	hashString(h, req.GetTable())
	hashString(h, req.GetIndex())
	hashUint64(h, uint64(len(req.GetColumns())))
	for _, column := range req.GetColumns() {
		hashString(h, column)
	}
	return h.Sum64()
}

func fingerprintExecuteSQLRequest(req *sppb.ExecuteSqlRequest) uint64 {
	h := fnv.New64a()
	hashString(h, req.GetSql())

	paramNames := make([]string, 0, len(req.GetParams().GetFields()))
	for name := range req.GetParams().GetFields() {
		paramNames = append(paramNames, name)
	}
	sort.Strings(paramNames)
	for _, name := range paramNames {
		hashString(h, name)
		if typ, ok := req.GetParamTypes()[name]; ok {
			typeBytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(typ)
			if err != nil {
				hashBytes(h, nil)
				continue
			}
			hashBytes(h, typeBytes)
		} else {
			hashUint64(h, uint64(valueKindCase(req.GetParams().GetFields()[name])))
		}
	}

	queryOptions := req.GetQueryOptions()
	if queryOptions == nil {
		queryOptions = &sppb.ExecuteSqlRequest_QueryOptions{}
	}
	queryOptionsBytes, _ := proto.MarshalOptions{Deterministic: true}.Marshal(queryOptions)
	hashBytes(h, queryOptionsBytes)
	return h.Sum64()
}

func (c *keyRecipeCache) addRecipes(recipeList *sppb.RecipeList) {
	if recipeList == nil {
		return
	}
	current := c.loadSchemaSnapshot()
	cmp := bytes.Compare(recipeList.GetSchemaGeneration(), current.schemaGeneration)
	if cmp < 0 {
		return
	}

	next := &keyRecipeSchemaSnapshot{
		schemaGeneration: append([]byte(nil), current.schemaGeneration...),
		schemaRecipes:    copyRecipeMap(current.schemaRecipes),
	}
	if cmp > 0 {
		next.schemaGeneration = append([]byte(nil), recipeList.GetSchemaGeneration()...)
		next.schemaRecipes = make(map[string]*keyRecipe, defaultSchemaRecipeCacheSize)
	}

	c.queryMu.Lock()
	if cmp > 0 {
		c.queryRecipes.Clear()
	}
	for _, recipeProto := range recipeList.GetRecipe() {
		recipe, err := newKeyRecipe(recipeProto)
		if err != nil {
			continue
		}
		switch recipeProto.GetTarget().(type) {
		case *sppb.KeyRecipe_TableName:
			next.schemaRecipes[recipeProto.GetTableName()] = recipe
		case *sppb.KeyRecipe_IndexName:
			next.schemaRecipes[recipeProto.GetIndexName()] = recipe
		case *sppb.KeyRecipe_OperationUid:
			c.queryRecipes.Put(recipeProto.GetOperationUid(), recipe)
		}
	}
	c.queryMu.Unlock()
	c.schemaSnapshot.Store(next)
}

func (c *keyRecipeCache) computeReadKeys(req *sppb.ReadRequest) {
	if req == nil {
		return
	}
	reqFP := fingerprintReadRequest(req)
	schemaSnapshot := c.loadSchemaSnapshot()

	hint := ensureReadRoutingHint(req)
	if len(schemaSnapshot.schemaGeneration) > 0 {
		hint.SchemaGeneration = append([]byte(nil), schemaSnapshot.schemaGeneration...)
	}
	prepared, ok := c.getOrPrepareRead(reqFP, req)
	if !ok {
		return
	}
	hint.OperationUid = prepared.operationUID
	recipeKey := req.GetTable()
	if req.GetIndex() != "" {
		recipeKey = req.GetIndex()
	}
	recipe := schemaSnapshot.schemaRecipes[recipeKey]

	if recipe == nil {
		return
	}
	target := recipe.keySetToTargetRange(req.GetKeySet())
	if target == nil {
		return
	}
	c.applyTargetRange(hint, target)
}

func (c *keyRecipeCache) computeQueryKeys(req *sppb.ExecuteSqlRequest) {
	if req == nil {
		return
	}
	reqFP := fingerprintExecuteSQLRequest(req)
	schemaSnapshot := c.loadSchemaSnapshot()

	hint := ensureExecuteSQLRoutingHint(req)
	if len(schemaSnapshot.schemaGeneration) > 0 {
		hint.SchemaGeneration = append([]byte(nil), schemaSnapshot.schemaGeneration...)
	}
	prepared, ok := c.getOrPrepareQuery(reqFP, req)
	if !ok {
		return
	}
	hint.OperationUid = prepared.operationUID
	c.queryMu.RLock()
	recipe, ok := c.queryRecipes.Get(prepared.operationUID)
	c.queryMu.RUnlock()

	if !ok || recipe == nil {
		return
	}
	target := recipe.queryParamsToTargetRange(req.GetParams())
	if target == nil {
		return
	}
	c.applyTargetRange(hint, target)
}

func (c *keyRecipeCache) mutationToTargetRange(mutation *sppb.Mutation) *targetRange {
	if mutation == nil {
		return nil
	}
	tableName := tableNameFromMutation(mutation)
	if tableName == "" {
		return nil
	}
	recipe := c.loadSchemaSnapshot().schemaRecipes[tableName]
	if recipe == nil {
		return nil
	}
	return recipe.mutationToTargetRange(mutation)
}

func (c *keyRecipeCache) applySchemaGeneration(hint *sppb.RoutingHint) {
	if hint == nil {
		return
	}
	schemaGeneration := c.loadSchemaSnapshot().schemaGeneration
	if len(schemaGeneration) > 0 {
		hint.SchemaGeneration = append([]byte(nil), schemaGeneration...)
	}
}

func (c *keyRecipeCache) applyTargetRange(hint *sppb.RoutingHint, target *targetRange) {
	if hint == nil || target == nil {
		return
	}
	hint.Key = append(hint.Key[:0], target.start...)
	hint.LimitKey = hint.LimitKey[:0]
	if len(target.limit) > 0 {
		hint.LimitKey = append(hint.LimitKey, target.limit...)
	}
}

func (c *keyRecipeCache) clear() {
	c.schemaSnapshot.Store(&keyRecipeSchemaSnapshot{
		schemaRecipes: make(map[string]*keyRecipe, defaultSchemaRecipeCacheSize),
	})
	c.preparedMu.Lock()
	defer c.preparedMu.Unlock()
	c.preparedReads.Clear()
	c.preparedQueries.Clear()
	c.queryMu.Lock()
	defer c.queryMu.Unlock()
	c.queryRecipes.Clear()
}

func (c *keyRecipeCache) loadSchemaSnapshot() *keyRecipeSchemaSnapshot {
	snapshot := c.schemaSnapshot.Load()
	if snapshot == nil {
		return &keyRecipeSchemaSnapshot{
			schemaRecipes: make(map[string]*keyRecipe, defaultSchemaRecipeCacheSize),
		}
	}
	return snapshot
}

func (c *keyRecipeCache) getOrPrepareRead(reqFP uint64, req *sppb.ReadRequest) (*preparedRead, bool) {
	c.preparedMu.RLock()
	prepared, ok := c.preparedReads.Get(reqFP)
	c.preparedMu.RUnlock()
	if ok {
		return prepared, prepared.matches(req)
	}

	c.preparedMu.Lock()
	defer c.preparedMu.Unlock()
	prepared, ok = c.preparedReads.Get(reqFP)
	if ok {
		return prepared, prepared.matches(req)
	}
	prepared = &preparedRead{table: req.GetTable(), columns: append([]string(nil), req.GetColumns()...)}
	prepared.operationUID = c.nextOperationUID.Add(1) - 1
	c.preparedReads.Put(reqFP, prepared)
	return prepared, true
}

func (c *keyRecipeCache) getOrPrepareQuery(reqFP uint64, req *sppb.ExecuteSqlRequest) (*preparedQuery, bool) {
	c.preparedMu.RLock()
	prepared, ok := c.preparedQueries.Get(reqFP)
	c.preparedMu.RUnlock()
	if ok {
		return prepared, prepared.matches(req)
	}

	c.preparedMu.Lock()
	defer c.preparedMu.Unlock()
	prepared, ok = c.preparedQueries.Get(reqFP)
	if ok {
		return prepared, prepared.matches(req)
	}
	prepared = newPreparedQuery(req)
	prepared.operationUID = c.nextOperationUID.Add(1) - 1
	c.preparedQueries.Put(reqFP, prepared)
	return prepared, true
}

func copyRecipeMap(src map[string]*keyRecipe) map[string]*keyRecipe {
	dst := make(map[string]*keyRecipe, len(src))
	for key, recipe := range src {
		dst[key] = recipe
	}
	return dst
}

func (p *preparedRead) matches(req *sppb.ReadRequest) bool {
	if req == nil {
		return false
	}
	if p.table != req.GetTable() || len(p.columns) != len(req.GetColumns()) {
		return false
	}
	for i := range p.columns {
		if p.columns[i] != req.GetColumns()[i] {
			return false
		}
	}
	return true
}

func newPreparedQuery(req *sppb.ExecuteSqlRequest) *preparedQuery {
	params := make([]preparedQueryParam, 0, len(req.GetParams().GetFields()))
	for name, value := range req.GetParams().GetFields() {
		if typ, ok := req.GetParamTypes()[name]; ok {
			params = append(params, preparedQueryParam{name: name, typeRef: typ, hasType: true})
		} else {
			params = append(params, preparedQueryParam{name: name, kind: valueKindCase(value)})
		}
	}
	sort.Slice(params, func(i, j int) bool { return params[i].name < params[j].name })
	queryOptions := req.GetQueryOptions()
	if queryOptions == nil {
		queryOptions = &sppb.ExecuteSqlRequest_QueryOptions{}
	}
	return &preparedQuery{sql: req.GetSql(), params: params, queryOptions: proto.Clone(queryOptions).(*sppb.ExecuteSqlRequest_QueryOptions)}
}

func (p *preparedQuery) matches(req *sppb.ExecuteSqlRequest) bool {
	if req == nil || p.sql != req.GetSql() || len(p.params) != len(req.GetParams().GetFields()) {
		return false
	}
	for _, param := range p.params {
		value, ok := req.GetParams().GetFields()[param.name]
		if !ok {
			return false
		}
		if param.hasType {
			typ, ok := req.GetParamTypes()[param.name]
			if !ok || !proto.Equal(param.typeRef, typ) {
				return false
			}
			continue
		}
		if _, ok := req.GetParamTypes()[param.name]; ok {
			return false
		}
		if param.kind != valueKindCase(value) {
			return false
		}
	}
	queryOptions := req.GetQueryOptions()
	if queryOptions == nil {
		queryOptions = &sppb.ExecuteSqlRequest_QueryOptions{}
	}
	return proto.Equal(p.queryOptions, queryOptions)
}

func tableNameFromMutation(mutation *sppb.Mutation) string {
	switch op := mutation.GetOperation().(type) {
	case *sppb.Mutation_Insert:
		return op.Insert.GetTable()
	case *sppb.Mutation_Update:
		return op.Update.GetTable()
	case *sppb.Mutation_InsertOrUpdate:
		return op.InsertOrUpdate.GetTable()
	case *sppb.Mutation_Replace:
		return op.Replace.GetTable()
	case *sppb.Mutation_Delete_:
		return op.Delete.GetTable()
	default:
		return ""
	}
}

func ensureReadRoutingHint(req *sppb.ReadRequest) *sppb.RoutingHint {
	if req.RoutingHint == nil {
		req.RoutingHint = &sppb.RoutingHint{}
	}
	return req.RoutingHint
}

func ensureExecuteSQLRoutingHint(req *sppb.ExecuteSqlRequest) *sppb.RoutingHint {
	if req.RoutingHint == nil {
		req.RoutingHint = &sppb.RoutingHint{}
	}
	return req.RoutingHint
}

func ensureBeginTransactionRoutingHint(req *sppb.BeginTransactionRequest) *sppb.RoutingHint {
	if req.RoutingHint == nil {
		req.RoutingHint = &sppb.RoutingHint{}
	}
	return req.RoutingHint
}

func ensureCommitRoutingHint(req *sppb.CommitRequest) *sppb.RoutingHint {
	if req.RoutingHint == nil {
		req.RoutingHint = &sppb.RoutingHint{}
	}
	return req.RoutingHint
}

func hashString(h hash.Hash64, s string) {
	hashBytes(h, []byte(s))
}

func hashUint64(h hash.Hash64, v uint64) {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], v)
	_, _ = h.Write(buf[:])
}

func hashBytes(h hash.Hash64, b []byte) {
	hashUint64(h, uint64(len(b)))
	if len(b) > 0 {
		_, _ = h.Write(b)
	}
}

func valueKindCase(value *structpb.Value) int32 {
	if value == nil || value.GetKind() == nil {
		return 0
	}
	switch value.GetKind().(type) {
	case *structpb.Value_NullValue:
		return 1
	case *structpb.Value_NumberValue:
		return 2
	case *structpb.Value_StringValue:
		return 3
	case *structpb.Value_BoolValue:
		return 4
	case *structpb.Value_StructValue:
		return 5
	case *structpb.Value_ListValue:
		return 6
	default:
		return 0
	}
}

// boundedCache is a non-thread-safe fixed-size map with insertion-order
// eviction. Cache hits do not mutate the structure.
type boundedCache[K comparable, V any] struct {
	maxSize int
	items   map[K]*list.Element
	order   *list.List
}

type boundedCacheEntry[K comparable, V any] struct {
	key   K
	value V
}

func newBoundedCache[K comparable, V any](maxSize int) *boundedCache[K, V] {
	if maxSize < 1 {
		maxSize = 1
	}
	return &boundedCache[K, V]{
		maxSize: maxSize,
		items:   make(map[K]*list.Element, maxSize),
		order:   list.New(),
	}
}

func (c *boundedCache[K, V]) Get(key K) (V, bool) {
	elem, ok := c.items[key]
	if !ok {
		var zero V
		return zero, false
	}
	return elem.Value.(*boundedCacheEntry[K, V]).value, true
}

func (c *boundedCache[K, V]) Put(key K, value V) {
	if elem, ok := c.items[key]; ok {
		entry := elem.Value.(*boundedCacheEntry[K, V])
		entry.value = value
		return
	}
	elem := c.order.PushBack(&boundedCacheEntry[K, V]{key: key, value: value})
	c.items[key] = elem

	for len(c.items) > c.maxSize {
		first := c.order.Front()
		if first == nil {
			return
		}
		c.order.Remove(first)
		delete(c.items, first.Value.(*boundedCacheEntry[K, V]).key)
	}
}

func (c *boundedCache[K, V]) Clear() {
	clear(c.items)
	c.order.Init()
}
