# pipelinecontrol

The `pipelinecontrol` package is a collection of functions in Go, used to manage resources in New Relic associated with Pipeline Control. Currently, this package supports managing Pipeline Cloud Rules via New Relic’s Entity Management API. It lets you create, read, update, delete pipeline cloud rules that process inbound telemetry (for example, drop logs) using NRQL.

- Create rules: define NRQL-based drop filters.
- Get rules: fetch full details of the pipeline cloud rule entity, and metadata.
- Update rules: change name, description, or NRQL.
- Delete rules: remove entities by ID.

## ⚠️ Important: NRQL Drop Rules Deprecation Notice and Upcoming EOL

NRQL Drop Rules are being deprecated and will reach their end-of-life on June 30, 2026; these shall be replaced by Pipeline Cloud Rules. If you manage your droprules via the New Relic Go Client `nrqldroprules` package, we recommend migrating your scripts using functions in `nrqldroprules` to the functions described in this package as soon as possible to ensure uninterrupted service and to take advantage of the new capabilities. These new Pipeline Cloud Rules provide enhanced functionality for managing telemetry data processing with improved performance and reliability.

## Install

```go
import "github.com/newrelic/newrelic-client-go/v2/pkg/pipelinecontrol"
```

## Create a client

```go
package main

import (
	"github.com/newrelic/newrelic-client-go/v2/pkg/config"
	"github.com/newrelic/newrelic-client-go/v2/pkg/pipelinecontrol"
)

func main() {
	cfg := config.New()
	cfg.PersonalAPIKey = "YOUR_API_KEY"
	// Optional: cfg.Region = "EU" // default is US

	client := pipelinecontrol.New(cfg)
	_ = client
}
```

## Key types

- `EntityManagementPipelineCloudRuleEntityCreateInput`
- `EntityManagementPipelineCloudRuleEntityUpdateInput`
- `EntityManagementScopedReferenceInput`
- `EntityManagementPipelineCloudRuleEntity`
- `EntityManagementEntityInterface`

Note: For NRQL values, use the `nrdb.NRQL` type when setting NRQL on inputs.

## Imports used in examples

```go
import (
	"fmt"
	"log"

	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
	"github.com/newrelic/newrelic-client-go/v2/pkg/pipelinecontrol"
)
```

## Create a rule (`EntityManagementCreatePipelineCloudRule`)

Purpose: Create a Pipeline Cloud Rule with a name, description, NRQL, and scope.

```go
func createRule() {
	// assume client created
	createInput := pipelinecontrol.EntityManagementPipelineCloudRuleEntityCreateInput{
		Name:        "drop-debug-logs",
		Description: "Drop DEBUG logs in production",
		NRQL:        nrdb.NRQL("DELETE FROM Log WHERE logLevel = 'DEBUG' AND environment = 'production'"),
		Scope: pipelinecontrol.EntityManagementScopedReferenceInput{
			Type: pipelinecontrol.EntityManagementEntityScopeTypes.ACCOUNT,
			ID:   "YOUR_ACCOUNT_ID",
		},
	}

	result, err := client.EntityManagementCreatePipelineCloudRule(createInput)
	if err != nil {
		log.Fatalf("create failed: %v", err)
	}

	fmt.Printf("Created rule: id=%s name=%s version=%d\n",
		result.Entity.ID, result.Entity.Name, result.Entity.Metadata.Version)
}
```

## Get a rule (`GetEntity`)

Purpose: Fetch the entity and access typed fields on a Pipeline Cloud Rule.

```go
func getRule(id string) {
	entity, err := client.GetEntity(id)
	if err != nil {
		log.Fatalf("get failed: %v", err)
	}
	rule, ok := (*entity).(*pipelinecontrol.EntityManagementPipelineCloudRuleEntity)
	if !ok {
		log.Fatalf("entity %s is not a PipelineCloudRuleEntity", id)
	}
	fmt.Printf("Rule: id=%s name=%s version=%d\n", rule.ID, rule.Name, rule.Metadata.Version)
	fmt.Printf("NRQL: %s\n", rule.NRQL)
}
```

## Update a rule (`EntityManagementUpdatePipelineCloudRule`)

Purpose: Change name, description, or NRQL. The API handles versioning internally.

```go
func updateRule(id string) {
	updateInput := pipelinecontrol.EntityManagementPipelineCloudRuleEntityUpdateInput{
		Name:        "drop-debug-logs-updated",
		Description: "Drop DEBUG logs everywhere",
		NRQL:        nrdb.NRQL("DELETE FROM Log WHERE logLevel = 'DEBUG'"),
	}
	result, err := client.EntityManagementUpdatePipelineCloudRule(id, updateInput)
	if err != nil {
		log.Fatalf("update failed: %v", err)
	}
	fmt.Printf("Updated rule: id=%s name=%s version=%d\n",
		result.Entity.ID, result.Entity.Name, result.Entity.Metadata.Version)
}
```

## Delete a rule (`EntityManagementDelete`)

Purpose: Remove an entity by ID.

```go
func deleteRule(id string) {
	del, err := client.EntityManagementDelete(id)
	if err != nil {
		log.Fatalf("delete failed: %v", err)
	}
	fmt.Printf("Deleted entity id=%s\n", del.ID)
}
```

## Common NRQL snippets

- Drop health checks: `DELETE FROM Log WHERE uri LIKE '%/health%'`
- Drop verbose levels: `DELETE FROM Log WHERE logLevel IN ('DEBUG','TRACE')`

## Links

Pipeline Cloud Rules references: 
- https://docs.newrelic.com/docs/new-relic-control/pipeline-control/cloud-rules-api/
- https://docs.newrelic.com/docs/new-relic-control/pipeline-control/create-pipeline-rules/
