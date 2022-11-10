package entities

import (
	"context"
	"errors"
	"strings"

	"github.com/newrelic/newrelic-client-go/pkg/common"
)

// Tag represents a New Relic One entity tag.
//
// Deprecated: Use EntityTag instead.
type Tag struct {
	Key    string
	Values []string
}

// TagValue represents a New Relic One entity tag and value pair.
//
// Deprecated: Use TaggingTagValueInput instead.
type TagValue struct {
	Key   string
	Value string
}

// GetTagsForEntity returns a collection of all tags (mutable and not) for a given
// entity by entity GUID.
func (e *Entities) GetTagsForEntity(guid common.EntityGUID) ([]*EntityTag, error) {
	return e.GetTagsForEntityWithContext(context.Background(), guid)
}

// GetTagsForEntityMutable returns a collection of all tags (mutable only) for a given
// entity by entity GUID.
func (e *Entities) GetTagsForEntityMutable(guid common.EntityGUID) ([]*EntityTag, error) {
	return e.GetTagsForEntityWithContextMutable(context.Background(), guid)
}

// GetTagsForEntityWithContext returns a collection of all tags (mutable and not) for a given
// entity by entity GUID.
func (e *Entities) GetTagsForEntityWithContext(ctx context.Context, guid common.EntityGUID) ([]*EntityTag, error) {
	resp := getTagsResponse{}
	vars := map[string]interface{}{
		"guid": guid,
	}

	if err := e.client.NerdGraphQueryWithContext(ctx, listTagsQuery, vars, &resp); err != nil {
		return nil, err
	}

	return resp.Actor.Entity.Tags, nil
}

// GetTagsForEntityWithContextMutable returns a collection of all tags (mutable only) for a given
// entity by entity GUID.
func (e *Entities) GetTagsForEntityWithContextMutable(ctx context.Context, guid common.EntityGUID) ([]*EntityTag, error) {
	resp := getTagsResponse{}
	vars := map[string]interface{}{
		"guid": guid,
	}

	if err := e.client.NerdGraphQueryWithContext(ctx, listTagsQuery, vars, &resp); err != nil {
		return nil, err
	}

	return filterEntityTagMutable(resp)
}

// filterMutable removes tag values that are read-only from the received response.
func filterEntityTagMutable(resp getTagsResponse) ([]*EntityTag, error) {
	var tags []*EntityTag

	for _, responseTag := range resp.Actor.Entity.TagsWithMetadata {
		if responseTag != nil {
			tag := EntityTag{
				Key: responseTag.Key,
			}

			mutable := 0
			for _, responseTagValue := range responseTag.Values {
				if responseTagValue.Mutable {
					mutable++
					tag.Values = append(tag.Values, responseTagValue.Value)
				}
			}

			// All values were mutable
			if len(responseTag.Values) == mutable {
				tags = append(tags, &tag)
			}

		}
	}

	return tags, nil
}

// ListTags returns a collection of mutable tags for a given entity by entity GUID.
//
// Deprecated: Use GetTagsForEntity instead.
func (e *Entities) ListTags(guid common.EntityGUID) ([]*Tag, error) {
	return e.ListTagsWithContext(context.Background(), guid)
}

// ListTagsWithContext returns a collection of mutable tags for a given entity by entity GUID.
//
// Deprecated: Use GetTagsForEntityWithContext instead.
func (e *Entities) ListTagsWithContext(ctx context.Context, guid common.EntityGUID) ([]*Tag, error) {
	resp := listTagsResponse{}
	vars := map[string]interface{}{
		"guid": guid,
	}

	if err := e.client.NerdGraphQueryWithContext(ctx, listTagsQuery, vars, &resp); err != nil {
		return nil, err
	}

	return filterMutable(resp)
}

// ListAllTags returns a collection of all tags (mutable and not) for a given
// entity by entity GUID.
//
// Deprecated: Use GetTagsForEntity instead.
func (e *Entities) ListAllTags(guid common.EntityGUID) ([]*Tag, error) {
	return e.ListAllTagsWithContext(context.Background(), guid)
}

