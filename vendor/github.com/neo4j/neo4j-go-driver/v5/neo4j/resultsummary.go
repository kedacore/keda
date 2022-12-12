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

package neo4j

import (
	"fmt"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/db"
	"time"
)

// StatementType defines the type of the statement
type StatementType int

const (
	// StatementTypeUnknown identifies an unknown statement type
	StatementTypeUnknown StatementType = 0
	// StatementTypeReadOnly identifies a read-only statement
	StatementTypeReadOnly StatementType = 1
	// StatementTypeReadWrite identifies a read-write statement
	StatementTypeReadWrite StatementType = 2
	// StatementTypeWriteOnly identifies a write-only statement
	StatementTypeWriteOnly StatementType = 3
	// StatementTypeSchemaWrite identifies a schema-write statement
	StatementTypeSchemaWrite StatementType = 4
)

func (st StatementType) String() string {
	switch st {
	case StatementTypeReadOnly:
		return "r"
	case StatementTypeReadWrite:
		return "rw"
	case StatementTypeWriteOnly:
		return "w"
	case StatementTypeSchemaWrite:
		return "s"
	default:
		return ""
	}
}

type ResultSummary interface {
	// Server returns basic information about the server where the statement is carried out.
	Server() ServerInfo
	// Query returns the query that has been executed.
	Query() Query
	// StatementType returns type of statement that has been executed.
	StatementType() StatementType
	// Counters returns statistics counts for the statement.
	Counters() Counters
	// Plan returns statement plan for the executed statement if available, otherwise null.
	Plan() Plan
	// Profile returns profiled statement plan for the executed statement if available, otherwise null.
	Profile() ProfiledPlan
	// Notifications returns a slice of notifications produced while executing the statement.
	// The list will be empty if no notifications produced while executing the statement.
	Notifications() []Notification
	// ResultAvailableAfter returns the time it took for the server to make the result available for consumption.
	// Since 5.0, this returns a negative duration if the server has not sent the corresponding statistic.
	ResultAvailableAfter() time.Duration
	// ResultConsumedAfter returns the time it took the server to consume the result.
	// Since 5.0, this returns a negative duration if the server has not sent the corresponding statistic.
	ResultConsumedAfter() time.Duration
	// Database returns information about the database where the result is obtained from
	// Returns nil for Neo4j versions prior to v4.
	// Returns the default "neo4j" database for Community Edition servers.
	Database() DatabaseInfo
}

// Counters contains statistics about the changes made to the database made as part
// of the statement execution.
type Counters interface {
	// Whether there were any updates at all, eg. any of the counters are greater than 0.
	ContainsUpdates() bool
	// The number of nodes created.
	NodesCreated() int
	// The number of nodes deleted.
	NodesDeleted() int
	// The number of relationships created.
	RelationshipsCreated() int
	// The number of relationships deleted.
	RelationshipsDeleted() int
	PropertiesSet() int
	// The number of labels added to nodes.
	LabelsAdded() int
	// The number of labels removed from nodes.
	LabelsRemoved() int
	// The number of indexes added to the schema.
	IndexesAdded() int
	// The number of indexes removed from the schema.
	IndexesRemoved() int
	// The number of constraints added to the schema.
	ConstraintsAdded() int
	// The number of constraints removed from the schema.
	ConstraintsRemoved() int
	// The number of system updates
	SystemUpdates() int
	// ContainsSystemUpdates indicates whether the query contains system updates
	ContainsSystemUpdates() bool
}

type Statement interface {
	Query
}

type Query interface {
	// Text returns the statement's text.
	Text() string
	// Parameters returns the statement's parameters.
	Parameters() map[string]any
}

// ServerInfo contains basic information of the server.
type ServerInfo interface {
	// Address returns the address of the server.
	Address() string
	Agent() string
	ProtocolVersion() db.ProtocolVersion
}

type simpleServerInfo struct {
	address         string
	agent           string
	protocolVersion db.ProtocolVersion
}

func (s simpleServerInfo) Address() string {
	return s.address
}

func (s simpleServerInfo) Agent() string {
	return s.agent
}

func (s simpleServerInfo) ProtocolVersion() db.ProtocolVersion {
	return s.protocolVersion
}

// DatabaseInfo contains basic information of the database the query result has been obtained from.
type DatabaseInfo interface {
	Name() string
}

