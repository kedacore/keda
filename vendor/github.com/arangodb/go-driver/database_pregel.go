//
// DISCLAIMER
//
// Copyright 2023-2025 ArangoDB GmbH, Cologne, Germany
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

import (
	"context"
	"time"
)

// Deprecated: It will be removed in version 3.12
//
// DatabasePregels provides access to all Pregel Jobs in a single database.
type DatabasePregels interface {
	// StartJob - Start the execution of a Pregel algorithm
	StartJob(ctx context.Context, options PregelJobOptions) (string, error)
	// GetJob - Get the status of a Pregel execution
	GetJob(ctx context.Context, id string) (*PregelJob, error)
	// GetJobs - Returns a list of currently running and recently finished Pregel jobs without retrieving their results.
	GetJobs(ctx context.Context) ([]*PregelJob, error)
	// CancelJob - Cancel an ongoing Pregel execution
	CancelJob(ctx context.Context, id string) error
}

type PregelAlgorithm string

const (
	PregelAlgorithmPageRank                        PregelAlgorithm = "pagerank"
	PregelAlgorithmSingleSourceShortestPath        PregelAlgorithm = "sssp"
	PregelAlgorithmConnectedComponents             PregelAlgorithm = "connectedcomponents"
	PregelAlgorithmWeaklyConnectedComponents       PregelAlgorithm = "wcc"
	PregelAlgorithmStronglyConnectedComponents     PregelAlgorithm = "scc"
	PregelAlgorithmHyperlinkInducedTopicSearch     PregelAlgorithm = "hits"
	PregelAlgorithmEffectiveCloseness              PregelAlgorithm = "effectivecloseness"
	PregelAlgorithmLineRank                        PregelAlgorithm = "linerank"
	PregelAlgorithmLabelPropagation                PregelAlgorithm = "labelpropagation"
	PregelAlgorithmSpeakerListenerLabelPropagation PregelAlgorithm = "slpa"
)

type PregelJobOptions struct {
	// Name of the algorithm
	Algorithm PregelAlgorithm `json:"algorithm"`
	// Name of a graph. Either this or the parameters VertexCollections and EdgeCollections are required.
	// Please note that there are special sharding requirements for graphs in order to be used with Pregel.
	GraphName string `json:"graphName,omitempty"`
	// List of vertex collection names. Please note that there are special sharding requirements for collections in order to be used with Pregel.
	VertexCollections []string `json:"vertexCollections,omitempty"`
	// List of edge collection names. Please note that there are special sharding requirements for collections in order to be used with Pregel.
	EdgeCollections []string `json:"edgeCollections,omitempty"`
	// General as well as algorithm-specific options.
	Params map[string]interface{} `json:"params,omitempty"`
}

type PregelJobState string

const (
	// PregelJobStateNone - The Pregel run did not yet start.
	PregelJobStateNone PregelJobState = "none"
	// PregelJobStateLoading - The graph is loaded from the database into memory before the execution of the algorithm.
	PregelJobStateLoading PregelJobState = "loading"
	// PregelJobStateRunning - The algorithm is executing normally.
	PregelJobStateRunning PregelJobState = "running"
	// PregelJobStateStoring - The algorithm finished, but the results are still being written back into the collections. Occurs only if the store parameter is set to true.
	PregelJobStateStoring PregelJobState = "storing"
	// PregelJobStateDone  - The execution is done. In version 3.7.1 and later, this means that storing is also done.
	// In earlier versions, the results may not be written back into the collections yet. This event is announced in the server log (requires at least info log level for the pregel log topic).
	PregelJobStateDone PregelJobState = "done"
	// PregelJobStateCanceled  - The execution was permanently canceled, either by the user or by an error.
	PregelJobStateCanceled PregelJobState = "canceled"
	// PregelJobStateFatalError - The execution has failed and cannot recover.
	PregelJobStateFatalError PregelJobState = "fatal error"
	// PregelJobStateInError - The execution is in an error state. This can be caused by DB-Servers being not reachable or being non-responsive.
	// The execution might recover later, or switch to "canceled" if it was not able to recover successfully.
	PregelJobStateInError PregelJobState = "in error"
	// PregelJobStateRecovering - (currently unused): The execution is actively recovering and switches back to running if the recovery is successful.
	PregelJobStateRecovering PregelJobState = "recovering"
)

