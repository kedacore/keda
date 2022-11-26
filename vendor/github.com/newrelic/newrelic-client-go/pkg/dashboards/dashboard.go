package dashboards

import (
	"context"

	"github.com/newrelic/newrelic-client-go/pkg/common"
	"github.com/newrelic/newrelic-client-go/pkg/entities"
	"github.com/newrelic/newrelic-client-go/pkg/errors"
)

// GetDashboardEntity is used to retrieve a single New Relic One Dashboard
func (d *Dashboards) GetDashboardEntity(gUID common.EntityGUID) (*entities.DashboardEntity, error) {
	return d.GetDashboardEntityWithContext(context.Background(), gUID)
}

// GetDashboardEntityWithContext is used to retrieve a single New Relic One Dashboard
func (d *Dashboards) GetDashboardEntityWithContext(ctx context.Context, gUID common.EntityGUID) (*entities.DashboardEntity, error) {
	resp := struct {
		Actor entities.Actor `json:"actor"`
	}{}
	vars := map[string]interface{}{
		"guid": gUID,
	}

	if err := d.client.NerdGraphQueryWithContext(ctx, getDashboardEntityQuery, vars, &resp); err != nil {
		return nil, err
	}

	if resp.Actor.Entity == nil {
		return nil, errors.NewNotFound("entity not found. GUID: '" + string(gUID) + "'")
	}

	return resp.Actor.Entity.(*entities.DashboardEntity), nil
}

// getDashboardEntityQuery is not auto-generated as tutone does not currently support
// generation of queries that return a specific interface.
const getDashboardEntityQuery = `query ($guid: EntityGuid!) {
  actor {
    entity(guid: $guid) {
      guid
      ... on DashboardEntity {
        __typename
        accountId
        createdAt
        dashboardParentGuid
        description
        indexedAt
        name
        owner { email userId }
        pages {
          createdAt
          description
          guid
          name
          owner { email userId }
          updatedAt
          widgets {
            rawConfiguration
            configuration {
              area { nrqlQueries { accountId query } }
              bar { nrqlQueries { accountId query } }
              billboard { nrqlQueries { accountId query } thresholds { alertSeverity value } }
              line { nrqlQueries { accountId query } }
              markdown { text }
              pie { nrqlQueries { accountId query } }
              table { nrqlQueries { accountId query } }
            }
            layout { column height row width }
            title
            visualization { id }
            id
            linkedEntities {
              __typename
              guid
              name
              accountId
              tags { key values }
              ... on DashboardEntityOutline {
                dashboardParentGuid
              }
            }
          }
        }
        permalink
        permissions
        tags { key values }
        tagsWithMetadata { key values { mutable value } }
        updatedAt
      }
    }
  }
}`
