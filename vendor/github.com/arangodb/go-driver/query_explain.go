//
// DISCLAIMER
//
// Copyright 2023 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//

package driver

type ExplainQueryOptimizerOptions struct {
	// A list of to-be-included or to-be-excluded optimizer rules can be put into this attribute,
	// telling the optimizer to include or exclude specific rules.
	//  To disable a rule, prefix its name with a "-", to enable a rule, prefix it with a "+".
	// There is also a pseudo-rule "all", which matches all optimizer rules. "-all" disables all rules.
	Rules []string `json:"rules,omitempty"`
}

type ExplainQueryOptions struct {
	// If set to true, all possible execution plans will be returned.
	// The default is false, meaning only the optimal plan will be returned.
	AllPlans bool `json:"allPlans,omitempty"`

	// An optional maximum number of plans that the optimizer is allowed to generate.
	// Setting this attribute to a low value allows to put a cap on the amount of work the optimizer does.
	MaxNumberOfPlans *int `json:"maxNumberOfPlans,omitempty"`

	// Options related to the query optimizer.
	Optimizer ExplainQueryOptimizerOptions `json:"optimizer,omitempty"`
}

type ExplainQueryResultExecutionNodeRaw map[string]interface{}
type ExplainQueryResultExecutionCollection cursorPlanCollection
type ExplainQueryResultExecutionVariable cursorPlanVariable

type ExplainQueryResultPlan struct {
	// Execution nodes of the plan.
	NodesRaw []ExplainQueryResultExecutionNodeRaw `json:"nodes,omitempty"`
	// List of rules the optimizer applied
	Rules []string `json:"rules,omitempty"`
	// List of collections used in the query
	Collections []ExplainQueryResultExecutionCollection `json:"collections,omitempty"`
	// List of variables used in the query (note: this may contain internal variables created by the optimizer)
	Variables []ExplainQueryResultExecutionVariable `json:"variables,omitempty"`
	// The total estimated cost for the plan. If there are multiple plans, the optimizer will choose the plan with the lowest total cost
	EstimatedCost float64 `json:"estimatedCost,omitempty"`
	// The estimated number of results.
	EstimatedNrItems int `json:"estimatedNrItems,omitempty"`
}

type ExplainQueryResultExecutionStats struct {
	RulesExecuted   int     `json:"rulesExecuted,omitempty"`
	RulesSkipped    int     `json:"rulesSkipped,omitempty"`
	PlansCreated    int     `json:"plansCreated,omitempty"`
	PeakMemoryUsage uint64  `json:"peakMemoryUsage,omitempty"`
	ExecutionTime   float64 `json:"executionTime,omitempty"`
}

type ExplainQueryResult struct {
	Plan  ExplainQueryResultPlan   `json:"plan,omitempty"`
	Plans []ExplainQueryResultPlan `json:"plans,omitempty"`
	// List of warnings that occurred during optimization or execution plan creation
	Warnings []string `json:"warnings,omitempty"`
	// Info about optimizer statistics
	Stats ExplainQueryResultExecutionStats `json:"stats,omitempty"`
	// Cacheable states whether the query results can be cached on the server if the query result cache were used.
	// This attribute is not present when allPlans is set to true.
	Cacheable *bool `json:"cacheable,omitempty"`
}
