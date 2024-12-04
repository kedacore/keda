package entities

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	DELIMITER = "|"
)

var EntityGUIDValidationErrorTypes = struct {
	INVALID_ENTITY_GUID_ERROR EntityGUIDValidationError
	EMPTY_ENTITY_TYPE_ERROR   EntityGUIDValidationError
	EMPTY_DOMAIN_ID_ERROR     EntityGUIDValidationError
}{
	INVALID_ENTITY_GUID_ERROR: errors.New("invalid entity GUID format"),
	EMPTY_ENTITY_TYPE_ERROR:   errors.New("entity type is required"),
	EMPTY_DOMAIN_ID_ERROR:     errors.New("domain ID is required"),
}

type EntityGUIDValidationError error

// DecodedEntity represents the decoded entity information
type DecodedEntity struct {
	AccountId  int64  `json:"accountId"`
	Domain     string `json:"domain"`
	EntityType string `json:"entityType"`
	DomainId   string `json:"domainId"`
}

// DecodeEntityGuid decodes a string representation of an entity GUID and returns an GenericEntity (replaced with struct)
func DecodeEntityGuid(entityGuid string) (*DecodedEntity, error) {
	decodedGuid, err := base64.RawStdEncoding.DecodeString(entityGuid)
	if err != nil {
		return nil, EntityGUIDValidationErrorTypes.INVALID_ENTITY_GUID_ERROR
	}

	parts := strings.Split(string(decodedGuid), "|")
	if len(parts) < 4 {
		return nil, fmt.Errorf("%s: expected at least 4 parts delimited by '%s': %s", EntityGUIDValidationErrorTypes.INVALID_ENTITY_GUID_ERROR, DELIMITER, entityGuid)
	}

	accountId, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid account ID: %w", err)
	}

	domain := parts[1]
	entityType := parts[2]
	domainId := parts[3]

	if entityType == "" {
		return nil, EntityGUIDValidationErrorTypes.EMPTY_ENTITY_TYPE_ERROR
	}

	if domainId == "" {
		return nil, EntityGUIDValidationErrorTypes.EMPTY_DOMAIN_ID_ERROR
	}

	return &DecodedEntity{
		AccountId:  accountId,
		Domain:     domain,
		EntityType: entityType,
		DomainId:   domainId,
	}, nil
}
