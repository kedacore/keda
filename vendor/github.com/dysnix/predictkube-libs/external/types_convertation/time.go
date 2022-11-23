package types_convertation

import (
	"errors"
	"strconv"
	"time"

	"github.com/ulikunitz/unixtime"
)

const (
	ZeroDurationErr = "zero time duration"
)

func ParseMillisecondUnixTimestamp(s interface{}) (res time.Time, err error) {
	var ts int64
	switch tmp := s.(type) {
	case string:
		ts, err = strconv.ParseInt(tmp, 10, 64)
		if err != nil {
			return time.Time{}, err
		}
	case int8:
		ts = int64(tmp)
	case int16:
		ts = int64(tmp)
	case int32:
		ts = int64(tmp)
	case int:
		ts = int64(tmp)
	case int64:
		ts = tmp
	case uint:
		ts = int64(tmp)
	case uint8:
		ts = int64(tmp)
	case uint16:
		ts = int64(tmp)
	case uint32:
		ts = int64(tmp)
	case uint64:
		ts = int64(tmp)
	}

	if ts > 0 {
		tmp := unixtime.FromMilli(ts)
		if tmp.IsZero() || tmp.Equal(time.Unix(0, 0)) ||
			tmp.Sub(time.Unix(0, 0)) < time.Hour*8760 {

			return time.Unix(0, 0).Add(time.Duration(ts) * time.Second), nil
		}

		return tmp, nil
	}

	return res, errors.New(ZeroDurationErr)
}
