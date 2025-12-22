// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.
package eh

// ConvertToInt64 converts any int-like value to be an int64.
func ConvertToInt64(intValue any) (int64, bool) {
	switch v := intValue.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return int64(v), true
	}

	return 0, false
}
