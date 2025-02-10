// Copyright Splunk Inc.
// SPDX-License-Identifier: Apache-2.0

package messages

import (
	"encoding/json"
	"time"
)

const (
	JobRunningResolution    = "JOB_RUNNING_RESOLUTION"
	JobDetectedLag          = "JOB_DETECTED_LAG"
	JobInitialMaxDelay      = "JOB_INITIAL_MAX_DELAY"
	FindLimitedResultSet    = "FIND_LIMITED_RESULT_SET"
	FindMatchedNoTimeseries = "FIND_MATCHED_NO_TIMESERIES"
	GroupByMissingProperty  = "GROUPBY_MISSING_PROPERTY"
)

type MessageBlock struct {
	TimestampedMessage
	Code               string `json:"messageCode"`
	Level              string `json:"messageLevel"`
	NumInputTimeseries int    `json:"numInputTimeSeries"`
	// If the messageCode field in the message is known, this will be an
	// instance that has more specific methods to access the known fields.  You
	// can always access the original content by treating this value as a
	// map[string]interface{}.
	Contents    interface{}            `json:"-"`
	ContentsRaw map[string]interface{} `json:"contents"`
}

type InfoMessage struct {
	BaseJSONChannelMessage
	LogicalTimestampMillis uint64 `json:"logicalTimestampMs"`
	MessageBlock           `json:"message"`
}

func (im *InfoMessage) UnmarshalJSON(raw []byte) error {
	type IM InfoMessage
	if err := json.Unmarshal(raw, (*IM)(im)); err != nil {
		return err
	}

	mb := &im.MessageBlock
	switch mb.Code {
	case JobRunningResolution:
		mb.Contents = JobRunningResolutionContents(mb.ContentsRaw)
	case JobDetectedLag:
		mb.Contents = JobDetectedLagContents(mb.ContentsRaw)
	case JobInitialMaxDelay:
		mb.Contents = JobInitialMaxDelayContents(mb.ContentsRaw)
	case FindLimitedResultSet:
		mb.Contents = FindLimitedResultSetContents(mb.ContentsRaw)
	case FindMatchedNoTimeseries:
		mb.Contents = FindMatchedNoTimeseriesContents(mb.ContentsRaw)
	case GroupByMissingProperty:
		mb.Contents = GroupByMissingPropertyContents(mb.ContentsRaw)
	default:
		mb.Contents = mb.ContentsRaw
	}

	return nil
}

func (im *InfoMessage) LogicalTimestamp() time.Time {
	return time.Unix(0, int64(im.LogicalTimestampMillis*uint64(time.Millisecond)))
}

type JobRunningResolutionContents map[string]interface{}

func (jm JobRunningResolutionContents) ResolutionMS() int {
	field, _ := jm["resolutionMs"].(float64)
	return int(field)
}

type JobDetectedLagContents map[string]interface{}

func (jm JobDetectedLagContents) LagMS() int {
	field, _ := jm["lagMs"].(float64)
	return int(field)
}

type JobInitialMaxDelayContents map[string]interface{}

func (jm JobInitialMaxDelayContents) MaxDelayMS() int {
	field, _ := jm["maxDelayMs"].(float64)
	return int(field)
}

type FindLimitedResultSetContents map[string]interface{}

func (jm FindLimitedResultSetContents) MatchedSize() int {
	field, _ := jm["matchedSize"].(float64)
	return int(field)
}

func (jm FindLimitedResultSetContents) LimitSize() int {
	field, _ := jm["limitSize"].(float64)
	return int(field)
}

type FindMatchedNoTimeseriesContents map[string]interface{}

func (jm FindMatchedNoTimeseriesContents) MatchedNoTimeseriesQuery() string {
	field, _ := jm["query"].(string)
	return field
}

type GroupByMissingPropertyContents map[string]interface{}

func (jm GroupByMissingPropertyContents) GroupByMissingProperties() []string {
	propNames := make([]string, len(jm["propertyNames"].([]interface{})))
	for i, v := range jm["propertyNames"].([]interface{}) {
		propNames[i] = v.(string)
	}
	return propNames
}

// ExpiredTSIDMessage is received when a timeseries has expired and is no
// longer relvant to a computation.
type ExpiredTSIDMessage struct {
	BaseJSONChannelMessage
	TSID string `json:"tsId"`
}
