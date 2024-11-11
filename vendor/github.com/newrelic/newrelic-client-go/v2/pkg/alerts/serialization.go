package alerts

import (
	"encoding/json"
	"strconv"
)

// UnmarshalJSON is responsible for unmarshaling the ConditionTerm type.
func (c *ConditionTerm) UnmarshalJSON(data []byte) error {
	var v map[string]interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	threshold, err := strconv.ParseFloat(v["threshold"].(string), 64)
	if err != nil {
		return err
	}

	duration, err := strconv.ParseInt(v["duration"].(string), 10, 32)
	if err != nil {
		return err
	}

	c.Threshold = threshold
	c.Duration = int(duration)
	c.Operator = OperatorType(v["operator"].(string))
	c.Priority = PriorityType(v["priority"].(string))
	c.TimeFunction = TimeFunctionType(v["time_function"].(string))

	return nil
}
