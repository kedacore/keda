package couchdb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-kivik/couchdb/v3/chttp"
	kivik "github.com/go-kivik/kivik/v3"
	"github.com/go-kivik/kivik/v3/driver"
)

type replicationError struct {
	status int
	reason string
}

func (re *replicationError) Error() string {
	return re.reason
}

func (re *replicationError) StatusCode() int {
	return re.status
}

func (re *replicationError) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &re.reason); err != nil {
		return err
	}
	switch (strings.SplitN(re.reason, ":", 2))[0] {
	case "db_not_found":
		re.status = http.StatusNotFound
	case "timeout":
		re.status = http.StatusRequestTimeout
	case "unauthorized":
		re.status = http.StatusUnauthorized
	default:
		re.status = http.StatusInternalServerError
	}
	return nil
}

type replicationStateTime time.Time

func (t *replicationStateTime) UnmarshalJSON(data []byte) error {
	input := string(bytes.Trim(data, `"`))
	if ts, err := time.Parse(time.RFC3339, input); err == nil {
		*t = replicationStateTime(ts)
		return nil
	}
	// Fallback for really old versions of CouchDB
	if seconds, err := strconv.ParseInt(input, 10, 64); err == nil {
		epochTime := replicationStateTime(time.Unix(seconds, 0).UTC())
		*t = epochTime
		return nil
	}
	return &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: fmt.Errorf("kivik: '%s' does not appear to be a valid timestamp", string(data))}
}

type replication struct {
	docID         string
	replicationID string
	source        string
	target        string
	startTime     time.Time
	endTime       time.Time
	state         string
	err           error

	// mu protects the above values
	mu sync.RWMutex

	*db
}

var _ driver.Replication = &replication{}

func (c *client) fetchReplication(ctx context.Context, docID string) *replication {
	rep := c.newReplication(docID)
	rep.db = &db{client: c, dbName: "_replicator"}
	// Do an update to get the initial state, but don't fail if there's an error
	// at this stage, because we successfully created the replication doc.
	_ = rep.updateMain(ctx)
	return rep
}

func (c *client) newReplication(docID string) *replication {
	return &replication{
		docID: docID,
		db: &db{
			client: c,
			dbName: "_replicator",
		},
	}
}

func (r *replication) readLock() func() {
	r.mu.RLock()
	return r.mu.RUnlock
}

func (r *replication) ReplicationID() string { defer r.readLock()(); return r.replicationID }
func (r *replication) Source() string        { defer r.readLock()(); return r.source }
func (r *replication) Target() string        { defer r.readLock()(); return r.target }
func (r *replication) StartTime() time.Time  { defer r.readLock()(); return r.startTime }
func (r *replication) EndTime() time.Time    { defer r.readLock()(); return r.endTime }
func (r *replication) State() string         { defer r.readLock()(); return r.state }
func (r *replication) Err() error            { defer r.readLock()(); return r.err }

func (r *replication) Update(ctx context.Context, state *driver.ReplicationInfo) error {
	if err := r.updateMain(ctx); err != nil {
		return err
	}
	if r.State() == "complete" {
		state.Progress = 100
		return nil
	}
	info, err := r.updateActiveTasks(ctx)
	if err != nil {
		if kivik.StatusCode(err) == http.StatusNotFound {
			// not listed in _active_tasks (because the replication is done, or
			// hasn't yet started), but this isn't an error
			return nil
		}
		return err
	}
	state.DocWriteFailures = info.DocWriteFailures
	state.DocsRead = info.DocsRead
	state.DocsWritten = info.DocsWritten
	// state.progress = info.Progress
	return nil
}

type activeTask struct {
	Type             string `json:"type"`
	ReplicationID    string `json:"replication_id"`
	DocsWritten      int64  `json:"docs_written"`
	DocsRead         int64  `json:"docs_read"`
	DocWriteFailures int64  `json:"doc_write_failures"`
}

func (r *replication) updateActiveTasks(ctx context.Context) (*activeTask, error) {
	resp, err := r.client.DoReq(ctx, http.MethodGet, "/_active_tasks", nil)
	if err != nil {
		return nil, err
	}
	if err = chttp.ResponseError(resp); err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	var tasks []*activeTask
	if err = json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return nil, &kivik.Error{HTTPStatus: http.StatusBadGateway, Err: err}
	}
	for _, task := range tasks {
		if task.Type != "replication" {
			continue
		}
		repIDparts := strings.SplitN(task.ReplicationID, "+", 2)
		if repIDparts[0] != r.replicationID {
			continue
		}
		return task, nil
	}
	return nil, &kivik.Error{HTTPStatus: http.StatusNotFound, Err: errors.New("task not found")}
}

