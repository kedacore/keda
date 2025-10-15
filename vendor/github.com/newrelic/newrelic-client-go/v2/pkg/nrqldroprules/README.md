# nrqldroprules

The `nrqldroprules` package provides a Go client for managing legacy NRQL Drop Rules in New Relic. Drop Rules allow you to instruct New Relic to drop (or strip attributes from) incoming telemetry events that match a NRQL pattern. This package lets you create, list, and delete drop rules programmatically.

- Create rules: submit one or more rule definitions.
- List rules: retrieve all existing rules for an account.
- Delete rules: remove rules by their IDs.

## ⚠️ Deprecation & Migration Notice

NRQL Drop Rules are deprecated and scheduled to reach end-of-life on **January 7, 2026**. They are being replaced by **Pipeline Cloud Rules** (see the `pipelinecontrol` package).  
If you currently:
- Use this package to create / delete / list drop rules, or
- Depend on the NRQL-based DROP_DATA or DROP_ATTRIBUTES behaviors,

you should begin migrating to Pipeline Cloud Rules.

See: `github.com/newrelic/newrelic-client-go/v2/pkg/pipelinecontrol`.

## Install

```go
import "github.com/newrelic/newrelic-client-go/v2/pkg/nrqldroprules"
```

## Create a client

```go
package main

import (
	"github.com/newrelic/newrelic-client-go/v2/pkg/config"
	"github.com/newrelic/newrelic-client-go/v2/pkg/nrqldroprules"
)

func main() {
	cfg := config.New()
	cfg.PersonalAPIKey = "YOUR_API_KEY"
	// Optional: cfg.Region = "EU"

	client := nrqldroprules.New(cfg)
	_ = client
}
```

## Key types

- `NRQLDropRulesCreateDropRuleInput`
- `NRQLDropRulesCreateDropRuleResult`
- `NRQLDropRulesDeleteDropRuleResult`
- `NRQLDropRulesListDropRulesResult`
- `NRQLDropRulesDropRule`
- `NRQLDropRulesAction` / `NRQLDropRulesActionTypes`
- `NRQLDropRulesError` / `NRQLDropRulesErrorReasonTypes`

Important action semantics:
- `DROP_DATA`: NRQL must be a `SELECT * FROM <EventType> WHERE ...` style query; all matching data is dropped.
- `DROP_ATTRIBUTES`: Strips specified attributes in the SELECT clause.
- `DROP_ATTRIBUTES_FROM_METRIC_AGGREGATES`: Similar stripping logic but targeted at Metric aggregates.

## Create rules (`NRQLDropRulesCreate`)

Purpose: Submit one or more rules in a single mutation. Mixed success is possible—check both `Successes` and `Failures`.

```go
func createRules(client nrqldroprules.Nrqldroprules, accountID int) {
	inputs := []nrqldroprules.NRQLDropRulesCreateDropRuleInput{
		{
			Description: "Drop debug logs",
			NRQL:        "SELECT * FROM Log WHERE logLevel = 'DEBUG'",
			Action:      nrqldroprules.NRQLDropRulesActionTypes.DROP_DATA,
		},
		{
			Description: "Strip attrs from noisy metrics",
			NRQL:        "SELECT foo,bar FROM Metric WHERE metricName LIKE 'temp.%'",
			Action:      nrqldroprules.NRQLDropRulesActionTypes.DROP_ATTRIBUTES_FROM_METRIC_AGGREGATES,
		},
	}

	res, err := client.NRQLDropRulesCreate(accountID, inputs)
	if err != nil {
		panic(err)
	}

	for _, s := range res.Successes {
		// s.PipelineCloudRuleEntityId may be present if the backend linked this drop rule
		println("Created rule ID:", s.ID, "NRQL:", s.NRQL)
	}
	for _, f := range res.Failures {
		println("Failed rule:", f.Submitted.Description, "Reason:", string(f.Error.Reason))
	}
}
```

Minimal single-rule example:

```go
res, _ := client.NRQLDropRulesCreate(accountID, []nrqldroprules.NRQLDropRulesCreateDropRuleInput{
	{
		Description: "Drop health checks",
		NRQL:        "SELECT * FROM Log WHERE request LIKE '%/health%'",
		Action:      nrqldroprules.NRQLDropRulesActionTypes.DROP_DATA,
	},
})
```

## List rules (`GetList`)

Purpose: Retrieve all drop rules for an account. Use this to audit existing rules or plan migration to Pipeline Cloud Rules.

```go
func listRules(client nrqldroprules.Nrqldroprules, accountID int) {
	out, err := client.GetList(accountID)
	if err != nil {
		panic(err)
	}
	if out.Error.Description != "" {
		println("List warning:", out.Error.Description)
	}
	for _, r := range out.Rules {
		println("Rule:", r.ID, r.Description, "Action:", string(r.Action))
		if r.PipelineCloudRuleEntityId != "" {
			println(" Linked PipelineCloudRuleEntityId:", r.PipelineCloudRuleEntityId)
		}
	}
}
```