// ListAllTagsWithContext returns a collection of all tags (mutable and not) for a given
// entity by entity GUID.
//
// Deprecated: Use GetTagsForEntityWithContext instead.
func (e *Entities) ListAllTagsWithContext(ctx context.Context, guid common.EntityGUID) ([]*Tag, error) {
	resp := listTagsResponse{}
	vars := map[string]interface{}{
		"guid": guid,
	}

	if err := e.client.NerdGraphQueryWithContext(ctx, listTagsQuery, vars, &resp); err != nil {
		return nil, err
	}

	return resp.Actor.Entity.Tags, nil
}

// filterMutable removes tag values that are read-only from the received response.
func filterMutable(resp listTagsResponse) ([]*Tag, error) {
	var tags []*Tag

	for _, responseTag := range resp.Actor.Entity.TagsWithMetadata {
		if responseTag != nil {
			tag := Tag{
				Key: responseTag.Key,
			}

			mutable := 0
			for _, responseTagValue := range responseTag.Values {
				if responseTagValue.Mutable {
					mutable++
					tag.Values = append(tag.Values, responseTagValue.Value)
				}
			}

			// All values were mutable
			if len(responseTag.Values) == mutable {
				tags = append(tags, &tag)
			}

		}
	}

	return tags, nil
}

// AddTags writes tags to the entity specified by the provided entity GUID.
//
// Deprecated: Use TaggingAddTagsToEntity instead.
func (e *Entities) AddTags(guid common.EntityGUID, tags []Tag) error {
	return e.AddTagsWithContext(context.Background(), guid, tags)
}

// AddTagsWithContext writes tags to the entity specified by the provided entity GUID.
//
// Deprecated: Use TaggingAddTagsToEntityWithContext instead.
func (e *Entities) AddTagsWithContext(ctx context.Context, guid common.EntityGUID, tags []Tag) error {
	resp := addTagsResponse{}
	vars := map[string]interface{}{
		"guid": guid,
		"tags": tags,
	}

	if err := e.client.NerdGraphQueryWithContext(ctx, addTagsMutation, vars, &resp); err != nil {
		return err
	}

	if len(resp.TaggingAddTagsToEntity.Errors) > 0 {
		return errors.New(parseTagMutationErrors(resp.TaggingAddTagsToEntity.Errors))
	}

	return nil
}

// ReplaceTags replaces the entity's entire set of tags with the provided tag set.
//
// Deprecated: Use TaggingReplaceTagsOnEntity instead.
func (e *Entities) ReplaceTags(guid common.EntityGUID, tags []Tag) error {
	return e.ReplaceTagsWithContext(context.Background(), guid, tags)
}

// ReplaceTagsWithContext replaces the entity's entire set of tags with the provided tag set.
//
// Deprecated: Use TaggingReplaceTagsOnEntityWithContext instead.
func (e *Entities) ReplaceTagsWithContext(ctx context.Context, guid common.EntityGUID, tags []Tag) error {
	resp := replaceTagsResponse{}
	vars := map[string]interface{}{
		"guid": guid,
		"tags": tags,
	}

	if err := e.client.NerdGraphQueryWithContext(ctx, replaceTagsMutation, vars, &resp); err != nil {
		return err
	}

	if len(resp.TaggingReplaceTagsOnEntity.Errors) > 0 {
		return errors.New(parseTagMutationErrors(resp.TaggingReplaceTagsOnEntity.Errors))
	}

	return nil
}

// DeleteTags deletes specific tag keys from the entity.
//
// Deprecated: Use TaggingDeleteTagFromEntity instead.
func (e *Entities) DeleteTags(guid common.EntityGUID, tagKeys []string) error {
	return e.DeleteTagsWithContext(context.Background(), guid, tagKeys)
}

