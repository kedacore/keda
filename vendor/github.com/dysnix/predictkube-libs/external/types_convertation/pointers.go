package types_convertation

import "time"

func TimePtrToTime(t *time.Time) (emptyTime time.Time) {
	if t != nil {
		return *t
	}
	return emptyTime
}

func TimeToTimePtr(t time.Time) *time.Time {
	return &t
}

func String(v string) *string {
	return &v
}

func StringPtr(v *string) string {
	if v != nil {
		return *v
	}
	return ""
}

func Int32(v int32) *int32 {
	return &v
}

func Int32Ptr(v *int32) int32 {
	if v != nil {
		return *v
	}
	return 0
}

func Uint(v uint) *uint {
	return &v
}

func UintPtr(v *uint) uint {
	if v != nil {
		return *v
	}
	return 0
}
