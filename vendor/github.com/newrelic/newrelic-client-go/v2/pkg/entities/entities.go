// Package entities provides a programmatic API for interacting with New Relic One entities.
package entities

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/newrelic/newrelic-client-go/v2/internal/http"
	"github.com/newrelic/newrelic-client-go/v2/pkg/config"
	"github.com/newrelic/newrelic-client-go/v2/pkg/logging"
)

// Entities is used to communicate with the New Relic Entities product.
type Entities struct {
	client http.Client
	logger logging.Logger
}

// New returns a new client for interacting with New Relic One entities.
func New(config config.Config) Entities {
	return Entities{
		client: http.NewClient(config),
		logger: config.GetLogger(),
	}
}

type EntitySearchParams struct {
	Name            string
	Domain          string
	Type            string
	AlertSeverity   string
	IsReporting     *bool
	IsCaseSensitive bool
	Tags            []map[string]string
}

func BuildEntitySearchNrqlQuery(params EntitySearchParams) string {
	paramsMap := map[string]string{
		"name":   params.Name,
		"domain": params.Domain,
		"type":   params.Type,
	}

	count := 0
	query := ""
	for k, v := range paramsMap {
		if v == "" {
			continue
		}

		// The default entity name search operator is `LIKE`.
		// Change to `=` if case-sensitive option set to `true`.
		matchOperator := "LIKE"
		if k == "name" && params.IsCaseSensitive {
			matchOperator = "="
		}

		if count == 0 {
			if k == "name" {
				// Handle case-sensitive name param
				query = fmt.Sprintf("%s %s '%s'", k, matchOperator, v)
			} else {
				query = fmt.Sprintf("%s = '%s'", k, v)
			}
		} else {
			if k == "name" {
				// Handle case-sensitive name param
				query = fmt.Sprintf("%s AND %s %s '%s'", query, k, matchOperator, v)
			} else {
				query = fmt.Sprintf("%s AND %s = '%s'", query, k, v)
			}

			query = fmt.Sprintf("%s AND %s = '%s'", query, k, v)
		}

		count++
	}

	if len(params.Tags) > 0 {
		if count > 0 {
			query = fmt.Sprintf("%s AND %s", query, BuildTagsNrqlQueryFragment(params.Tags))
		} else {
			query = BuildTagsNrqlQueryFragment(params.Tags)
		}
	}

	if params.IsReporting != nil {
		if count == 0 {
			query = fmt.Sprintf("reporting = '%s'", strconv.FormatBool(*params.IsReporting))
		} else {
			query = fmt.Sprintf("%s AND reporting = '%s'", query, strconv.FormatBool(*params.IsReporting))
		}
	}

	return query
}

func BuildTagsNrqlQueryFragment(tags []map[string]string) string {
	var query string

	for i, t := range tags {
		var q string
		if i > 0 {
			q = fmt.Sprintf(" AND tags.`%s` = '%s'", t["key"], t["value"])
		} else {
			q = fmt.Sprintf("tags.`%s` = '%s'", t["key"], t["value"])
		}

		query = fmt.Sprintf("%s%s", query, q)
	}

	return query
}

func ConvertTagsToMap(tags []string) ([]map[string]string, error) {
	tagBuilder := make([]map[string]string, 0)

	for _, x := range tags {
		if !strings.Contains(x, ":") {
			return []map[string]string{}, errors.New("tags must be specified as colon separated key:value pairs")
		}

		v := strings.SplitN(x, ":", 2)

		tagBuilder = append(tagBuilder, map[string]string{
			"key":   v[0],
			"value": v[1],
		})
	}
	return tagBuilder, nil
}
