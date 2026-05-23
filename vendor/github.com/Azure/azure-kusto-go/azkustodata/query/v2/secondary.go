package v2

import (
	"github.com/Azure/azure-kusto-go/azkustodata/errors"
	"github.com/Azure/azure-kusto-go/azkustodata/query"
	"github.com/google/uuid"
	"time"
)

// This file handles the parsing of the known secondary tables in v2 datasets.

// QueryProperties represents the query properties table, which arrives before the first result.
type QueryProperties struct {
	TableId int
	Key     string
	Value   map[string]interface{}
}

// QueryCompletionInformation represents the query completion information table, which arrives after the last result.
type QueryCompletionInformation struct {
	Timestamp        time.Time
	ClientRequestId  string
	ActivityId       uuid.UUID
	SubActivityId    uuid.UUID
	ParentActivityId uuid.UUID
	Level            int
	LevelName        string
	StatusCode       int
	StatusCodeName   string
	EventType        int
	EventTypeName    string
	Payload          string
}

const QueryPropertiesKind = "QueryProperties"
const QueryCompletionInformationKind = "QueryCompletionInformation"

func AsQueryProperties(table query.BaseTable) ([]QueryProperties, error) {
	if table.Kind() != QueryPropertiesKind {
		return nil, errors.ES(errors.OpQuery, errors.KWrongTableKind, "expected QueryProperties table, got %s", table.Kind())
	}

	return query.ToStructs[QueryProperties](table)
}

func AsQueryCompletionInformation(table query.BaseTable) ([]QueryCompletionInformation, error) {
	if table.Kind() != QueryCompletionInformationKind {
		return nil, errors.ES(errors.OpQuery, errors.KWrongTableKind, "expected QueryCompletionInformation table, got %s", table.Kind())
	}

	return query.ToStructs[QueryCompletionInformation](table)
}
