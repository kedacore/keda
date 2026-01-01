package gcp

import (
	"testing"

	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/stretchr/testify/assert"
)

func TestNewPubSubAggregator(t *testing.T) {
	for _, tc := range []struct {
		name        string
		aggregation string
		isError     bool
		errMsg      string
		aligner     monitoringpb.Aggregation_Aligner
		reducer     monitoringpb.Aggregation_Reducer
	}{
		{
			name:        "count aggregation",
			aggregation: "count",
			isError:     false,
			aligner:     monitoringpb.Aggregation_ALIGN_DELTA,
			reducer:     monitoringpb.Aggregation_REDUCE_SUM,
		},
		{
			name:        "sum aggregation",
			aggregation: "sum",
			isError:     false,
			aligner:     monitoringpb.Aggregation_ALIGN_DELTA,
			reducer:     monitoringpb.Aggregation_REDUCE_SUM,
		},
		{
			name:        "mean aggregation",
			aggregation: "mean",
			isError:     false,
			aligner:     monitoringpb.Aggregation_ALIGN_MEAN,
			reducer:     monitoringpb.Aggregation_REDUCE_MEAN,
		},
		{
			name:        "median aggregation",
			aggregation: "median",
			isError:     false,
			aligner:     monitoringpb.Aggregation_ALIGN_PERCENTILE_50,
			reducer:     monitoringpb.Aggregation_REDUCE_PERCENTILE_50,
		},
		{
			name:        "stddev aggregation",
			aggregation: "stddev",
			isError:     false,
			aligner:     monitoringpb.Aggregation_ALIGN_STDDEV,
			reducer:     monitoringpb.Aggregation_REDUCE_STDDEV,
		},
		{
			name:        "variance aggregation (maps to stddev)",
			aggregation: "variance",
			isError:     false,
			aligner:     monitoringpb.Aggregation_ALIGN_STDDEV,
			reducer:     monitoringpb.Aggregation_REDUCE_STDDEV,
		},
		{
			name:        "percentile99 aggregation",
			aggregation: "percentile99",
			isError:     false,
			aligner:     monitoringpb.Aggregation_ALIGN_PERCENTILE_99,
			reducer:     monitoringpb.Aggregation_REDUCE_PERCENTILE_99,
		},
		{
			name:        "percentile95 aggregation",
			aggregation: "percentile95",
			isError:     false,
			aligner:     monitoringpb.Aggregation_ALIGN_PERCENTILE_95,
			reducer:     monitoringpb.Aggregation_REDUCE_PERCENTILE_95,
		},
		{
			name:        "percentile50 aggregation",
			aggregation: "percentile50",
			isError:     false,
			aligner:     monitoringpb.Aggregation_ALIGN_PERCENTILE_50,
			reducer:     monitoringpb.Aggregation_REDUCE_PERCENTILE_50,
		},
		{
			name:        "percentile05 aggregation",
			aggregation: "percentile05",
			isError:     false,
			aligner:     monitoringpb.Aggregation_ALIGN_PERCENTILE_05,
			reducer:     monitoringpb.Aggregation_REDUCE_PERCENTILE_05,
		},
		{
			name:        "invalid percentile",
			aggregation: "percentile101",
			isError:     true,
			errMsg:      "unsupported percentile: 101 (only 99, 95, 50, 05 are supported)",
		},
		{
			name:        "unsupported aggregation function",
			aggregation: "max",
			isError:     true,
			errMsg:      "unsupported aggregation function: max",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			agg, err := NewPubSubAggregator(tc.aggregation)
			if tc.isError {
				assert.Error(t, err)
				assert.Equal(t, tc.errMsg, err.Error())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, agg)
				assert.Equal(t, tc.aligner, agg.PerSeriesAligner)
				assert.Equal(t, tc.reducer, agg.CrossSeriesReducer)
				assert.Equal(t, DefaultPubSubAlignmentPeriod, agg.AlignmentPeriod.Seconds)
			}
		})
	}
}

func TestGetActualProjectID(t *testing.T) {
	// There are three ways to get projectID
	// This is ordered from highest priority to lowest priority
	pidFromMetadata := "myproject0"
	pidFromClient := "myproject1"
	pidFromClientCreds := "myproject2"

	for _, tc := range []struct {
		name      string
		projectID string
		client    *StackDriverClient
		expected  string
	}{
		{
			"all projectID present",
			pidFromMetadata,
			&StackDriverClient{
				projectID: pidFromClient,
				credentials: GoogleApplicationCredentials{
					ProjectID: pidFromClientCreds,
				},
			},
			pidFromMetadata,
		},
		{
			"both projectID from metadata and client present",
			pidFromMetadata,
			&StackDriverClient{
				projectID: pidFromClient,
			},
			pidFromMetadata,
		},
		{
			"both projectID from metadata and client credentials present",
			pidFromMetadata,
			&StackDriverClient{
				credentials: GoogleApplicationCredentials{
					ProjectID: pidFromClientCreds,
				},
			},
			pidFromMetadata,
		},
		{
			"both projectID from client and client credentials present",
			"",
			&StackDriverClient{
				projectID: pidFromClient,
				credentials: GoogleApplicationCredentials{
					ProjectID: pidFromClientCreds,
				},
			},
			pidFromClient,
		},
		{
			"projectID from metadata only",
			pidFromMetadata,
			&StackDriverClient{},
			pidFromMetadata,
		},
		{
			"projectID from client only",
			"",
			&StackDriverClient{
				projectID: pidFromClient,
			},
			pidFromClient,
		},
		{
			"projectID from client credentials only",
			"",
			&StackDriverClient{
				projectID: "",
				credentials: GoogleApplicationCredentials{
					ProjectID: pidFromClientCreds,
				},
			},
			pidFromClientCreds,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pid := getActualProjectID(tc.client, tc.projectID)
			assert.Equal(t, pid, tc.expected)
		})
	}
}