## Delete rules (`NRQLDropRulesDelete`)

Purpose: Bulk delete rules by ID. Mixed success possible, identical shape to create responses.

```go
func deleteRules(client nrqldroprules.Nrqldroprules, accountID int, ids []string) {
	res, err := client.NRQLDropRulesDelete(accountID, ids)
	if err != nil {
		panic(err)
	}
	for _, s := range res.Successes {
		println("Deleted:", s.ID)
	}
	for _, f := range res.Failures {
		println("Delete failed:", f.Submitted.RuleId, "Reason:", string(f.Error.Reason))
	}
}
```

## End-to-end example (create → list → delete)

```go
func exampleFlow(client nrqldroprules.Nrqldroprules, accountID int) {
	create := []nrqldroprules.NRQLDropRulesCreateDropRuleInput{
		{
			Description: "Temporary test drop",
			NRQL:        "SELECT * FROM Log WHERE message LIKE '%temp-test%'",
			Action:      nrqldroprules.NRQLDropRulesActionTypes.DROP_DATA,
		},
	}
	cr, err := client.NRQLDropRulesCreate(accountID, create)
	if err != nil || len(cr.Successes) == 0 {
		panic("create failed")
	}
	id := cr.Successes[0].ID

	list, _ := client.GetList(accountID)
	println("Rule count:", len(list.Rules))

	_, _ = client.NRQLDropRulesDelete(accountID, []string{id})
}
```

## Context usage

All operations have `WithContext` variants:

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

res, err := client.NRQLDropRulesCreateWithContext(ctx, accountID, []nrqldroprules.NRQLDropRulesCreateDropRuleInput{
	{
		Description: "Ctx example",
		NRQL:        "SELECT * FROM Log WHERE hostname = 'example-host'",
		Action:      nrqldroprules.NRQLDropRulesActionTypes.DROP_DATA,
	},
})
if err != nil {
	if ctx.Err() != nil {
		println("timed out")
	}
	panic(err)
}
println("Created:", len(res.Successes))
```

## Error handling patterns

Failures are returned per submitted rule—do not rely solely on `error`:

```go
res, err := client.NRQLDropRulesCreate(accountID, inputs)
if err != nil {
	// transport / GraphQL level error
	panic(err)
}
for _, f := range res.Failures {
	switch f.Error.Reason {
	case nrqldroprules.NRQLDropRulesErrorReasonTypes.INVALID_QUERY:
		println("Invalid NRQL:", f.Submitted.NRQL)
	case nrqldroprules.NRQLDropRulesErrorReasonTypes.USER_NOT_AUTHORIZED:
		println("Permission issue")
	}
}
```

## Field notes

- `PipelineCloudRuleEntityId`: May link a drop rule to an underlying Pipeline Cloud Rule (migration / backend linkage), when Drop Rules are fetched using the list function described above.
- `Action`: Ensure you supply `SELECT *` form when using `DROP_DATA`. Non-compliant queries will fail with `INVALID_QUERY`.
- Mixed outcomes: Both create and delete operations may partially succeed—always inspect both arrays.

## Migration hints (Drop Rules → Pipeline Cloud Rules)

Example mapping for a DROP_DATA rule:

```go
// Original drop rule input
nrqlDrop := nrqldroprules.NRQLDropRulesCreateDropRuleInput{
	Description: "Drop debug logs",
	NRQL:        "SELECT * FROM Log WHERE logLevel = 'DEBUG'",
	Action:      nrqldroprules.NRQLDropRulesActionTypes.DROP_DATA,
}

// Equivalent Pipeline Cloud Rule (DELETE form)
pipelineInput := pipelinecontrol.EntityManagementPipelineCloudRuleEntityCreateInput{
	Name:        "drop-debug-logs",
	Description: nrqlDrop.Description,
	NRQL:        nrdb.NRQL("DELETE FROM Log WHERE logLevel = 'DEBUG'"),
	Scope: pipelinecontrol.EntityManagementScopedReferenceInput{
		Type: pipelinecontrol.EntityManagementEntityScopeTypes.ACCOUNT,
		ID:   fmt.Sprintf("%d", accountID),
	},
}
```
## Links

- Pipeline Cloud Rules (target after migration): https://docs.newrelic.com/docs/new-relic-control/pipeline-control/cloud-rules-api/
- Drop Rules (legacy): https://docs.newrelic.com/docs/logs/ui-data/drop-data-drop-filter-rules/
- NRQL reference: https://docs.newrelic.com/docs/query-your-data/nrql-new-relic-query-language/

---
Migration strongly recommended: begin transitioning automated workflows to the `pipelinecontrol` package to ensure continuity after the NRQL Drop Rules EOL.