// updateMain updates the "main" fields: those stored directly in r.
func (r *replication) updateMain(ctx context.Context) error {
	doc, err := r.getReplicatorDoc(ctx)
	if err != nil {
		return err
	}
	r.setFromReplicatorDoc(doc)
	return nil
}

func (r *replication) getReplicatorDoc(ctx context.Context) (*replicatorDoc, error) {
	row, err := r.db.Get(ctx, r.docID, nil)
	if err != nil {
		return nil, err
	}
	var doc replicatorDoc
	err = json.NewDecoder(row.Body).Decode(&doc)
	return &doc, err
}

func (r *replication) setFromReplicatorDoc(doc *replicatorDoc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch kivik.ReplicationState(doc.State) {
	case kivik.ReplicationStarted:
		r.startTime = time.Time(doc.StateTime)
	case kivik.ReplicationError, kivik.ReplicationComplete:
		r.endTime = time.Time(doc.StateTime)
	}
	r.state = doc.State
	if doc.Error != nil {
		r.err = doc.Error
	} else {
		r.err = nil
	}
	if r.source == "" {
		r.source = doc.Source
	}
	if r.target == "" {
		r.target = doc.Target
	}
	if r.replicationID == "" {
		r.replicationID = doc.ReplicationID
	}
}

func (r *replication) Delete(ctx context.Context) error {
	_, rev, err := r.GetMeta(ctx, r.docID, nil)
	if err != nil {
		return err
	}
	_, err = r.db.Delete(ctx, r.docID, rev, nil)
	return err
}

type replicatorDoc struct {
	DocID         string               `json:"_id"`
	ReplicationID string               `json:"_replication_id"`
	Source        string               `json:"source"`
	Target        string               `json:"target"`
	State         string               `json:"_replication_state"`
	StateTime     replicationStateTime `json:"_replication_state_time"`
	Error         *replicationError    `json:"_replication_state_reason,omitempty"`
}

func (c *client) GetReplications(ctx context.Context, options map[string]interface{}) ([]driver.Replication, error) {
	scheduler, err := c.schedulerSupported(ctx)
	if err != nil {
		return nil, err
	}
	if scheduler {
		return c.getReplicationsFromScheduler(ctx, options)
	}
	return c.legacyGetReplications(ctx, options)
}

func (c *client) legacyGetReplications(ctx context.Context, options map[string]interface{}) ([]driver.Replication, error) {
	if options == nil {
		options = map[string]interface{}{}
	}
	delete(options, "conflicts")
	delete(options, "update_seq")
	options["include_docs"] = true
	params, err := optionsToParams(options)
	if err != nil {
		return nil, err
	}
	var result struct {
		Rows []struct {
			Doc replicatorDoc `json:"doc"`
		} `json:"rows"`
	}
	path := "/_replicator/_all_docs?" + params.Encode()
	if _, err = c.DoJSON(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	reps := make([]driver.Replication, 0, len(result.Rows))
	for _, row := range result.Rows {
		if row.Doc.DocID == "_design/_replicator" {
			continue
		}
		rep := c.newReplication(row.Doc.DocID)
		rep.setFromReplicatorDoc(&row.Doc)
		reps = append(reps, rep)
	}
	return reps, nil
}

func (c *client) Replicate(ctx context.Context, targetDSN, sourceDSN string, options map[string]interface{}) (driver.Replication, error) {
	if options == nil {
		options = make(map[string]interface{})
	}
	// Allow overriding source and target with options, i.e. for auth options
	if _, ok := options["source"]; !ok {
		options["source"] = sourceDSN
	}
	if _, ok := options["target"]; !ok {
		options["target"] = targetDSN
	}
	if t := options["target"]; t == "" {
		return nil, missingArg("targetDSN")
	}
	if s := options["source"]; s == "" {
		return nil, missingArg("sourceDSN")
	}

	scheduler, err := c.schedulerSupported(ctx)
	if err != nil {
		return nil, err
	}
	opts := &chttp.Options{
		Body: chttp.EncodeBody(options),
	}

	var repStub struct {
		ID string `json:"id"`
	}
	if _, e := c.Client.DoJSON(ctx, http.MethodPost, "/_replicator", opts, &repStub); e != nil {
		return nil, e
	}
	if scheduler {
		return c.fetchSchedulerReplication(ctx, repStub.ID)
	}
	return c.fetchReplication(ctx, repStub.ID), nil
}