// DeleteTagsWithContext deletes specific tag keys from the entity.
//
// Deprecated: Use TaggingDeleteTagFromEntityWithContext instead.
func (e *Entities) DeleteTagsWithContext(ctx context.Context, guid common.EntityGUID, tagKeys []string) error {
	resp := deleteTagsResponse{}
	vars := map[string]interface{}{
		"guid":    guid,
		"tagKeys": tagKeys,
	}

	if err := e.client.NerdGraphQueryWithContext(ctx, deleteTagsMutation, vars, &resp); err != nil {
		return err
	}

	if len(resp.TaggingDeleteTagFromEntity.Errors) > 0 {
		return errors.New(parseTagMutationErrors(resp.TaggingDeleteTagFromEntity.Errors))
	}

	return nil
}

// DeleteTagValues deletes specific tag key and value pairs from the entity.
//
// Deprecated: Use TaggingDeleteTagValuesFromEntity instead.
func (e *Entities) DeleteTagValues(guid common.EntityGUID, tagValues []TagValue) error {
	return e.DeleteTagValuesWithContext(context.Background(), guid, tagValues)
}

// DeleteTagValuesWithContext deletes specific tag key and value pairs from the entity.
//
// Deprecated: Use TaggingDeleteTagValuesFromEntityWithContext instead.
func (e *Entities) DeleteTagValuesWithContext(ctx context.Context, guid common.EntityGUID, tagValues []TagValue) error {
	resp := deleteTagValuesResponse{}
	vars := map[string]interface{}{
		"guid":      guid,
		"tagValues": tagValues,
	}

	if err := e.client.NerdGraphQueryWithContext(ctx, deleteTagValuesMutation, vars, &resp); err != nil {
		return err
	}

	if len(resp.TaggingDeleteTagValuesFromEntity.Errors) > 0 {
		return errors.New(parseTagMutationErrors(resp.TaggingDeleteTagValuesFromEntity.Errors))
	}

	return nil
}

type tagMutationError struct {
	Type    string
	Message string
}

func parseTagMutationErrors(errors []tagMutationError) string {
	messages := []string{}
	for _, e := range errors {
		messages = append(messages, e.Message)
	}

	return strings.Join(messages, ", ")
}

var listTagsQuery = `
query($guid:EntityGuid!) { actor { entity(guid: $guid)  {
  tagsWithMetadata { key values { mutable value } }
  tags { key values }
 } } }`

type listTagsResponse struct {
	Actor struct {
		Entity struct {
			Tags             []*Tag
			TagsWithMetadata []*EntityTagWithMetadata
		}
	}
}

type getTagsResponse struct {
	Actor struct {
		Entity struct {
			Tags             []*EntityTag
			TagsWithMetadata []*EntityTagWithMetadata
		}
	}
}

var addTagsMutation = `
	mutation($guid: EntityGuid!, $tags: [TaggingTagInput!]!) {
		taggingAddTagsToEntity(guid: $guid, tags: $tags) {
			errors {
				type
				message
			}
		}
	}
`

type addTagsResponse struct {
	TaggingAddTagsToEntity struct {
		Errors []tagMutationError
	}
}

var replaceTagsMutation = `
	mutation($guid: EntityGuid!, $tags: [TaggingTagInput!]!) {
		taggingReplaceTagsOnEntity(guid: $guid, tags: $tags) {
			errors {
				type
				message
			}
		}
	}
`

type replaceTagsResponse struct {
	TaggingReplaceTagsOnEntity struct {
		Errors []tagMutationError
	}
}

var deleteTagsMutation = `
	mutation($guid: EntityGuid!, $tagKeys: [String!]!) {
		taggingDeleteTagFromEntity(guid: $guid, tagKeys: $tagKeys) {
			errors {
				type
				message
			}
		}
	}
`

type deleteTagsResponse struct {
	TaggingDeleteTagFromEntity struct {
		Errors []tagMutationError
	}
}

var deleteTagValuesMutation = `
	mutation($guid: EntityGuid!, $tagValues: [TaggingTagValueInput!]!) {
		taggingDeleteTagValuesFromEntity(guid: $guid, tagValues: $tagValues) {
			errors {
				type
				message
			}
		}
	}
`

type deleteTagValuesResponse struct {
	TaggingDeleteTagValuesFromEntity struct {
		Errors []tagMutationError
	}
}
