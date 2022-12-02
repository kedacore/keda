package types_convertation

import (
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/dysnix/predictkube-proto/external/proto/enums"
	tspb "github.com/golang/protobuf/ptypes/timestamp"
)

var (
	TimeIsEmptyOrZero = errors.New("time parameter is empty or zero")
)

func AdaptTimeToPbTimestamp(currentTime *time.Time) (*tspb.Timestamp, error) {
	if currentTime != nil && !(*currentTime).IsZero() {
		return timestamppb.New(TimePtrToTime(currentTime)), nil
	}
	return nil, TimeIsEmptyOrZero
}

func AdaptPbTimestampToTime(protoTime *tspb.Timestamp) (*time.Time, error) {
	if protoTime == nil || (protoTime.GetNanos() == 0 || protoTime.GetSeconds() == 0) {
		return nil, fmt.Errorf("proto time parameter is empty or zero")
	}
	return TimeToTimePtr(time.Unix(protoTime.GetSeconds(), int64(protoTime.GetNanos()))), nil
}

var _metrics = map[enums.MetricsType]string{
	enums.MetricsType_Memory:        "memory",
	enums.MetricsType_Cpu:           "cpu",
	enums.MetricsType_Disk:          "disk",
	enums.MetricsType_Network:       "network",
	enums.MetricsType_Nginx:         "nginx",
	enums.MetricsType_Logs:          "logs",
	enums.MetricsType_ReplicasCount: "replicas_count",
}

func GetMetricTypeStr(pb enums.MetricsType) string {
	return _metrics[pb]
}