type PregelJob struct {
	// The ID of the Pregel job, as a string.
	ID string `json:"id"`
	// The algorithm used by the job.
	Algorithm PregelAlgorithm `json:"algorithm,omitempty"`
	// The date and time when the job was created.
	Created time.Time `json:"created,omitempty"`
	// The date and time when the job results expire.
	// The expiration date is only meaningful for jobs that were completed, canceled or resulted in an error.
	// Such jobs are cleaned up by the garbage collection when they reach their expiration date/time.
	Started time.Time `json:"started,omitempty"`
	// The TTL (time to live) value for the job results, specified in seconds. The TTL is used to calculate the expiration date for the job’s results.
	TTL uint64 `json:"ttl,omitempty"`
	// The state of the execution.
	State PregelJobState `json:"state,omitempty"`
	// The number of global supersteps executed.
	Gss uint64 `json:"gss,omitempty"`
	// The total runtime of the execution up to now (if the execution is still ongoing).
	TotalRuntime float64 `json:"totalRuntime,omitempty"`
	// The startup runtime of the execution. The startup time includes the data loading time and can be substantial.
	StartupTime float64 `json:"startupTime,omitempty"`
	// The algorithm execution time. Is shown when the computation started.
	ComputationTime float64 `json:"computationTime,omitempty"`
	// The time for storing the results if the job includes results storage. Is shown when the storing started.
	StorageTime float64 `json:"storageTime,omitempty"`
	// Computation time of each global super step. Is shown when the computation started.
	GSSTimes []float64 `json:"gssTimes,omitempty"`
	// This attribute is used by Programmable Pregel Algorithms (air, experimental). The value is only populated once the algorithm has finished.
	Reports []map[string]interface{} `json:"reports,omitempty"`
	// The total number of vertices processed.
	VertexCount uint64 `json:"vertexCount,omitempty"`
	// The total number of edges processed.
	EdgeCount uint64 `json:"edgeCount,omitempty"`
	// UseMemoryMaps
	UseMemoryMaps *bool `json:"useMemoryMaps,omitempty"`
	// The Pregel run details.
	// Available from 3.10 arangod version.
	Detail *PregelRunDetails `json:"detail,omitempty"`
}

// PregelRunDetails - The Pregel run details.
// Available from 3.10 arangod version.
type PregelRunDetails struct {
	// The aggregated details of the full Pregel run. The values are totals of all the DB-Server.
	AggregatedStatus *AggregatedStatus `json:"aggregatedStatus,omitempty"`
	// The details of the Pregel for every DB-Server. Each object key is a DB-Server ID, and each value is a nested object similar to the aggregatedStatus attribute.
	// In a single server deployment, there is only a single entry with an empty string as key.
	WorkerStatus map[string]*AggregatedStatus `json:"workerStatus,omitempty"`
}

// AggregatedStatus The aggregated details of the full Pregel run. The values are totals of all the DB-Server.
type AggregatedStatus struct {
	// The time at which the status was measured.
	TimeStamp time.Time `json:"timeStamp,omitempty"`
	// The status of the in memory graph.
	GraphStoreStatus *GraphStoreStatus `json:"graphStoreStatus,omitempty"`
	//  Information about the global supersteps.
	AllGSSStatus *AllGSSStatus `json:"allGssStatus,omitempty"`
}

// GraphStoreStatus The status of the in memory graph.
type GraphStoreStatus struct {
	// The number of vertices that are loaded from the database into memory.
	VerticesLoaded uint64 `json:"verticesLoaded,omitempty"`
	// The number of edges that are loaded from the database into memory.
	EdgesLoaded uint64 `json:"edgesLoaded,omitempty"`
	// The number of bytes used in-memory for the loaded graph.
	MemoryBytesUsed uint64 `json:"memoryBytesUsed,omitempty"`
	// The number of vertices that are written back to the database after the Pregel computation finished. It is only set if the store parameter is set to true.
	VerticesStored uint64 `json:"verticesStored,omitempty"`
}

// AllGSSStatus Information about the global supersteps.
type AllGSSStatus struct {
	//  A list of objects with details for each global superstep.
	Items []GSSStatus `json:"items,omitempty"`
}

// GSSStatus Information about the global superstep
type GSSStatus struct {
	// The number of vertices that have been processed in this step.
	VerticesProcessed uint64 `json:"verticesProcessed,omitempty"`
	// The number of messages sent in this step.
	MessagesSent uint64 `json:"messagesSent,omitempty"`
	// The number of messages received in this step.
	MessagesReceived uint64 `json:"messagesReceived,omitempty"`
	// The number of bytes used in memory for the messages in this step.
	MemoryBytesUsedForMessages uint64 `json:"memoryBytesUsedForMessages,omitempty"`
}
