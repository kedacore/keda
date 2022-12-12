/*
 * Copyright (c) "Neo4j"
 * Neo4j Sweden AB [https://neo4j.com]
 *
 * This file is part of Neo4j.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package db

// Definitions of these should correspond to public API
type StatementType int

const (
	StatementTypeUnknown     StatementType = 0
	StatementTypeRead        StatementType = 1
	StatementTypeReadWrite   StatementType = 2
	StatementTypeWrite       StatementType = 3
	StatementTypeSchemaWrite StatementType = 4
)

// Counter key names
const (
	NodesCreated         = "nodes-created"
	NodesDeleted         = "nodes-deleted"
	RelationshipsCreated = "relationships-created"
	RelationshipsDeleted = "relationships-deleted"
	PropertiesSet        = "properties-set"
	LabelsAdded          = "labels-added"
	LabelsRemoved        = "labels-removed"
	IndexesAdded         = "indexes-added"
	IndexesRemoved       = "indexes-removed"
	ConstraintsAdded     = "constraints-added"
	ConstraintsRemoved   = "constraints-removed"
	SystemUpdates        = "system-updates"
)

// Plan describes the actual plan that the database planner produced and used (or will use) to execute your statement.
// This can be extremely helpful in understanding what a statement is doing, and how to optimize it. For more details,
// see the Neo4j Manual. The plan for the statement is a tree of plans - each sub-tree containing zero or more child
// plans. The statement starts with the root plan. Each sub-plan is of a specific operator, which describes what
// that part of the plan does - for instance, perform an index lookup or filter results.
// The Neo4j Manual contains a reference of the available operator types, and these may differ across Neo4j versions.
type Plan struct {
	// Operator is the operation this plan is performing.
	Operator string
	// Arguments for the operator.
	// Many operators have arguments defining their specific behavior. This map contains those arguments.
	Arguments map[string]any
	// List of identifiers used by this plan. Identifiers used by this part of the plan.
	// These can be both identifiers introduced by you, or automatically generated.
	Identifiers []string
	// Zero or more child plans. A plan is a tree, where each child is another plan.
	// The children are where this part of the plan gets its input records - unless this is an operator that
	// introduces new records on its own.
	Children []Plan
}

// ProfiledPlan is the same as a regular Plan - except this plan has been executed, meaning it also
// contains detailed information about how much work each step of the plan incurred on the database.
type ProfiledPlan struct {
	// Operator contains the operation this plan is performing.
	Operator string
	// Arguments contains the arguments for the operator used.
	// Many operators have arguments defining their specific behavior. This map contains those arguments.
	Arguments map[string]any
	// Identifiers contains a list of identifiers used by this plan. Identifiers used by this part of the plan.
	// These can be both identifiers introduced by you, or automatically generated.
	Identifiers []string
	// DbHits contains the number of times this part of the plan touched the underlying data stores/
	DbHits int64
	// Records contains the number of records this part of the plan produced.
	Records int64
	// Children contains zero or more child plans. A plan is a tree, where each child is another plan.
	// The children are where this part of the plan gets its input records - unless this is an operator that
	// introduces new records on its own.
	Children          []ProfiledPlan
	PageCacheMisses   int64
	PageCacheHits     int64
	PageCacheHitRatio float64
	Time              int64
}

// Notification represents notifications generated when executing a statement.
// A notification can be visualized in a client pinpointing problems or other information about the statement.
type Notification struct {
	// Code contains a notification code for the discovered issue of this notification.
	Code string
	// Title contains a short summary of this notification.
	Title string
	// Description contains a longer description of this notification.
	Description string
	// Position contains the position in the statement where this notification points to.
	// Not all notifications have a unique position to point to and in that case the position would be set to nil.
	Position *InputPosition
	// Severity contains the severity level of this notification.
	Severity string
}

// InputPosition contains information about a specific position in a statement
type InputPosition struct {
	// Offset contains the character offset referred to by this position; offset numbers start at 0.
	Offset int
	// Line contains the line number referred to by this position; line numbers start at 1.
	Line int
	// Column contains the column number referred to by this position; column numbers start at 1.
	Column int
}

type ProtocolVersion struct {
	Major int
	Minor int
}

type Summary struct {
	Bookmark              string
	StmntType             StatementType
	ServerName            string
	Agent                 string
	Major                 int
	Minor                 int
	Counters              map[string]int
	TFirst                int64
	TLast                 int64
	Plan                  *Plan
	ProfiledPlan          *ProfiledPlan
	Notifications         []Notification
	Database              string
	ContainsSystemUpdates *bool
	ContainsUpdates       *bool
}