// Plan describes the actual plan that the database planner produced and used (or will use) to execute your statement.
// This can be extremely helpful in understanding what a statement is doing, and how to optimize it. For more details,
// see the Neo4j Manual. The plan for the statement is a tree of plans - each sub-tree containing zero or more child
// plans. The statement starts with the root plan. Each sub-plan is of a specific operator, which describes what
// that part of the plan does - for instance, perform an index lookup or filter results.
// The Neo4j Manual contains a reference of the available operator types, and these may differ across Neo4j versions.
type Plan interface {
	// Operator returns the operation this plan is performing.
	Operator() string
	// Arguments returns the arguments for the operator used.
	// Many operators have arguments defining their specific behavior. This map contains those arguments.
	Arguments() map[string]any
	// Identifiers returns a list of identifiers used by this plan. Identifiers used by this part of the plan.
	// These can be both identifiers introduced by you, or automatically generated.
	Identifiers() []string
	// Children returns zero or more child plans. A plan is a tree, where each child is another plan.
	// The children are where this part of the plan gets its input records - unless this is an operator that
	// introduces new records on its own.
	Children() []Plan
}

// ProfiledPlan is the same as a regular Plan - except this plan has been executed, meaning it also
// contains detailed information about how much work each step of the plan incurred on the database.
type ProfiledPlan interface {
	// Operator returns the operation this plan is performing.
	Operator() string
	// Arguments returns the arguments for the operator used.
	// Many operators have arguments defining their specific behavior. This map contains those arguments.
	Arguments() map[string]any
	// Identifiers returns a list of identifiers used by this plan. Identifiers used by this part of the plan.
	// These can be both identifiers introduced by you, or automatically generated.
	Identifiers() []string
	// DbHits returns the number of times this part of the plan touched the underlying data stores/
	DbHits() int64
	// Records returns the number of records this part of the plan produced.
	Records() int64
	// Children returns zero or more child plans. A plan is a tree, where each child is another plan.
	// The children are where this part of the plan gets its input records - unless this is an operator that
	// introduces new records on its own.
	Children() []ProfiledPlan
	PageCacheMisses() int64
	PageCacheHits() int64
	PageCacheHitRatio() float64
	Time() int64
}

// Notification represents notifications generated when executing a statement.
// A notification can be visualized in a client pinpointing problems or other information about the statement.
type Notification interface {
	// Code returns a notification code for the discovered issue of this notification.
	Code() string
	// Title returns a short summary of this notification.
	Title() string
	// Description returns a longer description of this notification.
	Description() string
	// Position returns the position in the statement where this notification points to.
	// Not all notifications have a unique position to point to and in that case the position would be set to nil.
	Position() InputPosition
	// Severity returns the severity level of this notification.
	Severity() string
}

// InputPosition contains information about a specific position in a statement
type InputPosition interface {
	// Offset returns the character offset referred to by this position; offset numbers start at 0.
	Offset() int
	// Line returns the line number referred to by this position; line numbers start at 1.
	Line() int
	// Column returns the column number referred to by this position; column numbers start at 1.
	Column() int
}

type resultSummary struct {
	sum    *db.Summary
	cypher string
	params map[string]any
}

func (s *resultSummary) Agent() string {
	return s.sum.Agent
}

func (s *resultSummary) ProtocolVersion() db.ProtocolVersion {
	return db.ProtocolVersion{
		Major: s.sum.Major,
		Minor: s.sum.Minor,
	}
}

func (s *resultSummary) Server() ServerInfo {
	return s
}

func (s *resultSummary) Address() string {
	return s.sum.ServerName
}

func (s *resultSummary) Query() Query {
	return s
}

func (s *resultSummary) StatementType() StatementType {
	return StatementType(s.sum.StmntType)
}

func (s *resultSummary) Text() string {
	return s.cypher
}

func (s *resultSummary) Parameters() map[string]any {
	return s.params
}

func (s *resultSummary) Counters() Counters {
	return s
}

func (s *resultSummary) ContainsUpdates() bool {
	if s.sum.ContainsUpdates != nil {
		return *s.sum.ContainsUpdates
	}
	return s.ConstraintsAdded() > 0 || s.ConstraintsRemoved() > 0 ||
		s.IndexesAdded() > 0 || s.IndexesRemoved() > 0 ||
		s.NodesCreated() > 0 || s.NodesDeleted() > 0 ||
		s.RelationshipsCreated() > 0 || s.RelationshipsDeleted() > 0 ||
		s.LabelsAdded() > 0 || s.LabelsRemoved() > 0 ||
		s.PropertiesSet() > 0
}

func (s *resultSummary) getCounter(n string) int {
	if s.sum.Counters == nil {
		return 0
	}
	return s.sum.Counters[n]
}

func (s *resultSummary) SystemUpdates() int {
	return s.getCounter(db.SystemUpdates)
}

func (s *resultSummary) ContainsSystemUpdates() bool {
	if s.sum.ContainsSystemUpdates != nil {
		return *s.sum.ContainsSystemUpdates
	}
	return s.getCounter(db.SystemUpdates) > 0
}

