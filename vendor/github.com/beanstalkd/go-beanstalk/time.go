package beanstalk

import (
	"strconv"
	"time"
)

type dur time.Duration

func (d dur) String() string {
	return strconv.FormatInt(int64(time.Duration(d)/time.Second), 10)
}