func (s *resultSummary) NodesCreated() int {
	return s.getCounter(db.NodesCreated)
}

func (s *resultSummary) NodesDeleted() int {
	return s.getCounter(db.NodesDeleted)
}

func (s *resultSummary) RelationshipsCreated() int {
	return s.getCounter(db.RelationshipsCreated)
}

func (s *resultSummary) RelationshipsDeleted() int {
	return s.getCounter(db.RelationshipsDeleted)
}

func (s *resultSummary) PropertiesSet() int {
	return s.getCounter(db.PropertiesSet)
}

func (s *resultSummary) LabelsAdded() int {
	return s.getCounter(db.LabelsAdded)
}

func (s *resultSummary) LabelsRemoved() int {
	return s.getCounter(db.LabelsRemoved)
}

func (s *resultSummary) IndexesAdded() int {
	return s.getCounter(db.IndexesAdded)
}

func (s *resultSummary) IndexesRemoved() int {
	return s.getCounter(db.IndexesRemoved)
}

func (s *resultSummary) ConstraintsAdded() int {
	return s.getCounter(db.ConstraintsAdded)
}

func (s *resultSummary) ConstraintsRemoved() int {
	return s.getCounter(db.ConstraintsRemoved)
}

func (s *resultSummary) ResultAvailableAfter() time.Duration {
	return time.Duration(s.sum.TFirst) * time.Millisecond
}

func (s *resultSummary) ResultConsumedAfter() time.Duration {
	return time.Duration(s.sum.TLast) * time.Millisecond
}

func (s *resultSummary) Plan() Plan {
	if s.sum.Plan == nil {
		return nil
	}
	return &plan{plan: s.sum.Plan}
}

func (s *resultSummary) Database() DatabaseInfo {
	database := s.sum.Database
	if database == "" {
		return nil
	}
	return &databaseInfo{name: database}
}

type databaseInfo struct {
	name string
}

func (d *databaseInfo) Name() string {
	return d.name
}

type plan struct {
	plan *db.Plan
}

func (p *plan) Operator() string {
	return p.plan.Operator
}

func (p *plan) Arguments() map[string]any {
	return p.plan.Arguments
}

func (p *plan) Identifiers() []string {
	return p.plan.Identifiers
}

func (p *plan) Children() []Plan {
	children := make([]Plan, len(p.plan.Children))
	for i, c := range p.plan.Children {
		children[i] = &plan{plan: &c}
	}
	return children
}

func (s *resultSummary) Profile() ProfiledPlan {
	if s.sum.ProfiledPlan == nil {
		return nil
	}
	return &profile{profile: s.sum.ProfiledPlan}
}

type profile struct {
	profile *db.ProfiledPlan
}

func (p *profile) String() string {
	return fmt.Sprintf("%v", *p.profile)
}

func (p *profile) Operator() string {
	return p.profile.Operator
}

func (p *profile) Arguments() map[string]any {
	return p.profile.Arguments
}

func (p *profile) Identifiers() []string {
	return p.profile.Identifiers
}

func (p *profile) DbHits() int64 {
	return p.profile.DbHits
}

func (p *profile) Records() int64 {
	return p.profile.Records
}

func (p *profile) Children() []ProfiledPlan {
	children := make([]ProfiledPlan, len(p.profile.Children))
	for i, c := range p.profile.Children {
		child := c
		children[i] = &profile{profile: &child}
	}
	return children
}

func (p *profile) PageCacheMisses() int64 {
	return p.profile.PageCacheMisses
}

func (p *profile) PageCacheHits() int64 {
	return p.profile.PageCacheHits
}

func (p *profile) PageCacheHitRatio() float64 {
	return p.profile.PageCacheHitRatio
}

func (p *profile) Time() int64 {
	return p.profile.Time
}

func (s *resultSummary) Notifications() []Notification {
	if s.sum.Notifications == nil {
		return nil
	}
	notifications := make([]Notification, len(s.sum.Notifications))
	for i := range s.sum.Notifications {
		notifications[i] = &notification{notification: &s.sum.Notifications[i]}
	}
	return notifications
}

type notification struct {
	notification *db.Notification
}

func (n *notification) Code() string {
	return n.notification.Code
}

func (n *notification) Title() string {
	return n.notification.Title
}

func (n *notification) Description() string {
	return n.notification.Description
}

func (n *notification) Severity() string {
	return n.notification.Severity
}

func (n *notification) Position() InputPosition {
	if n.notification.Position == nil {
		return nil
	}
	return n
}

func (n *notification) Offset() int {
	return n.notification.Position.Offset
}
func (n *notification) Column() int {
	return n.notification.Position.Column
}
func (n *notification) Line() int {
	return n.notification.Position.Line
}
